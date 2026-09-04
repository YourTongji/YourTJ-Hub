package routes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jwtopt "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imConversations"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userSessions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// csrfOrderContractRouter assembles the production apiRoute CSRF-first chain
// (issue #406 follow-up: CSRFProtection runs before JWTAuthCheck in every
// cookie-write group) on the shared contract test DB, migrating the extra
// tables the exercised real handlers need.
func csrfOrderContractRouter(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, _ := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&eventNotification.Entity{},
		&imConversations.Entity{},
		&imUserChatConfigs.Entity{},
		&messages.Entity{},
		&rolePermissionRs.Entity{},
	); err != nil {
		t.Fatalf("migrate CSRF order contract tables: %v", err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiRoute(router)
	return conn, router
}

// csrfOrderRequest sends a cookie-authenticated request with an optional
// Origin header against the production assembly.
func csrfOrderRequest(t *testing.T, router http.Handler, path, body, token, origin string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "http://forum.example.test"+path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

// contractNearExpirySessionToken mints a valid session whose JWT and
// user_sessions row both expire in ttl. An authenticated request with such a
// token refreshes it (rotates the cookie via TokenSetting and extends the row
// via sessionservice.TouchExpiry), which makes the row a detector for
// authentication side effects on rejected requests.
func contractNearExpirySessionToken(t *testing.T, user *users.EntityComplete, ttl time.Duration) (token, jti string) {
	t.Helper()
	var err error
	jti, err = jwtopt.GenerateJti()
	if err != nil {
		t.Fatalf("generate near-expiry jti: %v", err)
	}
	token, err = jwtopt.Std().CreateToken(jwtopt.CustomClaims{
		UserId:           user.Id,
		TokenVersion:     user.TokenVersion,
		Jti:              jti,
		RegisteredClaims: jwtopt.GetBaseRegisteredClaims(ttl),
	})
	if err != nil {
		t.Fatalf("create near-expiry token: %v", err)
	}
	if err := sessionservice.Create(user.Id, jti, "contract-test", "127.0.0.1"); err != nil {
		t.Fatalf("create session row: %v", err)
	}
	if err := userSessions.UpdateExpiresAtByJti(jti, time.Now().Add(ttl)); err != nil {
		t.Fatalf("align session row expiry: %v", err)
	}
	return token, jti
}

// TestCSRFOrderRejectedCrossSitePostDoesNotRefreshSession is the P2
// regression: CSRFProtection now runs before JWTAuthCheck, so a cross-site
// cookie POST carrying a real near-expiry session is rejected 403 without
// rotating the cookie or extending the user_sessions row — a hostile subdomain
// can no longer repeatedly trigger rejected requests to prolong the victim's
// session. The same-origin control shows the refresh side effect does happen
// when CSRF lets the request through, proving the 403 path is what suppresses
// it.
func TestCSRFOrderRejectedCrossSitePostDoesNotRefreshSession(t *testing.T) {
	conn, router := csrfOrderContractRouter(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token, jti := contractNearExpirySessionToken(t, user, time.Hour)

	before := userSessions.GetByJti(jti)
	if before == nil {
		t.Fatal("session row missing before request")
	}
	expiryBefore := before.ExpiresAt

	recorder := csrfOrderRequest(t, router, "/api/forum/notification/mark-all-read", `{}`, token, "http://evil.example.test")
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("cross-site status = %d, want 403 (body %s)", recorder.Code, recorder.Body.String())
	}
	if envelope := decodeContractEnvelope(t, recorder); envelope.MessageCode != "auth.csrf.rejected" {
		t.Fatalf("cross-site messageCode = %q, want auth.csrf.rejected", envelope.MessageCode)
	}
	if strings.Contains(recorder.Header().Get("Set-Cookie"), "access_token") {
		t.Fatal("rejected cross-site POST rotated the session cookie")
	}
	if recorder.Header().Get("New-Token") != "" {
		t.Fatal("rejected cross-site POST emitted a refreshed New-Token header")
	}
	after := userSessions.GetByJti(jti)
	if after == nil {
		t.Fatal("session row disappeared after rejected request")
	}
	if delta := after.ExpiresAt.Sub(expiryBefore); delta > time.Minute || delta < -time.Minute {
		t.Fatalf("session row expiry moved from %v to %v on a rejected request", expiryBefore, after.ExpiresAt)
	}

	// Control: the same near-expiry session on a same-origin POST does rotate
	// the cookie and extend the row, so the assertions above are meaningful.
	control := csrfOrderRequest(t, router, "/api/forum/notification/mark-all-read", `{}`, token, "http://forum.example.test")
	if control.Code != http.StatusOK {
		t.Fatalf("same-origin status = %d, want 200 (body %s)", control.Code, control.Body.String())
	}
	if !strings.Contains(control.Header().Get("Set-Cookie"), "access_token") {
		t.Fatal("same-origin near-expiry POST should rotate the session cookie")
	}
	refreshed := userSessions.GetByJti(jti)
	if refreshed == nil {
		t.Fatal("session row missing after same-origin request")
	}
	if !refreshed.ExpiresAt.After(time.Now().Add(6 * time.Hour)) {
		t.Fatalf("same-origin request did not extend the session row (expiry %v)", refreshed.ExpiresAt)
	}
}

// TestCSRFOrderSameOriginCookiePostsReachBusiness proves chain completeness on
// the production assembly for every auth-first write group: with a real
// session and a same-origin Origin, the request passes CSRFProtection and
// JWTAuthCheck and executes the real controller (oierxjn should#1).
func TestCSRFOrderSameOriginCookiePostsReachBusiness(t *testing.T) {
	conn, router := csrfOrderContractRouter(t)

	t.Run("loginApi revoke-all succeeds", func(t *testing.T) {
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		recorder := csrfOrderRequest(t, router, "/api/user/sessions/revoke-all", `{}`, token, "http://forum.example.test")
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
		}
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.Code != 0 || envelope.MessageCode != "session.revokeAll.success" {
			t.Fatalf("envelope = %+v, want code 0 session.revokeAll.success", envelope)
		}
	})

	t.Run("forumLoginApi mark-all-read succeeds", func(t *testing.T) {
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		recorder := csrfOrderRequest(t, router, "/api/forum/notification/mark-all-read", `{}`, token, "http://forum.example.test")
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
		}
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.Code != 0 || envelope.MessageCode != "notification.markAllRead.success" {
			t.Fatalf("envelope = %+v, want code 0 notification.markAllRead.success", envelope)
		}
	})

	t.Run("chatApi messages read succeeds", func(t *testing.T) {
		user := createHTTPContractUser(t, conn, contractTestID())
		peer := createHTTPContractUser(t, conn, contractTestID())
		convID := contractTestID()
		createContractConversation(t, conn, convID, user.Id, peer.Id)
		token := contractSessionToken(t, user)
		recorder := csrfOrderRequest(t, router, "/api/forum/chat/messages",
			fmt.Sprintf(`{"convId":%d}`, convID), token, "http://forum.example.test")
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
		}
		if envelope := decodeContractEnvelope(t, recorder); envelope.Code != 0 {
			t.Fatalf("envelope code = %d, want 0 (messageCode %q)", envelope.Code, envelope.MessageCode)
		}
	})

	t.Run("adminApi traffic-overview succeeds", func(t *testing.T) {
		user := createHTTPContractUser(t, conn, contractTestID())
		grantContractPermission(t, conn, user.Id, permission.Admin)
		token := contractSessionToken(t, user)
		recorder := csrfOrderRequest(t, router, "/api/admin/traffic-overview", `{}`, token, "http://forum.example.test")
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
		}
		if envelope := decodeContractEnvelope(t, recorder); envelope.Code != 0 {
			t.Fatalf("envelope code = %d, want 0 (messageCode %q)", envelope.Code, envelope.MessageCode)
		}
	})
}

package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userSessions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Route-level CSRF matrix (issue #406). Unlike the middleware unit tests in
// app/http/middleware, these exercise the production route assembly
// (apiRoute): CSRFProtection is mounted exactly where production mounts it
// (logout before the handler; login/forum/admin groups after JWTAuthCheck).
// The logout matrix below needs no database (an invalid session JWT never
// reaches one); the chain-attachment and real-session tests bootstrap the
// minimal table set (users, user statistics, user sessions) so JWTAuthCheck
// can validate a minted session.

// csrfChainTestDB migrates the tables the auth chain needs and nothing else:
// the requests under test are rejected at the CSRF gate before any handler
// runs.
func csrfChainTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&userStatistics.Entity{},
		&userSessions.Entity{},
	); err != nil {
		t.Fatalf("migrate CSRF chain test tables: %v", err)
	}
	return conn
}

func csrfRouteRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiRoute(router)
	return router
}

func csrfRoutePost(t *testing.T, router http.Handler, cookie, authorization, origin, referer string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "http://forum.example.test/api/logout", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		request.AddCookie(&http.Cookie{Name: "access_token", Value: cookie})
	}
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	if referer != "" {
		request.Header.Set("Referer", referer)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func assertCsrfRouteRejected(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body %s)", recorder.Code, recorder.Body.String())
	}
	var envelope struct {
		Code        int    `json:"code"`
		MessageCode string `json:"messageCode"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode rejection body %q: %v", recorder.Body.String(), err)
	}
	if envelope.MessageCode != "auth.csrf.rejected" {
		t.Fatalf("messageCode = %q, want auth.csrf.rejected", envelope.MessageCode)
	}
	if strings.Contains(recorder.Header().Get("Set-Cookie"), "access_token") {
		t.Fatal("rejected request must not clear or set the session cookie")
	}
}

func assertCsrfRouteAllowed(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}
}

func TestCSRFLogoutRejectsCookiePostWithoutOrigin(t *testing.T) {
	router := csrfRouteRouter()
	assertCsrfRouteRejected(t, csrfRoutePost(t, router, "session-jwt", "", "", ""))
}

func TestCSRFLogoutRejectsCrossSiteOrigin(t *testing.T) {
	router := csrfRouteRouter()
	assertCsrfRouteRejected(t, csrfRoutePost(t, router, "session-jwt", "", "http://evil.example.test", ""))
}

func TestCSRFLogoutAllowsSameOriginCookiePost(t *testing.T) {
	router := csrfRouteRouter()
	// Invalid JWT in the cookie: the CSRF gate passes (same origin), then
	// logout clears the cookie and stays idempotent.
	recorder := csrfRoutePost(t, router, "not-a-jwt", "", "http://forum.example.test", "")
	assertCsrfRouteAllowed(t, recorder)
	if !strings.Contains(recorder.Header().Get("Set-Cookie"), "access_token") {
		t.Fatal("same-origin logout should still clear the access_token cookie")
	}
}

func TestCSRFLogoutAllowsRefererFallbackForCookiePost(t *testing.T) {
	router := csrfRouteRouter()
	recorder := csrfRoutePost(t, router, "not-a-jwt", "", "", "http://forum.example.test/p/1")
	assertCsrfRouteAllowed(t, recorder)
}

func TestCSRFLogoutAllowsApiClientWithAuthorizationEvenWithForeignOrigin(t *testing.T) {
	router := csrfRouteRouter()
	// Bearer API clients are exempt from the origin check even when a cookie
	// is also present and the Origin is foreign.
	recorder := csrfRoutePost(t, router, "session-jwt", "Bearer api-token", "http://evil.example.test", "")
	assertCsrfRouteAllowed(t, recorder)
}

func TestCSRFLogoutAllowsAnonymousPostWithoutOrigin(t *testing.T) {
	router := csrfRouteRouter()
	// No access_token cookie -> nothing cookie-authenticated to protect;
	// anonymous logout stays an idempotent success.
	recorder := csrfRoutePost(t, router, "", "", "", "")
	assertCsrfRouteAllowed(t, recorder)
}

// TestCSRFProtectedForumWriteRoute rejects cookie POSTs without Origin on the
// real /api/forum write chain (JWTAuthCheck + CSRFProtection + handler).
func TestCSRFProtectedForumWriteRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Reuse the full apiRoute assembly; only the middleware ordering matters
	// here and the request is rejected before any DB access.
	apiRoute(router)

	request := httptest.NewRequest(http.MethodPost, "http://forum.example.test/api/forum/topics/write", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "access_token", Value: "session-jwt"})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	// JWTAuthCheck runs first and fails on the bogus JWT before CSRF runs, so
	// an unauthenticated write stays a 401 (anonymous semantics preserved).
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body %s)", recorder.Code, recorder.Body.String())
	}
}

// TestCSRFProtectedWriteRoutesRegistered pins the registered paths of the
// cookie-authenticated write surface (a registration smoke test only:
// router.Routes() exposes method+path+final handler, not the middleware
// chain). TestCSRFGroupChainsRejectCrossSiteSessionWrites below is what
// proves the middleware is attached.
func TestCSRFProtectedWriteRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiRoute(router)
	fileServer(router)

	for _, route := range []string{
		"POST /api/logout",
		"POST /api/set-user-info",
		"POST /api/forum/topics/write",
		"POST /api/forum/posts/create",
		"POST /api/forum/moderation/topic-status",
		"POST /api/forum/chat/send",
		"POST /api/admin/traffic-overview",
		"POST /api/admin/save-site-settings",
		"POST /file/img-upload",
	} {
		found := false
		for _, r := range router.Routes() {
			if r.Method+" "+r.Path == route {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s was not registered", route)
		}
	}
}

// TestCSRFGroupChainsRejectCrossSiteSessionWrites proves CSRFProtection is
// actually in every production cookie-authenticated write chain. The group
// middleware added via .Use() is invisible to router.Routes(), so this fires
// a valid-session, foreign-Origin write at one representative route per
// protected group and requires the 403 auth.csrf.rejected envelope; if the
// middleware is dropped from a group the request proceeds past auth and the
// assertion fails.
func TestCSRFGroupChainsRejectCrossSiteSessionWrites(t *testing.T) {
	conn := csrfChainTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiRoute(router)
	fileServer(router)

	user := createHTTPContractUser(t, conn, contractTestID())
	sessionToken := contractSessionToken(t, user)

	for _, tc := range []struct{ group, path string }{
		{"loginApi", "/api/user/sessions/revoke"},
		{"forumLoginApi", "/api/forum/topics/write"},
		{"chatApi", "/api/forum/chat/send"},
		{"adminApi", "/api/admin/traffic-overview"},
		// The /file group mounts CSRFProtection at group level, before the
		// per-route auth — the mount-order exception documented in
		// middleware/csrfProtection.go. A rejection here (not a 401) pins it.
		{"fileGroup", "/file/img-upload"},
	} {
		request := httptest.NewRequest(http.MethodPost, "http://forum.example.test"+tc.path, strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(&http.Cookie{Name: "access_token", Value: sessionToken})
		request.Header.Set("Origin", "http://evil.example.test")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("%s: status = %d, want 403 (body %s)", tc.group, recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), "auth.csrf.rejected") {
			t.Fatalf("%s: body %q lacks auth.csrf.rejected", tc.group, recorder.Body.String())
		}
	}
}

// TestCSRFRealSessionSameOriginLogoutPasses guards against over-blocking: a
// legitimate same-origin write carrying a real session passes the gate and
// reaches the handler, which revokes the session row.
func TestCSRFRealSessionSameOriginLogoutPasses(t *testing.T) {
	conn := csrfChainTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiRoute(router)

	user := createHTTPContractUser(t, conn, contractTestID())
	sessionToken := contractSessionToken(t, user)

	request := httptest.NewRequest(http.MethodPost, "http://forum.example.test/api/logout", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "access_token", Value: sessionToken})
	request.Header.Set("Origin", "http://forum.example.test")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}
	// Revocation deletes the session row (sessionservice.RevokeByJti).
	var remaining int64
	if err := conn.Model(&userSessions.Entity{}).Where("user_id = ?", user.Id).Count(&remaining).Error; err != nil {
		t.Fatalf("count session rows: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("same-origin logout must revoke the session row, %d remain", remaining)
	}
}

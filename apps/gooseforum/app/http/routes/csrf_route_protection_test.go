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
// (apiRoute + fileServer): CSRFProtection is mounted before the stateful
// authentication middleware (JWTAuthCheck) in every cookie-write group
// (logout; login/forum/chat/admin/file), so a cross-site cookie POST is
// rejected 403 without ever reaching JWTAuthCheck — no JWT refresh, no
// user_sessions extension, no activity event (Codex review P2). Requests
// without a session cookie pass through CSRF untouched and keep their
// anonymous semantics (JWTAuthCheck still answers 401).
//
// The logout matrix below needs no database (an invalid session JWT never
// reaches one); the real-session chain-attachment tests in this file and the
// DB-backed no-refresh proofs in csrf_order_contract_test.go bootstrap the
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
	fileServer(router)
	return router
}

func csrfRoutePost(t *testing.T, router http.Handler, path, cookie, authorization, origin, referer string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "http://forum.example.test"+path, strings.NewReader(`{}`))
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
	// A rejection must never carry a session cookie: CSRF runs before
	// JWTAuthCheck, so the near-expiry JWT is not rotated on rejected
	// cross-site requests (Codex review P2).
	if strings.Contains(recorder.Header().Get("Set-Cookie"), "access_token") {
		t.Fatal("rejected request must not set or refresh the session cookie")
	}
	if recorder.Header().Get("New-Token") != "" {
		t.Fatal("rejected request must not emit a refreshed New-Token header")
	}
}

func assertCsrfRouteUnauthenticated(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body %s)", recorder.Code, recorder.Body.String())
	}
	var envelope struct {
		Code        int    `json:"code"`
		MessageCode string `json:"messageCode"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode 401 body %q: %v", recorder.Body.String(), err)
	}
	if envelope.MessageCode != "auth.required" {
		t.Fatalf("messageCode = %q, want auth.required", envelope.MessageCode)
	}
}

func assertCsrfRouteAllowed(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}
}

// csrfCookieWriteGroupRoutes are representative POST routes of the four
// authentication-first write groups in route4api.go. After the #406 follow-up
// every group mounts CSRFProtection before JWTAuthCheck.
var csrfCookieWriteGroupRoutes = []struct {
	name string
	path string
}{
	{"loginApi", "/api/user/sessions/revoke-all"},
	{"forumLoginApi", "/api/forum/notification/mark-all-read"},
	{"chatApi", "/api/forum/chat/messages"},
	{"adminApi", "/api/admin/traffic-overview"},
}

// TestCSRFCookieWriteGroupsRejectCrossSiteBeforeAuthentication proves the
// CSRF gate runs before JWTAuthCheck in every auth-first write group: a
// cookie POST from a foreign Origin is rejected 403 auth.csrf.rejected even
// though the cookie holds an invalid JWT — had JWTAuthCheck run first it
// would have answered 401 and the test would fail here.
func TestCSRFCookieWriteGroupsRejectCrossSiteBeforeAuthentication(t *testing.T) {
	router := csrfRouteRouter()
	for _, group := range csrfCookieWriteGroupRoutes {
		t.Run(group.name, func(t *testing.T) {
			recorder := csrfRoutePost(t, router, group.path, "bogus-session-jwt", "", "http://evil.example.test", "")
			assertCsrfRouteRejected(t, recorder)
		})
	}
}

// TestCSRFCookieWriteGroupsAnonymousPostStaysUnauthenticated pins the
// anonymous regression after the CSRF-first reorder: a POST without a
// session cookie passes the CSRF gate and JWTAuthCheck still answers 401.
func TestCSRFCookieWriteGroupsAnonymousPostStaysUnauthenticated(t *testing.T) {
	router := csrfRouteRouter()
	for _, group := range csrfCookieWriteGroupRoutes {
		t.Run(group.name, func(t *testing.T) {
			recorder := csrfRoutePost(t, router, group.path, "", "", "", "")
			assertCsrfRouteUnauthenticated(t, recorder)
		})
	}
}

// TestCSRFCookieWriteGroupsSameOriginReachesAuthentication ensures the CSRF
// gate does not over-block: a same-origin cookie POST passes through to
// JWTAuthCheck, which rejects the bogus JWT with 401 (not 403).
func TestCSRFCookieWriteGroupsSameOriginReachesAuthentication(t *testing.T) {
	router := csrfRouteRouter()
	for _, group := range csrfCookieWriteGroupRoutes {
		t.Run(group.name, func(t *testing.T) {
			recorder := csrfRoutePost(t, router, group.path, "bogus-session-jwt", "", "http://forum.example.test", "")
			assertCsrfRouteUnauthenticated(t, recorder)
		})
	}
}

// TestCSRFFileGroupRejectsCrossSiteBeforeAuthentication proves the file
// group's CSRF-first mount behaviorally: a foreign-Origin cookie POST to
// /file/img-upload is rejected 403 by CSRF, not 401 by the invalid-JWT auth
// check (oierxjn should#1).
func TestCSRFFileGroupRejectsCrossSiteBeforeAuthentication(t *testing.T) {
	router := csrfRouteRouter()
	recorder := csrfRoutePost(t, router, "/file/img-upload", "bogus-session-jwt", "", "http://evil.example.test", "")
	assertCsrfRouteRejected(t, recorder)
}

func TestCSRFFileGroupSameOriginReachesAuthentication(t *testing.T) {
	router := csrfRouteRouter()
	recorder := csrfRoutePost(t, router, "/file/img-upload", "bogus-session-jwt", "", "http://forum.example.test", "")
	assertCsrfRouteUnauthenticated(t, recorder)
}

func TestCSRFFileGroupAnonymousPostStaysUnauthenticated(t *testing.T) {
	router := csrfRouteRouter()
	recorder := csrfRoutePost(t, router, "/file/img-upload", "", "", "", "")
	assertCsrfRouteUnauthenticated(t, recorder)
}

// TestCSRFProtectedForumWriteRouteRejectsCookiePostWithoutOrigin pins the
// fail-closed branch on the real /api/forum chain: a cookie POST without
// Origin or Referer is rejected 403 before authentication runs.
func TestCSRFProtectedForumWriteRouteRejectsCookiePostWithoutOrigin(t *testing.T) {
	router := csrfRouteRouter()
	recorder := csrfRoutePost(t, router, "/api/forum/topics/write", "bogus-session-jwt", "", "", "")
	assertCsrfRouteRejected(t, recorder)
}

func TestCSRFLogoutRejectsCookiePostWithoutOrigin(t *testing.T) {
	router := csrfRouteRouter()
	assertCsrfRouteRejected(t, csrfRoutePost(t, router, "/api/logout", "session-jwt", "", "", ""))
}

func TestCSRFLogoutRejectsCrossSiteOrigin(t *testing.T) {
	router := csrfRouteRouter()
	assertCsrfRouteRejected(t, csrfRoutePost(t, router, "/api/logout", "session-jwt", "", "http://evil.example.test", ""))
}

func TestCSRFLogoutAllowsSameOriginCookiePost(t *testing.T) {
	router := csrfRouteRouter()
	// Invalid JWT in the cookie: the CSRF gate passes (same origin), then
	// logout clears the cookie and stays idempotent.
	recorder := csrfRoutePost(t, router, "/api/logout", "not-a-jwt", "", "http://forum.example.test", "")
	assertCsrfRouteAllowed(t, recorder)
	if !strings.Contains(recorder.Header().Get("Set-Cookie"), "access_token") {
		t.Fatal("same-origin logout should still clear the access_token cookie")
	}
}

func TestCSRFLogoutAllowsRefererFallbackForCookiePost(t *testing.T) {
	router := csrfRouteRouter()
	recorder := csrfRoutePost(t, router, "/api/logout", "not-a-jwt", "", "", "http://forum.example.test/p/1")
	assertCsrfRouteAllowed(t, recorder)
}

func TestCSRFLogoutAllowsApiClientWithAuthorizationEvenWithForeignOrigin(t *testing.T) {
	router := csrfRouteRouter()
	// Bearer API clients are exempt from the origin check even when a cookie
	// is also present and the Origin is foreign.
	recorder := csrfRoutePost(t, router, "/api/logout", "session-jwt", "Bearer api-token", "http://evil.example.test", "")
	assertCsrfRouteAllowed(t, recorder)
}

func TestCSRFLogoutAllowsAnonymousPostWithoutOrigin(t *testing.T) {
	router := csrfRouteRouter()
	// No access_token cookie -> nothing cookie-authenticated to protect;
	// anonymous logout stays an idempotent success.
	recorder := csrfRoutePost(t, router, "/api/logout", "", "", "", "")
	assertCsrfRouteAllowed(t, recorder)
}

// TestCSRFProtectedWriteRoutesRegistered pins the registered paths of the
// cookie-authenticated write surface (a registration smoke test only:
// router.Routes() exposes method+path+final handler, not the middleware
// chain). The behavioral chain-order coverage lives in
// TestCSRFGroupChainsRejectCrossSiteSessionWrites, the rejection/401 matrix
// above, and csrf_order_contract_test.go (oierxjn should#1).
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

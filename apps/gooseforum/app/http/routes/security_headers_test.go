package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/gin-gonic/gin"
)

// TestSecurityHeadersRealRouterHTMLRoutes asserts the real RegisterByGin
// assembly (issue #407) sends the unified security headers plus the page CSP
// on browser HTML routes: home, login, search, wiki, courses and topic
// detail. Headers are written before handlers run, so they must be present
// regardless of the resulting status.
func TestSecurityHeadersRealRouterHTMLRoutes(t *testing.T) {
	router := securityHeadersTestRouter(t, "production")

	for _, path := range []string{"/", "/login", "/search", "/terms", "/links", "/privacy", "/sponsors", "/courses"} {
		response := performSecurityHeadersRequest(router, http.MethodGet, path, "text/html")
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d, want 200 (body=%s)", path, response.Code, response.Body.String())
		}
		assertHTMLSecurityHeaders(t, path, response)
	}
}

// TestSecurityHeadersRealRouterErrorResponses asserts HTML error pages still
// carry the full header set including page CSP: a missing forum topic (404)
// and an engine NoRoute hit with browser Accept (HTML 404).
func TestSecurityHeadersRealRouterErrorResponses(t *testing.T) {
	router := securityHeadersTestRouter(t, "production")

	for _, tc := range []struct {
		path   string
		accept string
	}{
		{path: "/p/post/99999999", accept: "text/html"},        // forum topic 404 page
		{path: "/definitely-not-a-route", accept: "text/html"}, // engine NoRoute, HTML accept
	} {
		response := performSecurityHeadersRequest(router, http.MethodGet, tc.path, tc.accept)
		if response.Code != http.StatusNotFound {
			t.Fatalf("GET %s status=%d, want 404 error page", tc.path, response.Code)
		}
		assertHTMLSecurityHeaders(t, tc.path, response)
	}
}

// TestSecurityHeadersRealRouterWikiProjection covers the wiki home: it renders
// either an HTML 500 (empty/unmigrated wiki projection) or the HTML 200 page
// when a parallel test has seeded wiki tables. Either way the response must
// carry the full security header set, which is the invariant under test.
func TestSecurityHeadersRealRouterWikiProjection(t *testing.T) {
	router := securityHeadersTestRouter(t, "production")

	response := performSecurityHeadersRequest(router, http.MethodGet, "/wiki", "text/html")
	if response.Code != http.StatusOK && response.Code != http.StatusInternalServerError {
		t.Fatalf("GET /wiki status=%d, want 200 or 500 (body=%s)", response.Code, response.Body.String())
	}
	assertHTMLSecurityHeaders(t, "/wiki", response)
}

// TestSecurityHeadersRealRouterAPISurfaces asserts JSON/API/file/health
// responses receive the universal headers but no page CSP (layered policy).
func TestSecurityHeadersRealRouterAPISurfaces(t *testing.T) {
	router := securityHeadersTestRouter(t, "production")

	for _, tc := range []struct{ method, path, accept string }{
		{method: http.MethodGet, path: "/health", accept: "application/json"},
		{method: http.MethodPost, path: "/api/logout", accept: "application/json"},
		{method: http.MethodGet, path: "/file/img/nonexistent.webp", accept: "application/json"},
	} {
		response := performSecurityHeadersRequest(router, tc.method, tc.path, tc.accept)
		if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Fatalf("%s %s X-Content-Type-Options=%q, want nosniff", tc.method, tc.path, got)
		}
		if got := response.Header().Get("X-Frame-Options"); got != "DENY" {
			t.Fatalf("%s %s X-Frame-Options=%q, want DENY", tc.method, tc.path, got)
		}
		if got := response.Header().Get("Content-Security-Policy"); got != "" {
			t.Fatalf("%s %s Content-Security-Policy=%q, want withheld on non-HTML surfaces", tc.method, tc.path, got)
		}
	}
}

// TestSecurityHeadersDevVsProductionCSP pins the documented dev/prod
// difference: only non-production pages additionally allow the Vite HMR
// WebSocket schemes in connect-src.
func TestSecurityHeadersDevVsProductionCSP(t *testing.T) {
	previous := preferences.GetString("app.env", "production")
	t.Cleanup(func() { preferences.Set("app.env", previous) })

	preferences.Set("app.env", "local")
	router := securityHeadersTestRouter(t, "local")
	localCSP := performSecurityHeadersRequest(router, http.MethodGet, "/login", "text/html").
		Header().Get("Content-Security-Policy")
	if !strings.Contains(localCSP, "connect-src 'self' http: https: ws: wss:") {
		t.Fatalf("dev /login CSP missing HMR ws allowance: %q", localCSP)
	}

	preferences.Set("app.env", "production")
	router = securityHeadersTestRouter(t, "production")
	prodCSP := performSecurityHeadersRequest(router, http.MethodGet, "/login", "text/html").
		Header().Get("Content-Security-Policy")
	if strings.Contains(prodCSP, "ws: wss:") {
		t.Fatalf("production /login CSP must not allow ws: %q", prodCSP)
	}
}

func assertHTMLSecurityHeaders(t *testing.T, path string, response *httptest.ResponseRecorder) {
	t.Helper()
	if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("GET %s X-Content-Type-Options=%q, want nosniff", path, got)
	}
	if got := response.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("GET %s X-Frame-Options=%q, want DENY", path, got)
	}
	if got := response.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Fatalf("GET %s Referrer-Policy=%q", path, got)
	}
	if got := response.Header().Get("Permissions-Policy"); got != "camera=(), microphone=(), geolocation=()" {
		t.Fatalf("GET %s Permissions-Policy=%q", path, got)
	}
	if got := response.Header().Get("X-Powered-By"); got != "" {
		t.Fatalf("GET %s X-Powered-By=%q, want removed", path, got)
	}
	csp := response.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatalf("GET %s missing Content-Security-Policy", path)
	}
	if !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Fatalf("GET %s CSP missing frame-ancestors 'none': %q", path, csp)
	}
	if !strings.Contains(csp, "script-src 'self'") {
		t.Fatalf("GET %s CSP script-src must include 'self': %q", path, csp)
	}
	if strings.Contains(csp, "script-src 'unsafe-inline'") || strings.Contains(csp, "script-src *") {
		t.Fatalf("GET %s CSP script-src must not allow inline/global scripts: %q", path, csp)
	}
}

func securityHeadersTestRouter(t *testing.T, appEnv string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ratelimit.Default().ResetAll()
	t.Cleanup(func() { ratelimit.Default().ResetAll() })

	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate page_config: %v", err)
	}
	configureHTTPContractTestSettings(t, conn)

	previousEnv := preferences.GetString("app.env", "production")
	preferences.Set("app.env", appEnv)
	t.Cleanup(func() { preferences.Set("app.env", previousEnv) })

	router := gin.New()
	RegisterByGin(router)
	return router
}

func performSecurityHeadersRequest(router http.Handler, method, path, accept string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "http://localhost"+path, nil)
	if accept != "" {
		request.Header.Set("Accept", accept)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

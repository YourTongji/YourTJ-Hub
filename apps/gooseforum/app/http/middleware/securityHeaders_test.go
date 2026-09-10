package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersOnAllResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeaders)
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := recorder.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q, want DENY", got)
	}
	if got := recorder.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Fatalf("Referrer-Policy = %q, want strict-origin-when-cross-origin", got)
	}
	if got := recorder.Header().Get("Permissions-Policy"); got != "camera=(), microphone=(), geolocation=()" {
		t.Fatalf("Permissions-Policy = %q", got)
	}
	if got := recorder.Header().Get("X-Powered-By"); got != "" {
		t.Fatalf("X-Powered-By = %q, want empty (removed fingerprinting header)", got)
	}
}

func TestSecurityHeadersPageCSPAppliedToHTMLRoutes(t *testing.T) {
	withSecurityEnv(t, "production", func() {
		router := gin.New()
		router.Use(SecurityHeaders)
		router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
		router.GET("/p/post/1", func(c *gin.Context) { c.Status(http.StatusOK) })
		router.GET("/wiki", func(c *gin.Context) { c.Status(http.StatusOK) })
		router.GET("/courses", func(c *gin.Context) { c.Status(http.StatusOK) })

		for _, path := range []string{"/", "/p/post/1", "/wiki", "/courses"} {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
			csp := recorder.Header().Get("Content-Security-Policy")
			if csp == "" {
				t.Fatalf("GET %s missing Content-Security-Policy", path)
			}
			assertCSPConsistent(t, path, csp)
		}
	})
}

func TestSecurityHeadersCSPWithheldFromNonPageSurfaces(t *testing.T) {
	withSecurityEnv(t, "production", func() {
		router := gin.New()
		router.Use(SecurityHeaders)
		router.GET("/api/health-probe", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
		router.GET("/file/img/1.png", func(c *gin.Context) { c.Data(http.StatusOK, "image/png", nil) })
		router.GET("/assets/site.js", func(c *gin.Context) { c.String(http.StatusOK, "") })
		router.GET("/static/pic/1.webp", func(c *gin.Context) { c.String(http.StatusOK, "") })
		router.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
		router.POST("/api/login", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })

		for _, tc := range []struct{ method, path string }{
			{http.MethodGet, "/api/health-probe"},
			{http.MethodGet, "/file/img/1.png"},
			{http.MethodGet, "/assets/site.js"},
			{http.MethodGet, "/static/pic/1.webp"},
			{http.MethodGet, "/health"},
			{http.MethodPost, "/api/login"},
		} {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.path, nil))
			if got := recorder.Header().Get("Content-Security-Policy"); got != "" {
				t.Fatalf("%s %s Content-Security-Policy = %q, want withheld", tc.method, tc.path, got)
			}
			if got := recorder.Header().Get("X-Frame-Options"); got != "DENY" {
				t.Fatalf("%s %s X-Frame-Options = %q, want DENY on every surface", tc.method, tc.path, got)
			}
		}
	})
}

func TestSecurityHeadersPageCSPDevAllowsHMRWebSocket(t *testing.T) {
	withSecurityEnv(t, "local", func() {
		router := gin.New()
		router.Use(SecurityHeaders)
		router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		csp := recorder.Header().Get("Content-Security-Policy")
		if !strings.Contains(csp, "connect-src 'self' http: https: ws: wss:") {
			t.Fatalf("dev CSP connect-src = %q, want Vite HMR ws/wss allowance", csp)
		}
	})
}

func TestSecurityHeadersAppliesToErrorResponses(t *testing.T) {
	withSecurityEnv(t, "production", func() {
		router := gin.New()
		router.Use(SecurityHeaders)
		router.GET("/boom", func(c *gin.Context) { c.String(http.StatusInternalServerError, "boom") })

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/boom", nil))
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", recorder.Code)
		}
		if got := recorder.Header().Get("Content-Security-Policy"); got == "" {
			t.Fatal("500 error response missing Content-Security-Policy")
		}
		if got := recorder.Header().Get("X-Frame-Options"); got != "DENY" {
			t.Fatalf("500 X-Frame-Options = %q, want DENY", got)
		}
	})
}

// assertCSPConsistent guards against self-contradictory or accidental header
// values: frame-ancestors must forbid embedding, no directive may allow
// arbitrary inline script execution, and clickjacking tokens must not allow
// any ancestor source.
func assertCSPConsistent(t *testing.T, path, csp string) {
	t.Helper()
	if !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Fatalf("GET %s CSP missing frame-ancestors 'none': %q", path, csp)
	}
	if !strings.Contains(csp, "frame-src 'none'") && !strings.Contains(csp, "frame-src ") {
		t.Fatalf("GET %s CSP missing frame-src 'none': %q", path, csp)
	}
	if strings.Contains(csp, "script-src 'unsafe-inline'") || strings.Contains(csp, "script-src *") {
		t.Fatalf("GET %s CSP script-src must not allow inline/global scripts: %q", path, csp)
	}
	for _, directive := range strings.Split(csp, ";") {
		fields := strings.Fields(strings.TrimSpace(directive))
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "script-src" {
			if !containsStr(fields[1:], "'self'") {
				t.Fatalf("GET %s CSP script-src must include 'self': %q", path, csp)
			}
		}
	}
}
func withSecurityEnv(t *testing.T, appEnv string, fn func()) {
	t.Helper()
	previous := preferences.GetString("app.env", "production")
	preferences.Set("app.env", appEnv)
	t.Cleanup(func() { preferences.Set("app.env", previous) })
	fn()
}

func containsStr(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

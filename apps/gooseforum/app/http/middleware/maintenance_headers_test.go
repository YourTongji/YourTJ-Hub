package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/gin-gonic/gin"
)

// TestSiteMaintenanceUniversalHeaders guards the maintenance 503 surface:
// SiteMaintenance registers before SecurityHeaders in bridge.go and aborts
// the chain, so the universal security headers must be written by the
// maintenance path itself. Page CSP stays deliberately withheld (the
// maintenance page ships inline scripts, see securityHeaders.go).
func TestSiteMaintenanceUniversalHeaders(t *testing.T) {
	withSecurityEnv(t, "production", func() {
		preferences.Set("app.maintenance", true)
		t.Cleanup(func() { preferences.Set("app.maintenance", false) })

		router := gin.New()
		router.Use(SiteMaintenance)
		router.Use(SecurityHeaders)
		router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", recorder.Code)
		}
		if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
			t.Fatalf("Content-Type = %q, want text/html; charset=utf-8", got)
		}
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
		if got := recorder.Header().Get("Content-Security-Policy"); got != "" {
			t.Fatalf("Content-Security-Policy = %q, want empty (maintenance page ships inline scripts)", got)
		}
	})
}

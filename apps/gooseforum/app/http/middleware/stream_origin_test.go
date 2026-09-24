package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStreamOriginProtection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/events", StreamOriginProtection, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	for _, tc := range []struct {
		name, origin, bearer string
		status               int
	}{
		{"same origin cookie", "https://forum.example", "", 204},
		{"foreign origin cookie", "https://evil.example", "", 403},
		{"native bearer", "https://evil.example", "Bearer token", 204},
		{"native without origin", "", "", 204},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "https://forum.example/events", nil)
			req.Host = "forum.example"
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("Authorization", tc.bearer)
			req.AddCookie(&http.Cookie{Name: accessTokenCookieName, Value: "opaque"})
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("status=%d, want %d", w.Code, tc.status)
			}
		})
	}
}

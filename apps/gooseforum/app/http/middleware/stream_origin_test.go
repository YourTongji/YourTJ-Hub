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
		name, origin, referer, bearer string
		cookie                        bool
		status                        int
	}{
		{"same origin cookie", "https://forum.example", "", "", true, 204},
		{"same origin referer cookie", "", "https://forum.example/page", "", true, 204},
		{"foreign origin cookie", "https://evil.example", "", "", true, 403},
		{"originless cookie", "", "", "", true, 403},
		{"foreign referer cookie", "", "https://evil.example/page", "", true, 403},
		{"non-bearer header cannot exempt cookie", "", "", "Basic opaque", true, 403},
		{"native bearer", "https://evil.example", "", "Bearer token", true, 204},
		{"native without origin or cookie", "", "", "", false, 204},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "https://forum.example/events", nil)
			req.Host = "forum.example"
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("Referer", tc.referer)
			req.Header.Set("Authorization", tc.bearer)
			if tc.cookie {
				req.AddCookie(&http.Cookie{Name: accessTokenCookieName, Value: "opaque"})
			}
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("status=%d, want %d", w.Code, tc.status)
			}
		})
	}
}

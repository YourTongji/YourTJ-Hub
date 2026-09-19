package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCampusCallbackGuardStopsBeforeCodeExchange(t *testing.T) {
	for _, guard := range []gin.HandlerFunc{JWTAuthCheck, CheckWritableAccount} {
		app := gin.New()
		exchanged := false
		app.GET("/api/campus/tongji/callback", CampusCallbackPrivacy, guard, func(c *gin.Context) {
			exchanged = true
			c.Status(http.StatusNoContent)
		})
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/campus/tongji/callback?code=private&state=private", nil))
		if exchanged || w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/campus?authorization=failed" {
			t.Fatalf("guard did not abort with a clean redirect: exchanged=%v status=%d", exchanged, w.Code)
		}
	}
}

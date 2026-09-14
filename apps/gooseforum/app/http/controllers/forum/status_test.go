package forum

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStatusPublicPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/status", Status)
	for _, mode := range []string{"", "true"} {
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		req.Header.Set("X-Goose-Page", mode)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"component":"status.index"`) || !strings.Contains(rec.Body.String(), `/status`) {
			t.Fatalf("missing status page: %s", rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "umami_share_id") {
			t.Fatal("page payload exposes provider config")
		}
	}
}

package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/gin-gonic/gin"
)

func TestServerStatusHTTPContract(t *testing.T) {
	previous := preferences.GetBool("status.enabled")
	preferences.Set("status.enabled", false)
	t.Cleanup(func() { preferences.Set("status.enabled", previous) })
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/forum/status", api.ServerStatus)
	for _, tc := range []struct {
		path, fixture string
		code          int
	}{
		{"/api/forum/status", "status-unconfigured.json", http.StatusOK},
		{"/api/forum/status?range=365d", "status-invalid-range.json", http.StatusBadRequest},
		{"/api/forum/status?serverRange=30d", "status-invalid-range.json", http.StatusBadRequest},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if recorder.Code != tc.code || recorder.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("unexpected response %d: %s", recorder.Code, recorder.Body.String())
			}
			assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, tc.fixture))
		})
	}
	for _, period := range []string{"1h", "6h", "24h", "7d"} {
		t.Run("serverRange="+period, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/forum/status?range=30d&serverRange="+period, nil))
			var result struct{ Range, ServerRange string }
			if err := json.Unmarshal(decodeContractEnvelope(t, recorder).Result, &result); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusOK || result.Range != "30d" || result.ServerRange != period {
				t.Fatalf("range queries were not independently forwarded: %s", recorder.Body.String())
			}
		})
	}
}

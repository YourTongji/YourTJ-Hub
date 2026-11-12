package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/buildinfo"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers"
	"github.com/gin-gonic/gin"
)

func TestMCPServerCardAndOAuthDiscoveryStaySplit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	siteInfoRoute(router)
	router.NoRoute(controllers.NotFound)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://localhost/.well-known/mcp/server-card.json", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("server card status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("server card content-type = %q, want application/json", got)
	}
	var card map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &card); err != nil {
		t.Fatalf("server card is not JSON: %v body=%s", err, rec.Body.String())
	}
	info, _ := card["serverInfo"].(map[string]any)
	name, _ := info["name"].(string)
	if name == "" {
		name, _ = card["name"].(string)
	}
	if name == "" {
		t.Fatalf("server card has no name: %s", rec.Body.String())
	}
	transport, _ := card["transport"].(map[string]any)
	endpoint, _ := transport["endpoint"].(string)
	if !strings.Contains(endpoint, "/mcp") {
		t.Fatalf("transport endpoint = %q, want a /mcp path", endpoint)
	}
	if transport["type"] != "streamable-http" {
		t.Fatalf("transport type = %v, want streamable-http", transport["type"])
	}
	if _, ok := card["capabilities"].(map[string]any); !ok {
		t.Fatalf("server card capabilities missing: %s", rec.Body.String())
	}
	if got, _ := info["version"].(string); got != buildinfo.Get().Version {
		t.Fatalf("serverInfo.version = %q, want %q", got, buildinfo.Get().Version)
	}
	auth, _ := card["authentication"].(map[string]any)
	if auth["type"] != "bearer" || auth["required"] != true {
		t.Fatalf("authentication = %#v, want required static bearer", auth)
	}
	lower := strings.ToLower(rec.Body.String())
	for _, forbidden := range []string{"agt_", "oauth", "client_secret", "access_token"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("server card contains %q", forbidden)
		}
	}

	for _, path := range []string{
		"/.well-known/oauth-authorization-server",
		"/.well-known/oauth-protected-resource",
	} {
		miss := httptest.NewRecorder()
		router.ServeHTTP(miss, httptest.NewRequest(http.MethodGet, "http://localhost"+path, nil))
		if miss.Code != http.StatusNotFound {
			t.Fatalf("GET %s status = %d, want 404", path, miss.Code)
		}
	}
}

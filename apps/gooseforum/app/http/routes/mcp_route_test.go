package routes

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
)

// TestMcpRouteRegistered asserts the MCP endpoint is mounted when RegisterByGin
// is used (the production assembly), independent of the apiRoute-only router
// the agent tests drive.
func TestMcpRouteRegistered(t *testing.T) {
	preferences.Set("mcp.enabled", true)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterByGin(router)

	registered := map[string]bool{}
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		if !registered[method+" /mcp"] {
			t.Errorf("%s /mcp was not registered", method)
		}
	}
}

// TestMcpRouteDisabledHonorsPreference asserts the endpoint is not mounted when
// mcp.enabled is false.
func TestMcpRouteDisabledHonorsPreference(t *testing.T) {
	preferences.Set("mcp.enabled", false)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterByGin(router)

	registered := map[string]bool{}
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	if registered[http.MethodGet+" /mcp"] || registered[http.MethodPost+" /mcp"] {
		t.Fatal("/mcp should not be mounted when mcp.enabled=false")
	}
	preferences.Set("mcp.enabled", true)
}

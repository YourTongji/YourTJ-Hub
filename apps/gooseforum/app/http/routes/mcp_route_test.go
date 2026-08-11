package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/jsonopt"
	"github.com/leancodebox/GooseForum/app/bundles/ratelimit"
	"github.com/leancodebox/GooseForum/app/http/middleware"
	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
)

// setupMcpRouteTestDB migrates the page_config table the MCP settings live in
// and clears both the table and the hotdataserve cache so each test starts
// from a clean admin-panel configuration.
func setupMcpRouteTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate page_config: %v", err)
	}
	conn.Where("page_type = ?", pageConfig.MCPSettings).Delete(&pageConfig.Entity{})
	hotdataserve.ClearMCPSettingsConfigCache()
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.MCPSettings).Delete(&pageConfig.Entity{})
		hotdataserve.ClearMCPSettingsConfigCache()
	})
}

func writeMCPSettings(t *testing.T, cfg pageConfig.MCPSettingsConfig) {
	t.Helper()
	conn := db.Connect()
	conn.Where("page_type = ?", pageConfig.MCPSettings).Delete(&pageConfig.Entity{})
	if err := conn.Create(&pageConfig.Entity{PageType: pageConfig.MCPSettings, Config: jsonopt.Encode(cfg)}).Error; err != nil {
		t.Fatalf("write mcp settings: %v", err)
	}
	hotdataserve.ClearMCPSettingsConfigCache()
}

// TestMcpRouteAlwaysRegistered asserts the MCP endpoint is always registered
// when RegisterByGin is used (the production assembly), independent of the
// apiRoute-only router the agent tests drive. The gin route tree is immutable
// at runtime, so enabled/disabled is enforced per request (404) instead of at
// registration time.
func TestMcpRouteAlwaysRegistered(t *testing.T) {
	setupMcpRouteTestDB(t)
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

// TestMcpRouteDisabledReturns404 asserts that when the admin-panel MCP
// setting has enabled=false, the always-registered /mcp endpoint answers 404
// and exposes no MCP surface.
func TestMcpRouteDisabledReturns404(t *testing.T) {
	setupMcpRouteTestDB(t)
	writeMCPSettings(t, pageConfig.MCPSettingsConfig{Enabled: false, Writes: false})
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterByGin(router)

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		req := httptest.NewRequest(method, "/mcp", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s /mcp with enabled=false: status = %d, want 404", method, rec.Code)
		}
	}
}

// TestMcpRouteEnabledServes asserts that with enabled=true the /mcp endpoint
// is reachable: an unauthenticated request must be answered by the MCP
// bearer-auth layer (401) rather than the 404 disabled guard.
func TestMcpRouteEnabledServes(t *testing.T) {
	setupMcpRouteTestDB(t)
	writeMCPSettings(t, pageConfig.MCPSettingsConfig{Enabled: true, Writes: false})
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterByGin(router)

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("POST /mcp with enabled=true (no token): status = %d, want 401 from bearer auth", rec.Code)
	}
}

// setMcpAuthRateLimit persists a rate-limit config where the mcp.auth action
// allows only limitPerIp requests per minute per client IP, so tests can
// observe the 429 without flooding the default quota. The previous config row
// (if any) is restored on cleanup.
func setMcpAuthRateLimit(t *testing.T, limitPerIp int) {
	t.Helper()
	conn := db.Connect()
	var prev pageConfig.Entity
	hasPrev := conn.Where("page_type = ?", pageConfig.RateLimitSettings).First(&prev).Error == nil
	cfg := pageConfig.RateLimitConfig{
		Enabled:   true,
		SkipAdmin: false,
		Actions: []pageConfig.RateLimitRule{
			{Action: middleware.RateLimitMCPAuth, WindowSeconds: 60, LimitPerIp: limitPerIp, LimitPerUser: 0},
		},
	}
	conn.Where("page_type = ?", pageConfig.RateLimitSettings).Delete(&pageConfig.Entity{})
	if err := conn.Create(&pageConfig.Entity{PageType: pageConfig.RateLimitSettings, Config: jsonopt.Encode(cfg)}).Error; err != nil {
		t.Fatalf("write mcp.auth rate limit: %v", err)
	}
	hotdataserve.ClearRateLimitConfigCache()
	ratelimit.Default().ResetAll()
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.RateLimitSettings).Delete(&pageConfig.Entity{})
		if hasPrev {
			conn.Create(&prev)
		}
		hotdataserve.ClearRateLimitConfigCache()
		ratelimit.Default().ResetAll()
	})
}

// TestMcpRouteAuthFailureRateLimited asserts /mcp is protected by the shared
// mcp.auth per-IP rate limit. Every failed bearer attempt otherwise triggers a
// ResolveByToken DB query, so unauthenticated floods must be bounded by the
// same ratelimit store as the REST stack; the quota is charged before the MCP
// handler runs, keyed by gin's trusted-proxy-resolved client IP.
func TestMcpRouteAuthFailureRateLimited(t *testing.T) {
	setupMcpRouteTestDB(t)
	writeMCPSettings(t, pageConfig.MCPSettingsConfig{Enabled: true, Writes: false})
	setMcpAuthRateLimit(t, 3)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterByGin(router)

	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		want := http.StatusUnauthorized
		if i == 3 {
			want = http.StatusTooManyRequests
		}
		if rec.Code != want {
			t.Fatalf("request %d: status = %d, want %d", i+1, rec.Code, want)
		}
	}
}

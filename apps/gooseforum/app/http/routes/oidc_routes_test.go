package routes

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
	"github.com/leancodebox/GooseForum/app/bundles/ratelimit"
	"github.com/leancodebox/GooseForum/app/http/middleware"
	"github.com/leancodebox/GooseForum/app/models/forum/oidcAuthRequests"
	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
)

func setupOIDCRoutes(t *testing.T, limit int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	conn := db.Connect()
	if err := conn.AutoMigrate(&oidcAuthRequests.Entity{}, &pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate OIDC route test tables: %v", err)
	}

	preferences.Set("oidc.enabled", true)
	preferences.Set("oidc.issuer", "https://forum.example.com/api/oauth")
	preferences.Set("oidc.signing_key", "")
	preferences.Set("oidc.signing_key_file", filepath.Join(t.TempDir(), "signing_key.pem"))
	preferences.Set("oidc.clients", []map[string]any{{
		"id":            "web-client",
		"name":          "Web Client",
		"secret":        "web-secret",
		"redirect_uris": []any{"https://example.com/callback"},
	}})
	persistHTTPContractConfig(t, conn, pageConfig.RateLimitSettings, pageConfig.RateLimitConfig{
		Enabled: true,
		Actions: []pageConfig.RateLimitRule{
			{Action: middleware.RateLimitLogin, WindowSeconds: 60, LimitPerIp: limit},
			{Action: middleware.RateLimitOIDCAuthorize, WindowSeconds: 60, LimitPerIp: limit},
			{Action: middleware.RateLimitOIDCToken, WindowSeconds: 60, LimitPerIp: limit},
		},
	})
	hotdataserve.ClearRateLimitConfigCache()
	ratelimit.Default().ResetAll()
	t.Cleanup(func() {
		preferences.Set("oidc.enabled", false)
		preferences.Set("oidc.issuer", "")
		preferences.Set("oidc.signing_key", "")
		preferences.Set("oidc.clients", nil)
		conn.Where("page_type = ?", pageConfig.RateLimitSettings).Delete(&pageConfig.Entity{})
		hotdataserve.ClearRateLimitConfigCache()
		ratelimit.Default().ResetAll()
	})

	router := gin.New()
	apiRoute(router)
	return router
}

func oidcAuthorizeTarget() string {
	return "/api/oauth/authorize?client_id=web-client&redirect_uri=https%3A%2F%2Fexample.com%2Fcallback&response_type=code&scope=openid&state=state&nonce=nonce&code_challenge=7Z4R6pM5omWBJXqB4rwDDXcYuTVqWk0STbYJc1Jv4WM&code_challenge_method=S256"
}

func TestOIDCRoutesCoexistWithForumOAuthBindings(t *testing.T) {
	router := setupOIDCRoutes(t, 10)
	registered := map[string]bool{}
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	for _, route := range []string{
		"GET /api/oauth/.well-known/openid-configuration",
		"GET /api/oauth/authorize",
		"GET /api/oauth/authorize/callback",
		"GET /api/oauth/token",
		"POST /api/oauth/token",
		"POST /api/oauth/userinfo",
		"GET /api/oauth/userinfo",
		"GET /api/oauth/keys",
		"GET /api/oauth/bindings",
	} {
		if !registered[route] {
			t.Fatalf("%s was not registered", route)
		}
	}
}

func TestOIDCAuthorizeRateLimitRunsBeforePersistence(t *testing.T) {
	router := setupOIDCRoutes(t, 1)
	conn := db.Connect()
	var before int64
	if err := conn.Model(&oidcAuthRequests.Entity{}).Count(&before).Error; err != nil {
		t.Fatalf("count auth requests before test: %v", err)
	}

	first := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodGet, oidcAuthorizeTarget(), nil)
	firstRequest.RemoteAddr = "203.0.113.10:1234"
	router.ServeHTTP(first, firstRequest)
	if first.Code != http.StatusFound {
		t.Fatalf("first authorize status = %d, want 302 (body: %s)", first.Code, first.Body.String())
	}

	second := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodGet, oidcAuthorizeTarget(), nil)
	secondRequest.RemoteAddr = "203.0.113.10:5678"
	router.ServeHTTP(second, secondRequest)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second authorize status = %d, want 429 (body: %s)", second.Code, second.Body.String())
	}

	var after int64
	if err := conn.Model(&oidcAuthRequests.Entity{}).Count(&after).Error; err != nil {
		t.Fatalf("count auth requests after test: %v", err)
	}
	if after != before+1 {
		t.Fatalf("persisted auth requests = %d, want exactly one new row", after-before)
	}
}

func TestOIDCAuthorizeDoesNotConsumeLoginQuota(t *testing.T) {
	router := setupOIDCRoutes(t, 1)

	authorize := httptest.NewRecorder()
	authorizeRequest := httptest.NewRequest(http.MethodGet, oidcAuthorizeTarget(), nil)
	authorizeRequest.RemoteAddr = "203.0.113.20:1234"
	router.ServeHTTP(authorize, authorizeRequest)
	if authorize.Code != http.StatusFound {
		t.Fatalf("authorize status = %d, want 302", authorize.Code)
	}

	login := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader("{}"))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRequest.RemoteAddr = "203.0.113.20:5678"
	router.ServeHTTP(login, loginRequest)
	if login.Code == http.StatusTooManyRequests {
		t.Fatal("authorize request consumed the password-login quota")
	}
}

func TestOIDCTokenRateLimit(t *testing.T) {
	router := setupOIDCRoutes(t, 1)

	first := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodPost, "/api/oauth/token", strings.NewReader("grant_type=authorization_code&code=invalid"))
	firstRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	firstRequest.RemoteAddr = "203.0.113.30:1234"
	router.ServeHTTP(first, firstRequest)
	if first.Code == http.StatusTooManyRequests {
		t.Fatal("first token request was unexpectedly rate limited")
	}

	second := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodPost, "/api/oauth/token", strings.NewReader("grant_type=authorization_code&code=invalid"))
	secondRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	secondRequest.RemoteAddr = "203.0.113.30:5678"
	router.ServeHTTP(second, secondRequest)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second token status = %d, want 429 (body: %s)", second.Code, second.Body.String())
	}
}

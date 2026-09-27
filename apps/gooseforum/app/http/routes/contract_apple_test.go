package routes

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAppleRoutesContractAndAuthenticationGuards(t *testing.T) {
	conn, _ := setupHTTPContractTest(t)
	router := gin.New()
	apiRoute(router)
	user := createHTTPContractUser(t, conn, 895321)
	token := contractSessionToken(t, user)
	const body = `{"authorizationCode":"single-use","identityToken":"signed-token","nonce":"01234567890123456789012345678901"}`
	for _, item := range []struct {
		path, token string
		status      int
		fixture     string
	}{
		{"/api/auth/apple/exchange", "", 503, "apple-unavailable.json"},
		{"/api/auth/apple/bind", "", 401, "auth-required.json"},
		{"/api/auth/apple/bind", token, 503, "apple-unavailable.json"},
	} {
		w := serveAuthSecurityJSON(router, "POST", item.path, body, item.token)
		if w.Code != item.status {
			t.Fatalf("%s status=%d body=%s", item.path, w.Code, w.Body)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, w), contractFixture(t, item.fixture))
	}
	request := httptest.NewRequest(http.MethodPost, "http://localhost/api/auth/apple/bind", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://other.example")
	request.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, request)
	if w.Code != 403 {
		t.Fatalf("cross-site binding status=%d body=%s", w.Code, w.Body)
	}

	persistHTTPContractConfig(t, conn, pageConfig.RateLimitSettings, pageConfig.RateLimitConfig{Enabled: true, Actions: []pageConfig.RateLimitRule{{Action: middleware.RateLimitLogin, WindowSeconds: 60, LimitPerIp: 1}}})
	hotdataserve.ClearRateLimitConfigCache()
	ratelimit.Default().ResetAll()
	t.Cleanup(func() { hotdataserve.ClearRateLimitConfigCache(); ratelimit.Default().ResetAll() })
	first := serveAuthSecurityJSON(router, "POST", "/api/auth/apple/exchange", body, "")
	second := serveAuthSecurityJSON(router, "POST", "/api/auth/apple/exchange", body, "")
	if first.Code != 503 || second.Code != 429 {
		t.Fatalf("rate limit statuses=%d,%d", first.Code, second.Code)
	}
}

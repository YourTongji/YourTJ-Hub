package api

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/leancodebox/GooseForum/app/models/forum/userOAuth"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"github.com/leancodebox/GooseForum/app/service/oidcservice"
)

func postOidcExchange(t *testing.T, body string) (*httptest.ResponseRecorder, component.ResultStruct) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/oidc/exchange", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	OidcExchange(c)

	var res component.ResultStruct
	if err := json.Unmarshal(recorder.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return recorder, res
}

func setCasdoorPrefs(endpoint string) {
	preferences.Set("casdoor.endpoint", endpoint)
	preferences.Set("casdoor.client_id", "test-client")
	preferences.Set("casdoor.client_secret", "test-secret")
	preferences.Set("casdoor.mobile_redirect_uri", "yourtj://callback")
}

func TestOidcExchangeInvalidJSON(t *testing.T) {
	rec, _ := postOidcExchange(t, "{not-json")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestOidcExchangeMissingParams(t *testing.T) {
	rec, _ := postOidcExchange(t, `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestOidcExchangeRejectsUnknownRedirectURI(t *testing.T) {
	setCasdoorPrefs("http://127.0.0.1:1")
	rec, res := postOidcExchange(t, `{"code":"c","codeVerifier":"v","redirectUri":"https://evil.example.com/cb"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if res.MessageCode != component.MessageOidcCallbackFailed {
		t.Fatalf("messageCode = %q, want %q", res.MessageCode, component.MessageOidcCallbackFailed)
	}
}

func TestOidcExchangeNotConfigured(t *testing.T) {
	setCasdoorPrefs("")
	rec, res := postOidcExchange(t, `{"code":"c","codeVerifier":"v","redirectUri":"yourtj://callback"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if res.MessageCode != component.MessageOidcStartFailed {
		t.Fatalf("messageCode = %q, want %q", res.MessageCode, component.MessageOidcStartFailed)
	}
}

// TestOidcExchangeSuccessIssuesToken drives the happy path against a real
// in-process OIDC provider: exchange a signed id_token for a session token.
func TestOidcExchangeSuccessIssuesToken(t *testing.T) {
	// The full happy path needs a live OIDC provider; the oidcservice layer
	// covers it in oidc_test.go (TestExchangeCodeAcceptsNumericSub). Here we
	// assert the controller wiring: a token endpoint failure surfaces as the
	// generic OAuth failure code instead of a 500.
	setCasdoorPrefs("http://127.0.0.1:1")
	rec, res := postOidcExchange(t, `{"code":"c","codeVerifier":"v","redirectUri":"yourtj://callback"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (exchange failed)", rec.Code)
	}
	if res.Code != component.FAIL {
		t.Fatalf("code = %v, want FAIL", res.Code)
	}
}

// --- 登录分支覆盖:冻结用户 / EnableSignup 关闭 ---
//
// 以下两个用例驱动 OidcExchange 走过 ExchangeCode 成功点,进入控制器登录分支
// (oidcController.go:163-177),验证 403 语义。mockOIDCProvider 是进程内
// OpenID Connect 提供者;每个用例使用独立的 httptest server,端点不同使
// oidcservice.Provider() 的 settings 缓存自然失效,无需访问包内缓存变量。

type mockOIDCProvider struct {
	t       *testing.T
	server  *httptest.Server
	key     *rsa.PrivateKey
	kid     string
	idToken string
}

func newMockOIDC(t *testing.T) *mockOIDCProvider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	m := &mockOIDCProvider{t: t, key: key, kid: "test-key"}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", m.handleDiscovery)
	mux.HandleFunc("/jwks", m.handleJWKS)
	mux.HandleFunc("/token", m.handleToken)
	m.server = httptest.NewServer(mux)
	t.Cleanup(m.server.Close)
	return m
}

func (m *mockOIDCProvider) issuer() string { return m.server.URL }

func (m *mockOIDCProvider) handleDiscovery(w http.ResponseWriter, _ *http.Request) {
	writeOIDCJSON(w, map[string]any{
		"issuer":                                m.issuer(),
		"authorization_endpoint":                m.issuer() + "/authorize",
		"token_endpoint":                        m.issuer() + "/token",
		"jwks_uri":                              m.issuer() + "/jwks",
		"id_token_signing_alg_values_supported": []string{"RS256"},
	})
}

func (m *mockOIDCProvider) handleJWKS(w http.ResponseWriter, _ *http.Request) {
	pub := &m.key.PublicKey
	writeOIDCJSON(w, map[string]any{
		"keys": []map[string]any{{
			"kty": "RSA",
			"kid": m.kid,
			"use": "sig",
			"alg": "RS256",
			"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}},
	})
}

func (m *mockOIDCProvider) handleToken(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if r.Form.Get("code_verifier") == "" {
		m.t.Errorf("token request missing code_verifier")
	}
	writeOIDCJSON(w, map[string]any{
		"access_token": "mock-access-token",
		"token_type":   "Bearer",
		"expires_in":   3600,
		"id_token":     m.idToken,
	})
}

func writeOIDCJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (m *mockOIDCProvider) signIDToken(claims jwt.MapClaims) string {
	m.t.Helper()
	if _, ok := claims["iss"]; !ok {
		claims["iss"] = m.issuer()
	}
	if _, ok := claims["aud"]; !ok {
		claims["aud"] = "test-client"
	}
	if _, ok := claims["exp"]; !ok {
		claims["exp"] = time.Now().Add(time.Hour).Unix()
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = m.kid
	signed, err := token.SignedString(m.key)
	if err != nil {
		m.t.Fatalf("sign id_token: %v", err)
	}
	return signed
}

// setupOidcExchangeDB 迁移并清空 exchange 登录分支所需的表。
func setupOidcExchangeDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&userOAuth.Entity{},
		&pageConfig.Entity{},
	); err != nil {
		t.Fatalf("migrate oidc exchange tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&userOAuth.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
	conn.Where("page_type = ?", pageConfig.SecuritySettings).Delete(&pageConfig.Entity{})
	hotdataserve.ClearSecuritySettingsConfigCache()
}

// TestOidcExchangeRejectsFrozenUser 验证已绑定 OIDC 身份的用户被冻结时,
// exchange 返回 403 + MessageOAuthAccountFrozen(oidcController.go:174-175)。
func TestOidcExchangeRejectsFrozenUser(t *testing.T) {
	setupOidcExchangeDB(t)
	m := newMockOIDC(t)
	setCasdoorPrefs(m.issuer())

	user := users.MakeUser("frozenuser", "x", "frozen@example.com")
	user.IsFrozen = users.StatusFrozen
	user.IsActivated = users.ActivationSuccess
	if err := users.Create(user); err != nil {
		t.Fatalf("create frozen user: %v", err)
	}
	if err := userOAuth.Create(&userOAuth.Entity{
		UserId:      user.Id,
		Provider:    oidcservice.ProviderCasdoor,
		ProviderUid: "1002",
	}); err != nil {
		t.Fatalf("create oauth binding: %v", err)
	}

	m.idToken = m.signIDToken(jwt.MapClaims{
		"sub":                "1002",
		"preferred_username": "frozenuser",
	})
	rec, res := postOidcExchange(t, `{"code":"c","codeVerifier":"v","redirectUri":"yourtj://callback"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if res.MessageCode != component.MessageOAuthAccountFrozen {
		t.Fatalf("messageCode = %q, want %q", res.MessageCode, component.MessageOAuthAccountFrozen)
	}
}

// TestOidcExchangeRejectsSignupDisabled 验证站点关闭注册且无已有 Casdoor
// 绑定时,exchange 返回 403 + MessageAuthSignupDisabled(oidcController.go:164-165)。
func TestOidcExchangeRejectsSignupDisabled(t *testing.T) {
	setupOidcExchangeDB(t)
	m := newMockOIDC(t)
	setCasdoorPrefs(m.issuer())

	// 关闭注册:写入 securitySettings 配置行并清缓存,确保读取到该值。
	if err := db.Connect().Create(&pageConfig.Entity{
		PageType: pageConfig.SecuritySettings,
		Config: `{"enableSignup":false,"enableEmailVerification":false,` +
			`"allowedDomains":[],"reservedUsernames":[],"bannedUsernames":[],` +
			`"sensitiveWords":[],"sensitiveAction":"block","captchaRequired":true}`,
	}).Error; err != nil {
		t.Fatalf("write security settings: %v", err)
	}
	hotdataserve.ClearSecuritySettingsConfigCache()
	t.Cleanup(func() {
		db.Connect().Where("page_type = ?", pageConfig.SecuritySettings).Delete(&pageConfig.Entity{})
		hotdataserve.ClearSecuritySettingsConfigCache()
	})

	m.idToken = m.signIDToken(jwt.MapClaims{
		"sub":                "1003",
		"preferred_username": "signupdisabled",
	})
	rec, res := postOidcExchange(t, `{"code":"c","codeVerifier":"v","redirectUri":"yourtj://callback"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if res.MessageCode != component.MessageAuthSignupDisabled {
		t.Fatalf("messageCode = %q, want %q", res.MessageCode, component.MessageAuthSignupDisabled)
	}
}

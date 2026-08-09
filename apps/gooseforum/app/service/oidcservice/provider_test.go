package oidcservice

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/jwtopt"
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
	"github.com/leancodebox/GooseForum/app/models/forum/oidcAccessTokens"
	"github.com/leancodebox/GooseForum/app/models/forum/oidcAuthRequests"
	"github.com/leancodebox/GooseForum/app/models/forum/pointsRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/role"
	"github.com/leancodebox/GooseForum/app/models/forum/rolePermissionRs"
	"github.com/leancodebox/GooseForum/app/models/forum/userPoints"
	"github.com/leancodebox/GooseForum/app/models/forum/userSessions"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/sessionservice"
	"github.com/leancodebox/GooseForum/app/service/userservice"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

// setupProviderConfig 配置内建 OIDC Provider（每个用例独立 issuer 触发重建）。
func setupProviderConfig(t *testing.T, issuer string, clients []map[string]any) {
	t.Helper()
	preferences.Set("oidc.enabled", true)
	preferences.Set("oidc.issuer", issuer)
	preferences.Set("oidc.signing_key", "")
	preferences.Set("oidc.signing_key_file", filepath.Join(t.TempDir(), "signing_key.pem"))
	preferences.Set("oidc.clients", clients)
	// 清理全局 preferences，避免影响其他测试（provider 单例按 configKey 重建）。
	t.Cleanup(func() {
		preferences.Set("oidc.enabled", false)
		preferences.Set("oidc.issuer", "")
		preferences.Set("oidc.signing_key", "")
		preferences.Set("oidc.clients", nil)
	})
}

func defaultClients() []map[string]any {
	return []map[string]any{
		{
			"id":            "yourtj-mobile",
			"name":          "Mobile",
			"redirect_uris": []any{"yourtj://callback"},
		},
		{
			"id":            "web-client",
			"name":          "Web Client",
			"secret":        "web-secret",
			"redirect_uris": []any{"https://example.com/callback"},
		},
	}
}

func pkcePair(t *testing.T) (verifier, challenge string) {
	t.Helper()
	raw := []byte("test-verifier-0123456789-abcdefghijklmnopqrstuvwxyz")
	sum := sha256.Sum256(raw)
	return string(raw), base64.RawURLEncoding.EncodeToString(sum[:])
}

func authorizeURL(issuer string, clientID, redirectURI, state, nonce, verifier string) string {
	challenge := ""
	if verifier != "" {
		sum := sha256.Sum256([]byte(verifier))
		challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	}
	values := url.Values{}
	values.Set("client_id", clientID)
	values.Set("redirect_uri", redirectURI)
	values.Set("response_type", "code")
	values.Set("scope", "openid profile email")
	values.Set("state", state)
	values.Set("nonce", nonce)
	values.Set("code_challenge", challenge)
	values.Set("code_challenge_method", "S256")
	return issuer + "/authorize?" + values.Encode()
}

func doGet(t *testing.T, h http.Handler, target string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func doPostForm(t *testing.T, h http.Handler, target string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// loginCookie 创建真实论坛会话并返回 access_token cookie（模拟已登录用户）。
func loginCookie(t *testing.T, username string) (*http.Cookie, users.EntityComplete) {
	t.Helper()
	conn := db.Connect()
	models := []any{
		&users.EntityComplete{},
		&userSessions.Entity{},
		&userStatistics.Entity{},
		&userPoints.Entity{},
		&pointsRecord.Entity{},
		&role.Entity{},
		&rolePermissionRs.Entity{},
		&oidcAuthRequests.Entity{},
		&oidcAccessTokens.Entity{},
	}
	for _, model := range models {
		if err := conn.AutoMigrate(model); err != nil {
			t.Fatalf("migrate %T: %v", model, err)
		}
	}
	user, err := userservice.CreateUser(username, "password123", username+"@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	token, jti, err := jwtopt.CreateSessionToken(user.Id, user.TokenVersion)
	if err != nil {
		t.Fatalf("CreateSessionToken: %v", err)
	}
	if err := sessionservice.Create(user.Id, jti, "test-agent", "127.0.0.1"); err != nil {
		t.Fatalf("sessionservice.Create: %v", err)
	}
	return &http.Cookie{Name: "access_token", Value: token}, *user
}

// startAuthorize 发起 authorize 并返回 (authRequestID, bindingCookie, loginRedirectLocation)。
// 登录重定向格式为 /login?redirect=<urlencoded /api/oauth/authorize/callback?id=...>，
// authRequestID 位于 redirect 参数内部；浏览器绑定 cookie 由 Set-Cookie 下发。
func startAuthorize(t *testing.T, h http.Handler, target string) (string, *http.Cookie, string) {
	t.Helper()
	rec := doGet(t, h, target)
	if rec.Code != http.StatusFound {
		t.Fatalf("authorize status = %d, want 302 (body: %s)", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse location %q: %v", location, err)
	}
	redirectTarget := parsed.Query().Get("redirect")
	callback, err := url.Parse(redirectTarget)
	if err != nil || callback.Path != "/api/oauth/authorize/callback" {
		t.Fatalf("login redirect does not point to oauth callback: %q", location)
	}
	id := callback.Query().Get("id")
	if id == "" {
		t.Fatalf("login redirect missing id: %q", location)
	}
	var binding *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == browserBindingCookieName {
			binding = c
			break
		}
	}
	if binding == nil {
		t.Fatalf("authorize did not set browser binding cookie")
	}
	return id, binding, location
}

// completeAuthorize 走完登录桥（携带登录态 cookie 与浏览器绑定 cookie）并返回授权码。
func completeAuthorize(t *testing.T, h http.Handler, issuer, requestID string, cookies ...*http.Cookie) string {
	t.Helper()
	callbackTarget := issuer + "/authorize/callback?id=" + url.QueryEscape(requestID)
	rec := doGet(t, h, callbackTarget, cookies...)
	if rec.Code != http.StatusFound {
		t.Fatalf("callback status = %d, want 302 (body: %s)", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse callback location %q: %v", location, err)
	}
	code := parsed.Query().Get("code")
	if code == "" {
		t.Fatalf("callback location missing code: %q", location)
	}
	return code
}

func exchangeToken(t *testing.T, h http.Handler, tokenURL, code, redirectURI, clientID, secret, verifier string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", clientID)
	form.Set("code_verifier", verifier)
	if secret != "" {
		form.Set("client_secret", secret)
	}
	rec := doPostForm(t, h, tokenURL, form)
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec, body
}

// --- discovery 广告面 ---

func TestDiscoveryAdvertisesOnlyImplementedSurface(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	rec := doGet(t, h, issuer+"/.well-known/openid-configuration")
	if rec.Code != http.StatusOK {
		t.Fatalf("discovery status = %d", rec.Code)
	}
	var cfg oidc.DiscoveryConfiguration
	if err := json.Unmarshal(rec.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("decode discovery: %v", err)
	}
	if cfg.Issuer != issuer {
		t.Fatalf("issuer = %q, want %q", cfg.Issuer, issuer)
	}
	if len(cfg.GrantTypesSupported) != 1 || cfg.GrantTypesSupported[0] != oidc.GrantTypeCode {
		t.Fatalf("grant_types = %v, want [authorization_code]", cfg.GrantTypesSupported)
	}
	if len(cfg.ResponseTypesSupported) != 1 || cfg.ResponseTypesSupported[0] != string(oidc.ResponseTypeCode) {
		t.Fatalf("response_types = %v, want [code]", cfg.ResponseTypesSupported)
	}
	if cfg.RegistrationEndpoint != "" {
		t.Fatalf("registration_endpoint must be empty, got %q", cfg.RegistrationEndpoint)
	}
	if len(cfg.CodeChallengeMethodsSupported) != 1 || cfg.CodeChallengeMethodsSupported[0] != oidc.CodeChallengeMethodS256 {
		t.Fatalf("code_challenge_methods = %v, want [S256]", cfg.CodeChallengeMethodsSupported)
	}
	if cfg.EndSessionEndpoint != "" {
		t.Fatalf("end_session_endpoint must be empty, got %q", cfg.EndSessionEndpoint)
	}
}

func TestTokenEndpointRejectsGet(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	rec := doGet(t, h, issuer+"/token?grant_type=authorization_code&code=secret-code&code_verifier=secret-verifier")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET token status = %d, want 405", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodPost {
		t.Fatalf("Allow = %q, want POST", allow)
	}
}

// --- authorize 校验 ---

func TestAuthorizeRejectsUnknownClient(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	_, verifier := pkcePair(t)
	target := authorizeURL(issuer, "unknown-client", "https://example.com/callback", "st", "no", verifier)
	rec := doGet(t, h, target)
	if rec.Code == http.StatusFound {
		t.Fatalf("authorize with unknown client redirected; must not follow any redirect")
	}
}

func TestAuthorizeRejectsRedirectMismatch(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	_, verifier := pkcePair(t)
	// redirect_uri 必须精确匹配注册值，不能发生 open redirect
	target := authorizeURL(issuer, "web-client", "https://evil.example.com/callback", "st", "no", verifier)
	rec := doGet(t, h, target)
	if rec.Code == http.StatusFound {
		loc := rec.Header().Get("Location")
		if strings.Contains(loc, "https://evil.example.com") {
			t.Fatalf("authorize redirected to unregistered URI: %q", loc)
		}
	}
}

func TestAuthorizeRequiresPKCE(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	// 缺 code_challenge
	target := authorizeURL(issuer, "web-client", "https://example.com/callback", "st", "no", "")
	rec := doGet(t, h, target)
	if rec.Code == http.StatusFound && !strings.Contains(rec.Header().Get("Location"), "error") {
		t.Fatalf("authorize without PKCE must not reach login redirect: %q", rec.Header().Get("Location"))
	}
}

func TestAuthorizeRequiresStateAndNonce(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	_, verifier := pkcePair(t)
	values := url.Values{}
	values.Set("client_id", "web-client")
	values.Set("redirect_uri", "https://example.com/callback")
	values.Set("response_type", "code")
	values.Set("scope", "openid")
	values.Set("code_challenge", verifier)
	values.Set("code_challenge_method", "S256")
	// 无 state：库将错误重定向回注册的 redirect_uri（标准行为），
	// 但绝不能进入 /login 登录桥。
	rec := doGet(t, h, issuer+"/authorize?"+values.Encode())
	if strings.HasPrefix(rec.Header().Get("Location"), "/login") {
		t.Fatalf("authorize without state must not redirect to login: %q", rec.Header().Get("Location"))
	}
}

// --- 完整授权码流程 + claims ---

func TestFullCodeFlowIssuesOpaqueTokenAndIDToken(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	cookie, _ := loginCookie(t, "oidcflowuser")
	verifier, _ := pkcePair(t)
	state := "state-abc"
	nonce := "nonce-xyz"

	requestID, binding, loginLoc := startAuthorize(t, h, authorizeURL(issuer, "web-client", "https://example.com/callback", state, nonce, verifier))
	if !strings.HasPrefix(loginLoc, "/login?redirect=") {
		t.Fatalf("login redirect = %q, want /login?redirect=...", loginLoc)
	}

	code := completeAuthorize(t, h, issuer, requestID, cookie, binding)
	rec, body := exchangeToken(t, h, issuer+"/token", code, "https://example.com/callback", "web-client", "web-secret", verifier)
	if rec.Code != http.StatusOK {
		t.Fatalf("token status = %d, body: %s", rec.Code, rec.Body.String())
	}
	accessToken, _ := body["access_token"].(string)
	idToken, _ := body["id_token"].(string)
	if accessToken == "" || idToken == "" {
		t.Fatalf("token response missing access_token/id_token: %v", body)
	}
	if body["token_type"] != "Bearer" {
		t.Fatalf("token_type = %v, want Bearer", body["token_type"])
	}
}

func TestFullCodeFlowIDTokenClaims(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	cookie, _ := loginCookie(t, "claimuser")
	verifier, _ := pkcePair(t)
	state := "state-claims"
	nonce := "nonce-claims"

	requestID, binding, _ := startAuthorize(t, h, authorizeURL(issuer, "web-client", "https://example.com/callback", state, nonce, verifier))
	code := completeAuthorize(t, h, issuer, requestID, cookie, binding)
	_, body := exchangeToken(t, h, issuer+"/token", code, "https://example.com/callback", "web-client", "web-secret", verifier)
	idToken, _ := body["id_token"].(string)
	if idToken == "" {
		t.Fatalf("missing id_token: %v", body)
	}
	// 用持久 RSA 公钥验证 RS256 + iss/aud/sub/nonce claims
	claims := verifyIDToken(t, idToken)
	if claims["iss"] != issuer {
		t.Fatalf("iss = %v, want %q", claims["iss"], issuer)
	}
	// aud 是 OIDC 标准数组形式（单元素）
	audList, ok := claims["aud"].([]any)
	if !ok || len(audList) != 1 || audList[0] != "web-client" {
		t.Fatalf("aud = %v, want [web-client]", claims["aud"])
	}
	if _, ok := claims["sub"].(string); !ok {
		t.Fatalf("sub missing: %v", claims["sub"])
	}
	if claims["nonce"] != nonce {
		t.Fatalf("nonce = %v, want %q", claims["nonce"], nonce)
	}
}

func TestSetUserinfoFromRequestMapsGrantedClaims(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	_, user := loginCookie(t, "userinfohook")
	provider, err := Provider()
	if err != nil {
		t.Fatalf("Provider() error = %v", err)
	}
	st, ok := provider.Storage().(*storage)
	if !ok {
		t.Fatalf("provider storage = %T, want *storage", provider.Storage())
	}
	request := &authRequest{entity: &oidcAuthRequests.Entity{
		ClientId: "web-client",
		Subject:  user.Id,
	}}
	userinfo := new(oidc.UserInfo)
	if err := st.SetUserinfoFromRequest(
		context.Background(),
		userinfo,
		request,
		[]string{oidc.ScopeProfile, oidc.ScopeEmail},
	); err != nil {
		t.Fatalf("SetUserinfoFromRequest() error = %v", err)
	}
	wantSubject := strconv.FormatUint(user.Id, 10)
	if userinfo.Subject != wantSubject {
		t.Fatalf("userinfo subject = %q, want %q", userinfo.Subject, wantSubject)
	}
	wantName := user.Nickname
	if wantName == "" {
		wantName = user.Username
	}
	if userinfo.PreferredUsername != user.Username || userinfo.Name != wantName {
		t.Fatalf("userinfo profile = username %q, name %q; want username %q, name %q", userinfo.PreferredUsername, userinfo.Name, user.Username, wantName)
	}
	if userinfo.Email != user.Email {
		t.Fatalf("userinfo email = %q, want %q", userinfo.Email, user.Email)
	}
	wantVerified := oidc.Bool(user.IsActivated == users.ActivationSuccess)
	if userinfo.EmailVerified != wantVerified {
		t.Fatalf("userinfo email_verified = %v, want %v", userinfo.EmailVerified, wantVerified)
	}
}

func TestCodeSingleUseReplayRejected(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	cookie, _ := loginCookie(t, "replayuser")
	verifier, _ := pkcePair(t)

	requestID, binding, _ := startAuthorize(t, h, authorizeURL(issuer, "web-client", "https://example.com/callback", "st", "no", verifier))
	code := completeAuthorize(t, h, issuer, requestID, cookie, binding)

	first, _ := exchangeToken(t, h, issuer+"/token", code, "https://example.com/callback", "web-client", "web-secret", verifier)
	if first.Code != http.StatusOK {
		t.Fatalf("first token status = %d, body: %s", first.Code, first.Body.String())
	}

	second, _ := exchangeToken(t, h, issuer+"/token", code, "https://example.com/callback", "web-client", "web-secret", verifier)
	if second.Code == http.StatusOK {
		t.Fatalf("replayed code must be rejected, got 200: %s", second.Body.String())
	}
}

func TestWrongPKCEVerifierRejected(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	cookie, _ := loginCookie(t, "pkcuser")
	verifier, _ := pkcePair(t)

	requestID, binding, _ := startAuthorize(t, h, authorizeURL(issuer, "web-client", "https://example.com/callback", "st", "no", verifier))
	code := completeAuthorize(t, h, issuer, requestID, cookie, binding)

	rec, _ := exchangeToken(t, h, issuer+"/token", code, "https://example.com/callback", "web-client", "web-secret", "wrong-verifier-value")
	if rec.Code == http.StatusOK {
		t.Fatalf("wrong PKCE verifier must be rejected, got 200")
	}
}

// --- userinfo ---

func TestUserinfoHappyPathAndForumJWTRejected(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	cookie, _ := loginCookie(t, "userinfouser")
	verifier, _ := pkcePair(t)

	requestID, binding, _ := startAuthorize(t, h, authorizeURL(issuer, "web-client", "https://example.com/callback", "st", "no", verifier))
	code := completeAuthorize(t, h, issuer, requestID, cookie, binding)
	_, body := exchangeToken(t, h, issuer+"/token", code, "https://example.com/callback", "web-client", "web-secret", verifier)
	accessToken, _ := body["access_token"].(string)
	if accessToken == "" {
		t.Fatalf("missing access_token")
	}

	// 有效 opaque token → userinfo 返回 sub
	req := httptest.NewRequest(http.MethodGet, issuer+"/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("userinfo status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var info map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatalf("decode userinfo: %v", err)
	}
	if info["sub"] == "" {
		t.Fatalf("userinfo missing sub: %v", info)
	}

	// forum HS256 JWT 绝不能作为 OIDC access token 使用
	req2 := httptest.NewRequest(http.MethodGet, issuer+"/userinfo", nil)
	req2.Header.Set("Authorization", "Bearer "+cookie.Value)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code == http.StatusOK {
		t.Fatalf("userinfo must reject forum JWT, got 200")
	}
}

func TestUserinfoRespectsTokenVersionAndFrozen(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	cookie, user := loginCookie(t, "versionuser")
	verifier, _ := pkcePair(t)

	requestID, binding, _ := startAuthorize(t, h, authorizeURL(issuer, "web-client", "https://example.com/callback", "st", "no", verifier))
	code := completeAuthorize(t, h, issuer, requestID, cookie, binding)
	_, body := exchangeToken(t, h, issuer+"/token", code, "https://example.com/callback", "web-client", "web-secret", verifier)
	accessToken, _ := body["access_token"].(string)

	userinfo := func() int {
		req := httptest.NewRequest(http.MethodGet, issuer+"/userinfo", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	if code := userinfo(); code != http.StatusOK {
		t.Fatalf("userinfo pre-change status = %d", code)
	}

	// 改密/吊销全部设备 → TokenVersion 递增 → userinfo 拒绝
	users.IncrementTokenVersion(user.Id)
	if code := userinfo(); code == http.StatusOK {
		t.Fatalf("userinfo must reject token after TokenVersion bump")
	}

	// 冻结 → userinfo 拒绝
	_ = db.Connect().Model(&users.EntityComplete{}).Where("id = ?", user.Id).Update("is_frozen", users.StatusFrozen)
	if code := userinfo(); code == http.StatusOK {
		t.Fatalf("userinfo must reject frozen user")
	}
}

func TestUserinfoRespectsRevocation(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	cookie, _ := loginCookie(t, "revokeuser")
	verifier, _ := pkcePair(t)

	requestID, binding, _ := startAuthorize(t, h, authorizeURL(issuer, "web-client", "https://example.com/callback", "st", "no", verifier))
	code := completeAuthorize(t, h, issuer, requestID, cookie, binding)
	_, body := exchangeToken(t, h, issuer+"/token", code, "https://example.com/callback", "web-client", "web-secret", verifier)
	accessToken, _ := body["access_token"].(string)

	// 解密 bearer token 提取 tokenID 并标记撤销
	provider, err := Provider()
	if err != nil {
		t.Fatalf("Provider() error = %v", err)
	}
	plain, err := provider.Crypto().Decrypt(accessToken)
	if err != nil {
		t.Fatalf("decrypt access token: %v", err)
	}
	parts := strings.Split(plain, ":")
	if len(parts) != 2 {
		t.Fatalf("unexpected token payload: %q", plain)
	}
	if err := oidcAccessTokens.MarkRevoked(parts[0]); err != nil {
		t.Fatalf("mark revoked: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, issuer+"/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("userinfo must reject revoked token, got 200")
	}
}

// --- 登录桥 ---

func TestLoginBridgeRequiresAuthentication(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	verifier, _ := pkcePair(t)
	requestID, _, _ := startAuthorize(t, h, authorizeURL(issuer, "web-client", "https://example.com/callback", "st", "no", verifier))

	// 未登录访问 callback → 重定向到登录页（无 open redirect）
	rec := doGet(t, h, issuer+"/authorize/callback?id="+url.QueryEscape(requestID))
	if rec.Code != http.StatusFound {
		t.Fatalf("callback status = %d, want 302", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "/login?redirect=") {
		t.Fatalf("callback redirect = %q, want /login?redirect=", loc)
	}
	if strings.Contains(loc, "//") && !strings.HasPrefix(loc, "/login") {
		t.Fatalf("potential open redirect: %q", loc)
	}
}

// TestLoginBridgeRejectsMissingBinding 验证攻击者浏览器创建 request、
// 受害者登录态访问 callback（无 binding cookie）→ 400 且 request 未完成。
func TestLoginBridgeRejectsMissingBinding(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	cookie, _ := loginCookie(t, "victimbinding")
	verifier, _ := pkcePair(t)

	// 攻击者浏览器发起 authorize（获得 requestID 与攻击者的 binding cookie），
	// 但受害者浏览器只有登录态 cookie、没有 binding cookie。
	requestID, attackerBinding, _ := startAuthorize(t, h, authorizeURL(issuer, "web-client", "https://example.com/callback", "st", "no", verifier))
	_ = attackerBinding

	rec := doGet(t, h, issuer+"/authorize/callback?id="+url.QueryEscape(requestID), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("callback status = %d, want 400 (missing binding)", rec.Code)
	}

	// request 必须未完成：仍可被攻击者自己的 binding 完成（否则说明已被受害者完成）
	entity := oidcAuthRequests.GetByRequestId(requestID)
	if entity == nil {
		t.Fatalf("auth request row disappeared")
	}
	if entity.Done {
		t.Fatalf("auth request must not be completed without matching binding")
	}
	if entity.Subject != 0 {
		t.Fatalf("auth request subject must stay 0, got %d", entity.Subject)
	}
}

// TestLoginBridgeRejectsMismatchedBinding 验证攻击者浏览器创建 request、
// 受害者登录态 + 不同 binding cookie 访问 callback → 400 且 request 未完成。
func TestLoginBridgeRejectsMismatchedBinding(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	cookie, _ := loginCookie(t, "victimmismatch")
	verifier, _ := pkcePair(t)

	requestID, _, _ := startAuthorize(t, h, authorizeURL(issuer, "web-client", "https://example.com/callback", "st", "no", verifier))

	// 受害者浏览器带自己的 binding cookie（与 request 记录的 hash 不同）
	otherBinding := &http.Cookie{Name: browserBindingCookieName, Value: "victim-own-binding-value"}
	rec := doGet(t, h, issuer+"/authorize/callback?id="+url.QueryEscape(requestID), cookie, otherBinding)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("callback status = %d, want 400 (binding mismatch)", rec.Code)
	}
	entity := oidcAuthRequests.GetByRequestId(requestID)
	if entity == nil || entity.Done || entity.Subject != 0 {
		t.Fatalf("auth request must not be completed on binding mismatch")
	}
}

// TestBrowserBindingCookieAttributes 验证 binding cookie 属性：
// HttpOnly、SameSite=Lax、Path=/api/oauth、https issuer 时 Secure。
func TestBrowserBindingCookieAttributes(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	verifier, _ := pkcePair(t)
	rec := doGet(t, h, authorizeURL(issuer, "web-client", "https://example.com/callback", "st", "no", verifier))
	if rec.Code != http.StatusFound {
		t.Fatalf("authorize status = %d, want 302", rec.Code)
	}
	var binding *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == browserBindingCookieName {
			binding = c
			break
		}
	}
	if binding == nil {
		t.Fatal("browser binding cookie not set")
	}
	if !binding.HttpOnly {
		t.Fatal("binding cookie must be HttpOnly")
	}
	if binding.SameSite != http.SameSiteLaxMode {
		t.Fatalf("binding cookie SameSite = %v, want Lax", binding.SameSite)
	}
	if binding.Path != issuerPath {
		t.Fatalf("binding cookie Path = %q, want %q", binding.Path, issuerPath)
	}
	if !binding.Secure {
		t.Fatal("binding cookie must be Secure for https issuer")
	}
	if binding.Value == "" {
		t.Fatal("binding cookie value must not be empty")
	}
}

// TestBrowserBindingCookieMaxAgeFollowsAuthRequestTTL 验证 binding cookie 的
// MaxAge 跟随 oidc.auth_request_ttl（而非写死 10 分钟）。
func TestBrowserBindingCookieMaxAgeFollowsAuthRequestTTL(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	originalTTL := preferences.GetInt64("oidc.auth_request_ttl", int64(defaultAuthRequestTTL/time.Second))
	preferences.Set("oidc.auth_request_ttl", 42)
	t.Cleanup(func() { preferences.Set("oidc.auth_request_ttl", originalTTL) })
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	verifier, _ := pkcePair(t)
	rec := doGet(t, h, authorizeURL(issuer, "web-client", "https://example.com/callback", "st", "no", verifier))
	if rec.Code != http.StatusFound {
		t.Fatalf("authorize status = %d, want 302", rec.Code)
	}
	var binding *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == browserBindingCookieName {
			binding = c
			break
		}
	}
	if binding == nil {
		t.Fatal("browser binding cookie not set")
	}
	if binding.MaxAge != 42 {
		t.Fatalf("binding cookie MaxAge = %d, want 42 (auth_request_ttl)", binding.MaxAge)
	}
}

// TestCreateAuthRequestCleansExpiredRows 验证创建授权请求前会顺手清理过期行，
// 避免过期授权请求只等到启动/每日 cron 才被清除。
func TestCreateAuthRequestCleansExpiredRows(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	conn := db.Connect()
	if err := conn.AutoMigrate(&oidcAuthRequests.Entity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	expired := &oidcAuthRequests.Entity{
		RequestId:    "expired-row",
		ClientId:     "web-client",
		ResponseType: "code",
		ExpiresAt:    time.Now().Add(-time.Minute),
	}
	if err := oidcAuthRequests.Create(expired); err != nil {
		t.Fatalf("seed expired row: %v", err)
	}
	verifier, _ := pkcePair(t)
	rec := doGet(t, h, authorizeURL(issuer, "web-client", "https://example.com/callback", "st", "no", verifier))
	if rec.Code != http.StatusFound {
		t.Fatalf("authorize status = %d, want 302", rec.Code)
	}
	if entity := oidcAuthRequests.GetByRequestId("expired-row"); entity != nil {
		t.Fatal("expired auth request row must be cleaned up on new authorize")
	}
}

// TestDeriveCryptoKeyStickyFallback 验证无 app.signingKey 时回退加密密钥在
// 进程内保持稳定（provider 重建不更换 opaque-token 加密密钥）。
func TestDeriveCryptoKeyStickyFallback(t *testing.T) {
	original := preferences.GetString("app.signingKey", "")
	preferences.Set("app.signingKey", "")
	t.Cleanup(func() { preferences.Set("app.signingKey", original) })
	first := deriveCryptoKey()
	second := deriveCryptoKey()
	if first != second {
		t.Fatal("fallback crypto key must be sticky within the process")
	}
	if first == ([32]byte{}) {
		t.Fatal("fallback crypto key must not be zero")
	}
}

func TestCompleteLoginValidatesRequest(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&oidcAuthRequests.Entity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := CompleteLogin("", 1, time.Now(), "hash"); err == nil {
		t.Fatal("CompleteLogin with empty requestID must fail")
	}
	if err := CompleteLogin("missing-id", 1, time.Now(), "hash"); err == nil {
		t.Fatal("CompleteLogin with unknown requestID must fail")
	}
}

// TestExchangeCodeFullSuccess 覆盖 mobile 完整路径：
// mobile client authorize（public client + PKCE）→ 登录桥完成 →
// ExchangeCode 兑换，验证 nonce/client/redirect/PKCE 与 auth request 删除。
func TestExchangeCodeFullSuccess(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	cookie, _ := loginCookie(t, "mobileuser")
	verifier, _ := pkcePair(t)
	nonce := "mobile-nonce"

	// mobile client 发起 authorize（yourtj-mobile public client，redirect yourtj://callback）
	requestID, binding, _ := startAuthorize(t, h, authorizeURL(issuer, MobileClientID, "yourtj://callback", "st", nonce, verifier))
	code := completeAuthorize(t, h, issuer, requestID, cookie, binding)

	result, err := ExchangeCode(code, verifier, nonce, "yourtj://callback")
	if err != nil {
		t.Fatalf("ExchangeCode() error = %v", err)
	}
	if result.Sub == 0 {
		t.Fatalf("ExchangeCode() sub = 0, want the logged-in user")
	}
	// auth request 行必须已删除（同 token 端点语义）
	if entity := oidcAuthRequests.GetByRequestId(requestID); entity != nil {
		t.Fatalf("auth request row must be deleted after exchange, still present")
	}
}

// TestExchangeCodeRejectsWrongNonce 验证 nonce 不匹配拒绝兑换。
func TestExchangeCodeRejectsWrongNonce(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	h, err := Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	cookie, _ := loginCookie(t, "nonceuser")
	verifier, _ := pkcePair(t)

	requestID, binding, _ := startAuthorize(t, h, authorizeURL(issuer, MobileClientID, "yourtj://callback", "st", "real-nonce", verifier))
	code := completeAuthorize(t, h, issuer, requestID, cookie, binding)

	if _, err := ExchangeCode(code, verifier, "wrong-nonce", "yourtj://callback"); err == nil {
		t.Fatalf("ExchangeCode with wrong nonce must fail")
	}
}

// TestConfigKeyChangesWithTTLAndSecret 验证 configKey 覆盖 TTL、DevMode、
// inline key/client secret hash 与 names/redirects（敏感值不明文出现）。
func TestConfigKeyChangesWithTTLAndSecret(t *testing.T) {
	base := Config{
		Enabled:        true,
		Issuer:         "https://forum.example.com/api/oauth",
		KeyFile:        "/k.pem",
		KeyPEM:         "private-key-a",
		AccessTokenTTL: 3600,
		AuthRequestTTL: 600,
		IDTokenTTL:     3600,
		Clients: []ClientConfig{
			{ID: "c1", Name: "Client One", Secret: "secret-a", RedirectURIs: []string{"https://a/cb"}},
		},
	}
	baseKey := configKey(base)

	// TTL 变化 → key 变化
	ttlChanged := base
	ttlChanged.AccessTokenTTL = 7200
	if configKey(ttlChanged) == baseKey {
		t.Fatal("configKey must change when AccessTokenTTL changes")
	}

	// DevMode 变化 → key 变化
	devChanged := base
	devChanged.Clients[0].DevMode = true
	if configKey(devChanged) == baseKey {
		t.Fatal("configKey must change when DevMode changes")
	}

	// secret 变化 → key 变化（hash 参与）
	secretChanged := base
	secretChanged.Clients[0].Secret = "secret-b"
	if configKey(secretChanged) == baseKey {
		t.Fatal("configKey must change when client secret changes")
	}

	// inline private key 变化 → key 变化（hash 参与）
	keyPEMChanged := base
	keyPEMChanged.KeyPEM = "private-key-b"
	if configKey(keyPEMChanged) == baseKey {
		t.Fatal("configKey must change when inline private key changes")
	}

	// name 变化 → key 变化
	nameChanged := base
	nameChanged.Clients[0].Name = "Client Two"
	if configKey(nameChanged) == baseKey {
		t.Fatal("configKey must change when client name changes")
	}

	// redirect 变化 → key 变化
	redirectChanged := base
	redirectChanged.Clients[0].RedirectURIs = []string{"https://b/cb"}
	if configKey(redirectChanged) == baseKey {
		t.Fatal("configKey must change when redirect URIs change")
	}

	// 敏感值明文绝不出现在 key 中
	if strings.Contains(baseKey, "secret-a") || strings.Contains(baseKey, "secret-b") ||
		strings.Contains(baseKey, "private-key-a") || strings.Contains(baseKey, "private-key-b") {
		t.Fatalf("configKey must not contain plaintext secrets: %q", baseKey)
	}
}

func TestLoadConfigDefaultsNonPositiveAuthRequestTTL(t *testing.T) {
	issuer := "https://forum.example.com/api/oauth"
	setupProviderConfig(t, issuer, defaultClients())
	originalTTL := preferences.GetInt64("oidc.auth_request_ttl", int64(defaultAuthRequestTTL/time.Second))
	preferences.Set("oidc.auth_request_ttl", 0)
	t.Cleanup(func() { preferences.Set("oidc.auth_request_ttl", originalTTL) })

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.AuthRequestTTL != defaultAuthRequestTTL {
		t.Fatalf("AuthRequestTTL = %v, want %v", cfg.AuthRequestTTL, defaultAuthRequestTTL)
	}
}

func TestAddLoopbackPort(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		port int
		want string
	}{
		{name: "localhost", raw: "http://localhost", port: 5234, want: "http://localhost:5234"},
		{name: "IPv4 loopback", raw: "http://127.0.0.1", port: 5234, want: "http://127.0.0.1:5234"},
		{name: "IPv6 loopback", raw: "http://[::1]", port: 5234, want: "http://[::1]:5234"},
		{name: "preserve path", raw: "http://localhost/base", port: 5234, want: "http://localhost:5234/base"},
		{name: "preserve explicit port", raw: "http://localhost:8080", port: 5234, want: "http://localhost:8080"},
		{name: "preserve public host", raw: "https://forum.example.com", port: 5234, want: "https://forum.example.com"},
		{name: "preserve invalid port", raw: "http://localhost", port: 0, want: "http://localhost"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := addLoopbackPort(test.raw, test.port); got != test.want {
				t.Fatalf("addLoopbackPort(%q, %d) = %q, want %q", test.raw, test.port, got, test.want)
			}
		})
	}
}

// TestValidateIssuer 验证 issuer 校验规则。
func TestValidateIssuer(t *testing.T) {
	valid := []string{
		"https://forum.example.com/api/oauth",
		"http://localhost:5234/api/oauth",
		"http://127.0.0.1:5234/api/oauth",
		"http://[::1]:5234/api/oauth",
	}
	for _, issuer := range valid {
		if err := validateIssuer(issuer); err != nil {
			t.Fatalf("validateIssuer(%q) = %v, want nil", issuer, err)
		}
	}
	invalid := []string{
		"http://forum.example.com/api/oauth",      // http 非 loopback
		"https://forum.example.com/other",         // path 错误
		"https://forum.example.com/api/oauth?x=1", // query
		"https://forum.example.com/api/oauth#f",   // fragment
		"ftp://forum.example.com/api/oauth",       // scheme 错误
		"",                                        // 空
	}
	for _, issuer := range invalid {
		if err := validateIssuer(issuer); err == nil {
			t.Fatalf("validateIssuer(%q) = nil, want error", issuer)
		}
	}
}

// --- helpers ---

// verifyIDToken 用持久 RSA 公钥验证 RS256 签名并返回 claims。
func verifyIDToken(t *testing.T, idToken string) map[string]any {
	t.Helper()
	parsed, err := jose.ParseSigned(idToken, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		t.Fatalf("parse id_token: %v", err)
	}
	if len(parsed.Signatures) != 1 {
		t.Fatalf("signatures = %d, want 1", len(parsed.Signatures))
	}
	kid := parsed.Signatures[0].Header.KeyID
	provider, err := Provider()
	if err != nil {
		t.Fatalf("Provider() error = %v", err)
	}
	st, ok := provider.Storage().(*storage)
	if !ok {
		t.Fatalf("provider storage = %T, want *storage", provider.Storage())
	}
	if kid == "" || kid != st.km.KeyID() {
		t.Fatalf("kid = %q, want %q", kid, st.km.KeyID())
	}
	payload, err := parsed.Verify(st.km.PublicKey())
	if err != nil {
		t.Fatalf("verify id_token signature: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("decode claims: %v", err)
	}
	return claims
}

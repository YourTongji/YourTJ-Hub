package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/jwtopt"
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/oidcAccessTokens"
	"github.com/leancodebox/GooseForum/app/models/forum/oidcAuthRequests"
	"github.com/leancodebox/GooseForum/app/models/forum/pointsRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/role"
	"github.com/leancodebox/GooseForum/app/models/forum/rolePermissionRs"
	"github.com/leancodebox/GooseForum/app/models/forum/userPoints"
	"github.com/leancodebox/GooseForum/app/models/forum/userSessions"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/oidcservice"
	"github.com/leancodebox/GooseForum/app/service/sessionservice"
	"github.com/leancodebox/GooseForum/app/service/userservice"
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
	preferences.Set("oidc.enabled", true)
	preferences.Set("oidc.signing_key_file", filepath.Join(t.TempDir(), "key.pem"))
	rec, res := postOidcExchange(t, `{"code":"c","codeVerifier":"v","nonce":"n","redirectUri":"https://evil.example.com/cb"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if res.MessageCode != component.MessageOidcCallbackFailed {
		t.Fatalf("messageCode = %q, want %q", res.MessageCode, component.MessageOidcCallbackFailed)
	}
}

func TestOidcExchangeNotConfigured(t *testing.T) {
	preferences.Set("oidc.enabled", false)
	rec, res := postOidcExchange(t, `{"code":"c","codeVerifier":"v","nonce":"n","redirectUri":"yourtj://callback"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if res.MessageCode != component.MessageOidcStartFailed {
		t.Fatalf("messageCode = %q, want %q", res.MessageCode, component.MessageOidcStartFailed)
	}
}

// TestOidcExchangeRejectsInvalidCode 验证伪造/未知授权码被拒绝（401）。
func TestOidcExchangeRejectsInvalidCode(t *testing.T) {
	preferences.Set("oidc.enabled", true)
	preferences.Set("oidc.signing_key_file", filepath.Join(t.TempDir(), "key.pem"))
	preferences.Set("oidc.issuer", "http://127.0.0.1:1/api/oauth")
	rec, res := postOidcExchange(t, `{"code":"stale-code","codeVerifier":"v","nonce":"n","redirectUri":"yourtj://callback"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if res.Code != component.FAIL {
		t.Fatalf("code = %v, want FAIL", res.Code)
	}
}

// --- 真实 provider flow helper ---

const testIssuer = "https://forum.example.com/api/oauth"

func setupOIDCProviderTestDB(t *testing.T) {
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
	preferences.Set("oidc.enabled", true)
	preferences.Set("oidc.issuer", testIssuer)
	preferences.Set("oidc.signing_key", "")
	preferences.Set("oidc.signing_key_file", filepath.Join(t.TempDir(), "signing_key.pem"))
	preferences.Set("oidc.clients", []map[string]any{
		{"id": "yourtj-mobile", "name": "Mobile", "redirect_uris": []any{"yourtj://callback"}},
	})
}

func loginUserCookie(t *testing.T, username string) *http.Cookie {
	t.Helper()
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
	return &http.Cookie{Name: "access_token", Value: token}
}

// obtainMobileCode 走真实 provider 流程：mobile authorize → binding callback →
// 返回 (code, verifier, nonce)。
func obtainMobileCode(t *testing.T, cookie *http.Cookie) (code, verifier, nonce string) {
	t.Helper()
	handler, err := oidcservice.Router()
	if err != nil {
		t.Fatalf("Router() error = %v", err)
	}
	verifier = "test-verifier-0123456789-abcdefghijklmnopqrstuvwxyz"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	nonce = "mobile-nonce"

	values := url.Values{}
	values.Set("client_id", "yourtj-mobile")
	values.Set("redirect_uri", "yourtj://callback")
	values.Set("response_type", "code")
	values.Set("scope", "openid profile email")
	values.Set("state", "st")
	values.Set("nonce", nonce)
	values.Set("code_challenge", challenge)
	values.Set("code_challenge_method", "S256")
	target := testIssuer + "/authorize?" + values.Encode()

	authReq := httptest.NewRequest(http.MethodGet, target, nil)
	authRec := httptest.NewRecorder()
	handler.ServeHTTP(authRec, authReq)
	if authRec.Code != http.StatusFound {
		t.Fatalf("authorize status = %d, want 302", authRec.Code)
	}
	loc, err := url.Parse(authRec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse login redirect: %v", err)
	}
	callbackTarget := loc.Query().Get("redirect")
	callbackURL, err := url.Parse(callbackTarget)
	if err != nil || callbackURL.Path != "/api/oauth/authorize/callback" {
		t.Fatalf("login redirect does not point to oauth callback: %q", callbackTarget)
	}
	requestID := callbackURL.Query().Get("id")
	if requestID == "" {
		t.Fatalf("login redirect missing id")
	}
	var binding *http.Cookie
	for _, c := range authRec.Result().Cookies() {
		if c.Name == "yourtj_oidc_binding" {
			binding = c
			break
		}
	}
	if binding == nil {
		t.Fatalf("authorize did not set browser binding cookie")
	}

	cbReq := httptest.NewRequest(http.MethodGet, testIssuer+"/authorize/callback?id="+url.QueryEscape(requestID), nil)
	cbReq.AddCookie(cookie)
	cbReq.AddCookie(binding)
	cbRec := httptest.NewRecorder()
	handler.ServeHTTP(cbRec, cbReq)
	if cbRec.Code != http.StatusFound {
		t.Fatalf("callback status = %d, want 302 (body: %s)", cbRec.Code, cbRec.Body.String())
	}
	cbLoc, err := url.Parse(cbRec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse callback location: %v", err)
	}
	code = cbLoc.Query().Get("code")
	if code == "" {
		t.Fatalf("callback location missing code")
	}
	return code, verifier, nonce
}

// TestOidcExchangeSuccessIssuesForumJWT 验证真实 provider 完整路径下
// controller 返回 200 + forum JWT + session row。
func TestOidcExchangeSuccessIssuesForumJWT(t *testing.T) {
	setupOIDCProviderTestDB(t)
	cookie := loginUserCookie(t, "exchangeuser")

	code, verifier, nonce := obtainMobileCode(t, cookie)
	body, _ := json.Marshal(map[string]string{
		"code":         code,
		"codeVerifier": verifier,
		"nonce":        nonce,
		"redirectUri":  "yourtj://callback",
	})
	rec, res := postOidcExchange(t, string(body))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	token, _ := res.Result.(map[string]any)["token"].(string)
	if token == "" {
		t.Fatalf("response missing token: %v", res.Result)
	}
	// forum JWT 必须可验证且映射到 session row
	userID, _, _, ok := validateForumToken(t, token)
	if !ok || userID == 0 {
		t.Fatalf("issued forum JWT is not a valid session token")
	}
}

// TestOidcExchangeRejectsFrozenUser 真实 provider 流程兑换成功后，
// 冻结用户 → controller 返回 403 + MessageOAuthAccountFrozen。
func TestOidcExchangeRejectsFrozenUser(t *testing.T) {
	setupOIDCProviderTestDB(t)
	cookie := loginUserCookie(t, "frozenuser")

	code, verifier, nonce := obtainMobileCode(t, cookie)

	// 冻结用户（兑换成功后 controller 的冻结检查生效）
	claims, _, err := jwtopt.VerifyTokenWithFreshClaims(cookie.Value)
	if err != nil {
		t.Fatalf("verify login cookie: %v", err)
	}
	if err := db.Connect().Model(&users.EntityComplete{}).Where("id = ?", claims.UserId).Update("is_frozen", users.StatusFrozen).Error; err != nil {
		t.Fatalf("freeze user: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"code":         code,
		"codeVerifier": verifier,
		"nonce":        nonce,
		"redirectUri":  "yourtj://callback",
	})
	rec, res := postOidcExchange(t, string(body))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body: %s)", rec.Code, rec.Body.String())
	}
	if res.MessageCode != component.MessageOAuthAccountFrozen {
		t.Fatalf("messageCode = %q, want %q", res.MessageCode, component.MessageOAuthAccountFrozen)
	}
}

// validateForumToken 复刻 authsessionservice 校验（避免 import cycle 不便）。
func validateForumToken(t *testing.T, token string) (uint64, string, string, bool) {
	t.Helper()
	if token == "" {
		return 0, "", "", false
	}
	claims, refreshed, err := jwtopt.VerifyTokenWithFreshClaims(token)
	if err != nil {
		return 0, "", "", false
	}
	user, ok := userservice.GetUserInfo(claims.UserId)
	if !ok || user.TokenVersion != claims.TokenVersion {
		return 0, "", "", false
	}
	if claims.Jti == "" {
		return 0, "", "", false
	}
	entity := sessionservice.GetValidByJti(claims.Jti)
	if entity == nil || entity.UserId != claims.UserId {
		return 0, "", "", false
	}
	_ = strings.TrimSpace(refreshed)
	return claims.UserId, claims.Jti, refreshed, true
}

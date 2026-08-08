package oidcservice

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
	"github.com/leancodebox/GooseForum/app/models/forum/pointsRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/role"
	"github.com/leancodebox/GooseForum/app/models/forum/rolePermissionRs"
	"github.com/leancodebox/GooseForum/app/models/forum/userOAuth"
	"github.com/leancodebox/GooseForum/app/models/forum/userPoints"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
)

const (
	testClientID     = "test-client"
	testClientSecret = "test-secret"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// mockOIDC is an in-process OpenID Connect provider for tests.
type mockOIDC struct {
	t         *testing.T
	server    *httptest.Server
	key       *rsa.PrivateKey
	kid       string
	idToken   string
	failToken bool
}

func newMockOIDC(t *testing.T) *mockOIDC {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	m := &mockOIDC{t: t, key: key, kid: "test-key"}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", m.handleDiscovery)
	mux.HandleFunc("/jwks", m.handleJWKS)
	mux.HandleFunc("/token", m.handleToken)
	m.server = httptest.NewServer(mux)
	t.Cleanup(m.server.Close)
	return m
}

func (m *mockOIDC) issuer() string { return m.server.URL }

func (m *mockOIDC) handleDiscovery(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"issuer":                                m.issuer(),
		"authorization_endpoint":                m.issuer() + "/authorize",
		"token_endpoint":                        m.issuer() + "/token",
		"jwks_uri":                              m.issuer() + "/jwks",
		"id_token_signing_alg_values_supported": []string{"RS256"},
	})
}

func (m *mockOIDC) handleJWKS(w http.ResponseWriter, _ *http.Request) {
	pub := &m.key.PublicKey
	writeJSON(w, map[string]any{
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

func (m *mockOIDC) handleToken(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if r.Form.Get("code_verifier") == "" {
		m.t.Errorf("token request missing code_verifier")
	}
	if m.failToken {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]any{"error": "invalid_grant"})
		return
	}
	writeJSON(w, map[string]any{
		"access_token": "mock-access-token",
		"token_type":   "Bearer",
		"expires_in":   3600,
		"id_token":     m.idToken,
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (m *mockOIDC) signIDToken(claims jwt.MapClaims) string {
	m.t.Helper()
	if _, ok := claims["iss"]; !ok {
		claims["iss"] = m.issuer()
	}
	if _, ok := claims["aud"]; !ok {
		claims["aud"] = testClientID
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

func configureOIDC(endpoint string) {
	preferences.Set("casdoor.endpoint", endpoint)
	preferences.Set("casdoor.client_id", testClientID)
	preferences.Set("casdoor.client_secret", testClientSecret)
	providerMu.Lock()
	providerLoaded = false
	cachedProvider = nil
	cachedProviderErr = nil
	providerMu.Unlock()
}

func newGinContext(t *testing.T, method, target string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, target, nil)
	return c, w
}

func startLogin(t *testing.T) (string, *http.Cookie) {
	t.Helper()
	c, w := newGinContext(t, http.MethodGet, "/api/auth/oidc/login")
	authURL, err := StartLogin(c)
	if err != nil {
		t.Fatalf("StartLogin() error = %v", err)
	}
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == sessionName {
			return authURL, cookie
		}
	}
	t.Fatal("no oidc session cookie in response")
	return "", nil
}

func callbackContext(t *testing.T, target string, cookie *http.Cookie) *gin.Context {
	t.Helper()
	c, _ := newGinContext(t, http.MethodGet, target)
	if cookie != nil {
		c.Request.AddCookie(cookie)
	}
	return c
}

// runCallback drives the full flow: StartLogin, then HandleCallback with the
// id_token signed from claims. A nonce from the auth URL is added unless the
// claims already carry one.
func runCallback(t *testing.T, m *mockOIDC, claims jwt.MapClaims) (CallbackResult, error) {
	t.Helper()
	authURL, cookie := startLogin(t)
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}
	state := u.Query().Get("state")
	if state == "" {
		t.Fatal("auth url missing state")
	}
	if _, ok := claims["nonce"]; !ok {
		if nonce := u.Query().Get("nonce"); nonce != "" {
			claims["nonce"] = nonce
		}
	}
	m.idToken = m.signIDToken(claims)
	target := "/api/auth/oidc/callback?state=" + url.QueryEscape(state) + "&code=test-code"
	return HandleCallback(callbackContext(t, target, cookie))
}

func TestHandleCallbackAcceptsNumericSub(t *testing.T) {
	m := newMockOIDC(t)
	configureOIDC(m.issuer())

	result, err := runCallback(t, m, jwt.MapClaims{
		"sub":                "123456",
		"preferred_username": "alice",
		"name":               "Alice",
		"email":              "alice@example.com",
	})
	if err != nil {
		t.Fatalf("HandleCallback() error = %v", err)
	}
	if result.Sub != 123456 {
		t.Fatalf("Sub = %d, want 123456", result.Sub)
	}
	if result.Username != "alice" || result.Email != "alice@example.com" {
		t.Fatalf("result = %+v", result)
	}
}

func TestHandleCallbackRejectsNonNumericSub(t *testing.T) {
	m := newMockOIDC(t)
	configureOIDC(m.issuer())

	_, err := runCallback(t, m, jwt.MapClaims{
		"sub": "6f1c9b0e-8c2a-4b1e-9d3f-2a5c7e8f1a2b",
	})
	if !errors.Is(err, ErrNonNumericSub) {
		t.Fatalf("error = %v, want ErrNonNumericSub", err)
	}
}

func TestHandleCallbackRejectsZeroSub(t *testing.T) {
	m := newMockOIDC(t)
	configureOIDC(m.issuer())

	_, err := runCallback(t, m, jwt.MapClaims{
		"sub": "0",
	})
	if !errors.Is(err, ErrNonNumericSub) {
		t.Fatalf("error = %v, want ErrNonNumericSub", err)
	}
}

func TestHandleCallbackRejectsNonceMismatch(t *testing.T) {
	m := newMockOIDC(t)
	configureOIDC(m.issuer())

	_, err := runCallback(t, m, jwt.MapClaims{
		"sub":   "123456",
		"nonce": "attacker-nonce",
	})
	if !errors.Is(err, ErrNonceMismatch) {
		t.Fatalf("error = %v, want ErrNonceMismatch", err)
	}
}

func TestHandleCallbackRejectsIssuerMismatch(t *testing.T) {
	m := newMockOIDC(t)
	configureOIDC(m.issuer())

	_, err := runCallback(t, m, jwt.MapClaims{
		"sub": "123456",
		"iss": "https://evil.example.com",
	})
	if err == nil || errors.Is(err, ErrNonNumericSub) || errors.Is(err, ErrNonceMismatch) {
		t.Fatalf("error = %v, want id_token verification failure", err)
	}
}

func TestHandleCallbackRejectsAudienceMismatch(t *testing.T) {
	m := newMockOIDC(t)
	configureOIDC(m.issuer())

	_, err := runCallback(t, m, jwt.MapClaims{
		"sub": "123456",
		"aud": "another-client",
	})
	if err == nil || errors.Is(err, ErrNonNumericSub) || errors.Is(err, ErrNonceMismatch) {
		t.Fatalf("error = %v, want id_token verification failure", err)
	}
}

func TestHandleCallbackRejectsStateMismatch(t *testing.T) {
	m := newMockOIDC(t)
	configureOIDC(m.issuer())

	authURL, cookie := startLogin(t)
	u, _ := url.Parse(authURL)
	m.idToken = m.signIDToken(jwt.MapClaims{
		"sub":   "123456",
		"nonce": u.Query().Get("nonce"),
	})
	_, err := HandleCallback(callbackContext(t, "/api/auth/oidc/callback?state=wrong-state&code=test-code", cookie))
	if !errors.Is(err, ErrStateMismatch) {
		t.Fatalf("error = %v, want ErrStateMismatch", err)
	}
}

func TestStartLoginRequiresConfig(t *testing.T) {
	preferences.Set("casdoor.endpoint", "")
	preferences.Set("casdoor.client_id", "")
	preferences.Set("casdoor.client_secret", "")

	c, _ := newGinContext(t, http.MethodGet, "/api/auth/oidc/login")
	_, err := StartLogin(c)
	if !errors.Is(err, ErrOIDCNotConfigured) {
		t.Fatalf("error = %v, want ErrOIDCNotConfigured", err)
	}
}

// --- MatchOrCreateUser (DB-backed) ---

func setupUserDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	models := []any{
		&users.EntityComplete{},
		&userOAuth.Entity{},
		&userStatistics.Entity{},
		&userPoints.Entity{},
		&pointsRecord.Entity{},
		&role.Entity{},
		&rolePermissionRs.Entity{},
	}
	for _, model := range models {
		if err := conn.AutoMigrate(model); err != nil {
			t.Fatalf("migrate %T: %v", model, err)
		}
	}
	for _, model := range models {
		conn.Where("1 = 1").Delete(model)
	}
}

func TestMatchOrCreateUserCreatesNewAccount(t *testing.T) {
	setupUserDB(t)

	user, err := MatchOrCreateUser(CallbackResult{Sub: 1001, Username: "casdoor-alice", Email: "alice@example.com"})
	if err != nil {
		t.Fatalf("MatchOrCreateUser() error = %v", err)
	}
	if user.Username != "casdoor-alice" {
		t.Fatalf("Username = %q, want casdoor-alice", user.Username)
	}
	if user.Email != "alice@example.com" {
		t.Fatalf("Email = %q, want alice@example.com", user.Email)
	}
	oauth := userOAuth.GetByProviderAndUID(ProviderCasdoor, "1001")
	if oauth == nil || oauth.UserId != user.Id {
		t.Fatalf("oauth binding = %+v", oauth)
	}
}

func TestMatchOrCreateUserReturnsExistingBinding(t *testing.T) {
	setupUserDB(t)

	first, err := MatchOrCreateUser(CallbackResult{Sub: 2002, Username: "bob", Email: "bob@example.com"})
	if err != nil {
		t.Fatalf("first MatchOrCreateUser() error = %v", err)
	}
	second, err := MatchOrCreateUser(CallbackResult{Sub: 2002, Username: "bob", Email: ""})
	if err != nil {
		t.Fatalf("second MatchOrCreateUser() error = %v", err)
	}
	if second.Id != first.Id {
		t.Fatalf("user ids differ: %d vs %d", first.Id, second.Id)
	}
}

func TestMatchOrCreateUserAppendsSuffixOnUsernameConflict(t *testing.T) {
	setupUserDB(t)

	if _, err := MatchOrCreateUser(CallbackResult{Sub: 3003, Username: "samename", Email: "s1@example.com"}); err != nil {
		t.Fatalf("first MatchOrCreateUser() error = %v", err)
	}
	second, err := MatchOrCreateUser(CallbackResult{Sub: 3004, Username: "samename", Email: "s2@example.com"})
	if err != nil {
		t.Fatalf("second MatchOrCreateUser() error = %v", err)
	}
	if second.Username != "samename_1" {
		t.Fatalf("Username = %q, want samename_1", second.Username)
	}
}

func TestMatchOrCreateUserSkipsTakenEmail(t *testing.T) {
	setupUserDB(t)

	if _, err := MatchOrCreateUser(CallbackResult{Sub: 4004, Username: "owner", Email: "taken@example.com"}); err != nil {
		t.Fatalf("first MatchOrCreateUser() error = %v", err)
	}
	second, err := MatchOrCreateUser(CallbackResult{Sub: 4005, Username: "other", Email: "taken@example.com"})
	if err != nil {
		t.Fatalf("second MatchOrCreateUser() error = %v", err)
	}
	if second.Email != "" {
		t.Fatalf("Email = %q, want empty (taken)", second.Email)
	}
}

func TestMatchOrCreateUserFallsBackToNumericUsername(t *testing.T) {
	setupUserDB(t)

	user, err := MatchOrCreateUser(CallbackResult{Sub: 5005, Username: "中文名", Email: ""})
	if err != nil {
		t.Fatalf("MatchOrCreateUser() error = %v", err)
	}
	if user.Username != "user5005" {
		t.Fatalf("Username = %q, want user5005", user.Username)
	}
}

// --- ExchangeCode (mobile OIDC) ---

func configureMobileRedirect(t *testing.T, redirectURI string) {
	t.Helper()
	preferences.Set("casdoor.mobile_redirect_uri", redirectURI)
}

func TestExchangeCodeAcceptsNumericSub(t *testing.T) {
	m := newMockOIDC(t)
	configureOIDC(m.issuer())
	configureMobileRedirect(t, "yourtj://callback")

	m.idToken = m.signIDToken(jwt.MapClaims{
		"sub":                "123456",
		"preferred_username": "alice",
		"email":              "alice@example.com",
		"nonce":              "test-nonce",
	})
	result, err := ExchangeCode("test-code", "test-verifier", "test-nonce", "yourtj://callback")
	if err != nil {
		t.Fatalf("ExchangeCode() error = %v", err)
	}
	if result.Sub != 123456 || result.Username != "alice" || result.Email != "alice@example.com" {
		t.Fatalf("result = %+v", result)
	}
}

func TestExchangeCodeRejectsUnknownRedirectURI(t *testing.T) {
	m := newMockOIDC(t)
	configureOIDC(m.issuer())
	configureMobileRedirect(t, "yourtj://callback")

	_, err := ExchangeCode("test-code", "test-verifier", "test-nonce", "https://evil.example.com/callback")
	if !errors.Is(err, ErrInvalidMobileRedirectURI) {
		t.Fatalf("error = %v, want ErrInvalidMobileRedirectURI", err)
	}
}

func TestExchangeCodeRejectsMissingParams(t *testing.T) {
	m := newMockOIDC(t)
	configureOIDC(m.issuer())
	configureMobileRedirect(t, "yourtj://callback")

	if _, err := ExchangeCode("", "test-verifier", "test-nonce", "yourtj://callback"); !errors.Is(err, ErrInvalidExchangeRequest) {
		t.Fatalf("empty code error = %v, want ErrInvalidExchangeRequest", err)
	}
	if _, err := ExchangeCode("test-code", "", "test-nonce", "yourtj://callback"); !errors.Is(err, ErrInvalidExchangeRequest) {
		t.Fatalf("empty verifier error = %v, want ErrInvalidExchangeRequest", err)
	}
}

func TestExchangeCodeRejectsNonNumericSub(t *testing.T) {
	m := newMockOIDC(t)
	configureOIDC(m.issuer())
	configureMobileRedirect(t, "yourtj://callback")

	m.idToken = m.signIDToken(jwt.MapClaims{"sub": "uuid-sub", "nonce": "test-nonce"})
	_, err := ExchangeCode("test-code", "test-verifier", "test-nonce", "yourtj://callback")
	if !errors.Is(err, ErrNonNumericSub) {
		t.Fatalf("error = %v, want ErrNonNumericSub", err)
	}
}

func TestExchangeCodeRejectsNonceMismatch(t *testing.T) {
	m := newMockOIDC(t)
	configureOIDC(m.issuer())
	configureMobileRedirect(t, "yourtj://callback")

	m.idToken = m.signIDToken(jwt.MapClaims{
		"sub":                "123456",
		"preferred_username": "alice",
		"nonce":              "attacker-nonce",
	})
	if _, err := ExchangeCode("test-code", "test-verifier", "test-nonce", "yourtj://callback"); !errors.Is(err, ErrNonceMismatch) {
		t.Fatalf("error = %v, want ErrNonceMismatch", err)
	}
}

func TestExchangeCodeFailsWhenTokenEndpointRejects(t *testing.T) {
	m := newMockOIDC(t)
	configureOIDC(m.issuer())
	configureMobileRedirect(t, "yourtj://callback")
	m.failToken = true

	_, err := ExchangeCode("stale-code", "test-verifier", "test-nonce", "yourtj://callback")
	if err == nil {
		t.Fatal("ExchangeCode() error = nil, want token exchange failure")
	}
}

func TestExchangeCodeRequiresConfig(t *testing.T) {
	preferences.Set("casdoor.endpoint", "")
	preferences.Set("casdoor.client_id", "")
	preferences.Set("casdoor.client_secret", "")
	configureMobileRedirect(t, "yourtj://callback")

	_, err := ExchangeCode("test-code", "test-verifier", "test-nonce", "yourtj://callback")
	if !errors.Is(err, ErrOIDCNotConfigured) {
		t.Fatalf("error = %v, want ErrOIDCNotConfigured", err)
	}
}

func TestMobileRedirectURIDefault(t *testing.T) {
	preferences.Set("casdoor.mobile_redirect_uri", "")
	if got := MobileRedirectURI(); got != "yourtj://callback" {
		t.Fatalf("MobileRedirectURI() = %q, want yourtj://callback", got)
	}
}

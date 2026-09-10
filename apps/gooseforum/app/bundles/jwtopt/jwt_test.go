package jwtopt

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
)

func TestCreateNewToken(t *testing.T) {
	const userID uint64 = 123456
	token, err := CreateNewToken(userID, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	userId, newToken, err := VerifyTokenWithFresh(token)
	if err != nil {
		t.Fatal(err)
	}
	if userId != userID {
		t.Fatalf("VerifyTokenWithFresh userId = %d, want %d", userId, userID)
	}
	if newToken == "" {
		t.Fatal("expected refreshed token")
	}

	userId, err = VerifyToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if userId != userID {
		t.Fatalf("VerifyToken userId = %d, want %d", userId, userID)
	}

	time.Sleep(1100 * time.Millisecond)
	if _, err = VerifyToken(token); err == nil {
		t.Fatal("expected expired token error")
	}
}

func TestCreateNewTokenDefault(t *testing.T) {
	token, err := CreateNewTokenDefault(7)
	if err != nil {
		t.Fatal(err)
	}
	userID, err := VerifyToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if userID != 7 {
		t.Fatalf("userID = %d, want 7", userID)
	}
}

func TestCreateNewTokenWithVersion(t *testing.T) {
	const userID uint64 = 7
	const tokenVersion uint64 = 3

	token, err := CreateNewTokenWithVersion(userID, tokenVersion, 15*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	claims, newToken, err := VerifyTokenWithFreshClaims(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserId != userID || claims.TokenVersion != tokenVersion {
		t.Fatalf("claims = (%d, %d), want (%d, %d)", claims.UserId, claims.TokenVersion, userID, tokenVersion)
	}
	if newToken == token {
		t.Fatal("expected refreshed token")
	}

	refreshedClaims, _, err := VerifyTokenWithFreshClaims(newToken)
	if err != nil {
		t.Fatal(err)
	}
	if refreshedClaims.TokenVersion != tokenVersion {
		t.Fatalf("refreshed tokenVersion = %d, want %d", refreshedClaims.TokenVersion, tokenVersion)
	}
}

func TestGetGinAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	headerRecorder := httptest.NewRecorder()
	headerContext, _ := gin.CreateTestContext(headerRecorder)
	headerContext.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	headerContext.Request.Header.Set("Authorization", "Bearer header-token")
	if got := GetGinAccessToken(headerContext); got != "header-token" {
		t.Fatalf("header token = %q, want header-token", got)
	}

	cookieRecorder := httptest.NewRecorder()
	cookieContext, _ := gin.CreateTestContext(cookieRecorder)
	cookieContext.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	cookieContext.Request.AddCookie(&http.Cookie{Name: "access_token", Value: "cookie-token"})
	if got := GetGinAccessToken(cookieContext); got != "cookie-token" {
		t.Fatalf("cookie token = %q, want cookie-token", got)
	}
}

func TestTokenSettingAndClean(t *testing.T) {
	gin.SetMode(gin.TestMode)

	setRecorder := httptest.NewRecorder()
	setContext, _ := gin.CreateTestContext(setRecorder)
	TokenSetting(setContext, "fresh-token")
	if got := setRecorder.Header().Get("New-Token"); got != "fresh-token" {
		t.Fatalf("New-Token = %q, want fresh-token", got)
	}
	if cookies := setRecorder.Result().Cookies(); len(cookies) != 1 || cookies[0].Name != "access_token" || cookies[0].Value != "fresh-token" {
		t.Fatalf("set cookies = %#v", cookies)
	}

	cleanRecorder := httptest.NewRecorder()
	cleanContext, _ := gin.CreateTestContext(cleanRecorder)
	TokenClean(cleanContext)
	setCookie := cleanRecorder.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "access_token=") || !strings.Contains(setCookie, "Max-Age=0") {
		t.Fatalf("clear cookie header = %q", setCookie)
	}
}

// withEnv restores app.env and server.url after the test so package-level
// viper state does not leak across tests.
func withEnv(t *testing.T, appEnv, serverURL string) {
	t.Helper()
	prevEnv := preferences.GetString("app.env", "production")
	prevURL := preferences.GetString("server.url", "")
	preferences.Set("app.env", appEnv)
	preferences.Set("server.url", serverURL)
	t.Cleanup(func() {
		preferences.Set("app.env", prevEnv)
		preferences.Set("server.url", prevURL)
	})
}

// TestAccessTokenCookieSecureProductionHTTP reproduces issue #113: under the
// template defaults `app.env = "production"` + `server.url = "http://localhost"`
// the access_token cookie must still carry the Secure flag.
func TestAccessTokenCookieSecureProductionHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withEnv(t, "production", "http://localhost")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	TokenSetting(c, "tok")
	for _, ck := range rec.Result().Cookies() {
		if ck.Name != "access_token" {
			continue
		}
		if !ck.Secure {
			t.Fatalf("access_token cookie Secure=false under production+http://localhost, want Secure (CWE-614)")
		}
		if !ck.HttpOnly {
			t.Fatalf("access_token cookie HttpOnly=false")
		}
		if ck.SameSite != http.SameSiteLaxMode {
			t.Fatalf("access_token SameSite = %v, want Lax", ck.SameSite)
		}
	}

	cleanRec := httptest.NewRecorder()
	cleanC, _ := gin.CreateTestContext(cleanRec)
	cleanC.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	TokenClean(cleanC)
	for _, ck := range cleanRec.Result().Cookies() {
		if ck.Name == "access_token" && !ck.Secure {
			t.Fatalf("access_token clear cookie Secure=false under production+http://localhost, want Secure")
		}
	}
}

// TestAccessTokenCookieLocalKeepsSecureOff verifies local dev still drops Secure
// so plain-http 0.0.0.0 / LAN-IP access keeps the cookie.
func TestAccessTokenCookieLocalKeepsSecureOff(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withEnv(t, "local", "http://localhost:5234")

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	TokenSetting(c, "tok")
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == "access_token" && ck.Secure {
			t.Fatalf("access_token Secure=true under local, want false for dev over plain http")
		}
	}
}

func TestVerifyTokenWithFresh(t *testing.T) {
	const userID uint64 = 123456
	token, err := CreateNewToken(123456, 15*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	userId, newToken, err := VerifyTokenWithFresh(token)
	if err != nil {
		t.Fatal(err)
	}
	if userId != userID {
		t.Fatalf("VerifyTokenWithFresh userId = %d, want %d", userId, userID)
	}
	if newToken != token {
		token = newToken
	}

	time.Sleep(time.Second * 2)
	userId, err = VerifyToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if userId != userID {
		t.Fatalf("VerifyToken userId = %d, want %d", userId, userID)
	}
}

func TestGenerateJti(t *testing.T) {
	jti1, err := GenerateJti()
	if err != nil {
		t.Fatal(err)
	}
	jti2, err := GenerateJti()
	if err != nil {
		t.Fatal(err)
	}
	if len(jti1) != 32 {
		t.Fatalf("jti length = %d, want 32", len(jti1))
	}
	if jti1 == jti2 {
		t.Fatal("jti must be unique per call")
	}
}

func TestCreateSessionTokenJtiPreservedOnRefresh(t *testing.T) {
	token, jti, err := CreateSessionToken(42, 1)
	if err != nil {
		t.Fatal(err)
	}
	if jti == "" {
		t.Fatal("expected non-empty jti")
	}

	claims, newToken, err := VerifyTokenWithFreshClaims(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Jti != jti {
		t.Fatalf("claims.Jti = %q, want %q", claims.Jti, jti)
	}
	if claims.Purpose != "" {
		t.Fatalf("session token purpose = %q, want empty", claims.Purpose)
	}
	if newToken != token {
		refreshed, _, err := VerifyTokenWithFreshClaims(newToken)
		if err != nil {
			t.Fatal(err)
		}
		if refreshed.Jti != jti {
			t.Fatalf("refreshed jti = %q, want %q", refreshed.Jti, jti)
		}
	}
}

func TestCreateChallengeTokenPurpose(t *testing.T) {
	token, err := CreateChallengeToken(7, 0, PurposeTotpChallenge, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	claims, _, err := VerifyTokenWithFreshClaims(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Purpose != PurposeTotpChallenge {
		t.Fatalf("purpose = %q, want %q", claims.Purpose, PurposeTotpChallenge)
	}
	if claims.Jti == "" {
		t.Fatal("challenge token should carry a jti")
	}
	exp, err := claims.GetExpirationTime()
	if err != nil {
		t.Fatal(err)
	}
	if exp.Time.Before(time.Now()) {
		t.Fatal("challenge token should not be expired yet")
	}
}

// TestSigningKeyProblemForRejectsWeakKeys is the fail-closed gate for issue #106:
// the serve startup guard rejects any key SigningKeyProblemFor reports, and
// tokenservice shares this predicate to refuse signing reset/activation tokens.
func TestSigningKeyProblemForRejectsWeakKeys(t *testing.T) {
	cases := []struct {
		name string
		key  string
		want string
	}{
		{name: "empty", key: "", want: "empty signing key"},
		{name: "whitespace", key: "   \t\n ", want: "empty signing key"},
		{name: "built-in default", key: DefaultSigningKey, want: "built-in default signing key"},
		{name: "deploy placeholder", key: "REPLACE_SIGNING_KEY", want: "deploy template placeholder signing key"},
		{name: "strong random", key: "8KZx9wq0nJ6v3L2tRbMfYc+UeP1sDhGa", want: ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := SigningKeyProblemFor(tt.key)
			if got != tt.want {
				t.Fatalf("SigningKeyProblemFor(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

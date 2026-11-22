package jwtopt

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/gin-gonic/gin"
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

// withEnv restores app.env, server.url and server.cookieSecure after the test
// so package-level viper state does not leak across tests. preferences.Set
// cannot unset a key, so an absent cookieSecure is restored as "" — which
// resolves exactly like the unset default "auto".
func withEnv(t *testing.T, appEnv, serverURL, cookieSecure string) {
	t.Helper()
	prevEnv := preferences.GetString("app.env", "production")
	prevURL := preferences.GetString("server.url", "")
	prevCookieSecure := preferences.GetString("server.cookieSecure", "")
	preferences.Set("app.env", appEnv)
	preferences.Set("server.url", serverURL)
	preferences.Set("server.cookieSecure", cookieSecure)
	t.Cleanup(func() {
		preferences.Set("app.env", prevEnv)
		preferences.Set("server.url", prevURL)
		preferences.Set("server.cookieSecure", prevCookieSecure)
	})
}

// TestAccessTokenCookieSecureProductionHTTP reproduces issue #113: under the
// template defaults `app.env = "production"` + `server.url = "http://localhost"`
// the access_token cookie must still carry the Secure flag.
func TestAccessTokenCookieSecureProductionHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withEnv(t, "production", "http://localhost", "")

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
	withEnv(t, "local", "http://localhost:5234", "")

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

// TestAccessTokenCookieSecurePlainHTTPHost covers issue #1277: a production
// deployment served over plain http on a LAN/IP host must not set Secure on
// access_token (browsers drop it and web login silently fails), while an
// explicit server.cookieSecure = "on" restores the flag there.
func TestAccessTokenCookieSecurePlainHTTPHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name         string
		serverURL    string
		cookieSecure string
		want         bool
	}{
		{name: "auto http lan ip", serverURL: "http://10.0.0.5:5234", cookieSecure: "auto", want: false},
		{name: "forced on http lan ip", serverURL: "http://10.0.0.5:5234", cookieSecure: "on", want: true},
		{name: "forced off https", serverURL: "https://example.com", cookieSecure: "off", want: false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			withEnv(t, "production", tt.serverURL, tt.cookieSecure)

			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			TokenSetting(c, "tok")

			found := false
			for _, ck := range rec.Result().Cookies() {
				if ck.Name != "access_token" {
					continue
				}
				found = true
				if ck.Secure != tt.want {
					t.Fatalf("access_token Secure = %v (url=%q cookieSecure=%q), want %v", ck.Secure, tt.serverURL, tt.cookieSecure, tt.want)
				}
			}
			if !found {
				t.Fatal("no access_token cookie emitted")
			}
		})
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

func withJWTOptConfig(t *testing.T, validSecs, refreshSecs, maxLifeSecs int64) {
	t.Helper()
	prevValid := preferences.GetInt64("jwtopt.validTime", int64(DefaultValidTime/time.Second))
	prevRefresh := preferences.GetInt64("jwtopt.refreshInterval", int64(DefaultRefreshInterval/time.Second))
	prevMax := preferences.GetInt64("jwtopt.maxSessionLifetime", int64(DefaultMaxSessionLifetime/time.Second))
	preferences.Set("jwtopt.validTime", validSecs)
	preferences.Set("jwtopt.refreshInterval", refreshSecs)
	preferences.Set("jwtopt.maxSessionLifetime", maxLifeSecs)
	t.Cleanup(func() {
		preferences.Set("jwtopt.validTime", prevValid)
		preferences.Set("jwtopt.refreshInterval", prevRefresh)
		preferences.Set("jwtopt.maxSessionLifetime", prevMax)
	})
}

func TestVerifyTokenWithFreshClaimsSlidesAfterRefreshInterval(t *testing.T) {
	withJWTOptConfig(t, 30*86400, 86400, 180*86400)

	// A token with 28 days remaining (2 days after issue on a 30d TTL, well outside
	// the legacy final-24h window) must slide forward and preserve jti.
	agedToken, err := Std().CreateToken(CustomClaims{
		UserId:           42,
		TokenVersion:     1,
		Jti:              "slide-jti-001",
		RegisteredClaims: GetBaseRegisteredClaims(28 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, newToken, err := VerifyTokenWithFreshClaims(agedToken)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Jti != "slide-jti-001" {
		t.Fatalf("claims.Jti = %q, want slide-jti-001", claims.Jti)
	}
	if newToken == agedToken {
		t.Fatal("expected token with 28d remaining to refresh on active use, got unchanged token")
	}
	refreshedClaims, err := Std().ParseToken(newToken)
	if err != nil {
		t.Fatalf("parse refreshed token: %v", err)
	}
	if refreshedClaims.Jti != "slide-jti-001" {
		t.Fatalf("refreshed jti = %q, want slide-jti-001", refreshedClaims.Jti)
	}
	exp, err := refreshedClaims.GetExpirationTime()
	if err != nil || exp == nil {
		t.Fatalf("refreshed exp err = %v", err)
	}
	if delta := time.Until(exp.Time) - 30*24*time.Hour; delta < -time.Minute || delta > time.Minute {
		t.Fatalf("refreshed remaining TTL = %v, want ~30d", time.Until(exp.Time))
	}

	// Throttling: a freshly refreshed token must not re-sign immediately.
	_, secondToken, err := VerifyTokenWithFreshClaims(newToken)
	if err != nil {
		t.Fatal(err)
	}
	if secondToken != newToken {
		t.Fatal("fresh token within refreshInterval should not refresh again")
	}

	// Upgrade path: a legacy 7-day token upgrades to 30 days on its first request.
	legacyToken, err := Std().CreateToken(CustomClaims{
		UserId:           42,
		TokenVersion:     1,
		Jti:              "legacy-7d-jti",
		RegisteredClaims: GetBaseRegisteredClaims(6 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, upgradedToken, err := VerifyTokenWithFreshClaims(legacyToken)
	if err != nil {
		t.Fatal(err)
	}
	if upgradedToken == legacyToken {
		t.Fatal("legacy 7d token should immediately upgrade to 30d TTL")
	}
}

func TestRefreshTokenIfNeededRespectsMaxSessionLifetime(t *testing.T) {
	withJWTOptConfig(t, 30*86400, 86400, 180*86400)

	// Session created 170 days ago (10d left until 180d cap), token expires in 5d:
	// renewal must clamp to createdAt + 180d (~10d), not now + 30d.
	createdAt := time.Now().Add(-170 * 24 * time.Hour).Truncate(time.Second)
	token, err := Std().CreateToken(CustomClaims{
		UserId:           7,
		TokenVersion:     1,
		Jti:              "cap-jti-001",
		RegisteredClaims: GetBaseRegisteredClaims(5 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := Std().ParseToken(token)
	if err != nil {
		t.Fatal(err)
	}
	nextToken, nextExp, refreshed, err := RefreshTokenIfNeeded(token, claims, createdAt)
	if err != nil {
		t.Fatalf("RefreshTokenIfNeeded() error = %v", err)
	}
	if !refreshed || nextToken == token {
		t.Fatal("expected refresh up to hard deadline")
	}
	wantDeadline := createdAt.Add(180 * 24 * time.Hour)
	if diff := nextExp.Sub(wantDeadline); diff < -2*time.Second || diff > 2*time.Second {
		t.Fatalf("clamped exp = %v, want %v", nextExp, wantDeadline)
	}

	// Repeating with the capped token must not loop-refresh.
	cappedClaims, err := Std().ParseToken(nextToken)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, again, err := RefreshTokenIfNeeded(nextToken, cappedClaims, createdAt); err != nil || again {
		t.Fatalf("capped token re-refreshed: again=%t err=%v", again, err)
	}

	// Session past maxSessionLifetime must be rejected.
	expiredCreatedAt := time.Now().Add(-181 * 24 * time.Hour)
	if _, _, _, err := RefreshTokenIfNeeded(token, claims, expiredCreatedAt); err == nil {
		t.Fatal("session past maxSessionLifetime should fail")
	}

	// Setting maxSessionLifetime = 0 disables the cap.
	withJWTOptConfig(t, 30*86400, 86400, 0)
	_, unlimitedExp, unlimitedRefreshed, err := RefreshTokenIfNeeded(token, claims, expiredCreatedAt)
	if err != nil || !unlimitedRefreshed {
		t.Fatalf("unlimited cap refresh: refreshed=%t err=%v", unlimitedRefreshed, err)
	}
	if time.Until(unlimitedExp) < 29*24*time.Hour {
		t.Fatalf("unlimited exp = %v, want ~30d", unlimitedExp)
	}
}

func TestRefreshIntervalClampsToHalfValidTime(t *testing.T) {
	withJWTOptConfig(t, 30*86400, 6*3600, 0)
	if got := RefreshInterval(); got != 6*time.Hour {
		t.Fatalf("RefreshInterval() = %v, want 6h for explicit custom refreshInterval", got)
	}
	withJWTOptConfig(t, 3600, 86400, 0)
	if got := RefreshInterval(); got != 30*time.Minute {
		t.Fatalf("RefreshInterval() = %v, want 30m when validTime=1h and refreshInterval=24h", got)
	}
	withJWTOptConfig(t, 30*86400, 86400, 10*86400)
	if got := InitialSessionLifetime(); got != 10*24*time.Hour {
		t.Fatalf("InitialSessionLifetime() = %v, want 10d when maxSessionLifetime < validTime", got)
	}
}

func TestRefreshSessionTokenDistinguishesDBExtendFromClientCatchUp(t *testing.T) {
	withJWTOptConfig(t, 30*86400, 86400, 180*86400)

	createdAt := time.Now().Add(-2 * 24 * time.Hour).Truncate(time.Second)
	oldToken, err := Std().CreateToken(CustomClaims{
		UserId:           9,
		TokenVersion:     1,
		Jti:              "catchup-jti-001",
		RegisteredClaims: GetBaseRegisteredClaims(28 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := Std().ParseToken(oldToken)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Both JWT and DB are 28d remaining (>24h elapsed): must extend DB.
	persistedOld := time.Now().Add(28 * 24 * time.Hour).Truncate(time.Second)
	_, _, refreshed, needsPersist, err := RefreshSessionTokenIfNeeded(oldToken, claims, createdAt, persistedOld)
	if err != nil || !refreshed || !needsPersist {
		t.Fatalf("aged DB + JWT: refreshed=%t needsPersist=%t err=%v, want true/true/nil", refreshed, needsPersist, err)
	}

	// 2. DB was already extended to ~30d by a concurrent request, while this
	// client still sent the old 28d JWT: catch up JWT without another DB write.
	persistedFresh := time.Now().Add(30*24*time.Hour - 2*time.Second).Truncate(time.Second)
	catchUpToken, catchUpExp, refreshed, needsPersist, err := RefreshSessionTokenIfNeeded(oldToken, claims, createdAt, persistedFresh)
	if err != nil || !refreshed || needsPersist || catchUpToken == oldToken {
		t.Fatalf("catch-up: refreshed=%t needsPersist=%t err=%v, want true/false/nil", refreshed, needsPersist, err)
	}
	if !catchUpExp.Equal(persistedFresh) {
		t.Fatalf("catch-up exp = %v, want persisted %v", catchUpExp, persistedFresh)
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

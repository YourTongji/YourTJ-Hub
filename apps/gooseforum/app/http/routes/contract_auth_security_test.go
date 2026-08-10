package routes

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/jwtopt"
	"github.com/leancodebox/GooseForum/app/http/controllers/api"
	"github.com/leancodebox/GooseForum/app/http/middleware"
	"github.com/leancodebox/GooseForum/app/models/forum/userSessions"
	"github.com/leancodebox/GooseForum/app/models/forum/userTotp"
	"github.com/leancodebox/GooseForum/app/models/forum/userTotpChallenges"
	"github.com/leancodebox/GooseForum/app/models/forum/userTotpRecoveryCodes"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/totpservice"
	"github.com/leancodebox/GooseForum/app/service/userservice"
	otptotp "github.com/pquerna/otp/totp"
	"gorm.io/gorm"
)

type contractLoginPublicKeyResult struct {
	PublicKey string `json:"publicKey"`
	ServerTS  int64  `json:"serverTs"`
	Algorithm string `json:"algorithm"`
}

func setupAuthSecurityContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&userTotp.Entity{},
		&userTotpRecoveryCodes.Entity{},
		&userTotpChallenges.Entity{},
	); err != nil {
		t.Fatalf("migrate authentication security contract tables: %v", err)
	}
	router.GET("/api/login-public-key", api.LoginPublicKey)
	router.POST("/api/auth/totp/verify", middleware.TOTPChallengeAuth, api.TotpVerify)
	return conn, router
}

func serveAuthSecurityJSON(router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}
func serveAuthSecurityJSONWithCookie(router http.Handler, method, path, body, cookieToken string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if cookieToken != "" {
		request.AddCookie(&http.Cookie{Name: "access_token", Value: cookieToken})
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func assertResultObjectKeysMatchFixture(
	t *testing.T,
	actual contractEnvelope,
	fixture contractEnvelope,
) {
	t.Helper()
	if actual.Code != fixture.Code {
		t.Fatalf("code = %d, want fixture code %d", actual.Code, fixture.Code)
	}
	if actual.MessageCode != fixture.MessageCode {
		t.Fatalf("messageCode = %q, want fixture messageCode %q", actual.MessageCode, fixture.MessageCode)
	}
	var actualResult map[string]any
	if err := json.Unmarshal(actual.Result, &actualResult); err != nil {
		t.Fatalf("decode actual result %q: %v", actual.Result, err)
	}
	var fixtureResult map[string]any
	if err := json.Unmarshal(fixture.Result, &fixtureResult); err != nil {
		t.Fatalf("decode fixture result %q: %v", fixture.Result, err)
	}
	actualKeys := make([]string, 0, len(actualResult))
	for key := range actualResult {
		actualKeys = append(actualKeys, key)
	}
	fixtureKeys := make([]string, 0, len(fixtureResult))
	for key := range fixtureResult {
		fixtureKeys = append(fixtureKeys, key)
	}
	if strings.Join(sortedStrings(actualKeys), ",") != strings.Join(sortedStrings(fixtureKeys), ",") {
		t.Fatalf("result keys = %v, want fixture keys %v", actualKeys, fixtureKeys)
	}
}

func enableContractTotp(t *testing.T, userID uint64) (string, []string) {
	t.Helper()
	setup, err := totpservice.Setup(userID)
	if err != nil {
		t.Fatalf("set up TOTP: %v", err)
	}
	code, err := otptotp.GenerateCode(setup.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("generate current TOTP code: %v", err)
	}
	recoveryCodes, err := totpservice.Enable(userID, code)
	if err != nil {
		t.Fatalf("enable TOTP: %v", err)
	}
	return code, recoveryCodes
}

func contractTotpChallenge(t *testing.T, user *users.EntityComplete) string {
	t.Helper()
	token, jti, err := jwtopt.CreateChallengeTokenWithJti(
		user.Id,
		user.TokenVersion,
		jwtopt.PurposeTotpChallenge,
		5*time.Minute,
	)
	if err != nil {
		t.Fatalf("create TOTP challenge token: %v", err)
	}
	if err = totpservice.SaveChallenge(user.Id, jti, 5*time.Minute); err != nil {
		t.Fatalf("save TOTP challenge: %v", err)
	}
	return token
}

func contractSessionCount(t *testing.T, conn *gorm.DB, userID uint64) int64 {
	t.Helper()
	var count int64
	if err := conn.Model(&userSessions.Entity{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count contract sessions: %v", err)
	}
	return count
}

func TestLoginPublicKeyHTTPContract(t *testing.T) {
	_, router := setupAuthSecurityContractTest(t)
	before := time.Now().Add(-time.Second).UnixMilli()
	recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/login-public-key", "", "")
	after := time.Now().Add(time.Second).UnixMilli()
	if recorder.Code != http.StatusOK {
		t.Fatalf("login public key status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	response := decodeContractEnvelope(t, recorder)
	assertResultObjectKeysMatchFixture(t, response, contractFixture(t, "login-public-key-success.json"))

	var result contractLoginPublicKeyResult
	decoder := json.NewDecoder(bytes.NewReader(response.Result))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		t.Fatalf("decode login public key result %q: %v", response.Result, err)
	}
	block, _ := pem.Decode([]byte(result.PublicKey))
	if block == nil || block.Type != "PUBLIC KEY" {
		t.Fatal("publicKey is not a PEM public key")
	}
	if _, err := x509.ParsePKIXPublicKey(block.Bytes); err != nil {
		t.Fatalf("parse publicKey: %v", err)
	}
	if result.ServerTS < before || result.ServerTS > after {
		t.Fatalf("serverTs = %d, want current server milliseconds in [%d, %d]", result.ServerTS, before, after)
	}
	if result.Algorithm != "RSA-OAEP-256" {
		t.Fatalf("algorithm = %q, want RSA-OAEP-256", result.Algorithm)
	}
}

func TestTotpVerifyHTTPContract(t *testing.T) {
	t.Run("success consumes the challenge and creates one session", func(t *testing.T) {
		conn, router := setupAuthSecurityContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		code, _ := enableContractTotp(t, user.Id)
		challenge := contractTotpChallenge(t, user)

		body, err := json.Marshal(map[string]string{"code": code})
		if err != nil {
			t.Fatalf("marshal TOTP verification request: %v", err)
		}
		recorder := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			string(body),
			challenge,
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("TOTP verification status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "totp-verify-success.json"))
		if recorder.Header().Get("New-Token") == "" {
			t.Fatal("TOTP verification response missing New-Token header")
		}
		if cookie := recorder.Header().Get("Set-Cookie"); !strings.Contains(cookie, "access_token=") {
			t.Fatalf("TOTP verification response missing access_token cookie: %q", cookie)
		}
		if count := contractSessionCount(t, conn, user.Id); count != 1 {
			t.Fatalf("session count after TOTP verification = %d, want 1", count)
		}

		replay := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			string(body),
			challenge,
		)
		if replay.Code != http.StatusUnauthorized {
			t.Fatalf("consumed challenge status = %d, want 401", replay.Code)
		}
		assertFixtureEnvelope(
			t,
			decodeContractEnvelope(t, replay),
			contractFixture(t, "totp-verify-unauthenticated.json"),
		)
		if count := contractSessionCount(t, conn, user.Id); count != 1 {
			t.Fatalf("session count after challenge replay = %d, want 1", count)
		}
	})

	t.Run("a frozen account is rejected by the challenge middleware without consuming the challenge", func(t *testing.T) {
		conn, router := setupAuthSecurityContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		code, _ := enableContractTotp(t, user.Id)
		challenge := contractTotpChallenge(t, user)
		entity, err := users.Get(user.Id)
		if err != nil {
			t.Fatalf("load contract user for freeze: %v", err)
		}
		entity.IsFrozen = users.StatusFrozen
		if err := userservice.SaveUser(&entity); err != nil {
			t.Fatalf("freeze contract user: %v", err)
		}

		body, err := json.Marshal(map[string]string{"code": code})
		if err != nil {
			t.Fatalf("marshal frozen-account TOTP request: %v", err)
		}
		recorder := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			string(body),
			challenge,
		)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("frozen-account TOTP status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "totp-verify-account-frozen.json"))
		if count := contractSessionCount(t, conn, user.Id); count != 0 {
			t.Fatalf("session count after frozen-account TOTP verification = %d, want 0", count)
		}

		// The challenge stays unconsumed: unfreezing the account lets the same
		// challenge token complete the login.
		entity, err = users.Get(user.Id)
		if err != nil {
			t.Fatalf("load contract user for unfreeze: %v", err)
		}
		entity.IsFrozen = users.StatusNormal
		if err := userservice.SaveUser(&entity); err != nil {
			t.Fatalf("unfreeze contract user: %v", err)
		}
		retry := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			string(body),
			challenge,
		)
		if retry.Code != http.StatusOK {
			t.Fatalf("unfrozen retry status = %d, want 200: %s", retry.Code, retry.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, retry), contractFixture(t, "totp-verify-success.json"))
		if count := contractSessionCount(t, conn, user.Id); count != 1 {
			t.Fatalf("session count after unfrozen retry = %d, want 1", count)
		}
	})

	t.Run("recovery code consumes the challenge and creates one session", func(t *testing.T) {
		conn, router := setupAuthSecurityContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		_, recoveryCodes := enableContractTotp(t, user.Id)
		challenge := contractTotpChallenge(t, user)

		body, err := json.Marshal(map[string]string{"recoveryCode": recoveryCodes[0]})
		if err != nil {
			t.Fatalf("marshal recovery-code verification request: %v", err)
		}
		recorder := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			string(body),
			challenge,
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("recovery-code verification status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "totp-verify-success.json"))
		if recorder.Header().Get("New-Token") == "" {
			t.Fatal("recovery-code verification response missing New-Token header")
		}
		if count := contractSessionCount(t, conn, user.Id); count != 1 {
			t.Fatalf("session count after recovery-code verification = %d, want 1", count)
		}

		// The challenge is consumed by the successful verification: replaying the
		// same (now used) recovery code with the same challenge is rejected.
		replay := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			string(body),
			challenge,
		)
		if replay.Code != http.StatusUnauthorized {
			t.Fatalf("consumed-challenge replay status = %d, want 401", replay.Code)
		}
		assertFixtureEnvelope(
			t,
			decodeContractEnvelope(t, replay),
			contractFixture(t, "totp-verify-unauthenticated.json"),
		)
		if count := contractSessionCount(t, conn, user.Id); count != 1 {
			t.Fatalf("session count after recovery-code replay = %d, want 1", count)
		}
	})

	t.Run("code takes precedence over recovery code", func(t *testing.T) {
		conn, router := setupAuthSecurityContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		code, recoveryCodes := enableContractTotp(t, user.Id)
		challenge := contractTotpChallenge(t, user)

		// A valid recovery code in `recoveryCode` is ignored while `code` is
		// non-empty: the invalid `code` still fails.
		invalidCodeBody, err := json.Marshal(map[string]string{
			"code":         "not-a-valid-code",
			"recoveryCode": recoveryCodes[0],
		})
		if err != nil {
			t.Fatalf("marshal precedence request: %v", err)
		}
		recorder := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			string(invalidCodeBody),
			challenge,
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid-code-with-recovery status = %d, want 200", recorder.Code)
		}
		assertFixtureEnvelope(
			t,
			decodeContractEnvelope(t, recorder),
			contractFixture(t, "totp-verify-invalid-code.json"),
		)

		// A valid code still succeeds when a valid recovery code is also present.
		validBody, err := json.Marshal(map[string]string{
			"code":         code,
			"recoveryCode": recoveryCodes[0],
		})
		if err != nil {
			t.Fatalf("marshal precedence success request: %v", err)
		}
		success := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			string(validBody),
			challenge,
		)
		if success.Code != http.StatusOK {
			t.Fatalf("valid-code-with-recovery status = %d, want 200: %s", success.Code, success.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, success), contractFixture(t, "totp-verify-success.json"))
	})

	t.Run("empty request with a valid challenge stays an HTTP 200 business failure", func(t *testing.T) {
		conn, router := setupAuthSecurityContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		enableContractTotp(t, user.Id)
		challenge := contractTotpChallenge(t, user)

		recorder := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			`{}`,
			challenge,
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("empty-request status = %d, want 200", recorder.Code)
		}
		assertFixtureEnvelope(
			t,
			decodeContractEnvelope(t, recorder),
			contractFixture(t, "totp-verify-invalid-code.json"),
		)
	})

	t.Run("a normal session JWT is rejected by the challenge middleware", func(t *testing.T) {
		conn, router := setupAuthSecurityContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		sessionToken := contractSessionToken(t, user)

		recorder := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			`{"code":"123456"}`,
			sessionToken,
		)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("session-JWT status = %d, want 401", recorder.Code)
		}
		assertFixtureEnvelope(
			t,
			decodeContractEnvelope(t, recorder),
			contractFixture(t, "totp-verify-unauthenticated.json"),
		)
	})

	t.Run("challenge token via the access_token cookie authenticates", func(t *testing.T) {
		conn, router := setupAuthSecurityContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		code, _ := enableContractTotp(t, user.Id)
		challenge := contractTotpChallenge(t, user)

		body, err := json.Marshal(map[string]string{"code": code})
		if err != nil {
			t.Fatalf("marshal cookie-verification request: %v", err)
		}
		recorder := serveAuthSecurityJSONWithCookie(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			string(body),
			challenge,
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("cookie-challenge status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "totp-verify-success.json"))
	})

	t.Run("missing challenge returns 401", func(t *testing.T) {
		_, router := setupAuthSecurityContractTest(t)
		recorder := serveAuthSecurityJSON(router, http.MethodPost, "/api/auth/totp/verify", `{}`, "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("missing challenge status = %d, want 401", recorder.Code)
		}
		assertFixtureEnvelope(
			t,
			decodeContractEnvelope(t, recorder),
			contractFixture(t, "totp-verify-unauthenticated.json"),
		)
	})

	t.Run("malformed JSON remains an HTTP 200 business failure", func(t *testing.T) {
		conn, router := setupAuthSecurityContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		enableContractTotp(t, user.Id)
		challenge := contractTotpChallenge(t, user)
		recorder := serveAuthSecurityJSON(router, http.MethodPost, "/api/auth/totp/verify", "{", challenge)
		if recorder.Code != http.StatusOK {
			t.Fatalf("malformed TOTP request status = %d, want 200", recorder.Code)
		}
		assertFixtureEnvelope(
			t,
			decodeContractEnvelope(t, recorder),
			contractFixture(t, "totp-verify-invalid-format.json"),
		)
	})

	t.Run("unknown request fields are rejected by strict decoding", func(t *testing.T) {
		conn, router := setupAuthSecurityContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		enableContractTotp(t, user.Id)
		challenge := contractTotpChallenge(t, user)
		recorder := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			`{"code":"123456","extra":"not-in-contract"}`,
			challenge,
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown-field status = %d, want 200", recorder.Code)
		}
		assertFixtureEnvelope(
			t,
			decodeContractEnvelope(t, recorder),
			contractFixture(t, "totp-verify-invalid-format.json"),
		)
	})

	t.Run("trailing JSON value is rejected without consuming the challenge", func(t *testing.T) {
		conn, router := setupAuthSecurityContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		code, _ := enableContractTotp(t, user.Id)
		challenge := contractTotpChallenge(t, user)

		body, err := json.Marshal(map[string]string{"code": code})
		if err != nil {
			t.Fatalf("marshal trailing-value request: %v", err)
		}
		recorder := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			string(body)+" {}",
			challenge,
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("trailing-value status = %d, want 200", recorder.Code)
		}
		assertFixtureEnvelope(
			t,
			decodeContractEnvelope(t, recorder),
			contractFixture(t, "totp-verify-invalid-format.json"),
		)
		if count := contractSessionCount(t, conn, user.Id); count != 0 {
			t.Fatalf("session count after trailing-value request = %d, want 0", count)
		}

		// The challenge is not consumed: the same token completes login with a
		// well-formed single-document body.
		success := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			string(body),
			challenge,
		)
		if success.Code != http.StatusOK {
			t.Fatalf("well-formed retry status = %d, want 200: %s", success.Code, success.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, success), contractFixture(t, "totp-verify-success.json"))
		if count := contractSessionCount(t, conn, user.Id); count != 1 {
			t.Fatalf("session count after well-formed retry = %d, want 1", count)
		}
	})

	t.Run("null code is rejected without consuming the challenge", func(t *testing.T) {
		conn, router := setupAuthSecurityContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		code, _ := enableContractTotp(t, user.Id)
		challenge := contractTotpChallenge(t, user)

		recorder := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			`{"code":null}`,
			challenge,
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("null-code status = %d, want 200", recorder.Code)
		}
		assertFixtureEnvelope(
			t,
			decodeContractEnvelope(t, recorder),
			contractFixture(t, "totp-verify-invalid-format.json"),
		)
		if count := contractSessionCount(t, conn, user.Id); count != 0 {
			t.Fatalf("session count after null-code request = %d, want 0", count)
		}

		// The challenge is not consumed: the same token completes login with a
		// well-formed non-null body.
		body, err := json.Marshal(map[string]string{"code": code})
		if err != nil {
			t.Fatalf("marshal null-retry request: %v", err)
		}
		success := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			string(body),
			challenge,
		)
		if success.Code != http.StatusOK {
			t.Fatalf("null-retry status = %d, want 200: %s", success.Code, success.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, success), contractFixture(t, "totp-verify-success.json"))
		if count := contractSessionCount(t, conn, user.Id); count != 1 {
			t.Fatalf("session count after null-retry = %d, want 1", count)
		}
	})

	t.Run("invalid code remains an HTTP 200 business failure", func(t *testing.T) {
		conn, router := setupAuthSecurityContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		enableContractTotp(t, user.Id)
		challenge := contractTotpChallenge(t, user)
		recorder := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			`{"code":"not-a-valid-code"}`,
			challenge,
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid TOTP code status = %d, want 200", recorder.Code)
		}
		assertFixtureEnvelope(
			t,
			decodeContractEnvelope(t, recorder),
			contractFixture(t, "totp-verify-invalid-code.json"),
		)
	})

	t.Run("internal rate limit remains HTTP 200 without Retry-After", func(t *testing.T) {
		conn, router := setupAuthSecurityContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		enableContractTotp(t, user.Id)
		challenge := contractTotpChallenge(t, user)
		for attempt := 0; attempt < 10; attempt++ {
			recorder := serveAuthSecurityJSON(
				router,
				http.MethodPost,
				"/api/auth/totp/verify",
				`{"code":"not-a-valid-code"}`,
				challenge,
			)
			if recorder.Code != http.StatusOK {
				t.Fatalf("invalid attempt %d status = %d, want 200", attempt+1, recorder.Code)
			}
			assertFixtureEnvelope(
				t,
				decodeContractEnvelope(t, recorder),
				contractFixture(t, "totp-verify-invalid-code.json"),
			)
		}
		recorder := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			`{"code":"not-a-valid-code"}`,
			challenge,
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("rate-limited TOTP status = %d, want 200", recorder.Code)
		}
		assertFixtureEnvelope(
			t,
			decodeContractEnvelope(t, recorder),
			contractFixture(t, "totp-verify-rate-limited.json"),
		)
		if retryAfter := recorder.Header().Get("Retry-After"); retryAfter != "" {
			t.Fatalf("Retry-After = %q, want absent for internal TOTP limit", retryAfter)
		}
	})

	t.Run("session issuance failure after a valid code reports auth.login.failed", func(t *testing.T) {
		conn, router := setupAuthSecurityContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		code, _ := enableContractTotp(t, user.Id)
		challenge := contractTotpChallenge(t, user)

		// Drop the session table so sessionservice.Create cannot persist the
		// freshly issued session; the challenge is already consumed by then.
		if err := conn.Migrator().DropTable(&userSessions.Entity{}); err != nil {
			t.Fatalf("drop user_sessions for issuance-failure test: %v", err)
		}

		body, err := json.Marshal(map[string]string{"code": code})
		if err != nil {
			t.Fatalf("marshal issuance-failure request: %v", err)
		}
		recorder := serveAuthSecurityJSON(
			router,
			http.MethodPost,
			"/api/auth/totp/verify",
			string(body),
			challenge,
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("issuance-failure status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "totp-verify-login-failed.json"))
	})
}

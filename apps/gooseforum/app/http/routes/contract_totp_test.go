package routes

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/http/controllers/api"
	"github.com/leancodebox/GooseForum/app/http/middleware"
	otptotp "github.com/pquerna/otp/totp"
	"gorm.io/gorm"
)

func setupTotpSettingsContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupAuthSecurityContractTest(t)
	loginAPI := router.Group("/api").Use(middleware.JWTAuthCheck)
	loginAPI.POST("/user/totp/setup", UpButterReq(api.TotpSetup))
	loginAPI.POST("/user/totp/enable", UpButterReq(api.TotpEnable))
	loginAPI.POST("/user/totp/disable", UpButterReq(api.TotpDisable))
	loginAPI.GET("/user/totp/status", UpButterReq(api.TotpStatus))
	return conn, router
}

func TestTotpSettingsHTTPContract(t *testing.T) {
	t.Run("full setup enable status disable lifecycle", func(t *testing.T) {
		conn, router := setupTotpSettingsContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)

		setup := serveAuthSecurityJSON(router, http.MethodPost, "/api/user/totp/setup", `{"password":"secret123"}`, token)
		if setup.Code != http.StatusOK {
			t.Fatalf("setup status = %d, want 200: %s", setup.Code, setup.Body.String())
		}
		setupResponse := decodeContractEnvelope(t, setup)
		assertResultObjectKeysMatchFixture(t, setupResponse, contractFixture(t, "totp-setup-success.json"))
		var setupResult struct {
			Secret     string `json:"secret"`
			OtpauthURL string `json:"otpauthUrl"`
		}
		if err := json.Unmarshal(setupResponse.Result, &setupResult); err != nil {
			t.Fatalf("decode setup result: %v", err)
		}
		if setupResult.Secret == "" || setupResult.OtpauthURL == "" || !strings.HasPrefix(setupResult.OtpauthURL, "otpauth://totp/") {
			t.Fatalf("setup result has invalid secret/otpauthUrl: %#v", setupResult)
		}

		status := serveAuthSecurityJSON(router, http.MethodGet, "/api/user/totp/status", "", token)
		if status.Code != http.StatusOK {
			t.Fatalf("disabled status = %d, want 200", status.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, status), contractFixture(t, "totp-status-disabled.json"))

		code, err := otptotp.GenerateCode(setupResult.Secret, time.Now().UTC())
		if err != nil {
			t.Fatalf("generate enable code: %v", err)
		}
		enable := serveAuthSecurityJSON(router, http.MethodPost, "/api/user/totp/enable", `{"code":"`+code+`"}`, token)
		if enable.Code != http.StatusOK {
			t.Fatalf("enable status = %d, want 200: %s", enable.Code, enable.Body.String())
		}
		enableResponse := decodeContractEnvelope(t, enable)
		assertResultObjectKeysMatchFixture(t, enableResponse, contractFixture(t, "totp-enable-success.json"))
		var enableResult struct {
			RecoveryCodes []string `json:"recoveryCodes"`
		}
		if err := json.Unmarshal(enableResponse.Result, &enableResult); err != nil {
			t.Fatalf("decode enable result: %v", err)
		}
		if len(enableResult.RecoveryCodes) != 10 {
			t.Fatalf("recovery code count = %d, want 10", len(enableResult.RecoveryCodes))
		}

		status = serveAuthSecurityJSON(router, http.MethodGet, "/api/user/totp/status", "", token)
		if status.Code != http.StatusOK {
			t.Fatalf("enabled status = %d, want 200", status.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, status), contractFixture(t, "totp-status-enabled.json"))

		disable := serveAuthSecurityJSON(router, http.MethodPost, "/api/user/totp/disable", `{"code":"secret123"}`, token)
		if disable.Code != http.StatusOK {
			t.Fatalf("disable status = %d, want 200: %s", disable.Code, disable.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, disable), contractFixture(t, "totp-disable-success.json"))
	})

	t.Run("missing authentication returns 401 for every operation", func(t *testing.T) {
		_, router := setupTotpSettingsContractTest(t)
		requests := []struct {
			method string
			path   string
			body   string
		}{
			{http.MethodGet, "/api/user/totp/status", ""},
			{http.MethodPost, "/api/user/totp/setup", `{"password":"secret123"}`},
			{http.MethodPost, "/api/user/totp/enable", `{"code":"123456"}`},
			{http.MethodPost, "/api/user/totp/disable", `{"code":"secret123"}`},
		}
		for _, request := range requests {
			recorder := serveAuthSecurityJSON(router, request.method, request.path, request.body, "")
			if recorder.Code != http.StatusUnauthorized {
				t.Errorf("%s %s status = %d, want 401", request.method, request.path, recorder.Code)
				continue
			}
			assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "totp-settings-unauthenticated.json"))
		}
	})

	t.Run("business failures retain legacy HTTP 200 envelopes", func(t *testing.T) {
		conn, router := setupTotpSettingsContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)

		cases := []struct {
			name    string
			path    string
			body    string
			fixture string
		}{
			{"setup missing password", "/api/user/totp/setup", `{}`, "totp-settings-invalid-params.json"},
			{"setup wrong password", "/api/user/totp/setup", `{"password":"wrong"}`, "totp-setup-invalid-credentials.json"},
			{"enable without setup", "/api/user/totp/enable", `{"code":"123456"}`, "totp-not-enabled.json"},
			{"disable without setup", "/api/user/totp/disable", `{"code":"secret123"}`, "totp-not-enabled.json"},
		}
		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				recorder := serveAuthSecurityJSON(router, http.MethodPost, testCase.path, testCase.body, token)
				if recorder.Code != http.StatusOK {
					t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
				}
				assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, testCase.fixture))
			})
		}
	})
}

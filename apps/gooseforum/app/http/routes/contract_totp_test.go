package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	otptotp "github.com/pquerna/otp/totp"
	"gorm.io/gorm"
)

func setupTotpSettingsContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupAuthSecurityContractTest(t)
	loginAPI := router.Group("/api").Use(middleware.JWTAuthCheck)
	// 与真实挂载保持一致：setup/enable/disable 校验账户密码或 6 位验证码，挂 RateLimit 防暴力破解；
	// status 只读 enabled 标志，不限流。
	loginAPI.POST("/user/totp/setup", middleware.RateLimit(middleware.RateLimitTotpSetup), UpButterReq(api.TotpSetup))
	loginAPI.POST("/user/totp/enable", middleware.RateLimit(middleware.RateLimitTotpEnable), UpButterReq(api.TotpEnable))
	loginAPI.POST("/user/totp/disable", middleware.RateLimit(middleware.RateLimitTotpDisable), UpButterReq(api.TotpDisable))
	loginAPI.GET("/user/totp/status", UpButterReq(api.TotpStatus))
	return conn, router
}

// serveAuthSecurityJSONFromIP 用指定 RemoteAddr 发起契约请求，模拟不同客户端 IP，
// 用于验证用户维度限流（生产 limitPerUser 是防轮换 IP 绕过 IP 配额的主防线）。
func serveAuthSecurityJSONFromIP(router http.Handler, method, path, body, token, ip string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = ip + ":1234"
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
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

	t.Run("enable twice and setup after enable return totp.alreadyEnabled", func(t *testing.T) {
		conn, router := setupTotpSettingsContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)

		setup := serveAuthSecurityJSON(router, http.MethodPost, "/api/user/totp/setup", `{"password":"secret123"}`, token)
		if setup.Code != http.StatusOK {
			t.Fatalf("setup status = %d, want 200: %s", setup.Code, setup.Body.String())
		}
		var setupResult struct {
			Secret string `json:"secret"`
		}
		if err := json.Unmarshal(decodeContractEnvelope(t, setup).Result, &setupResult); err != nil {
			t.Fatalf("decode setup result: %v", err)
		}
		code, err := otptotp.GenerateCode(setupResult.Secret, time.Now().UTC())
		if err != nil {
			t.Fatalf("generate enable code: %v", err)
		}
		enable := serveAuthSecurityJSON(router, http.MethodPost, "/api/user/totp/enable", `{"code":"`+code+`"}`, token)
		if enable.Code != http.StatusOK {
			t.Fatalf("enable status = %d, want 200: %s", enable.Code, enable.Body.String())
		}

		enableAgain := serveAuthSecurityJSON(router, http.MethodPost, "/api/user/totp/enable", `{"code":"`+code+`"}`, token)
		if enableAgain.Code != http.StatusOK {
			t.Fatalf("enable-twice status = %d, want 200: %s", enableAgain.Code, enableAgain.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, enableAgain), contractFixture(t, "totp-already-enabled.json"))

		setupAgain := serveAuthSecurityJSON(router, http.MethodPost, "/api/user/totp/setup", `{"password":"secret123"}`, token)
		if setupAgain.Code != http.StatusOK {
			t.Fatalf("setup-after-enable status = %d, want 200: %s", setupAgain.Code, setupAgain.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, setupAgain), contractFixture(t, "totp-already-enabled.json"))
	})

	t.Run("frozen account is not rejected by TOTP settings routes", func(t *testing.T) {
		// S5 契约测试：这 4 个路由未挂 CheckWritableAccount，冻结账户不被拒绝是当前实际行为
		// （authsessionservice.ValidateToken 不检查 IsFrozen），契约如实描述并在此 pin 住，
		// 防止未来挂载 CheckWritableAccount 时静默改变契约行为。与 change-password（冻结 403）
		// 的行为差异是已知决策点：若后续给 TOTP 路由补上 CheckWritableAccount，
		// 需同步更新契约（403 + auth.account.frozen）与本测试。
		conn, router := setupTotpSettingsContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		entity, err := users.Get(user.Id)
		if err != nil {
			t.Fatalf("load contract user for freeze: %v", err)
		}
		entity.IsFrozen = users.StatusFrozen
		if err := userservice.SaveUser(&entity); err != nil {
			t.Fatalf("freeze contract user: %v", err)
		}
		token := contractSessionToken(t, user)

		setup := serveAuthSecurityJSON(router, http.MethodPost, "/api/user/totp/setup", `{"password":"secret123"}`, token)
		if setup.Code != http.StatusOK {
			t.Fatalf("frozen setup status = %d, want 200: %s", setup.Code, setup.Body.String())
		}
		assertResultObjectKeysMatchFixture(t, decodeContractEnvelope(t, setup), contractFixture(t, "totp-setup-success.json"))

		status := serveAuthSecurityJSON(router, http.MethodGet, "/api/user/totp/status", "", token)
		if status.Code != http.StatusOK {
			t.Fatalf("frozen status status = %d, want 200: %s", status.Code, status.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, status), contractFixture(t, "totp-status-disabled.json"))
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		cases := []struct {
			name    string
			path    string
			body    string
			action  string
			fixture string
			// preEnable 先完成 setup+enable，让 enable/disable 用例覆盖"已启用账户上
			// 暴力破解 6 位验证码/密码"的 M1 场景（命中 totp.code.invalid），
			// 而不是未启用账户的 totp.notEnabled 路径。
			preEnable bool
		}{
			{"setup", "/api/user/totp/setup", `{"password":"wrong"}`, middleware.RateLimitTotpSetup, "totp-setup-rate-limited.json", false},
			{"enable", "/api/user/totp/enable", `{"code":"000000"}`, middleware.RateLimitTotpEnable, "totp-enable-rate-limited.json", true},
			{"disable", "/api/user/totp/disable", `{"code":"wrong"}`, middleware.RateLimitTotpDisable, "totp-disable-rate-limited.json", true},
		}
		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				conn, router := setupTotpSettingsContractTest(t)
				user := createHTTPContractUser(t, conn, contractTestID())
				userToken := contractSessionToken(t, user)
				if testCase.preEnable {
					setup := serveAuthSecurityJSON(router, http.MethodPost, "/api/user/totp/setup", `{"password":"secret123"}`, userToken)
					if setup.Code != http.StatusOK {
						t.Fatalf("setup status = %d, want 200: %s", setup.Code, setup.Body.String())
					}
					var setupResult struct {
						Secret string `json:"secret"`
					}
					if err := json.Unmarshal(decodeContractEnvelope(t, setup).Result, &setupResult); err != nil {
						t.Fatalf("decode setup result: %v", err)
					}
					code, err := otptotp.GenerateCode(setupResult.Secret, time.Now().UTC())
					if err != nil {
						t.Fatalf("generate enable code: %v", err)
					}
					enable := serveAuthSecurityJSON(router, http.MethodPost, "/api/user/totp/enable", `{"code":"`+code+`"}`, userToken)
					if enable.Code != http.StatusOK {
						t.Fatalf("enable status = %d, want 200: %s", enable.Code, enable.Body.String())
					}
				}

				// IP 维度 limit=5：连发业务失败（HTTP 200）直至触发 429。预置成功的 action
				// 已消耗 1 次配额、触发点相应提前，故用循环检测而非固定次数。
				limited := false
				for attempt := 0; attempt < 6; attempt++ {
					recorder := serveAuthSecurityJSON(router, http.MethodPost, testCase.path, testCase.body, userToken)
					if recorder.Code == http.StatusTooManyRequests {
						limited = true
						response := decodeContractEnvelope(t, recorder)
						assertFixtureEnvelope(t, response, contractFixture(t, testCase.fixture))
						assertRetryAfter(t, recorder, response, testCase.action)
						break
					}
					if recorder.Code != http.StatusOK {
						t.Fatalf("attempt %d status = %d, want 200: %s", attempt+1, recorder.Code, recorder.Body.String())
					}
				}
				if !limited {
					t.Fatal("expected HTTP 429 rate limit after repeated requests")
				}
			})
		}
	})

	t.Run("user-dimension rate limit returns 429", func(t *testing.T) {
		conn, router := setupTotpSettingsContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		userToken := contractSessionToken(t, user)
		// 生产 limitPerUser=5 是防轮换 IP 绕过 IP 配额的主防线；用不同 RemoteAddr
		// 模拟多 IP，同一用户连发 setup，IP 维度每次仅计 1 次不触发，验证 user 维度 429。
		limited := false
		for attempt := 0; attempt < 6; attempt++ {
			ip := fmt.Sprintf("10.0.0.%d", attempt+1)
			recorder := serveAuthSecurityJSONFromIP(router, http.MethodPost, "/api/user/totp/setup", `{"password":"wrong"}`, userToken, ip)
			if recorder.Code == http.StatusTooManyRequests {
				limited = true
				response := decodeContractEnvelope(t, recorder)
				assertFixtureEnvelope(t, response, contractFixture(t, "totp-setup-rate-limited.json"))
				assertRetryAfter(t, recorder, response, middleware.RateLimitTotpSetup)
				break
			}
			if recorder.Code != http.StatusOK {
				t.Fatalf("attempt %d status = %d, want 200", attempt+1, recorder.Code)
			}
		}
		if !limited {
			t.Fatal("expected user-dimension HTTP 429 rate limit")
		}
	})
}

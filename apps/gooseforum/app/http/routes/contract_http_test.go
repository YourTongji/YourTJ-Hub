package routes

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/logincrypto"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/dailyStats"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserStat"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userActivities"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userBadges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userSessions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type contractEnvelope struct {
	Result      json.RawMessage `json:"result"`
	Code        int             `json:"code"`
	MessageCode string          `json:"messageCode"`
	Params      map[string]any  `json:"params"`
}

func setupHTTPContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ratelimit.Default().ResetAll()
	hotdataserve.ClearRateLimitConfigCache()
	hotdataserve.ClearSecuritySettingsConfigCache()
	hotdataserve.ClearPostingSettingsConfigCache()
	t.Cleanup(func() {
		ratelimit.Default().ResetAll()
		hotdataserve.ClearRateLimitConfigCache()
		hotdataserve.ClearSecuritySettingsConfigCache()
		hotdataserve.ClearPostingSettingsConfigCache()
	})

	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&userStatistics.Entity{},
		&userSessions.Entity{},
		&topics.Entity{},
		&postRevisions.Entity{},
		&posts.Entity{},
		&category.Entity{},
		&topicCategoryIndex.Entity{},
		&topicUserAction.Entity{},
		&topicUserStat.Entity{},
		&fileUsage.Entity{},
		&dailyStats.Entity{},
		&userActivities.Entity{},
		&userPoints.Entity{},
		&pointsRecord.Entity{},
		&userBadges.Entity{},
		&moderators.Entity{},
		&pageConfig.Entity{},
	); err != nil {
		t.Fatalf("migrate HTTP contract test tables: %v", err)
	}

	configureHTTPContractTestSettings(t, conn)
	router := gin.New()
	router.POST("/api/login", middleware.RateLimit(middleware.RateLimitLogin), api.Login)
	router.POST(
		"/api/auth/oidc/exchange",
		middleware.RateLimit(middleware.RateLimitLogin),
		api.OidcExchange,
	)
	forumAPI := router.Group("/api/forum")
	forumLoginAPI := forumAPI.Use(middleware.JWTAuthCheck)
	forumLoginAPI.POST(
		"/topics/write",
		middleware.CheckWritableAccount,
		middleware.RateLimit(middleware.RateLimitTopicWrite),
		UpButterReq(api.WriteTopic),
	)
	return conn, router
}

func configureHTTPContractTestSettings(t *testing.T, conn *gorm.DB) {
	t.Helper()
	pageTypes := []string{pageConfig.SecuritySettings, pageConfig.PostingSettings, pageConfig.RateLimitSettings}
	previous := make(map[string]*pageConfig.Entity, len(pageTypes))
	for _, pageType := range pageTypes {
		var entity pageConfig.Entity
		result := conn.Where("page_type = ?", pageType).First(&entity)
		if result.Error == nil {
			copy := entity
			previous[pageType] = &copy
		} else if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
			t.Fatalf("read existing %s config: %v", pageType, result.Error)
		}
	}
	t.Cleanup(func() {
		for _, pageType := range pageTypes {
			if entity := previous[pageType]; entity != nil {
				if err := conn.Save(entity).Error; err != nil {
					t.Errorf("restore %s config: %v", pageType, err)
				}
			} else if err := conn.Where("page_type = ?", pageType).Delete(&pageConfig.Entity{}).Error; err != nil {
				t.Errorf("delete test %s config: %v", pageType, err)
			}
		}
		hotdataserve.ClearRateLimitConfigCache()
		hotdataserve.ClearSecuritySettingsConfigCache()
		hotdataserve.ClearPostingSettingsConfigCache()
	})

	security := defaultconfig.GetDefaultSecuritySettingsConfig()
	security.CaptchaRequired = false
	security.EnableEmailVerification = false
	persistHTTPContractConfig(t, conn, pageConfig.SecuritySettings, security)
	persistHTTPContractConfig(t, conn, pageConfig.PostingSettings, defaultconfig.GetDefaultPostingSettingsConfig())
	persistHTTPContractConfig(t, conn, pageConfig.RateLimitSettings, pageConfig.RateLimitConfig{
		Enabled: true,
		Actions: []pageConfig.RateLimitRule{
			{Action: middleware.RateLimitLogin, WindowSeconds: 60, LimitPerIp: 5},
			{Action: middleware.RateLimitTopicWrite, WindowSeconds: 60, LimitPerIp: 5},
			{Action: middleware.RateLimitTotpSetup, WindowSeconds: 60, LimitPerIp: 5, LimitPerUser: 5},
			{Action: middleware.RateLimitTotpEnable, WindowSeconds: 60, LimitPerIp: 5, LimitPerUser: 5},
			{Action: middleware.RateLimitTotpDisable, WindowSeconds: 60, LimitPerIp: 5, LimitPerUser: 5},
		},
	})
	hotdataserve.ClearRateLimitConfigCache()
	hotdataserve.ClearSecuritySettingsConfigCache()
	hotdataserve.ClearPostingSettingsConfigCache()
}

func persistHTTPContractConfig(t *testing.T, conn *gorm.DB, pageType string, config any) {
	t.Helper()
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("encode %s config: %v", pageType, err)
	}
	entity := pageConfig.Entity{PageType: pageType, Config: string(encoded)}
	if err := conn.Where("page_type = ?", pageType).Assign(entity).FirstOrCreate(&entity).Error; err != nil {
		t.Fatalf("save %s config: %v", pageType, err)
	}
}

func contractTestID() uint64 {
	return uint64(time.Now().UnixNano())
}

func createHTTPContractUser(t *testing.T, conn *gorm.DB, id uint64) *users.EntityComplete {
	t.Helper()
	user := users.MakeUser("contract"+strconv.FormatUint(id, 10), "secret123", fmt.Sprintf("contract-%d@example.test", id))
	user.Id = id
	user.IsActivated = users.ActivationSuccess
	user.CreatedAt = time.Now().Add(-48 * time.Hour)
	if err := conn.Create(user).Error; err != nil {
		t.Fatalf("create contract user: %v", err)
	}
	if err := conn.Create(&userStatistics.Entity{UserId: id}).Error; err != nil {
		t.Fatalf("create contract user statistics: %v", err)
	}
	return user
}

func contractSessionToken(t *testing.T, user *users.EntityComplete) string {
	t.Helper()
	token, jti, err := jwtopt.CreateSessionToken(user.Id, user.TokenVersion)
	if err != nil {
		t.Fatalf("create session token: %v", err)
	}
	if err := sessionservice.Create(user.Id, jti, "contract-test", "127.0.0.1"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	return token
}

func encryptLoginPassword(t *testing.T, password string) string {
	t.Helper()
	return encryptLoginPasswordAt(t, password, time.Now().UnixMilli())
}

func encryptLoginPasswordAt(t *testing.T, password string, ts int64) string {
	t.Helper()
	block, _ := pem.Decode([]byte(logincrypto.PublicKeyPEM()))
	if block == nil {
		t.Fatal("decode login public key PEM")
	}
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatalf("parse login public key: %v", err)
	}
	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		t.Fatal("login public key is not RSA")
	}
	payload, err := json.Marshal(logincrypto.PasswordPayload{Password: password, Ts: ts})
	if err != nil {
		t.Fatalf("marshal encrypted password payload: %v", err)
	}
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPublicKey, payload, nil)
	if err != nil {
		t.Fatalf("encrypt login password: %v", err)
	}
	return base64.StdEncoding.EncodeToString(ciphertext)
}

func serveJSON(router http.Handler, path string, body string, token string) *httptest.ResponseRecorder {
	return serveAuthSecurityJSON(router, http.MethodPost, path, body, token)
}

func decodeContractEnvelope(t *testing.T, recorder *httptest.ResponseRecorder) contractEnvelope {
	t.Helper()
	var envelope contractEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	return envelope
}

func contractFixture(t *testing.T, filename string) contractEnvelope {
	t.Helper()
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve contract fixture path")
	}
	root := filepath.Join(filepath.Dir(testFile), "..", "..", "..", "..", "..")
	path := filepath.Join(root, "packages", "api-contract", "fixtures", filename)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", filename, err)
	}
	var fixture contractEnvelope
	if err := json.Unmarshal(contents, &fixture); err != nil {
		t.Fatalf("decode fixture %s: %v", filename, err)
	}
	return fixture
}

func assertFixtureEnvelope(t *testing.T, actual contractEnvelope, fixture contractEnvelope) {
	t.Helper()
	if actual.Code != fixture.Code {
		t.Fatalf("code = %d, want fixture code %d", actual.Code, fixture.Code)
	}
	if fixture.MessageCode != "" && actual.MessageCode != fixture.MessageCode {
		t.Fatalf("messageCode = %q, want fixture messageCode %q", actual.MessageCode, fixture.MessageCode)
	}
	assertFixtureResult(t, actual.Result, fixture.Result)
	assertFixtureParams(t, actual.Params, fixture.Params)
}

func assertFixtureResult(t *testing.T, actual json.RawMessage, fixture json.RawMessage) {
	t.Helper()
	var actualValue any
	if err := json.Unmarshal(actual, &actualValue); err != nil {
		t.Fatalf("decode response result %q: %v", actual, err)
	}
	var fixtureValue any
	if err := json.Unmarshal(fixture, &fixtureValue); err != nil {
		t.Fatalf("decode fixture result %q: %v", fixture, err)
	}

	switch fixtureValue.(type) {
	case nil, string, bool, map[string]any, []any:
		if !reflect.DeepEqual(actualValue, fixtureValue) {
			t.Fatalf("result = %#v, want fixture result %#v", actualValue, fixtureValue)
		}
	case float64:
		if _, ok := actualValue.(float64); !ok {
			t.Fatalf("result = %#v, want numeric result as in fixture", actualValue)
		}
	default:
		t.Fatalf("unsupported fixture result type %T", fixtureValue)
	}
}

func assertFixtureParams(t *testing.T, actual map[string]any, fixture map[string]any) {
	t.Helper()
	for name, fixtureValue := range fixture {
		actualValue, ok := actual[name]
		if !ok {
			t.Fatalf("params.%s is missing, want fixture value %#v", name, fixtureValue)
		}
		if name == "retryAfterSeconds" {
			// The fixture establishes the field's presence; its countdown varies with request time.
			if retryAfterSeconds, ok := actualValue.(float64); !ok || retryAfterSeconds < 1 {
				t.Fatalf("params.%s = %#v, want positive number", name, actualValue)
			}
			continue
		}
		if !reflect.DeepEqual(actualValue, fixtureValue) {
			t.Fatalf("params.%s = %#v, want fixture value %#v", name, actualValue, fixtureValue)
		}
	}
	// 断言实际响应不包含 fixture 未声明的额外参数，防止原始解析错误串等敏感信息泄漏回归
	// （例如 course-parse-failed.json 声明 params:{}，若 400 又带上 params.error 应在此失败）。
	for name := range actual {
		if _, declared := fixture[name]; !declared {
			t.Fatalf("params.%s = %#v is present but not declared in fixture", name, actual[name])
		}
	}
}

func TestLoginHTTPContract(t *testing.T) {
	conn, router := setupHTTPContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())

	body, err := json.Marshal(map[string]string{
		"username":          user.Username,
		"encryptedPassword": encryptLoginPassword(t, "secret123"),
	})
	if err != nil {
		t.Fatalf("marshal login request: %v", err)
	}
	recorder := serveJSON(router, "/api/login", string(body), "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	response := decodeContractEnvelope(t, recorder)
	assertFixtureEnvelope(t, response, contractFixture(t, "login-success.json"))
	if string(response.Result) != `"登录成功"` {
		t.Fatalf("login result = %s, want login success message", response.Result)
	}
	if recorder.Header().Get("New-Token") == "" {
		t.Fatal("login response missing New-Token header")
	}
	if !strings.Contains(recorder.Header().Get("Set-Cookie"), "access_token=") {
		t.Fatal("login response missing access_token session cookie")
	}
}

func TestLoginHTTPContractBusinessFailureAndRateLimit(t *testing.T) {
	t.Run("business failure remains HTTP 200", func(t *testing.T) {
		_, router := setupHTTPContractTest(t)
		body, err := json.Marshal(map[string]string{
			"username":          "missing-contract-user",
			"encryptedPassword": encryptLoginPassword(t, "incorrect123"),
		})
		if err != nil {
			t.Fatalf("marshal invalid login request: %v", err)
		}
		recorder := serveJSON(router, "/api/login", string(body), "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("business failure status = %d, want 200", recorder.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "login-failure.json"))
	})

	t.Run("stale encrypted password payload stays an invalid-request business failure", func(t *testing.T) {
		_, router := setupHTTPContractTest(t)
		body, err := json.Marshal(map[string]string{
			"username":          "missing-contract-user",
			"encryptedPassword": encryptLoginPasswordAt(t, "secret123", time.Now().Add(-10*time.Minute).UnixMilli()),
		})
		if err != nil {
			t.Fatalf("marshal stale-payload login request: %v", err)
		}
		recorder := serveJSON(router, "/api/login", string(body), "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("stale-payload status = %d, want 200", recorder.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "login-invalid-request.json"))
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		_, router := setupHTTPContractTest(t)
		for attempt := 0; attempt < 5; attempt++ {
			recorder := serveJSON(router, "/api/login", "{", "")
			if recorder.Code != http.StatusOK {
				t.Fatalf("attempt %d status = %d, want 200", attempt+1, recorder.Code)
			}
		}
		recorder := serveJSON(router, "/api/login", "{", "")
		if recorder.Code != http.StatusTooManyRequests {
			t.Fatalf("rate limited status = %d, want 429", recorder.Code)
		}
		response := decodeContractEnvelope(t, recorder)
		assertFixtureEnvelope(t, response, contractFixture(t, "login-rate-limited.json"))
		assertRetryAfter(t, recorder, response, middleware.RateLimitLogin)
	})
}

func TestOidcExchangeHTTPContractInvalidRequest(t *testing.T) {
	_, router := setupHTTPContractTest(t)
	recorder := serveJSON(router, "/api/auth/oidc/exchange", "{", "")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("OIDC exchange invalid request status = %d, want 400", recorder.Code)
	}
	assertFixtureEnvelope(
		t,
		decodeContractEnvelope(t, recorder),
		contractFixture(t, "oidc-exchange-invalid.json"),
	)
}

func TestWriteTopicHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupHTTPContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		categoryID := contractTestID()
		if err := conn.Create(&category.Entity{Id: categoryID, Name: "Contract", Slug: fmt.Sprintf("contract-%d", categoryID)}).Error; err != nil {
			t.Fatalf("create contract category: %v", err)
		}
		body := fmt.Sprintf(`{"title":"Contract topic","content":"Contract topic content is long enough for default posting rules.","categoryId":[%d],"topicStatus":1}`, categoryID)
		recorder := serveJSON(router, "/api/forum/topics/write", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("topic success status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		response := decodeContractEnvelope(t, recorder)
		assertFixtureEnvelope(t, response, contractFixture(t, "topic-write-success.json"))
		var topicID uint64
		if err := json.Unmarshal(response.Result, &topicID); err != nil || topicID == 0 {
			t.Fatalf("topic success result = %s, want positive numeric topic id: %v", response.Result, err)
		}
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupHTTPContractTest(t)
		recorder := serveJSON(router, "/api/forum/topics/write", `{}`, "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401", recorder.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "topic-write-unauthenticated.json"))
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupHTTPContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		if err := conn.Model(user).Update("is_frozen", users.StatusFrozen).Error; err != nil {
			t.Fatalf("freeze contract user: %v", err)
		}
		recorder := serveJSON(router, "/api/forum/topics/write", `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("frozen account status = %d, want 403", recorder.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "topic-write-forbidden.json"))
	})

	t.Run("malformed body remains a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupHTTPContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/topics/write", "{", contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid topic request status = %d, want 200", recorder.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "topic-write-invalid.json"))
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupHTTPContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		for attempt := 0; attempt < 5; attempt++ {
			recorder := serveJSON(router, "/api/forum/topics/write", "{", token)
			if recorder.Code != http.StatusOK {
				t.Fatalf("attempt %d status = %d, want 200", attempt+1, recorder.Code)
			}
		}
		recorder := serveJSON(router, "/api/forum/topics/write", "{", token)
		if recorder.Code != http.StatusTooManyRequests {
			t.Fatalf("rate limited status = %d, want 429", recorder.Code)
		}
		response := decodeContractEnvelope(t, recorder)
		assertFixtureEnvelope(t, response, contractFixture(t, "topic-write-rate-limited.json"))
		assertRetryAfter(t, recorder, response, middleware.RateLimitTopicWrite)
	})
}

func assertRetryAfter(t *testing.T, recorder *httptest.ResponseRecorder, response contractEnvelope, action string) {
	t.Helper()
	retryAfter, err := strconv.Atoi(recorder.Header().Get("Retry-After"))
	if err != nil || retryAfter < 1 {
		t.Fatalf("Retry-After = %q, want positive integer", recorder.Header().Get("Retry-After"))
	}
	if response.Params["action"] != action {
		t.Fatalf("rate limit action = %#v, want %q", response.Params["action"], action)
	}
	retryAfterSeconds, ok := response.Params["retryAfterSeconds"].(float64)
	if !ok || retryAfterSeconds < 1 {
		t.Fatalf("retryAfterSeconds = %#v, want positive number", response.Params["retryAfterSeconds"])
	}
	if int(retryAfterSeconds) != retryAfter {
		t.Fatalf("retryAfterSeconds = %v, Retry-After = %d", retryAfterSeconds, retryAfter)
	}
}

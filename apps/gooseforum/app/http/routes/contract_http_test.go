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
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/jwtopt"
	"github.com/leancodebox/GooseForum/app/bundles/logincrypto"
	"github.com/leancodebox/GooseForum/app/bundles/ratelimit"
	"github.com/leancodebox/GooseForum/app/http/controllers/api"
	"github.com/leancodebox/GooseForum/app/http/middleware"
	"github.com/leancodebox/GooseForum/app/models/defaultconfig"
	"github.com/leancodebox/GooseForum/app/models/forum/category"
	"github.com/leancodebox/GooseForum/app/models/forum/dailyStats"
	"github.com/leancodebox/GooseForum/app/models/forum/fileUsage"
	"github.com/leancodebox/GooseForum/app/models/forum/moderators"
	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/leancodebox/GooseForum/app/models/forum/pointsRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topicCategoryIndex"
	"github.com/leancodebox/GooseForum/app/models/forum/topicUserAction"
	"github.com/leancodebox/GooseForum/app/models/forum/topicUserStat"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/userActivities"
	"github.com/leancodebox/GooseForum/app/models/forum/userBadges"
	"github.com/leancodebox/GooseForum/app/models/forum/userPoints"
	"github.com/leancodebox/GooseForum/app/models/forum/userSessions"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"github.com/leancodebox/GooseForum/app/service/sessionservice"
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
	payload, err := json.Marshal(logincrypto.PasswordPayload{Password: password, Ts: time.Now().UnixMilli()})
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
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
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
	if string(fixture.Result) == "null" && string(actual.Result) != "null" {
		t.Fatalf("result = %s, want null as in fixture", actual.Result)
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

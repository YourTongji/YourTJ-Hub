package routes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/algorithm"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userSessions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/tokenservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupAccountRecoveryContractTest 挂载与生产 route4api.go 一致的
// register/forgot-password/reset-password 路由（含 RateLimit 中间件），
// 并迁移本域测试需要的表。复用 setupHTTPContractTest 的基础配置
// （CaptchaRequired=false、EnableSignup=true、限流动作已声明）。
func setupAccountRecoveryContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&taskQueue.Entity{},
		&rolePermissionRsEntity{},
		&moderationLogEntity{},
		&eventNotificationEntity{},
	); err != nil {
		t.Fatalf("migrate account recovery contract tables: %v", err)
	}
	// 与生产 ratelimit.json 默认值一致的 register/forgot-password/reset-password
	// 限流规则（基础测试配置只声明了 login/topic.write/totp.*）。
	configureAccountRecoveryRateLimits(t, conn)
	router.POST("/api/register", middleware.RateLimit(middleware.RateLimitRegister), api.Register)
	router.POST("/api/forgot-password", middleware.RateLimit(middleware.RateLimitForgotPassword), UpButterReq(api.ForgotPassword))
	router.POST("/api/reset-password", middleware.RateLimit(middleware.RateLimitResetPassword), UpButterReq(api.ResetPassword))
	return conn, router
}

// 注册成功路径会发布 UserSignUpEvent，事件处理器会触达 role_permission_rs/
// moderation_log/event_notification 等表；这些表在基础测试里未迁移，这里用
// 最小结构占位（真实表结构由 AutoMigrate 建出）。
type rolePermissionRsEntity struct {
	Id           uint64 `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	RoleId       uint64 `gorm:"column:role_id;not null;default:0;" json:"roleId"`
	PermissionId uint64 `gorm:"column:permission_id;not null;default:0;" json:"permissionId"`
	Effective    int    `gorm:"column:effective;not null;default:0;" json:"effective"`
}

func (rolePermissionRsEntity) TableName() string { return "role_permission_rs" }

type moderationLogEntity struct {
	Id uint64 `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
}

func (moderationLogEntity) TableName() string { return "moderation_logs" }

type eventNotificationEntity struct {
	Id uint64 `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
}

func (eventNotificationEntity) TableName() string { return "event_notifications" }

// serveAccountRecoveryJSON 发起 JSON 请求并返回 recorder。
func serveAccountRecoveryJSON(router http.Handler, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

// assertRegisterResponseCode 断言注册响应与 fixture 对齐（HTTP 200 业务信封）。
func assertRegisterResponseCode(t *testing.T, recorder *httptest.ResponseRecorder, fixtureName string) contractEnvelope {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("register status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	response := decodeContractEnvelope(t, recorder)
	assertFixtureEnvelope(t, response, contractFixture(t, fixtureName))
	return response
}

// cleanAccountRecoveryTables 清理本测试域写入的行，避免共享 in-memory DB 跨用例污染。
func cleanAccountRecoveryTables(t *testing.T, conn *gorm.DB) {
	t.Helper()
	conn.Unscoped().Where("1 = 1").Delete(&taskQueue.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&userSessions.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&users.EntityComplete{})
}

// createRecoveryUser 创建一个已激活的普通用户。
func createRecoveryUser(t *testing.T, conn *gorm.DB, id uint64, username, email string) *users.EntityComplete {
	t.Helper()
	user := users.MakeUser(username, "secret123", email)
	user.Id = id
	user.IsActivated = users.ActivationSuccess
	user.CreatedAt = time.Now().Add(-48 * time.Hour)
	if err := conn.Create(user).Error; err != nil {
		t.Fatalf("create recovery user: %v", err)
	}
	return user
}

// TestRegisterHTTPContractSuccess 验证注册成功路径：HTTP 200 +
// auth.login.success 信封 + New-Token 会话头 + Set-Cookie，并确认用户行已创建。
func TestRegisterHTTPContractSuccess(t *testing.T) {
	conn, router := setupAccountRecoveryContractTest(t)
	t.Cleanup(func() { cleanAccountRecoveryTables(t, conn) })

	body := `{"email":"newuser@example.com","userName":"brandnewuser","passWord":"password123"}`
	recorder := serveAccountRecoveryJSON(router, "/api/register", body)
	response := assertRegisterResponseCode(t, recorder, "login-success.json")
	if string(response.Result) != `"登录成功"` {
		t.Fatalf("register result = %s, want login success message", response.Result)
	}
	if recorder.Header().Get("New-Token") == "" {
		t.Fatal("register response missing New-Token header")
	}
	if !strings.Contains(recorder.Header().Get("Set-Cookie"), "access_token=") {
		t.Fatal("register response missing access_token session cookie")
	}
	if !users.ExistUsername("brandnewuser") {
		t.Fatal("register must create the user row")
	}
}

// TestRegisterHTTPContractEnumerationIdentical 固定注册反枚举不变量：
// 用户名占用与邮箱占用两种失败的响应体必须逐字节一致（auth.register.failed）。
func TestRegisterHTTPContractEnumerationIdentical(t *testing.T) {
	conn, router := setupAccountRecoveryContractTest(t)
	t.Cleanup(func() { cleanAccountRecoveryTables(t, conn) })
	createRecoveryUser(t, conn, 9001, "occupieduser", "occupied@example.com")

	recUsername := serveAccountRecoveryJSON(router, "/api/register",
		`{"email":"fresh@example.com","userName":"occupieduser","passWord":"password123"}`)
	recEmail := serveAccountRecoveryJSON(router, "/api/register",
		`{"email":"occupied@example.com","userName":"freshuser1","passWord":"password123"}`)

	if recUsername.Code != http.StatusOK || recEmail.Code != http.StatusOK {
		t.Fatalf("enumeration probes must stay HTTP 200: %d / %d", recUsername.Code, recEmail.Code)
	}
	if recUsername.Body.String() != recEmail.Body.String() {
		t.Fatalf("register enumeration responses differ:\nusername-occupied: %s\nemail-occupied:     %s",
			recUsername.Body.String(), recEmail.Body.String())
	}
	resUsername := decodeContractEnvelope(t, recUsername)
	assertFixtureEnvelope(t, resUsername, contractFixture(t, "register-failed.json"))
}

// TestRegisterHTTPContractHoneypotSilentRejects 验证注册蜜罐：website 非空时
// 返回与成功一致的 auth.login.success 信封，但绝不创建用户行。
func TestRegisterHTTPContractHoneypotSilentRejects(t *testing.T) {
	conn, router := setupAccountRecoveryContractTest(t)
	t.Cleanup(func() { cleanAccountRecoveryTables(t, conn) })

	body := `{"email":"bot@example.com","userName":"botuser","passWord":"password123","website":"http://spam.example"}`
	recorder := serveAccountRecoveryJSON(router, "/api/register", body)
	assertRegisterResponseCode(t, recorder, "login-success.json")
	if users.ExistUsername("botuser") || users.ExistEmail("bot@example.com") {
		t.Fatal("honeypot register must not create a user row")
	}
}

// TestRegisterHTTPContractSignupDisabled 验证注册开关关闭时返回 auth.signupDisabled。
func TestRegisterHTTPContractSignupDisabled(t *testing.T) {
	conn, router := setupAccountRecoveryContractTest(t)
	t.Cleanup(func() { cleanAccountRecoveryTables(t, conn) })

	persistAccountRecoverySecurity(t, conn, false, false)

	body := `{"email":"newuser@example.com","userName":"brandnewuser","passWord":"password123"}`
	recorder := serveAccountRecoveryJSON(router, "/api/register", body)
	assertRegisterResponseCode(t, recorder, "register-signup-disabled.json")
}

// TestRegisterHTTPContractRateLimit 验证注册限流：默认 20 次/小时/IP，
// 第 21 次返回 429 + Retry-After + common.rateLimited。
// 注意：前 20 次中首次请求会真实创建 rateuser 用户行，必须在 cleanup 中删除，
// 否则残留行（SQLite AUTOINCREMENT 会让其 id 顺延到 9002）会与后续
// createRecoveryUser(9002) 的显式主键冲突。
func TestRegisterHTTPContractRateLimit(t *testing.T) {
	conn, router := setupAccountRecoveryContractTest(t)
	t.Cleanup(func() { cleanAccountRecoveryTables(t, conn) })
	body := `{"email":"rate@example.com","userName":"rateuser","passWord":"password123"}`
	var recorder *httptest.ResponseRecorder
	for attempt := 0; attempt < 20; attempt++ {
		recorder = serveAccountRecoveryJSON(router, "/api/register", body)
		if recorder.Code != http.StatusOK {
			t.Fatalf("attempt %d status = %d, want 200: %s", attempt+1, recorder.Code, recorder.Body.String())
		}
	}
	recorder = serveAccountRecoveryJSON(router, "/api/register", body)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limited status = %d, want 429", recorder.Code)
	}
	response := decodeContractEnvelope(t, recorder)
	assertFixtureEnvelope(t, response, contractFixture(t, "register-rate-limited.json"))
	assertRetryAfter(t, recorder, response, middleware.RateLimitRegister)
}

// TestForgotPasswordHTTPContractIndistinguishable 固定找回密码反枚举不变量：
// 未知邮箱与已注册邮箱返回逐字节一致的 auth.passwordReset.mailQueued 成功信封。
func TestForgotPasswordHTTPContractIndistinguishable(t *testing.T) {
	conn, router := setupAccountRecoveryContractTest(t)
	t.Cleanup(func() { cleanAccountRecoveryTables(t, conn) })
	withRouteTestSigningKey(t, "route-test-signing-key-forgot")
	createRecoveryUser(t, conn, 9002, "knownuser", "knownuser@example.com")

	recUnknown := serveAccountRecoveryJSON(router, "/api/forgot-password", `{"email":"nobody@example.com"}`)
	recKnown := serveAccountRecoveryJSON(router, "/api/forgot-password", `{"email":"knownuser@example.com"}`)

	if recUnknown.Code != http.StatusOK || recKnown.Code != http.StatusOK {
		t.Fatalf("forgot-password probes must stay HTTP 200: %d / %d", recUnknown.Code, recKnown.Code)
	}
	if recUnknown.Body.String() != recKnown.Body.String() {
		t.Fatalf("forgot-password responses differ between known/unknown email:\nunknown: %s\nknown:   %s",
			recUnknown.Body.String(), recKnown.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recUnknown), contractFixture(t, "forgot-password-mail-queued.json"))
}

// TestForgotPasswordHTTPContractHoneypotSilentRejects 验证找回密码蜜罐：
// website 非空时返回成功信封但绝不入队任何重置邮件任务。
func TestForgotPasswordHTTPContractHoneypotSilentRejects(t *testing.T) {
	conn, router := setupAccountRecoveryContractTest(t)
	t.Cleanup(func() { cleanAccountRecoveryTables(t, conn) })
	withRouteTestSigningKey(t, "route-test-signing-key-forgot-hp")

	recorder := serveAccountRecoveryJSON(router, "/api/forgot-password",
		`{"email":"existing@example.com","website":"http://spam.example"}`)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "forgot-password-mail-queued.json"))

	var count int64
	if err := conn.Model(&taskQueue.Entity{}).Where("type = ?", "email.reset_password").Count(&count).Error; err != nil {
		t.Fatalf("count reset mail tasks: %v", err)
	}
	if count != 0 {
		t.Fatalf("honeypot forgot-password must not enqueue a reset mail (count = %d)", count)
	}
}

// TestForgotPasswordHTTPContractRateLimit 验证找回密码限流：默认 10 次/小时/IP。
func TestForgotPasswordHTTPContractRateLimit(t *testing.T) {
	_, router := setupAccountRecoveryContractTest(t)
	body := `{"email":"nobody@example.com"}`
	var recorder *httptest.ResponseRecorder
	for attempt := 0; attempt < 10; attempt++ {
		recorder = serveAccountRecoveryJSON(router, "/api/forgot-password", body)
		if recorder.Code != http.StatusOK {
			t.Fatalf("attempt %d status = %d, want 200: %s", attempt+1, recorder.Code, recorder.Body.String())
		}
	}
	recorder = serveAccountRecoveryJSON(router, "/api/forgot-password", body)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limited status = %d, want 429", recorder.Code)
	}
	response := decodeContractEnvelope(t, recorder)
	assertFixtureEnvelope(t, response, contractFixture(t, "forgot-password-rate-limited.json"))
	assertRetryAfter(t, recorder, response, middleware.RateLimitForgotPassword)
}

// TestResetPasswordHTTPContractLifecycle 覆盖重置密码 token 生命周期：
// 有效 token 成功、旧 token 重放被拒（token_version 绑定）、无效 token 被拒、
// 密码校验失败。
func TestResetPasswordHTTPContractLifecycle(t *testing.T) {
	conn, router := setupAccountRecoveryContractTest(t)
	t.Cleanup(func() { cleanAccountRecoveryTables(t, conn) })
	withRouteTestSigningKey(t, "route-test-signing-key-reset")

	user := createRecoveryUser(t, conn, 9003, "resetuser", "resetuser@example.com")
	user0, err := users.Get(user.Id)
	if err != nil {
		t.Fatalf("get user before reset: %v", err)
	}
	token, err := tokenservice.GeneratePasswordResetToken(user.Id, "resetuser@example.com", user0.TokenVersion)
	if err != nil {
		t.Fatalf("generate reset token: %v", err)
	}

	t.Run("valid token resets password and consumes the token", func(t *testing.T) {
		body := fmt.Sprintf(`{"token":%q,"newPassword":"brand-new-password-1"}`, token)
		recorder := serveAccountRecoveryJSON(router, "/api/reset-password", body)
		if recorder.Code != http.StatusOK {
			t.Fatalf("reset status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "reset-password-success.json"))

		userAfter, err := users.Get(user.Id)
		if err != nil {
			t.Fatalf("get user after reset: %v", err)
		}
		if userAfter.TokenVersion != user0.TokenVersion+1 {
			t.Fatalf("token_version after reset = %d, want %d", userAfter.TokenVersion, user0.TokenVersion+1)
		}
		if err := algorithm.VerifyEncryptPassword(userAfter.Password, "brand-new-password-1"); err != nil {
			t.Fatalf("password must be the new one after reset: %v", err)
		}
	})

	t.Run("replayed old token is rejected", func(t *testing.T) {
		body := fmt.Sprintf(`{"token":%q,"newPassword":"replay-password-2"}`, token)
		recorder := serveAccountRecoveryJSON(router, "/api/reset-password", body)
		if recorder.Code != http.StatusOK {
			t.Fatalf("replay status = %d, want 200 (business failure)", recorder.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "reset-password-token-invalid.json"))

		userAfter, err := users.Get(user.Id)
		if err != nil {
			t.Fatalf("get user after replay: %v", err)
		}
		if err := algorithm.VerifyEncryptPassword(userAfter.Password, "replay-password-2"); err == nil {
			t.Fatal("password must NOT be changed via replayed reset token")
		}
	})

	t.Run("garbage token is rejected", func(t *testing.T) {
		recorder := serveAccountRecoveryJSON(router, "/api/reset-password",
			`{"token":"not-a-real-token","newPassword":"password123"}`)
		if recorder.Code != http.StatusOK {
			t.Fatalf("garbage token status = %d, want 200 (business failure)", recorder.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "reset-password-token-invalid.json"))
	})

	t.Run("weak new password is rejected", func(t *testing.T) {
		userCurrent, err := users.Get(user.Id)
		if err != nil {
			t.Fatalf("get user for weak password: %v", err)
		}
		weakToken, err := tokenservice.GeneratePasswordResetToken(user.Id, "resetuser@example.com", userCurrent.TokenVersion)
		if err != nil {
			t.Fatalf("generate weak-password token: %v", err)
		}
		body := fmt.Sprintf(`{"token":%q,"newPassword":"short"}`, weakToken)
		recorder := serveAccountRecoveryJSON(router, "/api/reset-password", body)
		if recorder.Code != http.StatusOK {
			t.Fatalf("weak password status = %d, want 200 (business failure)", recorder.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "reset-password-password-invalid.json"))
	})
}

// TestResetPasswordHTTPContractFrozenAccount 验证冻结账号：token 有效但账号
// 冻结时返回 permission.userFrozen（params.action/actionCode），且不消耗 token。
func TestResetPasswordHTTPContractFrozenAccount(t *testing.T) {
	conn, router := setupAccountRecoveryContractTest(t)
	t.Cleanup(func() { cleanAccountRecoveryTables(t, conn) })
	withRouteTestSigningKey(t, "route-test-signing-key-reset-frozen")

	user := createRecoveryUser(t, conn, 9004, "frozenuser", "frozenuser@example.com")
	user0, err := users.Get(user.Id)
	if err != nil {
		t.Fatalf("get user before freeze: %v", err)
	}
	if err := conn.Model(user).Update("is_frozen", users.StatusFrozen).Error; err != nil {
		t.Fatalf("freeze user: %v", err)
	}
	token, err := tokenservice.GeneratePasswordResetToken(user.Id, "frozenuser@example.com", user0.TokenVersion)
	if err != nil {
		t.Fatalf("generate reset token: %v", err)
	}

	body := fmt.Sprintf(`{"token":%q,"newPassword":"password123"}`, token)
	recorder := serveAccountRecoveryJSON(router, "/api/reset-password", body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("frozen reset status = %d, want 200 (business failure)", recorder.Code)
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "account-frozen.json"))

	userAfter, err := users.Get(user.Id)
	if err != nil {
		t.Fatalf("get user after frozen reset: %v", err)
	}
	if userAfter.TokenVersion != user0.TokenVersion {
		t.Fatalf("frozen reset must not consume the token (token_version = %d, want %d)", userAfter.TokenVersion, user0.TokenVersion)
	}
	if err := algorithm.VerifyEncryptPassword(userAfter.Password, "secret123"); err != nil {
		t.Fatal("frozen reset must not change the password")
	}
}

// TestResetPasswordHTTPContractRateLimit 验证重置密码限流：默认 10 次/小时/IP。
func TestResetPasswordHTTPContractRateLimit(t *testing.T) {
	_, router := setupAccountRecoveryContractTest(t)
	body := `{"token":"not-a-real-token","newPassword":"password123"}`
	var recorder *httptest.ResponseRecorder
	for attempt := 0; attempt < 10; attempt++ {
		recorder = serveAccountRecoveryJSON(router, "/api/reset-password", body)
		if recorder.Code != http.StatusOK {
			t.Fatalf("attempt %d status = %d, want 200: %s", attempt+1, recorder.Code, recorder.Body.String())
		}
	}
	recorder = serveAccountRecoveryJSON(router, "/api/reset-password", body)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limited status = %d, want 429", recorder.Code)
	}
	response := decodeContractEnvelope(t, recorder)
	assertFixtureEnvelope(t, response, contractFixture(t, "reset-password-rate-limited.json"))
	assertRetryAfter(t, recorder, response, middleware.RateLimitResetPassword)
}

// configureAccountRecoveryRateLimits 在基础测试限流配置之上追加
// register/forgot-password/reset-password 规则（窗口 3600s，配额与生产
// ratelimit.json 默认一致：register 20/IP，forgot/reset 10/IP）。
// 基础测试的 configureHTTPContractTestSettings 已保存旧配置并在 cleanup 恢复，
// 这里只持久化增量规则即可。
func configureAccountRecoveryRateLimits(t *testing.T, conn *gorm.DB) {
	t.Helper()
	rateLimit := defaultconfig.GetDefaultRateLimitConfig()
	rateLimit.Enabled = true
	persistHTTPContractConfig(t, conn, pageConfig.RateLimitSettings, rateLimit)
	hotdataserve.ClearRateLimitConfigCache()
}

// persistAccountRecoverySecurity 持久化一份安全配置（注册开关/邮箱验证），
// 与 configureHTTPContractTestSettings 同款写法。保存旧值并在 cleanup 恢复，
// 避免测试在共享 in-memory DB 中永久改掉注册开关。
func persistAccountRecoverySecurity(t *testing.T, conn *gorm.DB, enableSignup, enableEmailVerification bool) {
	t.Helper()
	var previous *pageConfig.Entity
	var entity pageConfig.Entity
	result := conn.Where("page_type = ?", pageConfig.SecuritySettings).First(&entity)
	if result.Error == nil {
		copy := entity
		previous = &copy
	} else if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		t.Fatalf("read existing security config: %v", result.Error)
	}
	t.Cleanup(func() {
		if previous != nil {
			if err := conn.Save(previous).Error; err != nil {
				t.Errorf("restore security config: %v", err)
			}
		} else if err := conn.Where("page_type = ?", pageConfig.SecuritySettings).Delete(&pageConfig.Entity{}).Error; err != nil {
			t.Errorf("delete test security config: %v", err)
		}
		hotdataserve.ClearSecuritySettingsConfigCache()
	})

	security := defaultRecoverySecurity()
	security.EnableSignup = enableSignup
	security.EnableEmailVerification = enableEmailVerification
	persistHTTPContractConfig(t, conn, pageConfig.SecuritySettings, security)
	hotdataserve.ClearSecuritySettingsConfigCache()
}

func defaultRecoverySecurity() pageConfig.SecurityAndRegistration {
	return pageConfig.SecurityAndRegistration{
		EnableSignup:            true,
		EnableEmailVerification: false,
		CaptchaRequired:         false,
	}
}

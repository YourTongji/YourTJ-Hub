package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/algorithm"
	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/ratelimit"
	"github.com/leancodebox/GooseForum/app/http/controllers/api"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/http/middleware"
	"github.com/leancodebox/GooseForum/app/models/defaultconfig"
	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/leancodebox/GooseForum/app/models/forum/taskQueue"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"github.com/leancodebox/GooseForum/app/service/mailservice"
	"github.com/leancodebox/GooseForum/app/service/tokenservice"
)

// setupEmailChangeTestDB migrates the tables the email-change handler touches.
func setupEmailChangeTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &taskQueue.Entity{}, &pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate email change tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&taskQueue.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
}

func createEmailChangeUser(t *testing.T, id uint64, username, password string) {
	t.Helper()
	user := users.MakeUser(username, password, username+"@example.com")
	user.Id = id
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
}

// createEmailChangeUserActivated 创建一个已激活的用户，用于验证改邮箱后
// 激活状态确实被重置（避免对本就未激活的 fixture 做空洞断言）。
func createEmailChangeUserActivated(t *testing.T, id uint64, username, password string) {
	t.Helper()
	user := users.MakeUser(username, password, username+"@example.com")
	user.Id = id
	user.IsActivated = users.ActivationSuccess
	now := time.Now()
	user.ActivatedAt = &now
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
}

// createEmailChangeUserWithChangedAt 创建一个用户并预置邮箱变更时间。
func createEmailChangeUserWithChangedAt(t *testing.T, id uint64, username, password, email string, changedAt time.Time) {
	t.Helper()
	user := users.MakeUser(username, password, email)
	user.Id = id
	ts := changedAt
	user.EmailChangedAt = &ts
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
}

// emailChangeRouter registers the set-user-email route with an authenticated user
// injected via middleware, matching the production POST contract.
func emailChangeRouter(userID uint64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", userID)
		c.Next()
	})
	router.POST("api/set-user-email", UpButterReq(api.EditUserEmail))
	return router
}

func postEmailChange(t *testing.T, router http.Handler, body string) component.ResultStruct {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/set-user-email", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resp component.ResultStruct
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return resp
}

// countEmailTaskByType 统计 TaskJson 载荷里 Type 为 taskType 的邮件任务。
// 注意：队列行 type 带 "email." 前缀（AddToQueue 写入 email.<type>），
// 必须以载荷里的 EmailTask.Type 为准。
func countEmailTaskByType(t *testing.T, taskType string) int64 {
	t.Helper()
	var count int64
	for _, task := range getEmailTasks(t) {
		if task.Type == taskType {
			count++
		}
	}
	return count
}

func getEmailTasks(t *testing.T) []mailservice.EmailTask {
	t.Helper()
	conn := db.Connect()
	var entities []taskQueue.Entity
	if err := conn.Model(&taskQueue.Entity{}).Order("id asc").Find(&entities).Error; err != nil {
		t.Fatalf("query task queue: %v", err)
	}
	tasks := make([]mailservice.EmailTask, 0, len(entities))
	for _, e := range entities {
		var task mailservice.EmailTask
		if err := json.Unmarshal([]byte(e.TaskJson), &task); err != nil {
			t.Fatalf("unmarshal task json %q: %v", e.TaskJson, err)
		}
		tasks = append(tasks, task)
	}
	return tasks
}

// disableForgotPasswordCaptcha 持久化一份关闭验证码的安全配置，使
// ForgotPassword 在测试中能走到冷静期分支（默认配置 captchaRequired=true）。
func disableForgotPasswordCaptcha(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	security := defaultconfig.GetDefaultSecuritySettingsConfig()
	security.CaptchaRequired = false
	persistHTTPContractConfig(t, conn, pageConfig.SecuritySettings, security)
	hotdataserve.ClearSecuritySettingsConfigCache()
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.SecuritySettings).Delete(&pageConfig.Entity{})
		hotdataserve.ClearSecuritySettingsConfigCache()
	})
}

func TestSetUserEmailRouteRegisteredAsPost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiRoute(router)

	registered := map[string]bool{}
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	if !registered["POST /api/set-user-email"] {
		t.Fatal("POST /api/set-user-email was not registered")
	}
	if registered["GET /api/set-user-email"] {
		t.Fatal("GET /api/set-user-email should not be registered")
	}
	if !registered["POST /api/change-password"] {
		t.Fatal("POST /api/change-password was not registered")
	}
}

// TestSecurityRouteRateLimitActionsDefaulted 验证 set-user-email 与 change-password
// 对应的限流动作都已在默认配置中声明，存量部署经 mergeDefaultRateLimitActions 自动生效。
func TestSecurityRouteRateLimitActionsDefaulted(t *testing.T) {
	defaults := defaultconfig.GetDefaultRateLimitConfig()
	actions := map[string]bool{}
	for _, rule := range defaults.Actions {
		actions[rule.Action] = true
	}
	for _, action := range []string{middleware.RateLimitEmailChange, middleware.RateLimitPasswordChange} {
		if !actions[action] {
			t.Errorf("default rate limit config missing action %q", action)
		}
	}
}

func TestEmailChangeRateLimitPerUser(t *testing.T) {
	ratelimit.Default().ResetAll()
	t.Cleanup(ratelimit.Default().ResetAll)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", uint64(9201))
		c.Next()
	})
	router.POST("/", middleware.RateLimit(middleware.RateLimitEmailChange), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// email.change 默认 limitPerUser=5：同一用户第 6 次应 429。
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("attempt %d status = %d, want 200", i+1, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("6th request status = %d, want 429", rec.Code)
	}
}

// TestChangePasswordRateLimitPerUser 验证 change-password 也受 password.change 限流
// 保护（防持窃取 token 攻击者无节流爆破旧密码）。默认 limitPerUser=5：第 6 次 429。
func TestChangePasswordRateLimitPerUser(t *testing.T) {
	ratelimit.Default().ResetAll()
	t.Cleanup(ratelimit.Default().ResetAll)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", uint64(9202))
		c.Next()
	})
	router.POST("/", middleware.RateLimit(middleware.RateLimitPasswordChange), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("attempt %d status = %d, want 200", i+1, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("6th request status = %d, want 429", rec.Code)
	}
}

func TestEditUserEmailRequiresPassword(t *testing.T) {
	setupEmailChangeTestDB(t)
	const userID = uint64(9101)
	createEmailChangeUser(t, userID, "email-nopass", "correct-password-123")
	router := emailChangeRouter(userID)

	resp := postEmailChange(t, router, `{"email":"new-9101@example.com"}`)
	if resp.Code == component.SUCCESS {
		t.Fatal("email change without password should fail validation")
	}
	if resp.MessageCode != component.MessageRequestInvalidParams {
		t.Fatalf("messageCode = %q, want %q", resp.MessageCode, component.MessageRequestInvalidParams)
	}

	user, err := users.Get(userID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user.Email != "email-nopass@example.com" {
		t.Fatalf("email changed to %q despite rejected request", user.Email)
	}
	if user.EmailChangedAt != nil {
		t.Fatal("EmailChangedAt should stay nil on rejected request")
	}
}

func TestEditUserEmailRejectsWrongPassword(t *testing.T) {
	setupEmailChangeTestDB(t)
	const userID = uint64(9102)
	createEmailChangeUser(t, userID, "email-wrong", "correct-password-123")
	router := emailChangeRouter(userID)

	resp := postEmailChange(t, router, `{"email":"new-9102@example.com","password":"wrong-password"}`)
	if resp.Code == component.SUCCESS {
		t.Fatal("email change with wrong password should fail")
	}
	if !strings.Contains(string(resp.MessageCode), "oldInvalid") {
		t.Fatalf("messageCode = %q, want to contain oldInvalid", resp.MessageCode)
	}

	user, err := users.Get(userID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user.Email != "email-wrong@example.com" {
		t.Fatalf("email changed to %q despite wrong password", user.Email)
	}
	if user.EmailChangedAt != nil {
		t.Fatal("EmailChangedAt should stay nil on wrong-password rejection")
	}
}

func TestEditUserEmailSuccessQueuesTwoEmails(t *testing.T) {
	setupEmailChangeTestDB(t)
	const userID = uint64(9103)
	createEmailChangeUserActivated(t, userID, "email-ok", "correct-password-123")
	router := emailChangeRouter(userID)

	resp := postEmailChange(t, router, `{"email":"new-9103@example.com","password":"correct-password-123"}`)
	if resp.Code != component.SUCCESS {
		t.Fatalf("response code = %d, want success (body %+v)", resp.Code, resp)
	}
	if resp.MessageCode != component.MessageUserUpdateSuccess {
		t.Fatalf("messageCode = %q, want %q", resp.MessageCode, component.MessageUserUpdateSuccess)
	}

	user, err := users.Get(userID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user.Email != "new-9103@example.com" {
		t.Fatalf("email = %q, want new-9103@example.com", user.Email)
	}
	// fixture 已激活，改邮箱后必须重置为未激活——此断言是平凡的（非空洞）。
	if user.IsActivated != users.ActivationPending {
		t.Fatalf("IsActivated = %d, want %d (activation reset from activated fixture)", user.IsActivated, users.ActivationPending)
	}
	if user.ActivatedAt != nil {
		t.Fatal("ActivatedAt should be nil after email change")
	}
	if user.EmailChangedAt == nil {
		t.Fatal("EmailChangedAt should be set after email change")
	}

	tasks := getEmailTasks(t)
	if len(tasks) != 2 {
		t.Fatalf("queued %d tasks, want 2 (activation + email_changed): %+v", len(tasks), tasks)
	}
	var activation, changed *mailservice.EmailTask
	for i := range tasks {
		switch tasks[i].Type {
		case "activation":
			activation = &tasks[i]
		case "email_changed":
			changed = &tasks[i]
		}
	}
	if activation == nil || activation.To != "new-9103@example.com" {
		t.Fatalf("activation task missing or wrong To: %+v", tasks)
	}
	if changed == nil || changed.To != "email-ok@example.com" {
		t.Fatalf("email_changed task missing or wrong To (should be old email): %+v", tasks)
	}
	if changed.NewEmail != "new-9103@example.com" {
		t.Fatalf("email_changed task NewEmail = %q, want new-9103@example.com", changed.NewEmail)
	}
}

func TestForgotPasswordBlockedDuringEmailChangeCooldown(t *testing.T) {
	setupEmailChangeTestDB(t)
	const userID = uint64(9104)
	createEmailChangeUserWithChangedAt(t, userID, "email-cooldown", "correct-password-123", "cooldown@example.com", time.Now())
	disableForgotPasswordCaptcha(t)

	res := api.ForgotPassword(component.BetterRequest[api.ForgotPasswordReq]{
		Params: api.ForgotPasswordReq{Email: "cooldown@example.com"},
	})
	if res.Code != http.StatusOK || res.Data.Code != component.SUCCESS {
		t.Fatalf("response = %#v, want silent success", res)
	}
	if res.Data.MessageCode != component.MessageAuthResetMailQueued {
		t.Fatalf("messageCode = %q, want %q", res.Data.MessageCode, component.MessageAuthResetMailQueued)
	}
	if count := countEmailTaskByType(t, "reset_password"); count != 0 {
		t.Fatalf("forgot-password during cooldown must not enqueue reset mail (count = %d)", count)
	}
}

func TestForgotPasswordAllowedAfterEmailChangeCooldown(t *testing.T) {
	setupEmailChangeTestDB(t)
	const userID = uint64(9105)
	createEmailChangeUserWithChangedAt(t, userID, "email-cooldown-expired", "correct-password-123", "cooldown-expired@example.com", time.Now().Add(-25*time.Hour))
	disableForgotPasswordCaptcha(t)

	res := api.ForgotPassword(component.BetterRequest[api.ForgotPasswordReq]{
		Params: api.ForgotPasswordReq{Email: "cooldown-expired@example.com"},
	})
	if res.Code != http.StatusOK || res.Data.Code != component.SUCCESS {
		t.Fatalf("response = %#v, want success", res)
	}
	if res.Data.MessageCode != component.MessageAuthResetMailQueued {
		t.Fatalf("messageCode = %q, want %q", res.Data.MessageCode, component.MessageAuthResetMailQueued)
	}
	if count := countEmailTaskByType(t, "reset_password"); count != 1 {
		t.Fatalf("forgot-password after cooldown should enqueue exactly 1 reset mail (count = %d)", count)
	}
}

// TestResetPasswordRejectsOldEmailTokenAfterEmailChange 验证攻击链最后一环的纵深防线：
// 改邮箱前签发的重置令牌，在邮箱变更后调用 ResetPassword 必须被拒（claims.Email 不匹配
// 当前 user.Email）。即使攻击者拿到旧令牌，也无法在邮箱已变更的账号上复用。
func TestResetPasswordRejectsOldEmailTokenAfterEmailChange(t *testing.T) {
	setupEmailChangeTestDB(t)
	const userID = uint64(9106)
	createEmailChangeUser(t, userID, "email-oldtoken", "correct-password-123")

	// 1. 用旧邮箱签发一个有效重置令牌（模拟攻击者在邮箱变更前拿到的令牌）。
	tokenBeforeChange, err := tokenservice.GeneratePasswordResetToken(userID, "email-oldtoken@example.com")
	if err != nil {
		t.Fatalf("generate reset token for old email: %v", err)
	}

	// 2. 通过 set-user-email 把邮箱改掉（模拟攻击者完成改邮箱）。
	router := emailChangeRouter(userID)
	resp := postEmailChange(t, router, `{"email":"new-9106@example.com","password":"correct-password-123"}`)
	if resp.Code != component.SUCCESS {
		t.Fatalf("email change failed: %#v", resp)
	}

	// 3. 用旧邮箱令牌重置，必须被拒（claims.Email=old 与当前 user.Email=new 不匹配）。
	resAfter := api.ResetPassword(component.BetterRequest[api.ResetPasswordReq]{
		Params: api.ResetPasswordReq{Token: tokenBeforeChange, NewPassword: "another-new-password-2"},
	})
	if resAfter.Code != http.StatusOK || resAfter.Data.Code == component.SUCCESS {
		t.Fatalf("reset with old email token after email change must be rejected, got %#v", resAfter)
	}

	user, err := users.Get(userID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if err := algorithm.VerifyEncryptPassword(user.Password, "another-new-password-2"); err == nil {
		t.Fatal("password must NOT be changed via rejected reset token")
	}
}

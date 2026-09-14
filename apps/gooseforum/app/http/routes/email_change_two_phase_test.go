package routes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/tokenservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupTwoPhaseEmailTest 在账户找回 harness（register/forgot/reset，含限流）
// 之上补齐 set-user-email / resend-activation-email / activate 路由，
// 并开启邮箱验证（两阶段换绑的生效条件）。
func setupTwoPhaseEmailTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupAccountRecoveryContractTest(t)
	enableContractEmailVerification(t, conn)

	loginAPI := router.Group("/api").Use(middleware.JWTAuthCheck)
	loginAPI.POST("/set-user-email", middleware.CheckWritableAccountAllowPendingActivation, middleware.RateLimit(middleware.RateLimitEmailChange), UpButterReq(api.EditUserEmail))
	loginAPI.POST("/resend-activation-email", middleware.CheckWritableAccountAllowPendingActivation, UpButterReq(api.ResendActivationEmail))
	// 改密吊销换绑暂存的回归（issue #678 review P1）需要与生产一致的
	// change-password 挂载；password.change 未在限流配置中声明时中间件放行。
	loginAPI.POST("/change-password", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitPasswordChange), UpButterReq(api.ChangePassword))
	router.GET("/activate", controllers.ActivateAccount)
	return conn, router
}

// stageTwoPhaseEmailChange 发起两阶段换绑第一阶段并返回确认令牌
// （与 SendPendingEmailActivation 一致：令牌绑定暂存邮箱）。
func stageTwoPhaseEmailChange(t *testing.T, router *gin.Engine, user *users.EntityComplete, newEmail string) {
	t.Helper()
	withRouteTestSigningKey(t, "two-phase-signing-key-678")
	body := fmt.Sprintf(`{"email":%q,"password":"secret123"}`, newEmail)
	recorder := serveJSON(router, "/api/set-user-email", body, contractSessionToken(t, user))
	if recorder.Code != http.StatusOK {
		t.Fatalf("set-user-email status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if envelope := decodeContractEnvelope(t, recorder); envelope.Code != 0 || envelope.MessageCode != "user.updateSuccess" {
		t.Fatalf("set-user-email envelope = %+v, want user.updateSuccess", envelope)
	}
}

func getActivatePage(t *testing.T, router http.Handler, token string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/activate?token="+token, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

// TestTwoPhaseEmailChangeStageKeepsOldEmailOccupied 是 issue #678 的核心回归：
// 换绑第一阶段只暂存新邮箱，旧邮箱保持占用且仍可登录——封死「换绑释放旧邮箱
// →旧邮箱再注册新账号」的养号循环。
func TestTwoPhaseEmailChangeStageKeepsOldEmailOccupied(t *testing.T) {
	useContractTempKV(t)
	conn, router := setupTwoPhaseEmailTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	oldEmail := user.Email
	newEmail := fmt.Sprintf("two-phase-new-%d@example.test", user.Id)

	stageTwoPhaseEmailChange(t, router, user, newEmail)

	// 第一阶段只暂存：当前邮箱、激活状态、冷静期起点都不变。
	row, err := users.Get(user.Id)
	if err != nil {
		t.Fatalf("get staged user: %v", err)
	}
	if row.Email != oldEmail {
		t.Fatalf("email = %q, want unchanged %q (stage must not switch)", row.Email, oldEmail)
	}
	if row.PendingEmail != newEmail {
		t.Fatalf("pendingEmail = %q, want %q", row.PendingEmail, newEmail)
	}
	if row.IsActivated != users.ActivationSuccess {
		t.Fatalf("IsActivated = %d, want %d (activated account keeps write access)", row.IsActivated, users.ActivationSuccess)
	}
	if row.EmailChangedAt != nil {
		t.Fatal("EmailChangedAt must stay nil until the switch completes (old email keeps password recovery)")
	}

	// 核心 #678 断言：暂存期内用旧邮箱注册新账号必须被拒。
	regBody := fmt.Sprintf(`{"username":"twophase%d","password":"secret123","email":%q}`, user.Id, oldEmail)
	recorder := serveJSON(router, "/api/register", regBody, "")
	if envelope := decodeContractEnvelope(t, recorder); envelope.Code == 0 || envelope.MessageCode != "auth.register.failed" {
		t.Fatalf("old-email register envelope = %+v, want auth.register.failed", envelope)
	}

	// 暂存的新邮箱同样不可被他人抢注（防切换时唯一索引冲突）。
	hijackBody := fmt.Sprintf(`{"username":"hijack%d","password":"secret123","email":%q}`, user.Id, newEmail)
	recorder = serveJSON(router, "/api/register", hijackBody, "")
	if envelope := decodeContractEnvelope(t, recorder); envelope.Code == 0 || envelope.MessageCode != "auth.register.failed" {
		t.Fatalf("staged-email register envelope = %+v, want auth.register.failed", envelope)
	}

	// 旧邮箱仍可密码登录（保持账号可达性，受害者可自救）。
	if _, err := users.Verify(oldEmail, "secret123"); err != nil {
		t.Fatalf("old email must stay usable for login during staged switch: %v", err)
	}

	// 旧邮箱收到「换绑申请」通知，新邮箱收到确认邮件（Type=activation）。
	tasks := getEmailTasks(t)
	var pendingNotice, confirmation bool
	for _, task := range tasks {
		switch {
		case task.Type == "email_change_pending" && task.To == oldEmail:
			pendingNotice = true
		case task.Type == "activation" && task.To == newEmail:
			confirmation = true
		}
	}
	if !pendingNotice {
		t.Fatalf("email_change_pending notice to old email missing: %+v", tasks)
	}
	if !confirmation {
		t.Fatalf("activation confirmation to staged email missing: %+v", tasks)
	}
}

// TestTwoPhaseEmailChangeConfirmationSwitchesAtomically 第二阶段：确认链接
// 验证通过后原子切换邮箱、清暂存、置激活并起 24h 找回冷静期；切换完成后
// 旧邮箱才真正释放（可注册新账号）。
func TestTwoPhaseEmailChangeConfirmationSwitchesAtomically(t *testing.T) {
	useContractTempKV(t)
	conn, router := setupTwoPhaseEmailTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	oldEmail := user.Email
	newEmail := fmt.Sprintf("two-phase-confirm-%d@example.test", user.Id)

	stageTwoPhaseEmailChange(t, router, user, newEmail)
	confirmToken, err := tokenservice.GenerateActivationToken(user.Id, newEmail)
	if err != nil {
		t.Fatalf("generate confirmation token: %v", err)
	}

	// 切换前：冷静期未启动，新邮箱不能用于密码重置（仍指向旧邮箱状态）。
	recorder := getActivatePage(t, router, confirmToken)
	if recorder.Code != http.StatusOK {
		t.Fatalf("activate page status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "activationSuccess") &&
		!strings.Contains(recorder.Body.String(), "已成功激活") &&
		!strings.Contains(recorder.Body.String(), "activated") {
		t.Fatalf("activate page should render success state, got: %s", recorder.Body.String()[:min(len(recorder.Body.String()), 400)])
	}
	row, err := users.Get(user.Id)
	if err != nil {
		t.Fatalf("get switched user: %v", err)
	}
	if row.Email != newEmail || row.PendingEmail != "" {
		t.Fatalf("after switch: email = %q pending = %q, want %q/empty", row.Email, row.PendingEmail, newEmail)
	}
	if row.IsActivated != users.ActivationSuccess {
		t.Fatalf("IsActivated = %d, want %d", row.IsActivated, users.ActivationSuccess)
	}
	if row.EmailChangedAt == nil {
		t.Fatal("EmailChangedAt must be set at switch time (24h recovery cooldown)")
	}

	// 切换完成后旧邮箱才释放：此时注册旧邮箱成功。
	regBody := fmt.Sprintf(`{"username":"released%d","password":"secret123","email":%q}`, user.Id, oldEmail)
	recorder = serveJSON(router, "/api/register", regBody, "")
	if envelope := decodeContractEnvelope(t, recorder); envelope.Code != 0 {
		t.Fatalf("old-email register after switch envelope = %+v, want success", envelope)
	}

	// 旧邮箱收到最终「已变更」通知。
	var changedNotice bool
	for _, task := range getEmailTasks(t) {
		if task.Type == "email_changed" && task.To == oldEmail {
			changedNotice = true
		}
	}
	if !changedNotice {
		t.Fatal("email_changed final notice to old email missing after switch")
	}
}

// TestTwoPhaseEmailChangeRejectsExpiredConfirmation 过期暂存（超出占用窗口）
// 的确认链接不得切换邮箱，按无效链接处理。
func TestTwoPhaseEmailChangeRejectsExpiredConfirmation(t *testing.T) {
	useContractTempKV(t)
	conn, router := setupTwoPhaseEmailTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	oldEmail := user.Email
	newEmail := fmt.Sprintf("two-phase-expired-%d@example.test", user.Id)

	stageTwoPhaseEmailChange(t, router, user, newEmail)
	// 把暂存时间拨回窗口之外（7 天窗口，拨回 8 天前）。
	if err := conn.Model(&users.EntityComplete{}).Where("id = ?", user.Id).
		Update("pending_email_at", time.Now().Add(-8*24*time.Hour)).Error; err != nil {
		t.Fatalf("age pending_email_at: %v", err)
	}

	confirmToken, err := tokenservice.GenerateActivationToken(user.Id, newEmail)
	if err != nil {
		t.Fatalf("generate confirmation token: %v", err)
	}
	recorder := getActivatePage(t, router, confirmToken)
	if recorder.Code != http.StatusOK {
		t.Fatalf("activate page status = %d, want 200", recorder.Code)
	}

	row, err := users.Get(user.Id)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if row.Email != oldEmail {
		t.Fatalf("expired confirmation switched email to %q, want unchanged %q", row.Email, oldEmail)
	}
}

// TestActivateCurrentEmailClearsStagedSwitch 用户在暂存期内点击了发往当前
// 邮箱的旧激活链接（或重发后激活当前邮箱）：视为放弃换绑，暂存被清除。
func TestActivateCurrentEmailClearsStagedSwitch(t *testing.T) {
	useContractTempKV(t)
	conn, router := setupTwoPhaseEmailTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	oldEmail := user.Email
	newEmail := fmt.Sprintf("two-phase-abandon-%d@example.test", user.Id)

	stageTwoPhaseEmailChange(t, router, user, newEmail)

	// 用当前邮箱签发的激活链接（普通激活路径）。
	currentToken, err := tokenservice.GenerateActivationToken(user.Id, oldEmail)
	if err != nil {
		t.Fatalf("generate current-email token: %v", err)
	}
	recorder := getActivatePage(t, router, currentToken)
	if recorder.Code != http.StatusOK {
		t.Fatalf("activate page status = %d, want 200", recorder.Code)
	}

	row, err := users.Get(user.Id)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if row.Email != oldEmail || row.PendingEmail != "" {
		t.Fatalf("after current-email activation: email = %q pending = %q, want %q/empty", row.Email, row.PendingEmail, oldEmail)
	}
}

// TestResendActivationDuringPendingSwitch 已激活账号在换绑暂存期内调用
// resend-activation-email：重发对象是暂存邮箱的确认邮件（不是当前邮箱）。
func TestResendActivationDuringPendingSwitch(t *testing.T) {
	useContractTempKV(t)
	conn, router := setupTwoPhaseEmailTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	newEmail := fmt.Sprintf("two-phase-resend-%d@example.test", user.Id)

	stageTwoPhaseEmailChange(t, router, user, newEmail)

	recorder := serveJSON(router, "/api/resend-activation-email", `{}`, contractSessionToken(t, user))
	if recorder.Code != http.StatusOK {
		t.Fatalf("resend status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if envelope := decodeContractEnvelope(t, recorder); envelope.MessageCode != "auth.activation.resendSuccess" {
		t.Fatalf("resend messageCode = %q, want auth.activation.resendSuccess", envelope.MessageCode)
	}

	var resent bool
	for _, task := range getEmailTasks(t) {
		if task.Type == "activation" && task.To == newEmail {
			resent = true
		}
	}
	if !resent {
		t.Fatal("resend during staged switch must target the staged email")
	}
}

// TestChangePasswordCancelsStagedSwitch 改密必须吊销换绑暂存（issue #678
// review P1）：改密证明账号已收回，攻击者已拿到的新邮箱确认链接不得再生效
// ——激活令牌不绑定 token_version，只能靠清空 pending_email 使其失配。
func TestChangePasswordCancelsStagedSwitch(t *testing.T) {
	useContractTempKV(t)
	conn, router := setupTwoPhaseEmailTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	oldEmail := user.Email
	newEmail := fmt.Sprintf("two-phase-pwd-%d@example.test", user.Id)
	tokenVersionBefore := user.TokenVersion

	stageTwoPhaseEmailChange(t, router, user, newEmail)
	// 攻击者（或前主人）在改密前已拿到的确认令牌。
	confirmToken, err := tokenservice.GenerateActivationToken(user.Id, newEmail)
	if err != nil {
		t.Fatalf("generate confirmation token: %v", err)
	}

	recorder := serveJSON(router, "/api/change-password", `{"oldPassword":"secret123","newPassword":"newsecret456"}`, contractSessionToken(t, user))
	if recorder.Code != http.StatusOK {
		t.Fatalf("change-password status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if envelope := decodeContractEnvelope(t, recorder); envelope.Code != 0 || envelope.MessageCode != "auth.password.updateSuccess" {
		t.Fatalf("change-password envelope = %+v, want auth.password.updateSuccess", envelope)
	}

	// 改密同时清空暂存并自增 token_version（吊销全部会话与重置令牌）。
	row, err := users.Get(user.Id)
	if err != nil {
		t.Fatalf("get user after password change: %v", err)
	}
	if row.PendingEmail != "" || row.PendingEmailAt != nil {
		t.Fatalf("staged switch survived password change: pending = %q (at %v), want cleared", row.PendingEmail, row.PendingEmailAt)
	}
	if row.TokenVersion != tokenVersionBefore+1 {
		t.Fatalf("tokenVersion = %d, want %d", row.TokenVersion, tokenVersionBefore+1)
	}
	if _, err := users.Verify(oldEmail, "newsecret456"); err != nil {
		t.Fatalf("new password must verify: %v", err)
	}

	// 已签发的确认令牌因暂存被清而失配：按链接无效处理，邮箱不切换。
	recorder = getActivatePage(t, router, confirmToken)
	if recorder.Code != http.StatusOK {
		t.Fatalf("activate page status = %d, want 200", recorder.Code)
	}
	row, err = users.Get(user.Id)
	if err != nil {
		t.Fatalf("get user after stale confirm: %v", err)
	}
	if row.Email != oldEmail {
		t.Fatalf("staged confirmation after password change switched email to %q, want unchanged %q", row.Email, oldEmail)
	}
}

// TestTwoPhaseSwitchConcedesWhenTargetEmailTaken 注册与换绑暂存竞争同一邮箱
// 的收敛回归（issue #678 review P2）：并发注册抢先成为目标邮箱的当前持有者
// 后，确认链接按无效处理，暂存被撤销、邮箱留给注册者，绝不产生双占。
func TestTwoPhaseSwitchConcedesWhenTargetEmailTaken(t *testing.T) {
	useContractTempKV(t)
	conn, router := setupTwoPhaseEmailTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	oldEmail := user.Email
	newEmail := fmt.Sprintf("two-phase-raced-%d@example.test", user.Id)

	stageTwoPhaseEmailChange(t, router, user, newEmail)
	// 模拟竞态赢家：在暂存落库后、切换前，注册把目标邮箱插为自己的当前 email
	// （email 与 pending_email 两列唯一索引互不感知，直接建行绕过外层检查）。
	owner := users.MakeUser(fmt.Sprintf("race-owner-%d", user.Id), "secret123", newEmail)
	if err := conn.Create(owner).Error; err != nil {
		t.Fatalf("create email owner: %v", err)
	}

	confirmToken, err := tokenservice.GenerateActivationToken(user.Id, newEmail)
	if err != nil {
		t.Fatalf("generate confirmation token: %v", err)
	}
	recorder := getActivatePage(t, router, confirmToken)
	if recorder.Code != http.StatusOK {
		t.Fatalf("activate page status = %d, want 200", recorder.Code)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "无效") && !strings.Contains(body, "invalid") {
		t.Fatalf("activate page should render link-invalid state, got: %s", body[:min(len(body), 300)])
	}

	// 换绑方：邮箱不动、暂存撤销（切换永不可能成功，立即让步）。
	row, err := users.Get(user.Id)
	if err != nil {
		t.Fatalf("get stager: %v", err)
	}
	if row.Email != oldEmail || row.PendingEmail != "" {
		t.Fatalf("stager after concede: email = %q pending = %q, want %q/empty", row.Email, row.PendingEmail, oldEmail)
	}
	// 注册方：当前邮箱完好。
	ownerRow, err := users.Get(owner.Id)
	if err != nil {
		t.Fatalf("get owner: %v", err)
	}
	if ownerRow.Email != newEmail {
		t.Fatalf("owner email = %q, want %q (occupier keeps the email)", ownerRow.Email, newEmail)
	}
}

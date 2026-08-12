package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/http/controllers/api"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/service/mailservice"
)

// postRegister 通过 gin.CreateTestContext 直调 api.Register，返回 recorder 与
// 解出的响应体，与 honeypot 测试的调用方式一致。
func postRegister(t *testing.T, body string) (*httptest.ResponseRecorder, component.ResultStruct) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	api.Register(c)

	var res component.ResultStruct
	if err := json.Unmarshal(recorder.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode register response %q: %v", recorder.Body.String(), err)
	}
	return recorder, res
}

// TestRegisterOccupiedUsernameReturnsGenericError 验证注册用户名被占用时返回与其他
// 注册失败一致的通用错误体 auth.register.failed，而非之前的 auth.username.exists。
// 否则攻击者可一次请求确定性枚举用户名占用状态。
func TestRegisterOccupiedUsernameReturnsGenericError(t *testing.T) {
	setupEmailChangeTestDB(t)
	disableForgotPasswordCaptcha(t)
	createEmailChangeUser(t, 9201, "occupieduser", "correct-password-123")

	_, res := postRegister(t, `{"email":"brandnew@example.com","userName":"occupieduser","passWord":"password123"}`)
	if res.Code != component.FAIL {
		t.Fatalf("code = %d, want FAIL", res.Code)
	}
	if res.MessageCode != component.MessageAuthRegisterFailed {
		t.Fatalf("messageCode = %q, want %q", res.MessageCode, component.MessageAuthRegisterFailed)
	}
}

// TestRegisterOccupiedEmailReturnsGenericError 验证注册邮箱被占用时返回通用错误体
// auth.register.failed，而非之前的 auth.email.exists。否则攻击者可一次请求确定性
// 枚举邮箱注册状态（PII 级身份关联信息）。
func TestRegisterOccupiedEmailReturnsGenericError(t *testing.T) {
	setupEmailChangeTestDB(t)
	disableForgotPasswordCaptcha(t)
	createEmailChangeUser(t, 9202, "freshuser2", "correct-password-123")

	_, res := postRegister(t, `{"email":"freshuser2@example.com","userName":"brandnewuser1","passWord":"password123"}`)
	if res.Code != component.FAIL {
		t.Fatalf("code = %d, want FAIL", res.Code)
	}
	if res.MessageCode != component.MessageAuthRegisterFailed {
		t.Fatalf("messageCode = %q, want %q", res.MessageCode, component.MessageAuthRegisterFailed)
	}
}

// TestRegisterEnumerationResponsesIdentical 是核心枚举断言：用户名占用与邮箱占用两种
// 请求的响应体必须逐字节完全一致，且都解出 auth.register.failed。修复前两者返回
// 不同 messageCode，攻击者可据此区分到底哪个字段被占用。
func TestRegisterEnumerationResponsesIdentical(t *testing.T) {
	setupEmailChangeTestDB(t)
	disableForgotPasswordCaptcha(t)
	createEmailChangeUser(t, 9201, "occupieduser", "correct-password-123")
	createEmailChangeUser(t, 9202, "freshuser2", "correct-password-123")

	recUsername, resUsername := postRegister(t, `{"email":"brandnew@example.com","userName":"occupieduser","passWord":"password123"}`)
	recEmail, resEmail := postRegister(t, `{"email":"freshuser2@example.com","userName":"brandnewuser1","passWord":"password123"}`)

	if !bytes.Equal(recUsername.Body.Bytes(), recEmail.Body.Bytes()) {
		t.Fatalf("register enumeration responses differ:\nusername-occupied: %s\nemail-occupied:     %s",
			recUsername.Body.String(), recEmail.Body.String())
	}
	if resUsername.MessageCode != component.MessageAuthRegisterFailed {
		t.Fatalf("username-occupied messageCode = %q, want %q", resUsername.MessageCode, component.MessageAuthRegisterFailed)
	}
	if resEmail.MessageCode != component.MessageAuthRegisterFailed {
		t.Fatalf("email-occupied messageCode = %q, want %q", resEmail.MessageCode, component.MessageAuthRegisterFailed)
	}
}

// TestForgotPasswordUnknownEmailEnqueuesNoopOnly 验证未知邮箱路径执行等量 dummy 工作：
// 只入队一条 email.noop 任务（由邮件 worker 静默消费、不发邮件），绝不入队重置邮件，
// 返回与其他路径一致的 auth.passwordReset.mailQueued 成功响应。
func TestForgotPasswordUnknownEmailEnqueuesNoopOnly(t *testing.T) {
	setupEmailChangeTestDB(t)
	disableForgotPasswordCaptcha(t)

	res := api.ForgotPassword(component.BetterRequest[api.ForgotPasswordReq]{
		Params: api.ForgotPasswordReq{Email: "nobody@example.com"},
	})
	if res.Code != http.StatusOK || res.Data.Code != component.SUCCESS {
		t.Fatalf("response = %#v, want silent success", res)
	}
	if res.Data.MessageCode != component.MessageAuthResetMailQueued {
		t.Fatalf("messageCode = %q, want %q", res.Data.MessageCode, component.MessageAuthResetMailQueued)
	}
	if count := countEmailTaskByType(t, "noop"); count != 1 {
		t.Fatalf("unknown-email forgot-password should enqueue exactly 1 noop task (count = %d)", count)
	}
	if count := countEmailTaskByType(t, "reset_password"); count != 0 {
		t.Fatalf("unknown-email forgot-password must not enqueue reset mail (count = %d)", count)
	}

	// 等量负载断言（review #129 P2）：noop 任务必须携带与真实 reset_password 任务同量的
	// 载荷（非空 Token 签名、非空 Username 占位），保证 JSON 序列化与 DB 写入负载一致。
	var noopTask *mailservice.EmailTask
	tasks := getEmailTasks(t)
	for i := range tasks {
		if tasks[i].Type == "noop" {
			noopTask = &tasks[i]
			break
		}
	}
	if noopTask == nil {
		t.Fatal("no noop task found in queue")
	}
	if strings.TrimSpace(noopTask.Token) == "" {
		t.Fatal("noop task must carry an equal-size dummy reset token (P2)")
	}
	if len(noopTask.Username) < 6 {
		t.Fatalf("noop task Username placeholder too short: %q", noopTask.Username)
	}
}

// TestForgotPasswordKnownEmailResponseMatchesUnknown 验证已注册邮箱与未知邮箱的响应体
// 逐字节一致（静默成功），同时已注册邮箱确实入队 1 条重置邮件、未知探针入队 1 条 noop，
// 响应不泄露邮箱注册状态。
func TestForgotPasswordKnownEmailResponseMatchesUnknown(t *testing.T) {
	setupEmailChangeTestDB(t)
	disableForgotPasswordCaptcha(t)
	createEmailChangeUser(t, 9203, "knownuser", "correct-password-123")

	resUnknown := api.ForgotPassword(component.BetterRequest[api.ForgotPasswordReq]{
		Params: api.ForgotPasswordReq{Email: "nobody@example.com"},
	})
	resKnown := api.ForgotPassword(component.BetterRequest[api.ForgotPasswordReq]{
		Params: api.ForgotPasswordReq{Email: "knownuser@example.com"},
	})

	unknownBody, err := json.Marshal(resUnknown.Data)
	if err != nil {
		t.Fatalf("marshal unknown response: %v", err)
	}
	knownBody, err := json.Marshal(resKnown.Data)
	if err != nil {
		t.Fatalf("marshal known response: %v", err)
	}
	if !bytes.Equal(unknownBody, knownBody) {
		t.Fatalf("forgot-password responses differ between known/unknown email:\nunknown: %s\nknown:   %s", unknownBody, knownBody)
	}
	if resKnown.Data.Code != component.SUCCESS || resKnown.Data.MessageCode != component.MessageAuthResetMailQueued {
		t.Fatalf("known-email response = %#v, want success + %q", resKnown.Data, component.MessageAuthResetMailQueued)
	}
	if count := countEmailTaskByType(t, "reset_password"); count != 1 {
		t.Fatalf("known-email forgot-password should enqueue exactly 1 reset mail (count = %d)", count)
	}
	if count := countEmailTaskByType(t, "noop"); count != 1 {
		t.Fatalf("known-email forgot-password should enqueue exactly 1 noop for the unknown probe (count = %d)", count)
	}
}

// TestForgotPasswordCooldownEnqueuesNoopNotReset 验证邮箱变更冷静期内 forgot-password
// 静默返回成功但绝不入队重置邮件（只执行等量 noop dummy 工作），防止会话 token 被接管
// 后立刻用新邮箱重置密码，且不产生枚举差异。
func TestForgotPasswordCooldownEnqueuesNoopNotReset(t *testing.T) {
	setupEmailChangeTestDB(t)
	disableForgotPasswordCaptcha(t)
	createEmailChangeUserWithChangedAt(t, 9204, "cooldown-enum", "correct-password-123", "cooldown-enum@example.com", time.Now())

	res := api.ForgotPassword(component.BetterRequest[api.ForgotPasswordReq]{
		Params: api.ForgotPasswordReq{Email: "cooldown-enum@example.com"},
	})
	if res.Code != http.StatusOK || res.Data.Code != component.SUCCESS {
		t.Fatalf("response = %#v, want silent success", res)
	}
	if res.Data.MessageCode != component.MessageAuthResetMailQueued {
		t.Fatalf("messageCode = %q, want %q", res.Data.MessageCode, component.MessageAuthResetMailQueued)
	}
	if count := countEmailTaskByType(t, "noop"); count != 1 {
		t.Fatalf("cooldown forgot-password should enqueue exactly 1 noop task (count = %d)", count)
	}
	if count := countEmailTaskByType(t, "reset_password"); count != 0 {
		t.Fatalf("cooldown forgot-password must not enqueue reset mail (count = %d)", count)
	}
}

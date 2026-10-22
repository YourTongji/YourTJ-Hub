package routes

import (
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// issue #1284 B18：站点启用邮箱验证但邮件服务未配置时，注册过去仍返回「注册成功，
// 请前往邮箱验证」，账号停在未激活状态且激活邮件永远发不出去（SendActivationEmail
// 失败只记日志）。注册入口必须前置拒绝，且不创建账号。
func TestRegisterRejectedWhenEmailVerificationHasNoMailService(t *testing.T) {
	setupEmailChangeTestDB(t)
	disableForgotPasswordCaptcha(t)
	conn := db.Connect()

	security := emptySecurityConfig()
	security.MaxDailySignups = -1
	security.EnableEmailVerification = true
	persistSecurityPolicyConfig(t, conn, security)
	// 改动前登记还原：邮件服务行不得泄漏到同进程的后续用例（issue #1284 review）。
	restorePageConfigAfterTest(t, conn, pageConfig.EmailSettings)
	persistHTTPContractConfig(t, conn, pageConfig.EmailSettings, pageConfig.MailSettingsConfig{EnableMail: false})
	hotdataserve.ClearMailSettingsConfigCache()
	t.Cleanup(hotdataserve.ClearMailSettingsConfigCache)

	_, res := postRegister(t, `{"email":"mailoff@example.com","userName":"mailoff","passWord":"password123"}`)
	if res.Code != component.FAIL || res.MessageCode != component.MessageAuthMailNotConfigured {
		t.Fatalf("code = %d, messageCode = %q, want FAIL / %q", res.Code, res.MessageCode, component.MessageAuthMailNotConfigured)
	}
	if users.ExistEmailOrFreshPending("mailoff@example.com", 0) {
		t.Fatal("rejected registration must not create an account that can never be activated")
	}
}

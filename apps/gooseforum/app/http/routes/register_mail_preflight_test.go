package routes

import (
	"fmt"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/registrationservice"
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

func TestStaleOpenSignupAvailabilityCannotBypassFreshGates(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
		limit   int
		want    component.MessageCode
	}{
		{"disabled", false, -1, component.MessageAuthSignupDisabled},
		{"full", true, 0, component.MessageAuthRegisterDailyQuota},
		{"capacity", true, -1, component.MessageAuthRegisterDailyQuota},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupEmailChangeTestDB(t)
			disableForgotPasswordCaptcha(t)
			conn := db.Connect()
			config := emptySecurityConfig()
			config.EnableSignup = true
			config.EnableEmailVerification = false
			config.MaxDailySignups = -1
			if tc.name == "capacity" {
				config.MaxDailySignups = int(users.CountCreatedToday()) + 1
			}
			persistSecurityPolicyConfig(t, conn, config)
			if got := registrationservice.PasswordAvailability(config); got != "open" {
				t.Fatalf("preflight=%s", got)
			}
			config.EnableSignup = tc.enabled
			if tc.name == "capacity" {
				user := users.EntityComplete{Username: fmt.Sprintf("capacity-%d", contractTestID()), Email: fmt.Sprintf("capacity-%d@example.test", contractTestID())}
				if err := conn.Create(&user).Error; err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { conn.Unscoped().Delete(&user) })
			} else {
				config.MaxDailySignups = tc.limit
			}
			persistSecurityPolicyConfig(t, conn, config)
			_, res := postRegister(t, `{"email":"stale-signup@example.test","userName":"stale_signup","passWord":"password123"}`)
			if res.MessageCode != tc.want {
				t.Fatalf("stale public state bypassed gate: %+v", res)
			}
			if users.ExistEmailOrFreshPending("stale-signup@example.test", 0) {
				t.Fatal("freshly closed registration created an account")
			}
		})
	}
}

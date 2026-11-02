package routes

import (
	"fmt"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

func TestRegistrationActivationMailFollowsVerificationSwitch(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			conn, router := setupAccountRecoveryContractTest(t)
			cleanAccountRecoveryTables(t, conn)
			t.Cleanup(func() { cleanAccountRecoveryTables(t, conn) })
			security := emptySecurityConfig()
			security.MaxDailySignups = -1
			security.EnableEmailVerification = enabled
			persistSecurityPolicyConfig(t, conn, security)
			restorePageConfigAfterTest(t, conn, pageConfig.EmailSettings)
			persistHTTPContractConfig(t, conn, pageConfig.EmailSettings, pageConfig.MailSettingsConfig{EnableMail: true, SmtpHost: "smtp.example.com", SmtpPort: 587})
			hotdataserve.ClearMailSettingsConfigCache()
			t.Cleanup(hotdataserve.ClearMailSettingsConfigCache)
			w := serveAccountRecoveryJSON(router, "/api/register", `{"email":"mailpolicy@example.com","userName":"mailpolicy","passWord":"password123"}`)
			fixture := "login-success.json"
			if enabled {
				fixture = "register-email-verify.json"
			}
			assertRegisterResponseCode(t, w, fixture)
			if w.Header().Get("New-Token") == "" {
				t.Fatal("registration did not issue a session")
			}
			var tasks int64
			if err := conn.Model(&taskQueue.Entity{}).Where("type = ?", "email.activation").Count(&tasks).Error; err != nil {
				t.Fatal(err)
			}
			want := int64(0)
			if enabled {
				want = 1
			}
			if tasks != want {
				t.Fatalf("activation mail tasks=%d, want %d", tasks, want)
			}
			u, err := users.GetByEmail("mailpolicy@example.com")
			if err != nil || u.IsActivated != users.ActivationPending {
				t.Fatalf("verification truth changed: user=%+v err=%v", u, err)
			}
		})
	}
}

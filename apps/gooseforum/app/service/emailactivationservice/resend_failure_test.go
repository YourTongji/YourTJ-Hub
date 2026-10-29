package emailactivationservice

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/kvstore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/mailservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/tokenservice"
	"gorm.io/gorm"
)

func setupResend(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}, &taskQueue.Entity{}); err != nil {
		t.Fatal(err)
	}
	config := pageConfig.Entity{PageType: pageConfig.SecuritySettings, Config: `{"enableEmailVerification":true}`}
	if err := conn.Where("page_type = ?", config.PageType).Assign(config).FirstOrCreate(&config).Error; err != nil {
		t.Fatal(err)
	}
	hotdataserve.ClearSecuritySettingsConfigCache()
	t.Cleanup(hotdataserve.ClearSecuritySettingsConfigCache)
	old := preferences.GetString("app.signingKey")
	preferences.Set("app.signingKey", "resend-quota-regression-test-key")
	t.Cleanup(func() { preferences.Set("app.signingKey", old) })
}

func TestResendFailureReleasesQuota(t *testing.T) {
	for _, pending := range []bool{false, true} {
		for _, stage := range []string{"token", "queue"} {
			name := stage + "/current"
			if pending {
				name = stage + "/pending"
			}
			t.Run(name, func(t *testing.T) {
				setupResend(t)
				user := users.EntityComplete{Id: testUserID.Add(1), Username: "resend_user", Email: "current@example.com", IsActivated: users.ActivationPending}
				wantTo := user.Email
				if pending {
					now := time.Now()
					user.PendingEmail, user.PendingEmailAt = "pending@example.com", &now
					user.IsActivated = users.ActivationSuccess
					wantTo = user.PendingEmail
				}
				conn := db.Connect()
				var beforeCount int64
				if err := conn.Model(&taskQueue.Entity{}).Count(&beforeCount).Error; err != nil {
					t.Fatal(err)
				}
				var fail bool
				if stage == "token" {
					preferences.Set("app.signingKey", "")
				} else {
					fail = true
					if err := conn.Callback().Create().Before("gorm:create").Register("test:resend_queue_failure", func(tx *gorm.DB) {
						if fail && tx.Statement.Table == "task_queue" {
							tx.AddError(errors.New("injected queue failure"))
						}
					}); err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { _ = conn.Callback().Create().Remove("test:resend_queue_failure") })
				}
				if _, err := Resend(user); err == nil || errors.Is(err, ErrCooldown) || errors.Is(err, ErrDailyLimit) {
					t.Fatalf("first Resend() error = %v, want %s failure", err, stage)
				}
				var afterCount int64
				if err := conn.Model(&taskQueue.Entity{}).Count(&afterCount).Error; err != nil || afterCount != beforeCount {
					t.Fatalf("failed send queued a task: count %d → %d, %v", beforeCount, afterCount, err)
				}
				preferences.Set("app.signingKey", "resend-quota-regression-test-key")
				fail = false
				result, err := Resend(user)
				if err != nil || result.RemainingToday != 2 {
					t.Fatalf("retry after %s failure = %+v, %v; want success with 2 remaining", stage, result, err)
				}
				var task taskQueue.Entity
				if err := conn.Order("id desc").First(&task).Error; err != nil {
					t.Fatal(err)
				}
				var payload mailservice.EmailTask
				if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil || payload.To != wantTo {
					t.Fatalf("queued payload = %+v, %v; want target %q", payload, err, wantTo)
				}
				claims, err := tokenservice.ParseActivationToken(payload.Token)
				if err != nil || claims.UserId != user.Id || claims.Email != wantTo {
					t.Fatalf("token does not match resend target: %+v, %v", claims, err)
				}
				if _, err := Resend(user); !errors.Is(err, ErrCooldown) {
					t.Fatalf("successful queue must keep cooldown, got %v", err)
				}
			})
		}
	}
}

func TestResendIneligibleDoesNotReserve(t *testing.T) {
	setupResend(t)
	user := users.EntityComplete{Id: testUserID.Add(1), IsActivated: users.ActivationSuccess}
	for _, expiredPending := range []bool{false, true} {
		if expiredPending {
			at := time.Now().Add(-users.PendingEmailWindow - time.Hour)
			user.PendingEmail, user.PendingEmailAt = "expired@example.com", &at
		}
		if _, err := Resend(user); !errors.Is(err, ErrAlreadyVerified) {
			t.Fatalf("ineligible resend = %v", err)
		}
	}
	config := pageConfig.Entity{PageType: pageConfig.SecuritySettings, Config: `{"enableEmailVerification":false}`}
	if err := db.Connect().Where("page_type = ?", config.PageType).Assign(config).FirstOrCreate(&config).Error; err != nil {
		t.Fatal(err)
	}
	hotdataserve.ClearSecuritySettingsConfigCache()
	user.IsActivated = users.ActivationPending
	if _, err := Resend(user); !errors.Is(err, ErrDisabled) {
		t.Fatalf("disabled resend = %v", err)
	}
	key, _ := resendKey(user.Id, time.Now())
	if _, err := kvstore.GetBytes(key); !errors.Is(err, kvstore.ErrNotFound) {
		t.Fatalf("ineligible request touched quota: %v", err)
	}
}

func TestResendCurrentAndPendingShareQuota(t *testing.T) {
	setupResend(t)
	user := users.EntityComplete{Id: testUserID.Add(1), Username: "resend_user", Email: "current@example.com", IsActivated: users.ActivationPending}
	if _, err := Resend(user); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	user.PendingEmail, user.PendingEmailAt = "pending@example.com", &now
	user.IsActivated = users.ActivationSuccess
	if _, err := Resend(user); !errors.Is(err, ErrCooldown) {
		t.Fatalf("switching target bypassed account cooldown: %v", err)
	}
}

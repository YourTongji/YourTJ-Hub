package mailservice

import (
	"encoding/json"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/tokenservice"
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"testing"
	"time"
)

const queueExpiryTestKey = "queue-expiry-test-signing-key"

func expiredTaskFixture(t *testing.T, kind string, expired bool, signingKey string) string {
	t.Helper()
	expiry := time.Now().Add(time.Hour)
	if expired {
		expiry = time.Now().Add(-time.Minute)
	}
	claims := jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(expiry), IssuedAt: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour))}
	var payload jwt.Claims = &tokenservice.ActivationClaims{UserId: 12, Email: "owner@example.test", RegisteredClaims: claims}
	if kind == "reset_password" {
		payload = &tokenservice.PasswordResetClaims{UserId: 12, Email: "owner@example.test", TokenVersion: 3, RegisteredClaims: claims}
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, payload).SignedString([]byte(signingKey))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(EmailTask{Type: kind, To: "owner@example.test", Token: signed})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func withDisabledExpiryTestMail(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatal(err)
	}
	previous := pageConfig.GetByPageType(pageConfig.EmailSettings)
	disabled := previous
	disabled.PageType = pageConfig.EmailSettings
	disabled.Config = `{"enableMail":false}`
	if err := conn.Save(&disabled).Error; err != nil {
		t.Fatal(err)
	}
	hotdataserve.ClearMailSettingsConfigCache()
	oldKey := preferences.GetString("app.signingKey", "")
	preferences.Set("app.signingKey", queueExpiryTestKey)
	t.Cleanup(func() {
		preferences.Set("app.signingKey", oldKey)
		if previous.Id == 0 {
			conn.Delete(&disabled)
		} else {
			conn.Save(&previous)
		}
		hotdataserve.ClearMailSettingsConfigCache()
	})
}

func TestExpiredQueuedLinksAreTerminalWithoutSendOrRetry(t *testing.T) {
	withDisabledExpiryTestMail(t)
	for _, kind := range []string{"activation", "reset_password"} {
		t.Run(kind, func(t *testing.T) {
			_, running, lease := newClaimedEmailTask(t, expiredTaskFixture(t, kind, true, queueExpiryTestKey))
			email, outcome, sendErr := executeClaimedEmail(&running)
			if retrying := writeEmailTaskOutcome(&running, &running, email, outcome, sendErr, lease); retrying {
				t.Fatal("already expired link was dispatched and scheduled for retry")
			}
			var stored taskQueue.Entity
			if err := db.Connect().First(&stored, running.Id).Error; err != nil {
				t.Fatal(err)
			}
			if stored.Status != taskQueue.StatusFailed || stored.RetryCount != 0 {
				t.Fatalf("expired task status/retries = %d/%d", stored.Status, stored.RetryCount)
			}
		})
	}
}

func TestQueuedLinkExpiryPreservesRetryAndLeaseBoundaries(t *testing.T) {
	withDisabledExpiryTestMail(t)
	for _, kind := range []string{"activation", "reset_password"} {
		for _, item := range []struct {
			name    string
			expired bool
			key     string
		}{
			{"valid", false, queueExpiryTestKey}, {"wrong-signature", true, "another-test-key"},
		} {
			t.Run(kind+"/"+item.name, func(t *testing.T) {
				_, running, lease := newClaimedEmailTask(t, expiredTaskFixture(t, kind, item.expired, item.key))
				email, outcome, sendErr := executeClaimedEmail(&running)
				if outcome != emailOutcomeFailed {
					t.Fatalf("non-expiry path changed: %d", outcome)
				}
				if !writeEmailTaskOutcome(&running, &running, email, outcome, sendErr, lease) {
					t.Fatal("ordinary transport failure lost retry")
				}
			})
		}
	}
	_, running, lease := newClaimedEmailTask(t, expiredTaskFixture(t, "activation", true, queueExpiryTestKey))
	email, outcome, sendErr := executeClaimedEmail(&running)
	if sendErr == nil || !strings.Contains(sendErr.Error(), "expired before delivery") || strings.Contains(sendErr.Error(), email.Token) {
		t.Fatal("expiry error missing or contains token")
	}
	writeEmailTaskOutcome(&running, &running, email, outcome, sendErr, lease+"-stale")
	var stillRunning taskQueue.Entity
	if err := db.Connect().First(&stillRunning, running.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stillRunning.Status != taskQueue.StatusRunning {
		t.Fatal("stale worker changed expiry outcome")
	}
	if writeEmailTaskOutcome(&running, &running, email, outcome, sendErr, lease) {
		t.Fatal("expiry retried")
	}
}

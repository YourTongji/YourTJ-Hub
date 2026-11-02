package registrationservice

import (
	"errors"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestPasswordAvailabilityReflectsQuotaWithoutReserving(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"availability-one", "availability-two"} {
		user := users.EntityComplete{Username: name, Email: name + "@example.test"}
		if err := conn.Create(&user).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Unscoped().Delete(&user) })
	}
	yesterday := users.EntityComplete{Username: "availability-old", Email: "old@example.test", CreatedAt: time.Now().Add(-48 * time.Hour)}
	if err := conn.Create(&yesterday).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&yesterday) })
	for _, tc := range []struct {
		name            string
		enabled, verify bool
		limit           int
		want            string
	}{
		{"closed", false, false, 3, "disabled"}, {"mail not configured", true, true, -1, "mail_unavailable"}, {"zero", true, false, 0, "daily_limit"}, {"finite full", true, false, 2, "daily_limit"}, {"available", true, false, 3, "open"}, {"unlimited", true, false, -1, "open"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := pageConfig.SecurityAndRegistration{EnableSignup: tc.enabled, EnableEmailVerification: tc.verify, MaxDailySignups: tc.limit}
			for range 2 {
				if got := PasswordAvailability(config); got != tc.want {
					t.Fatalf("state=%s want%s", got, tc.want)
				}
			}
		})
	}
	count, err := users.CountCreatedTodayWithError()
	if err != nil || count != 2 {
		t.Fatalf("availability reserved or miscounted: %d %v", count, err)
	}
}

func TestPasswordAvailabilityDoesNotTreatCountFailureAsEmpty(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("injected count failure")
	const callback = "test:signup_count_failure"
	if err := conn.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == (&users.EntityComplete{}).TableName() {
			_ = tx.AddError(failure)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Query().Remove(callback) })
	if got := PasswordAvailability(pageConfig.SecurityAndRegistration{EnableSignup: true, MaxDailySignups: 3}); got != "unavailable" {
		t.Fatalf("failed query returned %s", got)
	}
}

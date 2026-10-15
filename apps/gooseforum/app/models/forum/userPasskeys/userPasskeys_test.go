package userPasskeys

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCleanupExpiredUsesExpiryIndexAndKeepsLiveChallenges(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.AutoMigrate(&ChallengeEntity{}); err != nil {
		t.Fatalf("migrate passkey challenges: %v", err)
	}
	if !conn.Migrator().HasIndex(&ChallengeEntity{}, "idx_passkey_challenge_expires_at") {
		t.Fatal("expiry-only index was not created for the expired-challenge cleanup query")
	}

	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	rows := []ChallengeEntity{
		{ID: "expired", Kind: "login", ExpiresAt: now.Add(-time.Second)},
		{ID: "live", Kind: "login", ExpiresAt: now.Add(time.Minute)},
	}
	if err := conn.Create(&rows).Error; err != nil {
		t.Fatalf("create passkey challenges: %v", err)
	}
	if err := CleanupExpired(conn, now); err != nil {
		t.Fatalf("clean expired passkey challenges: %v", err)
	}
	var remaining []ChallengeEntity
	if err := conn.Order("id").Find(&remaining).Error; err != nil {
		t.Fatalf("query remaining passkey challenges: %v", err)
	}
	if len(remaining) != 1 || remaining[0].ID != "live" {
		t.Fatalf("cleanup left unexpected challenges: %#v", remaining)
	}
}

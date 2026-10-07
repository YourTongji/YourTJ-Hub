package feed

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestSeenExpiryUsesTheInstantAcrossTimeZones(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := conn.AutoMigrate(&SeenState{}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 7, 8, 0, 0, 0, time.UTC)
	row := SeenState{UserID: 12, TopicID: 31, LastSeenAt: now.Add(-29 * 24 * time.Hour), SeenContentAt: now.Add(-time.Hour), LastSeenProofID: "proof", ExpiresAt: now.Add(time.Hour)}
	if err := conn.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	result, err := SeenAmong(conn, 12, []uint64{31}, now.In(time.FixedZone("UTC+8", 8*60*60)))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result[31]; !ok {
		t.Fatal("a non-UTC reader clock prematurely expired the same instant")
	}
	result, err = SeenAmong(conn, 12, []uint64{31}, row.ExpiresAt)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatal("expired state affected recommendations before physical cleanup")
	}
}

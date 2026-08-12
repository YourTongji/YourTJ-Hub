package networkAccessLog

import (
	"testing"
	"time"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
)

func TestRecordAndExpireBefore(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	conn.Where("path = ?", "/accept-network-log-test").Delete(&Entity{})

	if err := Record(Entity{
		Method: "GET", Path: "/accept-network-log-test", Route: "/accept-network-log-test",
		Status: 200, UserId: 0, LatencyMs: 3,
	}); err != nil {
		t.Fatalf("Record: %v", err)
	}

	// Force old timestamp via raw SQL: CreatedAt is create-only for GORM writes.
	old := time.Now().Add(-200 * 24 * time.Hour)
	if err := conn.Exec(
		"UPDATE network_access_logs SET created_at = ? WHERE path = ?",
		old, "/accept-network-log-test",
	).Error; err != nil {
		t.Fatalf("backdate: %v", err)
	}

	n, err := ExpireBefore(time.Now().Add(-Retention), 100)
	if err != nil {
		t.Fatalf("ExpireBefore: %v", err)
	}
	if n < 1 {
		t.Fatalf("ExpireBefore deleted %d, want >=1", n)
	}
	var remain int64
	conn.Model(&Entity{}).Where("path = ?", "/accept-network-log-test").Count(&remain)
	if remain != 0 {
		t.Fatalf("remain=%d after expire", remain)
	}
}

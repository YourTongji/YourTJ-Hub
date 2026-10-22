package feed

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"testing"
	"time"
)

func seenTestDB(t *testing.T) *gorm.DB {
	t.Helper()
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
	return conn
}

func readSeen(t *testing.T, conn *gorm.DB, userID, topicID uint64) SeenState {
	t.Helper()
	var row SeenState
	if err := conn.Where("user_id = ? AND topic_id = ?", userID, topicID).Take(&row).Error; err != nil {
		t.Fatal(err)
	}
	row.LastSeenAt = row.LastSeenAt.Truncate(time.Microsecond)
	row.SeenContentAt = row.SeenContentAt.Truncate(time.Microsecond)
	row.ExpiresAt = row.ExpiresAt.Truncate(time.Microsecond)
	return row
}

// A replayed proof must not move the cutoff or extend retention after an ACK
// was lost; only a new proof may advance the row.
func TestSeenUpsertKeepsDuplicateProofReplay(t *testing.T) {
	conn := seenTestDB(t)
	now := time.Date(2026, 10, 7, 8, 0, 0, 0, time.UTC).Truncate(time.Microsecond)
	first := SeenState{UserID: 12, TopicID: 31, LastSeenAt: now.Add(-2 * time.Hour), SeenContentAt: now.Add(-2 * time.Hour), LastSeenProofID: "proof-a", ExpiresAt: now.Add(time.Hour)}
	if err := UpsertSeenTx(conn, []SeenState{first}); err != nil {
		t.Fatal(err)
	}
	replay := SeenState{UserID: 12, TopicID: 31, LastSeenAt: now.Add(-time.Hour), SeenContentAt: now.Add(-time.Hour), LastSeenProofID: "proof-a", ExpiresAt: now.Add(2 * time.Hour)}
	if err := UpsertSeenTx(conn, []SeenState{replay}); err != nil {
		t.Fatalf("replayed proof failed: %v", err)
	}
	if got := readSeen(t, conn, 12, 31); got != first {
		t.Fatalf("replayed proof changed the row: %#v != %#v", got, first)
	}
	// Control: a different proof advances the watermark and retention.
	next := SeenState{UserID: 12, TopicID: 31, LastSeenAt: now.Add(-time.Minute), SeenContentAt: now.Add(-time.Minute), LastSeenProofID: "proof-b", ExpiresAt: now.Add(3 * time.Hour)}
	if err := UpsertSeenTx(conn, []SeenState{next}); err != nil {
		t.Fatal(err)
	}
	if got := readSeen(t, conn, 12, 31); got != next {
		t.Fatalf("a new proof did not advance the row: %#v != %#v", got, next)
	}
}

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

package agentEvents

import (
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func schemaCheck(t *testing.T, conn *gorm.DB) {
	t.Helper()
	if err := conn.AutoMigrate(&ReplayState{}, &Publication{}, &Intent{}, &Entity{}, &agents.Entity{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	e := Entity{ID: "evt_one", InstanceID: "schema", AgentID: 41, Seq: 1, SourceIntentID: "src_one", Type: "agent.mentioned", Reasons: []string{"mention"}, OccurredAt: now, ExpiresAt: now.Add(time.Hour)}
	if err := conn.Create(&e).Error; err != nil {
		t.Fatal(err)
	}
	dup := e
	dup.ID = "evt_duplicate"
	dup.SourceIntentID = "src_other"
	if err := conn.Create(&dup).Error; err == nil {
		t.Fatal("duplicate per-Agent seq accepted")
	}
	dup.Seq = 2
	dup.SourceIntentID = e.SourceIntentID
	if err := conn.Create(&dup).Error; err == nil {
		t.Fatal("duplicate source Agent accepted")
	}
	dup.ID = "evt_other_instance"
	dup.InstanceID = "other"
	if err := conn.Create(&dup).Error; err != nil {
		t.Fatalf("instance namespace not independent %v", err)
	}
	intent := Intent{ID: "src_list", InstanceID: "schema", PostID: 3, Version: 1, Recipients: []Recipient{{AgentID: 41, Reasons: []string{"mention"}, SubscriptionGeneration: 1}}, Status: "failed", ExpiresAt: now.Add(time.Hour)}
	if err := conn.Create(&intent).Error; err != nil {
		t.Fatal(err)
	}
	rows, total, err := PageIntentsForAgentTx(conn, "schema", 41, 1, 10)
	if err != nil || len(rows) != 1 || total != 1 {
		t.Fatalf("recipient page %#v %d %v", rows, total, err)
	}
	_, total, err = PageIntentsForAgentTx(conn, "schema", 42, 1, 10)
	if err != nil || total != 0 {
		t.Fatalf("recipient isolation %d %v", total, err)
	}
}
func TestSchemaNamespaceAndUniqueSQLite(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	schemaCheck(t, conn)
}
func pgDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		dsn = os.Getenv("TEST_PG_DSN")
	}
	if dsn == "" {
		t.Skip("PostgreSQL DSN not configured")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("agent_events_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if parsed, err := url.Parse(dsn); err == nil && (parsed.Scheme == "postgres" || parsed.Scheme == "postgresql") {
		q := parsed.Query()
		q.Set("search_path", schema)
		parsed.RawQuery = q.Encode()
		dsn = parsed.String()
	} else {
		dsn += " search_path=" + schema
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sqlDB, _ := conn.DB()
		_ = sqlDB.Close()
		_ = admin.Exec("DROP SCHEMA " + schema + " CASCADE").Error
		adminDB, _ := admin.DB()
		_ = adminDB.Close()
	})
	return conn
}
func TestPostgreSQLAgentEventSchemaIsolation(t *testing.T) { schemaCheck(t, pgDB(t)) }
func TestPostgreSQLEventSequenceHoldsCommitOrder(t *testing.T) {
	conn := pgDB(t)
	if err := conn.AutoMigrate(&agents.Entity{}, &Entity{}); err != nil {
		t.Fatal(err)
	}
	agent := agents.Entity{UserId: 51, TokenPrefix: "seq-agent", Enabled: 1}
	if err := conn.Create(&agent).Error; err != nil {
		t.Fatal(err)
	}
	first := conn.Begin()
	defer first.Rollback()
	seq, err := agents.ReserveEventSeqTx(first, agent.UserId)
	if err != nil || seq != 1 {
		t.Fatalf("first %d %v", seq, err)
	}
	now := time.Now().UTC()
	one := Entity{ID: "evt_first", InstanceID: "commit-order", AgentID: agent.UserId, Seq: seq, SourceIntentID: "src_first", Reasons: []string{}, OccurredAt: now, ExpiresAt: now.Add(time.Hour)}
	if err := CreateEventTx(first, &one); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	secondReserved := make(chan uint64, 1)
	done := make(chan error, 1)
	go func() {
		done <- conn.Transaction(func(tx *gorm.DB) error {
			close(started)
			seq, err := agents.ReserveEventSeqTx(tx, agent.UserId)
			if err != nil {
				return err
			}
			secondReserved <- seq
			two := one
			two.ID = "evt_second"
			two.SourceIntentID = "src_second"
			two.Seq = seq
			return CreateEventTx(tx, &two)
		})
	}()
	<-started
	select {
	case seq := <-secondReserved:
		t.Fatalf("later commit allocated %d before first commit", seq)
	case <-time.After(100 * time.Millisecond):
	}
	rows, err := ListTx(conn, "commit-order", agent.UserId, 0, 10)
	if err != nil || len(rows) != 0 {
		t.Fatalf("uncommitted source visible %#v %v", rows, err)
	}
	if err := first.Commit().Error; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("second writer stuck")
	}
	rows, err = ListTx(conn, "commit-order", agent.UserId, 0, 10)
	if err != nil || len(rows) != 2 || rows[0].Seq != 1 || rows[1].Seq != 2 {
		t.Fatalf("committed stream %#v %v", rows, err)
	}
}

func TestExpiredIntentCleanupDoesNotCollide(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&Intent{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for index := 1; index <= 2; index++ {
		row := Intent{ID: fmt.Sprintf("expired-%d", index), InstanceID: "expiry", PostID: uint64(index), Version: 1, Recipients: []Recipient{}, ExpiresAt: now.Add(-time.Hour)}
		if err := conn.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := ExpireIntentsTx(conn, "expiry", now, 500); err != nil {
		t.Fatalf("cleanup failed for multiple expired source occurrences: %v", err)
	}
	var count int64
	if err := conn.Model(&Intent{}).Where("status = ?", "expired").Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("expired count %d, %v", count, err)
	}
}

func TestPurgedDiagnosticsKeepNamespacedReplayFloor(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&Entity{}, &Intent{}, &ReplayState{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for seq := uint64(1); seq <= 2; seq++ {
		expiry := now.Add(-40 * 24 * time.Hour)
		if seq == 1 {
			expiry = now.Add(time.Hour)
		}
		e := Entity{ID: fmt.Sprintf("floor-%d", seq), InstanceID: "floor", AgentID: 1, Seq: seq, SourceIntentID: fmt.Sprintf("src-%d", seq), Reasons: []string{}, OccurredAt: now, ExpiresAt: expiry}
		if err := conn.Create(&e).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { return PurgeMetadataTx(tx, "floor", now.Add(-30*24*time.Hour), 500) }); err != nil {
		t.Fatal(err)
	}
	floor, err := ReplayFloorTx(conn, "floor", 1, now)
	if err != nil || floor != 0 {
		t.Fatalf("purged higher sequence skipped retained earlier event: %d %v", floor, err)
	}
	if err := conn.Model(&Entity{}).Where("id = ?", "floor-1").Update("expires_at", now.Add(-40*24*time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { return PurgeMetadataTx(tx, "floor", now.Add(-30*24*time.Hour), 500) }); err != nil {
		t.Fatal(err)
	}
	floor, err = ReplayFloorTx(conn, "floor", 1, now)
	if err != nil || floor != 2 {
		t.Fatalf("purged floor went backwards: %d %v", floor, err)
	}
	foreign, err := ReplayFloorTx(conn, "other", 1, now)
	if err != nil || foreign != 0 {
		t.Fatalf("foreign instance floor leaked: %d %v", foreign, err)
	}
}

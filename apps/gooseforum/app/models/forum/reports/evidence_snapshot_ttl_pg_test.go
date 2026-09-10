package reports

import (
	"os"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/sqlconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
)

// TestClearExpiredEvidenceSnapshotsPostgres 在 PostgreSQL 上验证证据快照 TTL 清理
// 查询（review MEDIUM）：`evidence_snapshot != ''` 会对 json 列做 `json <> unknown`
// 比较，PG 抛 42883。修复后 SQL 层不再与空字符串比较（只保留 IS NOT NULL），
// 空快照交给 Go 层 evidenceSnapshotIsEmpty 过滤。由 TEST_PG_DSN 门控。
func TestClearExpiredEvidenceSnapshotsPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set; skipping PostgreSQL integration test")
	}

	conn := sqlconnect.GetConnect(sqlconnect.Config{
		Connection:         "postgres",
		DbUrl:              dsn,
		MaxIdleConnections: 1,
		MaxOpenConnections: 2,
		MaxLifeSeconds:     60,
	})
	if conn.Error != nil {
		t.Fatalf("connect postgres: %v", conn.Error)
	}
	if conn.Connect == nil {
		t.Fatal("postgres connection is nil")
	}
	sqlDB, err := conn.Connect.DB()
	if err != nil {
		t.Fatalf("sqlDB: %v", err)
	}
	defer sqlDB.Close()

	db := conn.Connect
	if err := db.AutoMigrate(&Entity{}, &topics.Entity{}); err != nil {
		t.Fatalf("AutoMigrate reports/topics on postgres: %v", err)
	}
	db.Exec("DELETE FROM reports WHERE id IN (?,?,?)", 880001, 880002, 880003)
	db.Exec("DELETE FROM topics WHERE id IN (?,?)", 880101, 880102)

	now := time.Now()
	oldHandled := now.Add(-200 * 24 * time.Hour)
	snap := EvidenceSnapshotData{TargetType: TargetTopic, TargetID: 880101, TopicID: 880101, Title: "pg-ttl", CreatedAt: now}

	if err := db.Create(&topics.Entity{Id: 880101, Title: "normal", RetentionStatus: topics.RetentionNormal, Status: 1}).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	if err := db.Create(&topics.Entity{Id: 880102, Title: "hold", RetentionStatus: topics.RetentionLegalHold, Status: 1}).Error; err != nil {
		t.Fatalf("create hold topic: %v", err)
	}
	closed := Entity{
		Id: 880001, TargetType: TargetTopic, TargetId: 880101, TopicId: 880101,
		ReporterId: 1, Reason: ReasonSpam, Status: StatusResolved, Resolution: ResolutionBanned,
		HandlerId: 1, HandledAt: &oldHandled, EvidenceSnapshot: snap,
	}
	held := Entity{
		Id: 880002, TargetType: TargetTopic, TargetId: 880102, TopicId: 880102,
		ReporterId: 2, Reason: ReasonIllegal, Status: StatusResolved, Resolution: ResolutionBanned,
		HandlerId: 1, HandledAt: &oldHandled, EvidenceSnapshot: snap,
	}
	emptySnap := Entity{
		Id: 880003, TargetType: TargetTopic, TargetId: 880101, TopicId: 880101,
		ReporterId: 3, Reason: ReasonOther, Status: StatusResolved, Resolution: ResolutionIgnored,
		HandlerId: 1, HandledAt: &oldHandled,
	}
	for _, e := range []Entity{closed, held, emptySnap} {
		if err := db.Create(&e).Error; err != nil {
			t.Fatalf("create report %d: %v", e.Id, err)
		}
	}

	cleared, err := clearExpiredEvidenceSnapshots(db, now.Add(-180*24*time.Hour), 50)
	if err != nil {
		t.Fatalf("clearExpiredEvidenceSnapshots on postgres failed: %v", err)
	}
	if cleared < 1 {
		t.Fatalf("cleared=%d, want >=1 (closed non-empty snapshot)", cleared)
	}

	// 用 PG 连接直接查证（Get 走全局 builder，指向测试库 sqlite，不能用于断言 PG 数据）。
	var gotClosed Entity
	if err := db.Where("id = ?", 880001).First(&gotClosed).Error; err != nil {
		t.Fatalf("read closed report on PG: %v", err)
	}
	if gotClosed.EvidenceSnapshot.Title != "" {
		t.Fatalf("closed snapshot title=%q, want empty after TTL on PG", gotClosed.EvidenceSnapshot.Title)
	}
	var gotHeld Entity
	if err := db.Where("id = ?", 880002).First(&gotHeld).Error; err != nil {
		t.Fatalf("read held report on PG: %v", err)
	}
	if gotHeld.EvidenceSnapshot.Title != "pg-ttl" {
		t.Fatalf("LEGAL_HOLD snapshot cleared unexpectedly on PG: %q", gotHeld.EvidenceSnapshot.Title)
	}
}

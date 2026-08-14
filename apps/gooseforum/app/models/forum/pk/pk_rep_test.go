package pk

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
)

// review: CleanupOldFetchLogs 必须 Unscoped 物理删除，否则软删的 stale running 行
// 仍占用 running_key 唯一索引（uniq_pk_fetch_log_running_key），同 calendar 的
// 新 CreateFetchLog 会唯一冲突。
func TestCleanupOldFetchLogsPhysicallyRemovesStaleRunningRow(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&FetchLogEntity{}); err != nil {
		t.Fatalf("migrate fetch log: %v", err)
	}
	const calendarID = uint64(982001)

	created, err := CreateFetchLog(calendarID)
	if err != nil {
		t.Fatalf("CreateFetchLog: %v", err)
	}
	// 回拨 created_at 至 30 天前，使其落入清理窗口。
	// 注意：created_at 带 `<-:create` 标签，gorm 会过滤对该列的 Update，
	// 必须走原始 SQL 才能回拨（review P2 测试修正）。
	past := time.Now().Add(-31 * 24 * time.Hour)
	if err := conn.Exec(
		"UPDATE "+fetchLogTableName+" SET created_at = ? WHERE id = ?",
		past, created.Id,
	).Error; err != nil {
		t.Fatalf("backdate created_at: %v", err)
	}

	if err := CleanupOldFetchLogs(); err != nil {
		t.Fatalf("CleanupOldFetchLogs: %v", err)
	}

	// 物理删除：Unscoped 也查不到残留行。
	var remain int64
	if err := conn.Unscoped().Table(fetchLogTableName).Where("id = ?", created.Id).Count(&remain).Error; err != nil {
		t.Fatalf("count remaining: %v", err)
	}
	if remain != 0 {
		t.Fatalf("stale fetch log row still present after cleanup: %d rows", remain)
	}

	// running_key 唯一索引已释放：同一 calendar 可重新创建 running 日志。
	if _, err := CreateFetchLog(calendarID); err != nil {
		t.Fatalf("CreateFetchLog after cleanup: %v (running_key unique index still occupied?)", err)
	}
}

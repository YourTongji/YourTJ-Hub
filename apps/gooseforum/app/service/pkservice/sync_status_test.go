package pkservice

import (
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

// TestResolveSyncTerm 覆盖数字 calendarId / 学期名反查 / 空与未知参数。
func TestResolveSyncTerm(t *testing.T) {
	migratePkTables(t)

	t.Run("空参数报错", func(t *testing.T) {
		if _, _, err := ResolveSyncTerm("  "); err == nil {
			t.Fatal("expected error for empty term")
		}
	})

	t.Run("数字 calendarId 直接解析", func(t *testing.T) {
		id, term, err := ResolveSyncTerm("121")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != 121 || term != "121" {
			t.Fatalf("got id=%d term=%q, want 121/121", id, term)
		}
	})

	t.Run("学期名经 pk_calendar 反查", func(t *testing.T) {
		now := time.Now()
		conn := db.Connect()
		if err := conn.Create(&pk.CalendarEntity{
			CalendarId: 121, CalendarIdI18n: "2025-2026-1", SchemaVersion: pk.PKDataSchemaVersion, SyncedAt: &now,
		}).Error; err != nil {
			t.Fatalf("seed calendar: %v", err)
		}
		id, term, err := ResolveSyncTerm("2025-2026-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != 121 || term != "2025-2026-1" {
			t.Fatalf("got id=%d term=%q, want 121/2025-2026-1", id, term)
		}
	})

	t.Run("未知学期名报错", func(t *testing.T) {
		if _, _, err := ResolveSyncTerm("2020-2021-9"); err == nil {
			t.Fatal("expected error for unknown term")
		}
	})
}

// TestSyncStatusOverview 覆盖骨架学期 + 附日志 + 失败未写入学期 + 倒序。
func TestSyncStatusOverview(t *testing.T) {
	migratePkTables(t)
	conn := db.Connect()
	now := time.Now()

	seed := func(id uint64, i18n string) {
		t.Helper()
		if err := conn.Create(&pk.CalendarEntity{
			CalendarId: id, CalendarIdI18n: i18n, SchemaVersion: pk.PKDataSchemaVersion, SyncedAt: &now,
		}).Error; err != nil {
			t.Fatalf("seed calendar %d: %v", id, err)
		}
	}
	seed(121, "2025-2026-1")
	seed(120, "2025-2026-2")
	seed(119, "2025-2026-3")

	seedLog := func(id uint64, status string, rows int, errMsg string) {
		t.Helper()
		if err := conn.Create(&pk.FetchLogEntity{
			CalendarId: id, Status: status, RowsWritten: rows, ErrorMsg: errMsg, StartedAt: &now, FinishedAt: &now, SchemaVersion: pk.PKDataSchemaVersion,
		}).Error; err != nil {
			t.Fatalf("seed fetch log %d: %v", id, err)
		}
	}
	// 121 成功一次 + 再失败一次（应取最新失败）。
	seedLog(121, pk.FetchStatusCompleted, 3000, "")
	seedLog(121, pk.FetchStatusFailed, 1500, "cookie 失效")
	// 120 运行中。
	seedLog(120, pk.FetchStatusRunning, 800, "")
	// 118 无 calendar 行（首次同步失败未写入）。
	seedLog(118, pk.FetchStatusFailed, 0, "缺少 cookie")
	// 117 无 calendar 行、running 已超 1 小时窗口（进程中断）→ 视为失败。
	staleAt := now.Add(-2 * time.Hour)
	if err := conn.Create(&pk.FetchLogEntity{
		CalendarId: 117, Status: pk.FetchStatusRunning, RowsWritten: 100,
		StartedAt: &staleAt, SchemaVersion: pk.PKDataSchemaVersion,
	}).Error; err != nil {
		t.Fatalf("seed stale fetch log 117: %v", err)
	}

	items, err := SyncStatusOverview()
	if err != nil {
		t.Fatalf("SyncStatusOverview: %v", err)
	}

	// 倒序且 118/117 补入：121,120,119,118,117。
	if len(items) != 5 {
		t.Fatalf("got %d items, want 5: %+v", len(items), items)
	}
	expected := []struct {
		id     uint64
		status string
	}{
		{121, pk.FetchStatusFailed},
		{120, pk.FetchStatusRunning},
		{119, ""},
		{118, pk.FetchStatusFailed},
		{117, pk.FetchStatusFailed},
	}
	for i, want := range expected {
		got := items[i]
		if got.CalendarId != want.id {
			t.Errorf("item %d: calendarId=%d, want %d", i, got.CalendarId, want.id)
		}
		if got.Status != want.status {
			t.Errorf("item %d: status=%q, want %q", i, got.Status, want.status)
		}
	}
	// 121 的 rowsWritten 应为最新一次（1500），errorMsg 非空。
	if items[0].RowsWritten != 1500 {
		t.Errorf("121 rowsWritten=%d, want 1500", items[0].RowsWritten)
	}
	// 119 无 fetchlog：rowsWritten 归零。
	if items[2].RowsWritten != 0 {
		t.Errorf("119 rowsWritten=%d, want 0", items[2].RowsWritten)
	}
	// 117 stale running 应标记 failed 且带中断说明。
	if items[4].ErrorMsg == "" {
		t.Error("117 stale running should carry an interruption error message")
	}
}

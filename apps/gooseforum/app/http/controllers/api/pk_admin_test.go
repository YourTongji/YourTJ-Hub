package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pkservice"
)

// setupPkAdminTest 迁移并清空 PK 域表与操作审计表（controller 测试用）。
func setupPkAdminTest(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	models := []any{&pk.CalendarEntity{}, &pk.FetchLogEntity{}, &optRecord.Entity{}}
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate pk tables: %v", err)
	}
	for _, m := range models {
		if err := conn.Unscoped().Where("1 = 1").Delete(m).Error; err != nil {
			t.Fatalf("clean pk table: %v", err)
		}
	}
}

func TestSyncPkCalendarRejectsInvalidParams(t *testing.T) {
	setupPkAdminTest(t)
	t.Setenv("ONESYSTEM_COOKIE", "JWTUser=abc")

	// 空学期。
	res := SyncPkCalendar(component.BetterRequest[SyncPkCalendarReq]{Params: SyncPkCalendarReq{Term: "  "}})
	if res.Data.Code != component.FAIL {
		t.Errorf("empty term: expected FAIL, got %v", res.Data.Code)
	}
	// 未知学期名。
	res = SyncPkCalendar(component.BetterRequest[SyncPkCalendarReq]{Params: SyncPkCalendarReq{Term: "2020-2021-9"}})
	if res.Data.Code != component.FAIL {
		t.Errorf("unknown term: expected FAIL, got %v", res.Data.Code)
	}
}

func TestSyncPkCalendarRejectsMissingCookie(t *testing.T) {
	setupPkAdminTest(t)
	t.Setenv("ONESYSTEM_COOKIE", "")

	// 数字 calendarId 可解析，但无任何 cookie 来源 → 拒绝。
	res := SyncPkCalendar(component.BetterRequest[SyncPkCalendarReq]{Params: SyncPkCalendarReq{Term: "121"}})
	if res.Data.Code != component.FAIL {
		t.Errorf("missing cookie: expected FAIL, got %v", res.Data.Code)
	}
}

func TestSyncPkCalendarStartsAsyncSync(t *testing.T) {
	setupPkAdminTest(t)
	t.Setenv("ONESYSTEM_COOKIE", "JWTUser=abc")

	var gotCalendarId uint64
	var gotDepth int
	syncCalled := make(chan struct{})
	orig := runPkSync
	runPkSync = func(_ context.Context, _ string, calendarId uint64, depth int, _ bool) (*pkservice.SyncReport, error) {
		gotCalendarId = calendarId
		gotDepth = depth
		close(syncCalled)
		return &pkservice.SyncReport{}, nil
	}
	t.Cleanup(func() { runPkSync = orig })

	res := SyncPkCalendar(component.BetterRequest[SyncPkCalendarReq]{Params: SyncPkCalendarReq{Term: "121", Depth: 0}})
	if res.Code != http.StatusOK || res.Data.Code != component.SUCCESS {
		t.Fatalf("sync start failed: code=%d data=%+v", res.Code, res.Data)
	}
	result, ok := res.Data.Result.(map[string]any)
	if !ok || result["started"] != true {
		t.Fatalf("result = %+v, want started=true", res.Data.Result)
	}

	select {
	case <-syncCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("async sync stub was not invoked")
	}
	if gotCalendarId != 121 {
		t.Errorf("stub calendarId = %d, want 121", gotCalendarId)
	}
	if gotDepth != 1 {
		t.Errorf("stub depth = %d, want 1 (default)", gotDepth)
	}
}

func TestSyncPkCalendarClampsDepth(t *testing.T) {
	setupPkAdminTest(t)
	t.Setenv("ONESYSTEM_COOKIE", "JWTUser=abc")

	var gotDepth int
	syncCalled := make(chan struct{})
	orig := runPkSync
	runPkSync = func(_ context.Context, _ string, _ uint64, depth int, _ bool) (*pkservice.SyncReport, error) {
		gotDepth = depth
		close(syncCalled)
		return &pkservice.SyncReport{}, nil
	}
	t.Cleanup(func() { runPkSync = orig })

	res := SyncPkCalendar(component.BetterRequest[SyncPkCalendarReq]{Params: SyncPkCalendarReq{Term: "121", Depth: 99}})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("sync start failed: %+v", res.Data)
	}
	select {
	case <-syncCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("async sync stub was not invoked")
	}
	if gotDepth != maxPkSyncDepth {
		t.Errorf("stub depth = %d, want clamped %d", gotDepth, maxPkSyncDepth)
	}
}

func TestPkSyncStatusReturnsOverview(t *testing.T) {
	setupPkAdminTest(t)
	conn := db.Connect()
	now := time.Now()
	if err := conn.Create(&pk.CalendarEntity{
		CalendarId: 121, CalendarIdI18n: "2025-2026-1", SchemaVersion: pk.PKDataSchemaVersion, SyncedAt: &now,
	}).Error; err != nil {
		t.Fatalf("seed calendar: %v", err)
	}
	if err := conn.Create(&pk.FetchLogEntity{
		CalendarId: 121, Status: pk.FetchStatusCompleted, RowsWritten: 3000,
		StartedAt: &now, FinishedAt: &now, SchemaVersion: pk.PKDataSchemaVersion,
	}).Error; err != nil {
		t.Fatalf("seed fetch log: %v", err)
	}

	res := PkSyncStatus(component.BetterRequest[component.Null]{})
	if res.Code != http.StatusOK || res.Data.Code != component.SUCCESS {
		t.Fatalf("sync status failed: code=%d data=%+v", res.Code, res.Data)
	}
	items, ok := res.Data.Result.([]pkservice.SyncStatusItem)
	if !ok {
		t.Fatalf("result type = %T, want []pkservice.SyncStatusItem", res.Data.Result)
	}
	if len(items) != 1 || items[0].CalendarId != 121 || items[0].Status != pk.FetchStatusCompleted {
		t.Fatalf("items = %+v, want 1 item calendarId=121 status=completed", items)
	}
}

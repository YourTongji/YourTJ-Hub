package pkservice

import (
	"context"
	"strings"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// persistSyncSchedule runs the schedule config with the given fields and clears the hot cache.
func persistSyncSchedule(t *testing.T, cfg pageConfig.PkSyncScheduleConfig) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate page_config: %v", err)
	}
	pageConfig.UpdatePkSyncScheduleConfig(func(_ pageConfig.PkSyncScheduleConfig) pageConfig.PkSyncScheduleConfig {
		return cfg
	})
	hotdataserve.ClearPkSyncScheduleConfigCache()
	t.Cleanup(hotdataserve.ClearPkSyncScheduleConfigCache)
}

func defaultScheduledConfig() pageConfig.PkSyncScheduleConfig {
	return pageConfig.PkSyncScheduleConfig{Enabled: true, Schedule: "30 2 * * *", Term: "121", Depth: 1, Audience: "undergraduate"}
}

func TestResolveScheduledSyncTerm(t *testing.T) {
	migratePkTables(t)
	conn := db.Connect()
	// 无任何已同步学期：空 term 报错。
	if _, err := resolveScheduledSyncTerm(AudienceUndergraduate, ""); err == nil {
		t.Fatal("empty term with no calendar: expected error")
	}
	// 数值 calendarId 直接解析。
	id, err := resolveScheduledSyncTerm(AudienceUndergraduate, "121")
	if err != nil || id != 121 {
		t.Fatalf("numeric term: id=%d err=%v", id, err)
	}
	// 空 term 取最近已同步学期（calendarId 倒序第一）。
	now := time.Now()
	for _, c := range []*pk.CalendarEntity{
		{CalendarId: 120, CalendarIdI18n: "2024-2025-2", SchemaVersion: pk.PKDataSchemaVersion, SyncedAt: &now},
		{CalendarId: 121, CalendarIdI18n: "2025-2026-1", SchemaVersion: pk.PKDataSchemaVersion, SyncedAt: &now},
	} {
		if err := conn.Create(c).Error; err != nil {
			t.Fatal(err)
		}
	}
	id, err = resolveScheduledSyncTerm(AudienceUndergraduate, "")
	if err != nil || id != 121 {
		t.Fatalf("latest term: id=%d err=%v", id, err)
	}
}

func TestRunScheduledSyncDisabled(t *testing.T) {
	migratePkTables(t)
	persistSyncSchedule(t, pageConfig.PkSyncScheduleConfig{Enabled: false})
	if err := RunScheduledSync(context.Background()); err == nil || !strings.Contains(err.Error(), "未启用") {
		t.Fatalf("disabled: err = %v, want 未启用", err)
	}
}

func TestRunScheduledSyncInvalidAudience(t *testing.T) {
	migratePkTables(t)
	cfg := defaultScheduledConfig()
	cfg.Audience = "master"
	persistSyncSchedule(t, cfg)
	if err := RunScheduledSync(context.Background()); err == nil || !strings.Contains(err.Error(), "数据来源") {
		t.Fatalf("invalid audience: err = %v", err)
	}
}

func TestRunScheduledSyncEmptyTermWithoutCalendar(t *testing.T) {
	migratePkTables(t)
	cfg := defaultScheduledConfig()
	cfg.Term = ""
	persistSyncSchedule(t, cfg)
	t.Setenv("ONESYSTEM_COOKIE", "JWTUser=abc")
	if err := RunScheduledSync(context.Background()); err == nil || !strings.Contains(err.Error(), "pk_calendar") {
		t.Fatalf("empty term without calendar: err = %v", err)
	}
}

func TestRunScheduledSyncMissingCookie(t *testing.T) {
	migratePkTables(t)
	persistSyncSchedule(t, defaultScheduledConfig())
	t.Setenv("ONESYSTEM_COOKIE", "")
	if err := RunScheduledSync(context.Background()); err == nil || !strings.Contains(err.Error(), "Cookie") {
		t.Fatalf("missing cookie: err = %v", err)
	}
}

func TestRunScheduledSyncExecutesPipelineWithClaim(t *testing.T) {
	migratePkTables(t)
	persistSyncSchedule(t, defaultScheduledConfig())
	t.Setenv("ONESYSTEM_COOKIE", "JWTUser=abc")

	type call struct {
		audience    Audience
		calendar    uint64
		depth       int
		claim       *pk.FetchLogEntity
		resume      bool
		materialize bool
	}
	got := make(chan call, 1)
	orig := scheduledSyncExecutor
	scheduledSyncExecutor = func(_ context.Context, _ string, audience Audience, calendarId uint64, depth int, materialize bool, claim *pk.FetchLogEntity, resume bool) (*SyncReport, error) {
		got <- call{audience, calendarId, depth, claim, resume, materialize}
		return &SyncReport{TeachingClassInserted: 3, CalendarIDs: []uint64{121}}, nil
	}
	t.Cleanup(func() { scheduledSyncExecutor = orig })

	if err := RunScheduledSync(context.Background()); err != nil {
		t.Fatalf("scheduled sync failed: %v", err)
	}
	c := <-got
	if c.audience != AudienceUndergraduate || c.calendar != 121 || c.depth != 1 || !c.materialize || c.claim == nil {
		t.Fatalf("executor call = %+v", c)
	}
	if c.resume {
		t.Fatalf("fresh run must not resume")
	}

	// 失败遗留 fetchlog → 续跑（resume=true）。先清掉首个 stub 运行遗留的 running
	// 租约（真实完成路径会由 finishCalendarSync 置为 completed），再写入 failed 行。
	if err := db.Connect().Unscoped().Where("1 = 1").Delete(&pk.FetchLogEntity{}).Error; err != nil {
		t.Fatal(err)
	}
	failedAt := time.Now()
	if err := db.Connect().Create(&pk.FetchLogEntity{
		CalendarId: 121, Status: pk.FetchStatusFailed, ErrorMsg: "前次失败", StartedAt: &failedAt, FinishedAt: &failedAt,
		SchemaVersion: pk.PKDataSchemaVersion,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := RunScheduledSync(context.Background()); err != nil {
		t.Fatalf("resume sync failed: %v", err)
	}
	c = <-got
	if !c.resume || c.claim == nil || c.claim.Status != pk.FetchStatusRunning {
		t.Fatalf("resume executor call = %+v", c)
	}
}

func TestRunScheduledSyncConcurrentGuard(t *testing.T) {
	migratePkTables(t)
	persistSyncSchedule(t, defaultScheduledConfig())
	t.Setenv("ONESYSTEM_COOKIE", "JWTUser=abc")

	// 1 小时 running 窗口内的并发触发被重入保护拒绝。
	started := time.Now()
	if err := db.Connect().Create(&pk.FetchLogEntity{
		CalendarId: 121, Status: pk.FetchStatusRunning, StartedAt: &started, SchemaVersion: pk.PKDataSchemaVersion,
	}).Error; err != nil {
		t.Fatal(err)
	}
	err := RunScheduledSync(context.Background())
	if err == nil || !strings.Contains(err.Error(), "同步正在进行中") {
		t.Fatalf("concurrent trigger: err = %v", err)
	}
}

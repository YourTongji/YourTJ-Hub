package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
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

	var gotMaterialize bool
	var gotCalendarId uint64
	var gotDepth int
	syncCalled := make(chan struct{})
	orig := runPkSync
	runPkSync = func(_ context.Context, _ string, calendarId uint64, depth int, materialize bool, _ *pk.FetchLogEntity, _ bool) (*pkservice.SyncReport, error) {
		gotMaterialize = materialize
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
	if !gotMaterialize {
		t.Error("admin sync must materialize the course catalog")
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
	runPkSync = func(_ context.Context, _ string, _ uint64, depth int, _ bool, _ *pk.FetchLogEntity, _ bool) (*pkservice.SyncReport, error) {
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

func TestSyncPkCalendarRejectsConcurrentStart(t *testing.T) {
	setupPkAdminTest(t)
	t.Setenv("ONESYSTEM_COOKIE", "JWTUser=abc")

	syncStarted := make(chan struct{})
	releaseSync := make(chan struct{})
	syncFinished := make(chan struct{})
	orig := runPkSync
	runPkSync = func(_ context.Context, _ string, _ uint64, _ int, _ bool, _ *pk.FetchLogEntity, _ bool) (*pkservice.SyncReport, error) {
		close(syncStarted)
		<-releaseSync
		close(syncFinished)
		return &pkservice.SyncReport{}, nil
	}
	t.Cleanup(func() { runPkSync = orig })

	first := SyncPkCalendar(component.BetterRequest[SyncPkCalendarReq]{Params: SyncPkCalendarReq{Term: "121"}})
	if first.Data.Code != component.SUCCESS {
		t.Fatalf("first start failed: %+v", first.Data)
	}
	select {
	case <-syncStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("async sync stub was not invoked")
	}

	second := SyncPkCalendar(component.BetterRequest[SyncPkCalendarReq]{Params: SyncPkCalendarReq{Term: "121"}})
	if second.Data.Code != component.FAIL {
		t.Fatalf("concurrent start: expected FAIL, got %+v", second.Data)
	}
	close(releaseSync)
	select {
	case <-syncFinished:
	case <-time.After(2 * time.Second):
		t.Fatal("async sync stub did not finish")
	}
}

func TestSyncPkCalendarMarksClaimFailedAfterPanic(t *testing.T) {
	setupPkAdminTest(t)
	t.Setenv("ONESYSTEM_COOKIE", "JWTUser=abc")

	panicFinished := make(chan struct{})
	orig := runPkSync
	runPkSync = func(_ context.Context, _ string, _ uint64, _ int, _ bool, _ *pk.FetchLogEntity, _ bool) (*pkservice.SyncReport, error) {
		defer close(panicFinished)
		panic("injected sync panic")
	}
	t.Cleanup(func() { runPkSync = orig })

	res := SyncPkCalendar(component.BetterRequest[SyncPkCalendarReq]{Params: SyncPkCalendarReq{Term: "121"}})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("sync start failed: %+v", res.Data)
	}
	select {
	case <-panicFinished:
	case <-time.After(2 * time.Second):
		t.Fatal("panic sync stub was not invoked")
	}

	deadline := time.After(2 * time.Second)
	var log pk.FetchLogEntity
	for {
		var ok bool
		log, ok = pk.LatestFetchLogByCalendar(121)
		if ok && log.Status == pk.FetchStatusFailed {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("fetch log after panic = %+v, want failed", log)
		case <-time.After(10 * time.Millisecond):
		}
	}
	if log.ErrorMsg == "" {
		t.Fatal("panic failure should be recorded in fetch log")
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

func TestValidatePkCredentialReturnsValidResult(t *testing.T) {
	orig := validatePkCredential
	validatePkCredential = func(_ context.Context, _ pkservice.Audience, _ string) (pkservice.CredentialValidation, error) {
		return pkservice.CredentialValidation{Valid: true}, nil
	}
	t.Cleanup(func() { validatePkCredential = orig })

	res := ValidatePkCredential(component.BetterRequest[ValidatePkCredentialReq]{
		Params: ValidatePkCredentialReq{Audience: "undergraduate", Credential: "JWTUser=abc"},
	})
	if res.Code != http.StatusOK || res.Data.Code != component.SUCCESS {
		t.Fatalf("validation result: code=%d data=%+v", res.Code, res.Data)
	}
	result, ok := res.Data.Result.(map[string]any)
	if !ok || result["valid"] != true || result["message"] != "" {
		t.Fatalf("result = %+v, want valid=true message=\"\"", res.Data.Result)
	}
}

func TestValidatePkCredentialInvalidIsBusinessResult(t *testing.T) {
	// 凭证失效（401/业务失败）是业务结果：成功信封 + valid=false + 脱敏 message，不是失败信封。
	orig := validatePkCredential
	validatePkCredential = func(_ context.Context, _ pkservice.Audience, _ string) (pkservice.CredentialValidation, error) {
		return pkservice.CredentialValidation{Valid: false, Message: `一系统请求失败: HTTP 401 {"message":"未登录或会话失效"}`}, nil
	}
	t.Cleanup(func() { validatePkCredential = orig })

	res := ValidatePkCredential(component.BetterRequest[ValidatePkCredentialReq]{
		Params: ValidatePkCredentialReq{Audience: "undergraduate", Credential: "bad"},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("credential failure must be a success envelope, got data=%+v", res.Data)
	}
	result, ok := res.Data.Result.(map[string]any)
	if !ok || result["valid"] != false {
		t.Fatalf("result = %+v, want valid=false", res.Data.Result)
	}
	if message, _ := result["message"].(string); !strings.Contains(message, "401") {
		t.Errorf("message = %q, want sanitized 401 hint", result["message"])
	}
}

func TestValidatePkCredentialRejectsInvalidAudience(t *testing.T) {
	res := ValidatePkCredential(component.BetterRequest[ValidatePkCredentialReq]{
		Params: ValidatePkCredentialReq{Audience: "bogus"},
	})
	if res.Data.Code != component.FAIL {
		t.Fatalf("invalid audience: expected FAIL, got %+v", res.Data)
	}
}

func TestValidatePkCredentialMapsHardErrorToFailure(t *testing.T) {
	orig := validatePkCredential
	validatePkCredential = func(_ context.Context, _ pkservice.Audience, _ string) (pkservice.CredentialValidation, error) {
		return pkservice.CredentialValidation{}, errors.New("缺少本科一系统 Cookie")
	}
	t.Cleanup(func() { validatePkCredential = orig })

	res := ValidatePkCredential(component.BetterRequest[ValidatePkCredentialReq]{
		Params: ValidatePkCredentialReq{Audience: "undergraduate"},
	})
	if res.Data.Code != component.FAIL {
		t.Fatalf("missing credential source: expected FAIL, got %+v", res.Data)
	}
}

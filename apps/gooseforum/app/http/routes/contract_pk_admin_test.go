package routes

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pkservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 本文件覆盖 SiteManager 权限组的排课数据同步管理端点（issue #248）的契约测试：
// POST /api/admin/pk/sync-calendar 与 GET /api/admin/pk/sync-status。中间件链与
// route4api.go 生产注册一致（JWTAuthCheck + CheckWritableAccount + CheckPermission(SiteManager)）。

// setupPkAdminContractTest 注册两条 PK 管理路由，迁移并清空 PK 域表与操作审计表。
func setupPkAdminContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(&pk.CalendarEntity{}, &pk.FetchLogEntity{}, &optRecord.Entity{}, &rolePermissionRs.Entity{}); err != nil {
		t.Fatalf("migrate pk admin contract tables: %v", err)
	}
	cleanupPkAdminTables(t, conn)
	t.Cleanup(func() { cleanupPkAdminTables(t, conn) })

	admin := router.Group("/api/admin",
		middleware.JWTAuthCheck,
		middleware.CheckWritableAccount,
		middleware.CheckPermission(permission.SiteManager),
	)
	admin.POST("/pk/sync-calendar", UpButterReq(api.SyncPkCalendar))
	admin.GET("/pk/sync-status", UpButterReq(api.PkSyncStatus))
	return conn, router
}

func cleanupPkAdminTables(t *testing.T, conn *gorm.DB) {
	t.Helper()
	for _, model := range []any{&pk.CalendarEntity{}, &pk.FetchLogEntity{}, &optRecord.Entity{}} {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean pk admin table: %v", err)
		}
	}
}

// adminPkGuardScenarios 跑两条路由公共的中间件守卫场景：
// 未登录 401 / 冻结账号 403 / 无 SiteManager 权限 403（params.permission="站点管理"）。
func adminPkGuardScenarios(t *testing.T, method, path, fixturePrefix string) {
	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupPkAdminContractTest(t)
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "auth-required.json"))
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupPkAdminContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		if err := conn.Model(user).Update("is_frozen", users.StatusFrozen).Error; err != nil {
			t.Fatalf("freeze contract user: %v", err)
		}
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("frozen account status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "account-frozen.json"))
	})

	t.Run("user without SiteManager returns 403", func(t *testing.T) {
		conn, router := setupPkAdminContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("permission denied status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-ai-summary-settings-permission-denied.json"))
	})
}

func TestAdminSyncPkCalendarHTTPContract(t *testing.T) {
	path := "/api/admin/pk/sync-calendar"

	t.Run("success starts an async sync", func(t *testing.T) {
		t.Setenv("ONESYSTEM_COOKIE", "JWTUser=contract-test")
		t.Cleanup(api.SetRunPkSyncForTest(func(_ context.Context, _ string, _ uint64, _ int, _ bool, _ *pk.FetchLogEntity, _ bool) (*pkservice.SyncReport, error) {
			return &pkservice.SyncReport{}, nil
		}))
		conn, router := setupPkAdminContractTest(t)
		manager := createContractSiteManager(t, conn)
		recorder := serveAuthSecurityJSON(router, http.MethodPost, path, `{"term":"121"}`, contractSessionToken(t, manager))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "pk-sync-calendar-success.json"))
	})

	t.Run("missing term fails request validation", func(t *testing.T) {
		conn, router := setupPkAdminContractTest(t)
		manager := createContractSiteManager(t, conn)
		recorder := serveAuthSecurityJSON(router, http.MethodPost, path, `{}`, contractSessionToken(t, manager))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "invalid-params.json"))
	})

	adminPkGuardScenarios(t, http.MethodPost, path, "pk-sync-calendar")
}

func TestAdminGetPkSyncStatusHTTPContract(t *testing.T) {
	path := "/api/admin/pk/sync-status"

	t.Run("success summarizes per-term sync status", func(t *testing.T) {
		conn, router := setupPkAdminContractTest(t)
		seedPkSyncStatus(t, conn)
		manager := createContractSiteManager(t, conn)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, path, "", contractSessionToken(t, manager))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "pk-sync-status-success.json"))
	})

	adminPkGuardScenarios(t, http.MethodGet, path, "pk-sync-status")
}

// seedPkSyncStatus 写入与 pk-sync-status-success.json 完全一致的数据：
// calendarId 121 学期已完成（含 pk_calendar 行），calendarId 120 首次同步失败（仅 fetchlog、
// 无 calendar 行，验证"有 fetchlog 但 calendar 尚未写入"的补充条目）。
func seedPkSyncStatus(t *testing.T, conn *gorm.DB) {
	t.Helper()
	completedAt := time.Date(2026, 8, 1, 3, 0, 0, 0, time.UTC)
	completedEnd := time.Date(2026, 8, 1, 3, 5, 12, 0, time.UTC)
	failedAt := time.Date(2026, 8, 16, 11, 10, 0, 0, time.UTC)
	failedEnd := time.Date(2026, 8, 16, 11, 11, 2, 0, time.UTC)
	if err := conn.Create(&pk.CalendarEntity{
		CalendarId: 121, CalendarIdI18n: "2025-2026-1", SchemaVersion: pk.PKDataSchemaVersion,
	}).Error; err != nil {
		t.Fatalf("seed calendar 121: %v", err)
	}
	for _, row := range []*pk.FetchLogEntity{
		{CalendarId: 121, Status: pk.FetchStatusCompleted, RowsWritten: 120, TotalPages: 12, LastCommittedPage: 12, ErrorMsg: "", StartedAt: &completedAt, FinishedAt: &completedEnd, SchemaVersion: pk.PKDataSchemaVersion},
		{CalendarId: 120, Status: pk.FetchStatusFailed, RowsWritten: 0, TotalPages: 3, LastCommittedPage: 0, ErrorMsg: "抓取一系统失败：HTTP 502", StartedAt: &failedAt, FinishedAt: &failedEnd, SchemaVersion: pk.PKDataSchemaVersion},
	} {
		if err := conn.Create(row).Error; err != nil {
			t.Fatalf("seed fetch log: %v", err)
		}
	}
}

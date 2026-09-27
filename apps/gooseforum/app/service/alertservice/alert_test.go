package alertservice

import (
	"errors"
	"fmt"
	"sync"

	"gorm.io/gorm"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/role"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
)

// setupAlertTestDB 迁移并清空告警链路涉及的表格（测试用内存 sqlite，跨用例共享）。
func setupAlertTestDB(t *testing.T) {
	t.Helper()
	models := []any{
		&users.EntityComplete{},
		&role.Entity{},
		&rolePermissionRs.Entity{},
		&eventNotification.Entity{},
	}
	conn := db.Connect()
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate alert tables: %v", err)
	}
	for _, m := range models {
		if err := conn.Unscoped().Where("1 = 1").Delete(m).Error; err != nil {
			t.Fatalf("clean alert table: %v", err)
		}
	}
	// 清掉告警去重标记与角色-权限缓存，保证用例从干净状态开始。
	pkAlertCache.Clear()
}

// newTestRole 建角色并绑定权限（默认 SiteManager），返回角色 id。
func newTestRole(t *testing.T, name string, perm permission.Enum) uint64 {
	t.Helper()
	r := &role.Entity{RoleName: name, Effective: 1}
	if err := role.SaveOrCreateById(r); err != nil {
		t.Fatalf("create role %s: %v", name, err)
	}
	if r.Id == 0 {
		t.Fatalf("role %s got id 0", name)
	}
	if affected := rolePermissionRs.SaveOrCreateById(&rolePermissionRs.Entity{RoleId: r.Id, PermissionId: perm.Id()}); affected != 1 {
		t.Fatalf("bind permission %d to role %s: rows affected = %d", perm.Id(), name, affected)
	}
	// permission.GetPermissionByRoleId 有 10 分钟本地缓存：立即失效，让 CheckRole
	// 读取本次写入的关系。
	permission.InvalidateRole(r.Id)
	return r.Id
}

func newTestUser(t *testing.T, username string, roleID uint64, softDelete bool) uint64 {
	t.Helper()
	u := users.MakeUser(username, "secret123", username+"@example.com")
	u.RoleId = roleID
	if err := users.Create(u); err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
	if u.Id == 0 {
		t.Fatalf("user %s got id 0", username)
	}
	if softDelete {
		if err := db.Connect().Where("id = ?", u.Id).Delete(&users.EntityComplete{}).Error; err != nil {
			t.Fatalf("soft delete user %s: %v", username, err)
		}
	}
	return u.Id
}

// systemNotificationCount 返回某用户收到的 system 通知条数。
func systemNotificationCount(t *testing.T, userID uint64) int {
	t.Helper()
	var count int64
	if err := db.Connect().Model(&eventNotification.Entity{}).
		Where("user_id = ? AND event_type = ?", userID, eventNotification.EventTypeSystem).
		Count(&count).Error; err != nil {
		t.Fatalf("count system notifications: %v", err)
	}
	return int(count)
}

func TestNotifyPkSyncFailedSendsEachSiteManagerOnce(t *testing.T) {
	setupAlertTestDB(t)

	adminRole := newTestRole(t, "alert-admin", permission.Admin)
	smRole := newTestRole(t, "alert-sm", permission.SiteManager)
	otherRole := newTestRole(t, "alert-other", permission.TopicsManager)

	adminUser := newTestUser(t, "alert-admin-user", adminRole, false)
	smUser := newTestUser(t, "alert-sm-user", smRole, false)
	otherUser := newTestUser(t, "alert-other-user", otherRole, false)
	closedAdmin := newTestUser(t, "alert-closed-admin", adminRole, true)

	NotifyPkSyncFailed("undergraduate", "401 Unauthorized: 未登录或会话失效")

	// Admin 与 SiteManager 用户各收 1 条；无权限用户和已注销用户 0 条。
	if got := systemNotificationCount(t, adminUser); got != 1 {
		t.Errorf("admin user notifications = %d, want 1", got)
	}
	if got := systemNotificationCount(t, smUser); got != 1 {
		t.Errorf("site manager user notifications = %d, want 1", got)
	}
	if got := systemNotificationCount(t, otherUser); got != 0 {
		t.Errorf("plain role user notifications = %d, want 0 (no SiteManager/Admin permission)", got)
	}
	if got := systemNotificationCount(t, closedAdmin); got != 0 {
		t.Errorf("closed admin user notifications = %d, want 0 (soft-deleted excluded)", got)
	}

	// 同一受众窗口内第二次失败：去重，不再发。
	NotifyPkSyncFailed("undergraduate", "401 Unauthorized: 未登录或会话失效")
	if got := systemNotificationCount(t, adminUser); got != 1 {
		t.Errorf("after dedupe admin user notifications = %d, want still 1", got)
	}
	if got := systemNotificationCount(t, smUser); got != 1 {
		t.Errorf("after dedupe site manager user notifications = %d, want still 1", got)
	}

	// 不同受众（研究生）是独立去重 key，会再发一组。
	NotifyPkSyncFailed("graduate", "session storage expired")
	if got := systemNotificationCount(t, adminUser); got != 2 {
		t.Errorf("after graduate alert admin user notifications = %d, want 2", got)
	}
	if got := systemNotificationCount(t, smUser); got != 2 {
		t.Errorf("after graduate alert site manager user notifications = %d, want 2", got)
	}

	// 空受众回落本科：窗口内未过期，仍被去重。
	NotifyPkSyncFailed("", "empty audience")
	if got := systemNotificationCount(t, adminUser); got != 2 {
		t.Errorf("empty-audience alert admin user notifications = %d, want 2 (undergraduate deduped)", got)
	}
}

func TestNotifyPkSyncFailedDedupeWindowExpires(t *testing.T) {
	setupAlertTestDB(t)

	smRole := newTestRole(t, "alert-window-sm", permission.SiteManager)
	smUser := newTestUser(t, "alert-window-sm-user", smRole, false)

	NotifyPkSyncFailed("graduate", "first failure")

	// 窗口内重复失败去重。
	NotifyPkSyncFailed("graduate", "second failure")
	if got := systemNotificationCount(t, smUser); got != 1 {
		t.Fatalf("within window notifications = %d, want 1", got)
	}

	// 模拟时间流逝：清掉本地缓存标记后（等价于 alertWindow 过期）再次失败应重新告警。
	pkAlertCache.Delete(pkAlertKey("graduate", smUser))
	NotifyPkSyncFailed("graduate", "third failure after window")
	if got := systemNotificationCount(t, smUser); got != 2 {
		t.Errorf("after window expiry notifications = %d, want 2", got)
	}
}

func TestNotifyPkSyncFailedNoRecipientsDoesNotPanic(t *testing.T) {
	setupAlertTestDB(t)

	// 没有任何角色/用户（或存在但无权限）时，告警应为静默 no-op，不 panic。
	newTestRole(t, "alert-no-sm", permission.TopicsManager)

	NotifyPkSyncFailed("undergraduate", "boom")
	NotifyPkSyncFailed("graduate", "boom")
}

func TestNotifyPkSyncFailedNowHookRecordsTime(t *testing.T) {
	setupAlertTestDB(t)

	originalNow := now
	t.Cleanup(func() { now = originalNow })

	now = func() time.Time { return time.Unix(1_800_000_000, 0) }

	smRole := newTestRole(t, "alert-now-sm", permission.SiteManager)
	smUser := newTestUser(t, "alert-now-sm-user", smRole, false)
	NotifyPkSyncFailed("undergraduate", "boom")

	key := pkAlertKey("undergraduate", smUser)
	at, _ := pkAlertCache.GetOrLoadE(key, func() (time.Time, error) {
		return time.Time{}, errors.New("missing alert marker")
	}, alertWindow)
	if !at.Equal(time.Unix(1_800_000_000, 0)) {
		t.Errorf("recorded alert time = %v, want %v", at, time.Unix(1_800_000_000, 0))
	}
	if got := systemNotificationCount(t, smUser); got != 1 {
		t.Errorf("notifications = %d, want 1", got)
	}
}

// 通知 payload 需携带标题与正文原样内容（Web/移动端按此渲染），且不泄露
// 凭证敏感信息之外的内容——message 来自已脱敏的错误文本。
func TestNotifyPkSyncFailedPayloadContent(t *testing.T) {
	setupAlertTestDB(t)

	smRole := newTestRole(t, "alert-payload-sm", permission.SiteManager)
	smUser := newTestUser(t, "alert-payload-sm-user", smRole, false)

	NotifyPkSyncFailed("undergraduate", "401 Unauthorized")

	var n eventNotification.Entity
	if err := db.Connect().Where("user_id = ? AND event_type = ?", smUser, eventNotification.EventTypeSystem).First(&n).Error; err != nil {
		t.Fatalf("load system notification: %v", err)
	}
	if n.Payload.Title == "" {
		t.Error("system notification title is empty")
	}
	if n.Payload.Content == "" {
		t.Error("system notification content is empty")
	}
	if want := "【本科】同步失败：401 Unauthorized"; n.Payload.Content != want {
		t.Errorf("system notification content = %q, want %q", n.Payload.Content, want)
	}
}

// A failed insert must remain retryable, without re-notifying recipients whose
// notification was already committed during a partially successful attempt.
func TestNotifyPkSyncFailedRetriesUndeliveredRecipients(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(fmt.Sprintf("partial=%v", partial), func(t *testing.T) {
			setupAlertTestDB(t)
			roleID := newTestRole(t, "retry-role", permission.SiteManager)
			failedUser := newTestUser(t, "retry-failed", roleID, false)
			healthyUser := newTestUser(t, "retry-healthy", roleID, false)
			callback := "test:notification-outage"
			conn := db.Connect()
			if err := conn.Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
				n, ok := tx.Statement.Dest.(*eventNotification.Entity)
				if ok && (!partial || n.UserId == failedUser) {
					_ = tx.AddError(errors.New("temporary notification write outage"))
				}
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = conn.Callback().Create().Remove(callback) })
			NotifyPkSyncFailed("undergraduate", "expired credential")
			if got := systemNotificationCount(t, failedUser); got != 0 {
				t.Fatalf("failed delivery count = %d", got)
			}
			if err := conn.Callback().Create().Remove(callback); err != nil {
				t.Fatal(err)
			}
			NotifyPkSyncFailed("undergraduate", "delivery recovered")
			for _, id := range []uint64{failedUser, healthyUser} {
				if got := systemNotificationCount(t, id); got != 1 {
					t.Errorf("recipient %d count = %d, want 1", id, got)
				}
			}
		})
	}
}

func TestNotifyPkSyncFailedConcurrentAttemptsSendOnce(t *testing.T) {
	setupAlertTestDB(t)
	roleID := newTestRole(t, "concurrent-role", permission.SiteManager)
	userID := newTestUser(t, "concurrent-user", roleID, false)
	var wg sync.WaitGroup
	for range 12 {
		wg.Go(func() { NotifyPkSyncFailed("graduate", "concurrent failure") })
	}
	wg.Wait()
	if got := systemNotificationCount(t, userID); got != 1 {
		t.Fatalf("concurrent notifications = %d, want 1", got)
	}
}

func TestNotifyPkSyncFailedNewRecipientIsNotSuppressed(t *testing.T) {
	setupAlertTestDB(t)
	NotifyPkSyncFailed("undergraduate", "no recipients yet")
	roleID := newTestRole(t, "new-role", permission.SiteManager)
	first := newTestUser(t, "new-first", roleID, false)
	NotifyPkSyncFailed("undergraduate", "first admin added")
	second := newTestUser(t, "new-second", roleID, false)
	NotifyPkSyncFailed("undergraduate", "second admin added")
	for _, id := range []uint64{first, second} {
		if got := systemNotificationCount(t, id); got != 1 {
			t.Errorf("new recipient %d count = %d, want 1", id, got)
		}
	}
}

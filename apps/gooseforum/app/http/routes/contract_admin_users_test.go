package routes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/badges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/role"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userBadges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/badgeservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 本文件覆盖 UserManager 权限组 5 条路由的契约测试（issue #277 P3 切片三）。
// 种子数据全部使用 8xxx/9xxx 段固定 ID，cleanup 用 Unscoped 删除
// （users/role 表软删 + username 唯一约束，软删后同名重建会撞唯一索引）。

const (
	contractUserListTargetID uint64 = 8011
	contractUserEditTargetID uint64 = 8012
	contractUserEditBotID    uint64 = 8013
	contractBadgeTargetID    uint64 = 8014
	contractRoleOptionID     uint64 = 9001
)

var (
	contractUserListCreated    = time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	contractUserListLastActive = time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC)
)

// setupAdminUsersContractTest 在共享 harness（setupHTTPContractTest）之上注册
// UserManager 权限组 5 条路由，中间件链与 route4api.go 的生产注册保持一致
// （JWTAuthCheck + CheckWritableAccount 公共链 + CheckPermission(UserManager) 子组）。
// role/badges 表在各子测试间清空并失效徽章定义缓存，避免同进程共享库的数据串扰。
func setupAdminUsersContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&role.Entity{},
		&rolePermissionRs.Entity{},
		&optRecord.Entity{},
		&badges.Entity{},
		&eventNotification.Entity{},
	); err != nil {
		t.Fatalf("migrate admin users contract tables: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&role.Entity{})
	conn.Where("1 = 1").Delete(&badges.Entity{})
	badgeservice.InvalidateDefinitions()
	t.Cleanup(func() {
		badgeservice.InvalidateDefinitions()
	})

	usersAPI := router.Group("/api/admin",
		middleware.JWTAuthCheck,
		middleware.CheckWritableAccount,
		middleware.CheckPermission(permission.UserManager),
	)
	usersAPI.POST("/user-list", UpButterReq(api.UserList))
	usersAPI.POST("/user-edit", UpButterReq(api.EditUser))
	usersAPI.POST("/user-badge-options", UpButterReq(api.UserBadgeOptions))
	usersAPI.POST("/save-user-badges", UpButterReq(api.SaveUserBadges))
	usersAPI.GET("/get-all-role-item", UpButterReq(api.GetAllRoleItem))
	return conn, router
}

// createContractUserManager 创建登录用户并授予 UserManager 权限
// （复用 grantContractPermission：独立角色 ID，规避 10min 权限缓存串扰）。
func createContractUserManager(t *testing.T, conn *gorm.DB) *users.EntityComplete {
	t.Helper()
	user := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, user.Id, permission.UserManager)
	return user
}

// serveAdminUsersOK 以 UserManager 身份调用路由并断言 HTTP 200 + fixture 信封。
func serveAdminUsersOK(t *testing.T, conn *gorm.DB, router *gin.Engine, method, path, body, fixture string) {
	t.Helper()
	recorder := serveAdminUsersRaw(t, conn, router, method, path, body)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixture))
}

// serveAdminUsersRaw 以 UserManager 身份调用路由，返回原始 recorder 供结构化断言。
func serveAdminUsersRaw(t *testing.T, conn *gorm.DB, router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	manager := createContractUserManager(t, conn)
	recorder := serveAuthSecurityJSON(router, method, path, body, contractSessionToken(t, manager))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	return recorder
}

// adminUsersGuardScenarios 跑 5 条路由公共的中间件守卫场景：
// 未登录 401 / 冻结账号 403 / 无 UserManager 权限 403（params.permission="用户管理"）。
// get-all-role-item 是 GET 路由，guard 请求统一走 method 参数。
func adminUsersGuardScenarios(t *testing.T, method, path, fixturePrefix string) {
	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAdminUsersContractTest(t)
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "auth-required.json"))
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAdminUsersContractTest(t)
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

	t.Run("user without UserManager returns 403", func(t *testing.T) {
		conn, router := setupAdminUsersContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("permission denied status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-get-all-role-item-permission-denied.json"))
	})
}

// createContractEditableUser 以固定 ID 造目标用户（无头像、已激活、未冻结），
// cleanup 用 Unscoped 删除规避软删 + username 唯一约束的重建冲突。
func createContractEditableUser(t *testing.T, conn *gorm.DB, id uint64, username string) *users.EntityComplete {
	t.Helper()
	user := users.MakeUser(username, "secret123", fmt.Sprintf("%s@example.test", username))
	user.Id = id
	user.AvatarUrl = ""
	user.IsActivated = users.ActivationSuccess
	user.CreatedAt = contractUserListCreated
	if err := conn.Create(user).Error; err != nil {
		t.Fatalf("create contract editable user: %v", err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", id).Delete(&users.EntityComplete{})
	})
	return user
}

// prepareContractUserListTarget 造 user-list 确定性目标用户（固定注册/活跃时间）。
func prepareContractUserListTarget(t *testing.T, conn *gorm.DB) {
	t.Helper()
	user := createContractEditableUser(t, conn, contractUserListTargetID, "list_target")
	user.Email = "list-target@example.test"
	if err := conn.Save(user).Error; err != nil {
		t.Fatalf("set contract list target email: %v", err)
	}
	if err := conn.Create(&userStatistics.Entity{
		UserId:         contractUserListTargetID,
		LastActiveTime: contractUserListLastActive,
	}).Error; err != nil {
		t.Fatalf("create contract list target statistics: %v", err)
	}
	t.Cleanup(func() {
		conn.Where("user_id = ?", contractUserListTargetID).Delete(&userStatistics.Entity{})
	})
}

// seedContractRoleEntity 造固定 ID/名称/创建时间的角色行（role 软删，cleanup 用 Unscoped）。
func seedContractRoleEntity(t *testing.T, conn *gorm.DB, id uint64, roleName string) {
	t.Helper()
	if err := conn.Create(&role.Entity{
		Id:        id,
		RoleName:  roleName,
		Effective: 1,
		CreatedAt: contractUserListCreated,
	}).Error; err != nil {
		t.Fatalf("seed contract role %d: %v", id, err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", id).Delete(&role.Entity{})
	})
}

func TestAdminUserListHTTPContract(t *testing.T) {
	path := "/api/admin/user-list"

	t.Run("success with exact user id filter", func(t *testing.T) {
		conn, router := setupAdminUsersContractTest(t)
		prepareContractUserListTarget(t, conn)
		// 按 userId 精确过滤，屏蔽同进程共享库中其他测试造的用户，保证分页响应确定性。
		serveAdminUsersOK(t, conn, router, http.MethodPost, path,
			`{"userId":8011,"page":1,"pageSize":10}`, "admin-user-list-success.json")
	})

	adminUsersGuardScenarios(t, http.MethodPost, path, "admin-user-list")
}

func TestAdminUserEditHTTPContract(t *testing.T) {
	path := "/api/admin/user-edit"

	t.Run("success freezes the target user", func(t *testing.T) {
		conn, router := setupAdminUsersContractTest(t)
		createContractEditableUser(t, conn, contractUserEditTargetID, "edit_target")
		serveAdminUsersOK(t, conn, router, http.MethodPost, path,
			`{"userId":8012,"status":1,"validate":1,"roleId":0}`, "admin-agent-disable-success.json")
		target, err := users.Get(contractUserEditTargetID)
		if err != nil || target.IsFrozen != users.StatusFrozen {
			t.Fatalf("target user IsFrozen = %v, want frozen: %v", target.IsFrozen, err)
		}
	})

	t.Run("unknown user returns business failure", func(t *testing.T) {
		conn, router := setupAdminUsersContractTest(t)
		serveAdminUsersOK(t, conn, router, http.MethodPost, path,
			`{"userId":987654321,"status":1,"validate":1,"roleId":0}`, "admin-user-edit-target-fetch-failed.json")
	})

	t.Run("granting a role to a bot account returns business failure", func(t *testing.T) {
		conn, router := setupAdminUsersContractTest(t)
		bot := createContractEditableUser(t, conn, contractUserEditBotID, "edit_bot")
		if err := conn.Model(bot).Update("actor_type", users.ActorTypeBot).Error; err != nil {
			t.Fatalf("mark contract user as bot: %v", err)
		}
		serveAdminUsersOK(t, conn, router, http.MethodPost, path,
			`{"userId":8013,"status":0,"validate":1,"roleId":9001}`, "admin-category-moderator-add-agent-role-not-allowed.json")
	})

	adminUsersGuardScenarios(t, http.MethodPost, path, "admin-user-edit")
}

func TestAdminUserBadgeOptionsHTTPContract(t *testing.T) {
	path := "/api/admin/user-badge-options"

	t.Run("success lists the built-in manual badges and no active badges", func(t *testing.T) {
		conn, router := setupAdminUsersContractTest(t)
		createContractEditableUser(t, conn, contractBadgeTargetID, "badge_target")
		serveAdminUsersOK(t, conn, router, http.MethodPost, path,
			`{"userId":8014}`, "admin-user-badge-options-success.json")
	})

	adminUsersGuardScenarios(t, http.MethodPost, path, "admin-user-badge-options")
}

func TestAdminSaveUserBadgesHTTPContract(t *testing.T) {
	path := "/api/admin/save-user-badges"

	t.Run("success grants the manual badge", func(t *testing.T) {
		conn, router := setupAdminUsersContractTest(t)
		createContractEditableUser(t, conn, contractBadgeTargetID, "badge_target")
		t.Cleanup(func() {
			conn.Where("user_id = ?", contractBadgeTargetID).Delete(&userBadges.Entity{})
			conn.Where("user_id = ?", contractBadgeTargetID).Delete(&eventNotification.Entity{})
		})
		serveAdminUsersOK(t, conn, router, http.MethodPost, path,
			`{"userId":8014,"badgeCodes":["early_member"]}`, "admin-agent-disable-success.json")
		active := userBadges.GetActiveByUserID(contractBadgeTargetID)
		if len(active) != 1 || active[0].BadgeCode != badgeservice.CodeEarlyMember {
			t.Fatalf("active badges = %#v, want early_member", active)
		}
	})

	t.Run("zero user id returns business failure", func(t *testing.T) {
		conn, router := setupAdminUsersContractTest(t)
		serveAdminUsersOK(t, conn, router, http.MethodPost, path,
			`{"userId":0,"badgeCodes":["early_member"]}`, "admin-save-user-badges-user-not-found.json")
	})

	adminUsersGuardScenarios(t, http.MethodPost, path, "admin-save-user-badges")
}

func TestAdminGetAllRoleItemHTTPContract(t *testing.T) {
	path := "/api/admin/get-all-role-item"

	t.Run("success", func(t *testing.T) {
		conn, router := setupAdminUsersContractTest(t)
		seedContractRoleEntity(t, conn, contractRoleOptionID, "见习版主")
		serveAdminUsersOK(t, conn, router, http.MethodGet, path, `{}`, "admin-get-all-role-item-success.json")
	})

	adminUsersGuardScenarios(t, http.MethodGet, path, "admin-get-all-role-item")
}

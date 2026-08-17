package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/role"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 本文件覆盖 RoleManager 权限组 4 条路由的契约测试（issue #277 P3 切片三）。
// role 表在各子测试间清空（Unscoped，role 软删），grantContractPermission 只写
// rolePermissionRs 不建 role 行，因此清空 role 表即可保证分页/选项响应确定性。

const (
	contractRoleListRoleID   uint64 = 9001
	contractRoleDeleteRoleID uint64 = 9002
	contractRoleListRsID     uint64 = 6001
)

// setupAdminRolesContractTest 在共享 harness（setupHTTPContractTest）之上注册
// RoleManager 权限组 4 条路由，中间件链与 route4api.go 的生产注册保持一致
// （JWTAuthCheck + CheckWritableAccount 公共链 + CheckPermission(RoleManager) 子组）。
func setupAdminRolesContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&role.Entity{},
		&rolePermissionRs.Entity{},
	); err != nil {
		t.Fatalf("migrate admin roles contract tables: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&role.Entity{})

	rolesAPI := router.Group("/api/admin",
		middleware.JWTAuthCheck,
		middleware.CheckWritableAccount,
		middleware.CheckPermission(permission.RoleManager),
	)
	rolesAPI.POST("/get-permission-list", UpButterReq(api.GetPermissionList))
	rolesAPI.POST("/role-list", UpButterReq(api.RoleList))
	rolesAPI.POST("/role-save", UpButterReq(api.RoleSave))
	rolesAPI.POST("/role-delete", UpButterReq(api.RoleDel))
	return conn, router
}

// createContractRoleManager 创建登录用户并授予 RoleManager 权限
// （复用 grantContractPermission：独立角色 ID，规避 10min 权限缓存串扰）。
func createContractRoleManager(t *testing.T, conn *gorm.DB) *users.EntityComplete {
	t.Helper()
	user := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, user.Id, permission.RoleManager)
	return user
}

// serveAdminRolesOK 以 RoleManager 身份调用路由并断言 HTTP 200 + fixture 信封。
func serveAdminRolesOK(t *testing.T, conn *gorm.DB, router *gin.Engine, path, body, fixture string) {
	t.Helper()
	recorder := serveAdminRolesRaw(t, conn, router, path, body)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixture))
}

// serveAdminRolesRaw 以 RoleManager 身份调用路由，返回原始 recorder 供结构化断言。
func serveAdminRolesRaw(t *testing.T, conn *gorm.DB, router *gin.Engine, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	manager := createContractRoleManager(t, conn)
	recorder := serveJSON(router, path, body, contractSessionToken(t, manager))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	return recorder
}

// adminRolesGuardScenarios 跑 4 条路由公共的中间件守卫场景：
// 未登录 401 / 冻结账号 403 / 无 RoleManager 权限 403（params.permission="角色管理"）。
func adminRolesGuardScenarios(t *testing.T, path, fixturePrefix string) {
	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAdminRolesContractTest(t)
		assertInteractionUnauthenticated(t, router, path, `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAdminRolesContractTest(t)
		assertInteractionForbidden(t, conn, router, path, `{}`, "account-frozen.json")
	})

	t.Run("user without RoleManager returns 403", func(t *testing.T) {
		conn, router := setupAdminRolesContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, path, `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("permission denied status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-get-permission-list-permission-denied.json"))
	})
}

// prepareContractRoleListData 造 role-list 确定性角色与权限授予行（固定创建时间）。
func prepareContractRoleListData(t *testing.T, conn *gorm.DB) {
	t.Helper()
	seedContractRoleEntity(t, conn, contractRoleListRoleID, "见习版主")
	if err := conn.Create(&rolePermissionRs.Entity{
		Id:           contractRoleListRsID,
		RoleId:       contractRoleListRoleID,
		PermissionId: permission.TopicsManager.Id(),
		Effective:    1,
	}).Error; err != nil {
		t.Fatalf("seed contract role permission: %v", err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", contractRoleListRsID).Delete(&rolePermissionRs.Entity{})
	})
}

func TestAdminGetPermissionListHTTPContract(t *testing.T) {
	path := "/api/admin/get-permission-list"

	t.Run("success returns all seven localized permission options", func(t *testing.T) {
		conn, router := setupAdminRolesContractTest(t)
		serveAdminRolesOK(t, conn, router, path, `{}`, "admin-get-permission-list-success.json")
	})

	adminRolesGuardScenarios(t, path, "admin-get-permission-list")
}

func TestAdminRoleListHTTPContract(t *testing.T) {
	path := "/api/admin/role-list"

	t.Run("success", func(t *testing.T) {
		conn, router := setupAdminRolesContractTest(t)
		prepareContractRoleListData(t, conn)
		serveAdminRolesOK(t, conn, router, path, `{}`, "admin-role-list-success.json")
	})

	adminRolesGuardScenarios(t, path, "admin-role-list")
}

func TestAdminRoleSaveHTTPContract(t *testing.T) {
	path := "/api/admin/role-save"

	t.Run("success creates the role with the submitted permission set", func(t *testing.T) {
		conn, router := setupAdminRolesContractTest(t)
		serveAdminRolesOK(t, conn, router, path,
			`{"id":0,"roleName":"契约角色","permissions":[2]}`, "result-true.json")
		var created role.Entity
		if err := conn.Where("role_name = ?", "契约角色").First(&created).Error; err != nil {
			t.Fatalf("created role not found: %v", err)
		}
		permissionIds := rolePermissionRs.GetPermissionIdsByRoleIds([]uint64{created.Id})
		if len(permissionIds) != 1 || permissionIds[0] != permission.TopicsManager.Id() {
			t.Fatalf("created role permission ids = %#v, want [2]", permissionIds)
		}
		t.Cleanup(func() {
			conn.Unscoped().Where("role_id = ?", created.Id).Delete(&rolePermissionRs.Entity{})
			conn.Unscoped().Where("id = ?", created.Id).Delete(&role.Entity{})
		})
	})

	t.Run("missing permissions stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupAdminRolesContractTest(t)
		serveAdminRolesOK(t, conn, router, path,
			`{"id":0,"roleName":"只有名字"}`, "invalid-params.json")
	})

	adminRolesGuardScenarios(t, path, "admin-role-save")
}

func TestAdminRoleDeleteHTTPContract(t *testing.T) {
	path := "/api/admin/role-delete"

	t.Run("success soft-deletes the role and its grants", func(t *testing.T) {
		conn, router := setupAdminRolesContractTest(t)
		seedContractRoleEntity(t, conn, contractRoleDeleteRoleID, "待删角色")
		serveAdminRolesOK(t, conn, router, path, `{"id":9002}`, "result-true.json")
		if got := role.Get(contractRoleDeleteRoleID); got.Id != 0 {
			t.Fatalf("role %d still readable after delete", contractRoleDeleteRoleID)
		}
	})

	t.Run("unknown role returns business failure", func(t *testing.T) {
		conn, router := setupAdminRolesContractTest(t)
		serveAdminRolesOK(t, conn, router, path, `{"id":987654321}`, "admin-role-delete-not-found.json")
	})

	adminRolesGuardScenarios(t, path, "admin-role-delete")
}

package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 本文件覆盖 TopicsManager 权限组 category/moderator 8 条路由的契约测试
// （issue #277 P3 切片三）。category/moderators 表在各子测试间清空并清分类缓存
// （hotdataserve 分类缓存进程内共享），种子数据全部使用固定 ID。

const (
	contractCategoryID           uint64 = 5001
	contractCategorySecondID     uint64 = 5002
	contractCategorySingleID     uint64 = 5010
	contractCategoryModeratorID  uint64 = 7001
	contractGlobalModeratorID    uint64 = 7002
	contractCategoryModeratorID2 uint64 = 7003
	contractModeratorUserID      uint64 = 8021
	contractGlobalModTargetID    uint64 = 8022
	contractGlobalModBotID       uint64 = 8023
	contractCategoryModTargetID  uint64 = 8024
	contractCategoryModBotID     uint64 = 8025
	contractCategoryTopicID      uint64 = 9201
)

// setupAdminCategoriesContractTest 在共享 harness（setupHTTPContractTest）之上注册
// TopicsManager 权限组 category/moderator 8 条路由，中间件链与 route4api.go 的
// 生产注册保持一致（JWTAuthCheck + CheckWritableAccount 公共链 +
// CheckPermission(TopicsManager) 子组）。
func setupAdminCategoriesContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&rolePermissionRs.Entity{},
		&optRecord.Entity{},
	); err != nil {
		t.Fatalf("migrate admin categories contract tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&moderators.Entity{})
	conn.Where("1 = 1").Delete(&category.Entity{})
	hotdataserve.ClearCategoryCache()
	t.Cleanup(func() {
		hotdataserve.ClearCategoryCache()
	})

	categoriesAPI := router.Group("/api/admin",
		middleware.JWTAuthCheck,
		middleware.CheckWritableAccount,
		middleware.CheckPermission(permission.TopicsManager),
	)
	categoriesAPI.POST("/category-list", UpButterReq(api.GetCategoryList))
	categoriesAPI.POST("/category-save", UpButterReq(api.SaveCategory))
	categoriesAPI.POST("/category-delete", UpButterReq(api.DeleteCategory))
	categoriesAPI.POST("/global-moderator-list", UpButterReq(api.GetGlobalModeratorList))
	categoriesAPI.POST("/global-moderator-add", UpButterReq(api.AddGlobalModerator))
	categoriesAPI.POST("/global-moderator-delete", UpButterReq(api.DeleteGlobalModerator))
	categoriesAPI.POST("/category-moderator-add", UpButterReq(api.AddCategoryModerator))
	categoriesAPI.POST("/category-moderator-delete", UpButterReq(api.DeleteCategoryModerator))
	return conn, router
}

// serveAdminCategoriesOK 以 TopicsManager 身份调用路由并断言 HTTP 200 + fixture 信封。
func serveAdminCategoriesOK(t *testing.T, conn *gorm.DB, router *gin.Engine, path, body, fixture string) {
	t.Helper()
	manager := createContractTopicsManager(t, conn)
	recorder := serveJSON(router, path, body, contractSessionToken(t, manager))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixture))
}

// serveAdminCategoriesRaw 以 TopicsManager 身份调用路由，返回原始 recorder 供结构化断言。
func serveAdminCategoriesRaw(t *testing.T, conn *gorm.DB, router *gin.Engine, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	manager := createContractTopicsManager(t, conn)
	recorder := serveJSON(router, path, body, contractSessionToken(t, manager))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	return recorder
}

// adminCategoriesGuardScenarios 跑 8 条路由公共的中间件守卫场景：
// 未登录 401 / 冻结账号 403 / 无 TopicsManager 权限 403（params.permission="话题管理"）。
func adminCategoriesGuardScenarios(t *testing.T, path, fixturePrefix string) {
	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAdminCategoriesContractTest(t)
		assertInteractionUnauthenticated(t, router, path, `{}`, fixturePrefix+"-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		assertInteractionForbidden(t, conn, router, path, `{}`, fixturePrefix+"-forbidden.json")
	})

	t.Run("user without TopicsManager returns 403", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		assertAdminTopicsPermissionDenied(t, conn, router, path, fixturePrefix+"-permission-denied.json")
	})
}

// seedContractCategory 造固定 ID 分类（category 硬删，cleanup 直接按 ID 删除并清缓存）。
func seedContractCategory(t *testing.T, conn *gorm.DB, id uint64, name, desc, icon, color, slug string, sort int) {
	t.Helper()
	if err := conn.Create(&category.Entity{
		Id:    id,
		Name:  name,
		Desc:  desc,
		Icon:  icon,
		Color: color,
		Slug:  slug,
		Sort:  sort,
	}).Error; err != nil {
		t.Fatalf("seed contract category %d: %v", id, err)
	}
	hotdataserve.ClearCategoryCache()
	t.Cleanup(func() {
		conn.Where("id = ?", id).Delete(&category.Entity{})
		hotdataserve.ClearCategoryCache()
	})
}

// createContractModeratorCandidate 造版主目标用户（无头像，响应里回退默认头像）。
func createContractModeratorCandidate(t *testing.T, conn *gorm.DB, id uint64, username string, bot bool) {
	t.Helper()
	user := users.MakeUser(username, "secret123", username+"@example.test")
	user.Id = id
	user.AvatarUrl = ""
	user.IsActivated = users.ActivationSuccess
	if bot {
		user.ActorType = users.ActorTypeBot
	}
	if err := conn.Create(user).Error; err != nil {
		t.Fatalf("create contract moderator candidate: %v", err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", id).Delete(&users.EntityComplete{})
	})
}

// seedContractModerator 造固定 ID 版主行（moderators 硬删，cleanup 直接按 ID 删除）。
func seedContractModerator(t *testing.T, conn *gorm.DB, id, userID uint64, scopeType string, scopeID uint64) {
	t.Helper()
	if err := conn.Create(&moderators.Entity{
		Id:        id,
		UserId:    userID,
		ScopeType: scopeType,
		ScopeId:   scopeID,
		Status:    moderators.StatusEnabled,
		CreatedBy: contractModerationOperatorID,
	}).Error; err != nil {
		t.Fatalf("seed contract moderator %d: %v", id, err)
	}
	t.Cleanup(func() {
		conn.Where("id = ?", id).Delete(&moderators.Entity{})
	})
}

func TestAdminCategoryListHTTPContract(t *testing.T) {
	path := "/api/admin/category-list"

	t.Run("success", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		createContractModeratorCandidate(t, conn, contractModeratorUserID, "mod_user", false)
		seedContractCategory(t, conn, contractCategoryID, "学习交流", "课程与学习讨论", "book", "#3b82f6", "study", 1)
		seedContractCategory(t, conn, contractCategorySecondID, "生活广场", "校园生活分享", "life", "#f59e0b", "life", 2)
		seedContractModerator(t, conn, contractCategoryModeratorID, contractModeratorUserID, moderators.ScopeCategory, contractCategoryID)
		serveAdminCategoriesOK(t, conn, router, path, `{}`, "admin-category-list-success.json")
	})

	adminCategoriesGuardScenarios(t, path, "admin-category-list")
}

func TestAdminCategorySaveHTTPContract(t *testing.T) {
	path := "/api/admin/category-save"

	t.Run("success creates the category", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		serveAdminCategoriesOK(t, conn, router, path,
			`{"id":0,"category":"契约新板块","desc":"契约测试板块","slug":"contract-board","sort":9}`,
			"admin-category-save-success.json")
		var created category.Entity
		if err := conn.Where("slug = ?", "contract-board").First(&created).Error; err != nil {
			t.Fatalf("created category not found: %v", err)
		}
		if created.Name != "契约新板块" || created.Sort != 9 {
			t.Fatalf("created category = %#v, want 契约新板块", created)
		}
		t.Cleanup(func() {
			conn.Where("id = ?", created.Id).Delete(&category.Entity{})
			hotdataserve.ClearCategoryCache()
		})
	})

	t.Run("whitespace-only name returns business failure", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		serveAdminCategoriesOK(t, conn, router, path,
			`{"id":0,"category":"   "}`, "admin-category-save-name-required.json")
	})

	t.Run("unknown id returns business failure", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		serveAdminCategoriesOK(t, conn, router, path,
			`{"id":987654321,"category":"不存在的板块"}`, "admin-category-save-data-not-found.json")
	})

	adminCategoriesGuardScenarios(t, path, "admin-category-save")
}

func TestAdminCategoryDeleteHTTPContract(t *testing.T) {
	path := "/api/admin/category-delete"

	t.Run("success hard-deletes the category", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		seedContractCategory(t, conn, contractCategoryID, "学习交流", "课程与学习讨论", "book", "#3b82f6", "study", 1)
		seedContractCategory(t, conn, contractCategorySecondID, "生活广场", "校园生活分享", "life", "#f59e0b", "life", 2)
		serveAdminCategoriesOK(t, conn, router, path, `{"id":5002}`, "admin-category-delete-success.json")
		if got := category.Get(contractCategorySecondID); got.Id != 0 {
			t.Fatalf("category %d still readable after delete", contractCategorySecondID)
		}
	})

	t.Run("unknown category returns business failure", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		serveAdminCategoriesOK(t, conn, router, path, `{"id":987654321}`, "admin-category-delete-not-found.json")
	})

	t.Run("deleting the last category returns business failure", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		seedContractCategory(t, conn, contractCategorySingleID, "唯一板块", "唯一板块", "only", "#3b82f6", "only", 1)
		serveAdminCategoriesOK(t, conn, router, path, `{"id":5010}`, "admin-category-delete-keep-one.json")
	})

	t.Run("category with topic bindings returns business failure", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		seedContractCategory(t, conn, contractCategoryID, "学习交流", "课程与学习讨论", "book", "#3b82f6", "study", 1)
		seedContractCategory(t, conn, contractCategorySecondID, "生活广场", "校园生活分享", "life", "#f59e0b", "life", 2)
		if err := conn.Create(&topicCategoryIndex.Entity{
			TopicId:    contractCategoryTopicID,
			CategoryId: contractCategoryID,
			Effective:  1,
		}).Error; err != nil {
			t.Fatalf("seed contract topic category index: %v", err)
		}
		t.Cleanup(func() {
			conn.Where("topic_id = ?", contractCategoryTopicID).Delete(&topicCategoryIndex.Entity{})
		})
		serveAdminCategoriesOK(t, conn, router, path, `{"id":5001}`, "admin-category-delete-has-topics.json")
	})

	adminCategoriesGuardScenarios(t, path, "admin-category-delete")
}

func TestAdminGlobalModeratorListHTTPContract(t *testing.T) {
	path := "/api/admin/global-moderator-list"

	t.Run("success", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		createContractModeratorCandidate(t, conn, contractModeratorUserID, "mod_user", false)
		seedContractModerator(t, conn, contractCategoryModeratorID, contractModeratorUserID, moderators.ScopeGlobal, 0)
		serveAdminCategoriesOK(t, conn, router, path, `{}`, "admin-global-moderator-list-success.json")
	})

	adminCategoriesGuardScenarios(t, path, "admin-global-moderator-list")
}

func TestAdminGlobalModeratorAddHTTPContract(t *testing.T) {
	path := "/api/admin/global-moderator-add"

	t.Run("success grants the global scope", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		createContractModeratorCandidate(t, conn, contractGlobalModTargetID, "global_mod_target", false)
		serveAdminCategoriesOK(t, conn, router, path, `{"userId":8022}`, "admin-global-moderator-add-success.json")
		granted := moderators.GetByUserScope(contractGlobalModTargetID, moderators.ScopeGlobal, 0)
		if granted.Id == 0 || granted.Status != moderators.StatusEnabled {
			t.Fatalf("global moderator grant = %#v, want enabled row", granted)
		}
		t.Cleanup(func() {
			conn.Where("id = ?", granted.Id).Delete(&moderators.Entity{})
		})
	})

	t.Run("missing user reference returns business failure", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		serveAdminCategoriesOK(t, conn, router, path, `{}`, "admin-global-moderator-add-user-required.json")
	})

	t.Run("unknown user returns business failure", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		serveAdminCategoriesOK(t, conn, router, path, `{"userId":987654321}`, "admin-global-moderator-add-user-not-found.json")
	})

	t.Run("bot account returns business failure", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		createContractModeratorCandidate(t, conn, contractGlobalModBotID, "global_mod_bot", true)
		serveAdminCategoriesOK(t, conn, router, path, `{"userId":8023}`, "admin-global-moderator-add-agent-role-not-allowed.json")
	})

	adminCategoriesGuardScenarios(t, path, "admin-global-moderator-add")
}

func TestAdminGlobalModeratorDeleteHTTPContract(t *testing.T) {
	path := "/api/admin/global-moderator-delete"

	t.Run("success hard-deletes the moderator row", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		createContractModeratorCandidate(t, conn, contractModeratorUserID, "mod_user", false)
		seedContractModerator(t, conn, contractGlobalModeratorID, contractModeratorUserID, moderators.ScopeGlobal, 0)
		serveAdminCategoriesOK(t, conn, router, path, `{"id":7002}`, "admin-global-moderator-delete-success.json")
		if got := moderators.Get(contractGlobalModeratorID); got.Id != 0 {
			t.Fatalf("moderator %d still readable after delete", contractGlobalModeratorID)
		}
	})

	t.Run("unknown moderator returns business failure", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		serveAdminCategoriesOK(t, conn, router, path, `{"id":987654321}`, "admin-global-moderator-delete-not-found.json")
	})

	t.Run("missing id stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		serveAdminCategoriesOK(t, conn, router, path, `{}`, "admin-global-moderator-delete-invalid-params.json")
	})

	adminCategoriesGuardScenarios(t, path, "admin-global-moderator-delete")
}

func TestAdminCategoryModeratorAddHTTPContract(t *testing.T) {
	path := "/api/admin/category-moderator-add"

	t.Run("success grants the category scope", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		createContractModeratorCandidate(t, conn, contractCategoryModTargetID, "cat_mod_target", false)
		seedContractCategory(t, conn, contractCategoryID, "学习交流", "课程与学习讨论", "book", "#3b82f6", "study", 1)
		serveAdminCategoriesOK(t, conn, router, path,
			`{"categoryId":5001,"userId":8024}`, "admin-category-moderator-add-success.json")
		granted := moderators.GetByUserScope(contractCategoryModTargetID, moderators.ScopeCategory, contractCategoryID)
		if granted.Id == 0 || granted.Status != moderators.StatusEnabled {
			t.Fatalf("category moderator grant = %#v, want enabled row", granted)
		}
		t.Cleanup(func() {
			conn.Where("id = ?", granted.Id).Delete(&moderators.Entity{})
		})
	})

	t.Run("unknown category returns business failure", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		serveAdminCategoriesOK(t, conn, router, path,
			`{"categoryId":987654321,"userId":8024}`, "admin-category-moderator-add-category-not-found.json")
	})

	t.Run("missing user reference returns business failure", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		seedContractCategory(t, conn, contractCategoryID, "学习交流", "课程与学习讨论", "book", "#3b82f6", "study", 1)
		serveAdminCategoriesOK(t, conn, router, path,
			`{"categoryId":5001}`, "admin-category-moderator-add-user-required.json")
	})

	t.Run("unknown user returns business failure", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		seedContractCategory(t, conn, contractCategoryID, "学习交流", "课程与学习讨论", "book", "#3b82f6", "study", 1)
		serveAdminCategoriesOK(t, conn, router, path,
			`{"categoryId":5001,"userId":987654321}`, "admin-category-moderator-add-user-not-found.json")
	})

	t.Run("bot account returns business failure", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		createContractModeratorCandidate(t, conn, contractCategoryModBotID, "cat_mod_bot", true)
		seedContractCategory(t, conn, contractCategoryID, "学习交流", "课程与学习讨论", "book", "#3b82f6", "study", 1)
		serveAdminCategoriesOK(t, conn, router, path,
			`{"categoryId":5001,"userId":8025}`, "admin-category-moderator-add-agent-role-not-allowed.json")
	})

	adminCategoriesGuardScenarios(t, path, "admin-category-moderator-add")
}

func TestAdminCategoryModeratorDeleteHTTPContract(t *testing.T) {
	path := "/api/admin/category-moderator-delete"

	t.Run("success hard-deletes the moderator row", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		createContractModeratorCandidate(t, conn, contractModeratorUserID, "mod_user", false)
		seedContractCategory(t, conn, contractCategoryID, "学习交流", "课程与学习讨论", "book", "#3b82f6", "study", 1)
		seedContractModerator(t, conn, contractCategoryModeratorID2, contractModeratorUserID, moderators.ScopeCategory, contractCategoryID)
		serveAdminCategoriesOK(t, conn, router, path, `{"id":7003}`, "admin-category-moderator-delete-success.json")
		if got := moderators.Get(contractCategoryModeratorID2); got.Id != 0 {
			t.Fatalf("moderator %d still readable after delete", contractCategoryModeratorID2)
		}
	})

	t.Run("unknown moderator returns business failure", func(t *testing.T) {
		conn, router := setupAdminCategoriesContractTest(t)
		serveAdminCategoriesOK(t, conn, router, path, `{"id":987654321}`, "admin-category-moderator-delete-not-found.json")
	})

	adminCategoriesGuardScenarios(t, path, "admin-category-moderator-delete")
}

package routes

import (
	"net/http"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/contentDeleteEvent"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupAdminTopicsContractTest 在共享 harness（setupHTTPContractTest）之上注册 admin
// topics 组 8 条路由，中间件链与 route4api.go 的生产注册保持一致
// （JWTAuthCheck + CheckWritableAccount 公共链 + CheckPermission(TopicsManager) 子组）。
func setupAdminTopicsContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&rolePermissionRs.Entity{},
		&optRecord.Entity{},
		&moderationLog.Entity{},
		&contentDeleteEvent.Entity{},
		&eventNotification.Entity{},
	); err != nil {
		t.Fatalf("migrate admin topics contract tables: %v", err)
	}

	topicsAPI := router.Group("/api/admin",
		middleware.JWTAuthCheck,
		middleware.CheckWritableAccount,
		middleware.CheckPermission(permission.TopicsManager),
	)
	topicsAPI.POST("/topics/list", UpButterReq(api.TopicsList))
	topicsAPI.POST("/topics/source", UpButterReq(api.TopicSource))
	topicsAPI.POST("/topics/edit", UpButterReq(api.EditTopic))
	topicsAPI.POST("/topics/delete", UpButterReq(api.DeleteTopic))
	topicsAPI.POST("/topics/restore", UpButterReq(api.RestoreTopic))
	topicsAPI.POST("/posts/delete", UpButterReq(api.DeletePostAsModerator))
	topicsAPI.POST("/topics/pin-edit", UpButterReq(api.EditTopicPin))
	topicsAPI.POST("/topics/categories-edit", UpButterReq(api.EditTopicCategories))
	return conn, router
}

// createContractTopicsManager 创建登录用户并授予 TopicsManager 权限
// （复用 grantContractPermission：独立角色 ID，规避 10min 权限缓存串扰）。
func createContractTopicsManager(t *testing.T, conn *gorm.DB) *users.EntityComplete {
	t.Helper()
	user := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, user.Id, permission.TopicsManager)
	return user
}

// assertAdminTopicsPermissionDenied 断言无 TopicsManager 权限的登录用户被
// CheckPermission 中间件拦截为 HTTP 403 + permission.denied（params 带权限名）。
func assertAdminTopicsPermissionDenied(t *testing.T, conn *gorm.DB, router *gin.Engine, path string, fixture string) {
	t.Helper()
	user := createHTTPContractUser(t, conn, contractTestID())
	recorder := serveJSON(router, path, `{}`, contractSessionToken(t, user))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("permission denied status = %d, want 403: %s", recorder.Code, recorder.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixture))
}

// adminTopicsGuardScenarios 跑 8 条路由公共的中间件守卫场景：
// 未登录 401 / 冻结账号 403 / 无 TopicsManager 权限 403。
func adminTopicsGuardScenarios(t *testing.T, path string, fixturePrefix string) {
	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAdminTopicsContractTest(t)
		assertInteractionUnauthenticated(t, router, path, `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		assertInteractionForbidden(t, conn, router, path, `{}`, "account-frozen.json")
	})

	t.Run("user without TopicsManager returns 403", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		assertAdminTopicsPermissionDenied(t, conn, router, path, "admin-category-delete-permission-denied.json")
	})
}

// serveAdminTopicsOK 以 TopicsManager 身份调用路由并断言 HTTP 200 + fixture 信封。
func serveAdminTopicsOK(t *testing.T, conn *gorm.DB, router *gin.Engine, path string, body string, fixture string) {
	t.Helper()
	manager := createContractTopicsManager(t, conn)
	recorder := serveJSON(router, path, body, contractSessionToken(t, manager))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixture))
}

func TestAdminTopicsListHTTPContract(t *testing.T) {
	path := "/api/admin/topics/list"

	t.Run("success", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		prepareContractModerationTopic(t, conn)
		// 按作者过滤，屏蔽同进程共享库中其他测试造的话题，保证分页响应确定性。
		serveAdminTopicsOK(t, conn, router, path, `{"userId":1024}`, "admin-topics-list-success.json")
	})

	adminTopicsGuardScenarios(t, path, "admin-topics-list")
}

func TestAdminTopicSourceHTTPContract(t *testing.T) {
	path := "/api/admin/topics/source"

	t.Run("success", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		prepareContractModerationTopic(t, conn)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":1201}`, "admin-topic-source-success.json")
	})

	t.Run("unknown topic returns business failure", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":987654321}`, "admin-topic-categories-edit-topic-not-found.json")
	})

	adminTopicsGuardScenarios(t, path, "admin-topic-source")
}

func TestAdminTopicEditHTTPContract(t *testing.T) {
	path := "/api/admin/topics/edit"

	t.Run("success", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		prepareContractModerationTopic(t, conn)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":1201,"processStatus":1}`, "admin-post-delete-success.json")
	})

	t.Run("unknown topic returns business failure", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":987654321,"processStatus":1}`, "admin-topic-categories-edit-topic-not-found.json")
	})

	t.Run("invalid processStatus stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":1201,"processStatus":5}`, "invalid-params.json")
	})

	adminTopicsGuardScenarios(t, path, "admin-topic-edit")
}

func TestAdminTopicDeleteHTTPContract(t *testing.T) {
	path := "/api/admin/topics/delete"

	t.Run("success", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		prepareContractModerationTopic(t, conn)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":1201,"reason":"违反社区规范"}`, "admin-post-delete-success.json")
	})

	t.Run("wiki topic returns operation denied", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		prepareContractModerationTopic(t, conn)
		if err := conn.Model(&topics.Entity{}).
			Where("id = ?", contractModerationTopicID).
			Update("topic_type", topics.TopicTypeWiki).Error; err != nil {
			t.Fatalf("mark contract topic as wiki: %v", err)
		}
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":1201,"reason":"违反社区规范"}`, "admin-topic-delete-operation-denied.json")
	})

	t.Run("unknown topic returns business failure", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":987654321,"reason":"违反社区规范"}`, "admin-topic-categories-edit-topic-not-found.json")
	})

	t.Run("missing reason stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":1201}`, "invalid-params.json")
	})

	adminTopicsGuardScenarios(t, path, "admin-topic-delete")
}

func TestAdminTopicRestoreHTTPContract(t *testing.T) {
	path := "/api/admin/topics/restore"

	t.Run("success", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		prepareContractModerationTopic(t, conn)
		markContractModerationTopicDeleted(t, conn)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":1201}`, "admin-topic-restore-success.json")
	})

	t.Run("active topic is not recoverable", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		prepareContractModerationTopic(t, conn)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":1201}`, "admin-topic-restore-not-recoverable.json")
	})

	t.Run("unknown topic returns business failure", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":987654321}`, "admin-topic-categories-edit-topic-not-found.json")
	})

	adminTopicsGuardScenarios(t, path, "admin-topic-restore")
}

func TestAdminPostDeleteHTTPContract(t *testing.T) {
	path := "/api/admin/posts/delete"

	t.Run("success", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		prepareContractModerationTopic(t, conn)
		createContractReplyPost(t, conn, contractModerationReplyPostID, contractModerationTopicID, contractModerationAuthorID)
		serveAdminTopicsOK(t, conn, router, path, `{"postId":1302,"reason":"垃圾信息"}`, "admin-post-delete-success.json")
	})

	t.Run("unknown post returns business failure", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		serveAdminTopicsOK(t, conn, router, path, `{"postId":987654321,"reason":"垃圾信息"}`, "admin-post-delete-post-not-found.json")
	})

	t.Run("missing reason stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		serveAdminTopicsOK(t, conn, router, path, `{"postId":1302}`, "invalid-params.json")
	})

	adminTopicsGuardScenarios(t, path, "admin-post-delete")
}

func TestAdminTopicPinEditHTTPContract(t *testing.T) {
	path := "/api/admin/topics/pin-edit"

	t.Run("success", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		prepareContractModerationTopic(t, conn)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":1201,"pinWeight":10}`, "admin-post-delete-success.json")
	})

	t.Run("unknown topic returns business failure", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":987654321,"pinWeight":10}`, "admin-topic-categories-edit-topic-not-found.json")
	})

	t.Run("pinWeight out of range stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":1201,"pinWeight":1000001}`, "invalid-params.json")
	})

	adminTopicsGuardScenarios(t, path, "admin-topic-pin-edit")
}

func TestAdminTopicCategoriesEditHTTPContract(t *testing.T) {
	path := "/api/admin/topics/categories-edit"

	t.Run("success", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		prepareContractModerationTopic(t, conn)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":1201,"categoryId":[3]}`, "admin-post-delete-success.json")
	})

	t.Run("unknown category returns business failure", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		prepareContractModerationTopic(t, conn)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":1201,"categoryId":[999999999]}`, "admin-category-delete-not-found.json")
	})

	t.Run("unknown topic returns business failure", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		prepareContractModerationTopic(t, conn)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":987654321,"categoryId":[3]}`, "admin-topic-categories-edit-topic-not-found.json")
	})

	t.Run("empty category list stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupAdminTopicsContractTest(t)
		serveAdminTopicsOK(t, conn, router, path, `{"topicId":1201,"categoryId":[]}`, "invalid-params.json")
	})

	adminTopicsGuardScenarios(t, path, "admin-topic-categories-edit")
}
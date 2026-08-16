package routes

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/badges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/badgeservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 本文件覆盖 SiteManager 权限组 badges/review-queue/file-resources 6 条内容运营路由的
// 契约测试（issue #277 P3 切片六）。badges 表行与徽章定义缓存在子测试间清理；
// 审核队列的种子主题/回复、file-resources 的种子文件（独立 db4fileconnect 连接）
// 均在各子测试首尾清删，避免同进程共享库串扰。

// setupAdminContentOpsContractTest 在共享 harness（setupHTTPContractTest）之上注册
// badges/review-queue/file-resources 6 条路由，中间件链与 route4api.go 的生产注册
// 保持一致（JWTAuthCheck + CheckWritableAccount + CheckPermission(SiteManager)）。
func setupAdminContentOpsContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&rolePermissionRs.Entity{},
		&badges.Entity{},
		&optRecord.Entity{},
	); err != nil {
		t.Fatalf("migrate admin content-ops contract tables: %v", err)
	}
	// filedata 走独立的 db4fileconnect 连接（测试模式同样各自 :memory:），
	// 需在文件库上单独迁移。
	if err := db4fileconnect.Connect().AutoMigrate(&filedata.Entity{}); err != nil {
		t.Fatalf("migrate filedata contract table: %v", err)
	}
	badgeservice.InvalidateDefinitions()
	t.Cleanup(badgeservice.InvalidateDefinitions)

	contentAPI := router.Group("/api/admin",
		middleware.JWTAuthCheck,
		middleware.CheckWritableAccount,
		middleware.CheckPermission(permission.SiteManager),
	)
	contentAPI.GET("/badges", UpButterReq(api.BadgeList))
	contentAPI.POST("/badge-save", UpButterReq(api.SaveBadge))
	contentAPI.POST("/badge-delete", UpButterReq(api.DeleteBadge))
	contentAPI.POST("/review-queue", UpButterReq(api.ReviewQueue))
	contentAPI.POST("/review-action", UpButterReq(api.ReviewAction))
	contentAPI.POST("/file-resources", UpButterReq(api.FileResourcePage))
	return conn, router
}

// adminContentOpsGuardScenarios 跑本文件 6 条路由公共的中间件守卫场景：
// 未登录 401 / 冻结账号 403 / 无 SiteManager 权限 403（params.permission="站点管理"）。
func adminContentOpsGuardScenarios(t *testing.T, method, path, fixturePrefix string) {
	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAdminContentOpsContractTest(t)
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixturePrefix+"-unauthenticated.json"))
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		if err := conn.Model(user).Update("is_frozen", users.StatusFrozen).Error; err != nil {
			t.Fatalf("freeze contract user: %v", err)
		}
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("frozen account status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixturePrefix+"-forbidden.json"))
	})

	t.Run("user without SiteManager returns 403", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("permission denied status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixturePrefix+"-permission-denied.json"))
	})
}

// seedContractReviewTopic 插入一条契约主题并在子测试结束后删除。
func seedContractReviewTopic(t *testing.T, conn *gorm.DB, entity topics.Entity) topics.Entity {
	t.Helper()
	if err := conn.Create(&entity).Error; err != nil {
		t.Fatalf("seed topic %d: %v", entity.Id, err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Delete(&topics.Entity{}, entity.Id)
	})
	return entity
}

// seedContractReviewPost 插入一条契约回复并在子测试结束后删除。
func seedContractReviewPost(t *testing.T, conn *gorm.DB, entity posts.Entity) posts.Entity {
	t.Helper()
	if err := conn.Create(&entity).Error; err != nil {
		t.Fatalf("seed post %d: %v", entity.Id, err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Delete(&posts.Entity{}, entity.Id)
	})
	return entity
}

func TestAdminListBadgesHTTPContract(t *testing.T) {
	path := "/api/admin/badges"

	t.Run("success returns the built-in system badges", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		// 清空共享 badges 表，保证列表恰为 15 个内建系统徽章（与 fixture 一致）。
		conn.Where("1 = 1").Delete(&badges.Entity{})
		badgeservice.InvalidateDefinitions()
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-badges-success.json")
	})

	adminContentOpsGuardScenarios(t, http.MethodGet, path, "admin-badges")
}

func TestAdminSaveBadgeHTTPContract(t *testing.T) {
	path := "/api/admin/badge-save"

	t.Run("success creates a custom badge with an explicit code", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		t.Cleanup(func() {
			conn.Where("code = ?", "custom_contract_badge").Delete(&badges.Entity{})
			badgeservice.InvalidateDefinitions()
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"code":"custom_contract_badge","type":"custom","grantMode":"manual","name":"契约徽章","description":"契约测试徽章","iconType":"asset","iconUrl":"/static/badges/custom.svg","color":"blue","level":"bronze","isEnabled":true,"isWearable":true,"sortOrder":5}`,
			"admin-badge-save-success.json")
		stored := badges.GetByCode("custom_contract_badge")
		if stored.Id == 0 || stored.Name != "契约徽章" || stored.Type != badges.TypeCustom {
			t.Fatalf("stored badge = %#v, want the submitted custom badge", stored)
		}
	})

	t.Run("blank name fails with nameRequired", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"name":"  "}`, "admin-badge-save-name-required.json")
	})

	t.Run("unknown type fails with typeInvalid", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"name":"契约徽章","type":"weird"}`, "admin-badge-save-type-invalid.json")
	})

	t.Run("system badge without a code fails with codeRequired", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"name":"契约徽章","type":"system"}`, "admin-badge-save-code-required.json")
	})

	t.Run("unknown grantMode fails with grantModeInvalid", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"name":"契约徽章","type":"custom","grantMode":"weird"}`, "admin-badge-save-grant-mode-invalid.json")
	})

	t.Run("unknown system code fails with systemNotFound", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"name":"契约徽章","type":"system","code":"no_such_system_badge"}`, "admin-badge-save-system-not-found.json")
	})

	adminContentOpsGuardScenarios(t, http.MethodPost, path, "admin-badge-save")
}

func TestAdminDeleteBadgeHTTPContract(t *testing.T) {
	path := "/api/admin/badge-delete"

	t.Run("success deletes a custom badge", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		t.Cleanup(func() {
			conn.Where("code = ?", "custom_contract_badge").Delete(&badges.Entity{})
			badgeservice.InvalidateDefinitions()
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, "/api/admin/badge-save",
			`{"code":"custom_contract_badge","type":"custom","name":"契约徽章"}`,
			"admin-badge-save-success.json")
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"code":"custom_contract_badge"}`, "admin-badge-delete-success.json")
		if stored := badges.GetByCode("custom_contract_badge"); stored.Id != 0 {
			t.Fatalf("badge still present after delete: %#v", stored)
		}
	})

	t.Run("blank code fails with codeRequired", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"code":"  "}`, "admin-badge-delete-code-required.json")
	})

	t.Run("system badge fails with systemDeleteBlocked", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"code":"early_member"}`, "admin-badge-delete-system-delete-blocked.json")
	})

	adminContentOpsGuardScenarios(t, http.MethodPost, path, "admin-badge-delete")
}

func TestAdminListReviewQueueHTTPContract(t *testing.T) {
	path := "/api/admin/review-queue"

	t.Run("success lists the seeded pending topic", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		author := createHTTPContractUser(t, conn, contractTestID())
		topic := seedContractReviewTopic(t, conn, topics.Entity{
			Id:            contractTestID(),
			Title:         "契约待审主题",
			UserId:        author.Id,
			Status:        1,
			ProcessStatus: topics.ProcessStatusPending,
			TopicType:     topics.TopicTypeForum,
			CreatedAt:     contractTaskCreatedAt,
			UpdatedAt:     contractTaskCreatedAt,
		})
		result := decodeSiteResult(t, serveAdminSiteRaw(t, conn, router, http.MethodPost, path, `{"kind":"topic"}`))
		if result["page"] != float64(1) || result["pageSize"] != float64(20) {
			t.Fatalf("page/pageSize = %#v/%#v, want 1/20", result["page"], result["pageSize"])
		}
		items, ok := result["items"].([]any)
		if !ok {
			t.Fatalf("result.items = %#v, want an array", result["items"])
		}
		item := findContractItem(items, float64(topic.Id))
		if item == nil {
			t.Fatalf("seeded topic %d not found in review queue items %#v", topic.Id, items)
		}
		if item["title"] != "契约待审主题" || item["excerpt"] != "契约待审主题" {
			t.Fatalf("item title/excerpt = %#v/%#v, want the seeded title (excerpt falls back to title)", item["title"], item["excerpt"])
		}
		if item["username"] != author.Username {
			t.Fatalf("item username = %#v, want %q", item["username"], author.Username)
		}
		if item["processStatus"] != float64(2) || item["createdAt"] != "2026-02-03T04:05:06Z" {
			t.Fatalf("item processStatus/createdAt = %#v/%#v, want 2 and the seeded timestamp", item["processStatus"], item["createdAt"])
		}
		if _, exists := item["topicId"]; exists {
			t.Fatalf("topic item must omit topicId, got %#v", item["topicId"])
		}
	})

	t.Run("success lists the seeded pending post with topicId and postNo", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		author := createHTTPContractUser(t, conn, contractTestID())
		topic := seedContractReviewTopic(t, conn, topics.Entity{
			Id:            contractTestID(),
			Title:         "契约宿主主题",
			UserId:        author.Id,
			Status:        1,
			ProcessStatus: topics.ProcessStatusNormal,
			TopicType:     topics.TopicTypeForum,
			CreatedAt:     contractTaskCreatedAt,
			UpdatedAt:     contractTaskCreatedAt,
		})
		post := seedContractReviewPost(t, conn, posts.Entity{
			Id:               contractTestID(),
			TopicId:          topic.Id,
			PostNo:           2,
			UserId:           author.Id,
			Content:          "契约待审回复内容",
			ProcessStatus:    posts.ProcessStatusPending,
			VisibilityStatus: posts.VisibilityActive,
			CreatedAt:        contractTaskCreatedAt,
		})
		result := decodeSiteResult(t, serveAdminSiteRaw(t, conn, router, http.MethodPost, path, `{"kind":"post"}`))
		items, ok := result["items"].([]any)
		if !ok {
			t.Fatalf("result.items = %#v, want an array", result["items"])
		}
		item := findContractItem(items, float64(post.Id))
		if item == nil {
			t.Fatalf("seeded post %d not found in review queue items %#v", post.Id, items)
		}
		if item["title"] != "契约宿主主题" || item["excerpt"] != "契约待审回复内容" {
			t.Fatalf("item title/excerpt = %#v/%#v, want the topic title and post content", item["title"], item["excerpt"])
		}
		if item["topicId"] != float64(topic.Id) || item["postNo"] != float64(2) {
			t.Fatalf("item topicId/postNo = %#v/%#v, want %d/2", item["topicId"], item["postNo"], topic.Id)
		}
	})

	t.Run("unknown kind fails request validation", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"kind":"comment"}`, "admin-review-queue-invalid-params.json")
	})

	adminContentOpsGuardScenarios(t, http.MethodPost, path, "admin-review-queue")
}

// findContractItem 在审核队列 items 中按 id 查找条目。
func findContractItem(items []any, id float64) map[string]any {
	for _, raw := range items {
		if item, ok := raw.(map[string]any); ok && item["id"] == id {
			return item
		}
	}
	return nil
}

func TestAdminReviewActionHTTPContract(t *testing.T) {
	path := "/api/admin/review-action"

	t.Run("approve marks the pending topic and its first post as normal", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		author := createHTTPContractUser(t, conn, contractTestID())
		topicID := contractTestID()
		firstPost := seedContractReviewPost(t, conn, posts.Entity{
			Id:               contractTestID(),
			TopicId:          topicID,
			PostNo:           1,
			UserId:           author.Id,
			Content:          "契约待审首楼",
			ProcessStatus:    posts.ProcessStatusPending,
			VisibilityStatus: posts.VisibilityActive,
			CreatedAt:        contractTaskCreatedAt,
		})
		seedContractReviewTopic(t, conn, topics.Entity{
			Id:            topicID,
			Title:         "契约待审主题",
			UserId:        author.Id,
			Status:        1,
			ProcessStatus: topics.ProcessStatusPending,
			TopicType:     topics.TopicTypeForum,
			FirstPostId:   firstPost.Id,
			CreatedAt:     contractTaskCreatedAt,
			UpdatedAt:     contractTaskCreatedAt,
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			fmt.Sprintf(`{"kind":"topic","id":%d,"approve":true}`, topicID),
			"admin-review-action-success.json")
		if got := topics.Get(topicID); got.ProcessStatus != topics.ProcessStatusNormal {
			t.Fatalf("topic processStatus = %d, want normal after approve", got.ProcessStatus)
		}
		if got := posts.Get(firstPost.Id); got.ProcessStatus != posts.ProcessStatusNormal {
			t.Fatalf("first post processStatus = %d, want normal after approve", got.ProcessStatus)
		}
	})

	t.Run("reject marks the pending post as blocked", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		author := createHTTPContractUser(t, conn, contractTestID())
		topic := seedContractReviewTopic(t, conn, topics.Entity{
			Id:            contractTestID(),
			Title:         "契约宿主主题",
			UserId:        author.Id,
			Status:        1,
			ProcessStatus: topics.ProcessStatusNormal,
			TopicType:     topics.TopicTypeForum,
			CreatedAt:     contractTaskCreatedAt,
			UpdatedAt:     contractTaskCreatedAt,
		})
		post := seedContractReviewPost(t, conn, posts.Entity{
			Id:               contractTestID(),
			TopicId:          topic.Id,
			PostNo:           2,
			UserId:           author.Id,
			Content:          "契约待审回复内容",
			ProcessStatus:    posts.ProcessStatusPending,
			VisibilityStatus: posts.VisibilityActive,
			CreatedAt:        contractTaskCreatedAt,
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			fmt.Sprintf(`{"kind":"post","id":%d,"approve":false}`, post.Id),
			"admin-review-action-success.json")
		if got := posts.Get(post.Id); got.ProcessStatus != posts.ProcessStatusBlocked {
			t.Fatalf("post processStatus = %d, want blocked after reject", got.ProcessStatus)
		}
	})

	t.Run("unknown target fails with notFound", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"kind":"topic","id":999999999,"approve":true}`, "admin-review-action-not-found.json")
	})

	t.Run("already processed target fails with processed", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		author := createHTTPContractUser(t, conn, contractTestID())
		topic := seedContractReviewTopic(t, conn, topics.Entity{
			Id:            contractTestID(),
			Title:         "契约正常主题",
			UserId:        author.Id,
			Status:        1,
			ProcessStatus: topics.ProcessStatusNormal,
			TopicType:     topics.TopicTypeForum,
			CreatedAt:     contractTaskCreatedAt,
			UpdatedAt:     contractTaskCreatedAt,
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			fmt.Sprintf(`{"kind":"topic","id":%d,"approve":true}`, topic.Id),
			"admin-review-action-processed.json")
	})

	t.Run("wiki topic fails with targetInvalid", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		author := createHTTPContractUser(t, conn, contractTestID())
		topic := seedContractReviewTopic(t, conn, topics.Entity{
			Id:            contractTestID(),
			Title:         "契约 wiki 主题",
			UserId:        author.Id,
			Status:        1,
			ProcessStatus: topics.ProcessStatusPending,
			TopicType:     topics.TopicTypeWiki,
			CreatedAt:     contractTaskCreatedAt,
			UpdatedAt:     contractTaskCreatedAt,
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			fmt.Sprintf(`{"kind":"topic","id":%d,"approve":true}`, topic.Id),
			"admin-review-action-target-invalid.json")
	})

	t.Run("missing kind and id fail request validation", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{}`, "admin-review-action-invalid-params.json")
	})

	adminContentOpsGuardScenarios(t, http.MethodPost, path, "admin-review-action")
}

func TestAdminListFileResourcesHTTPContract(t *testing.T) {
	path := "/api/admin/file-resources"

	t.Run("success lists the seeded file with the uploader username", func(t *testing.T) {
		conn, router := setupAdminContentOpsContractTest(t)
		manager := createContractSiteManager(t, conn)
		fileName := fmt.Sprintf("contract/%d.png", contractTestID())
		if _, err := filedata.SaveFile(manager.Id, fileName, "image/png", contractTinyPNG); err != nil {
			t.Fatalf("seed file resource: %v", err)
		}
		t.Cleanup(func() {
			db4fileconnect.Connect().Where("name = ?", fileName).Delete(&filedata.Entity{})
		})
		recorder := serveAuthSecurityJSON(router, http.MethodPost, path, `{"page":1,"pageSize":10}`, contractSessionToken(t, manager))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		result := decodeSiteResult(t, recorder)
		if result["page"] != float64(1) || result["size"] != float64(10) {
			t.Fatalf("page/size = %#v/%#v, want 1/10", result["page"], result["size"])
		}
		list, ok := result["list"].([]any)
		if !ok {
			t.Fatalf("result.list = %#v, want an array", result["list"])
		}
		var item map[string]any
		for _, raw := range list {
			candidate, _ := raw.(map[string]any)
			if candidate["name"] == fileName {
				item = candidate
				break
			}
		}
		if item == nil {
			t.Fatalf("seeded file %q not found in list %#v", fileName, list)
		}
		if item["url"] != "/file/img/"+fileName {
			t.Fatalf("item url = %#v, want /file/img/%s", item["url"], fileName)
		}
		if item["uploaderUsername"] != manager.Username {
			t.Fatalf("item uploaderUsername = %#v, want %q", item["uploaderUsername"], manager.Username)
		}
		if item["size"] != float64(len(contractTinyPNG)) || item["type"] != "image/png" {
			t.Fatalf("item size/type = %#v/%#v, want %d/image/png", item["size"], item["type"], len(contractTinyPNG))
		}
	})

	adminContentOpsGuardScenarios(t, http.MethodPost, path, "admin-file-resources")
}

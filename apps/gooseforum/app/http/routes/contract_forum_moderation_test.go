package routes

import (
	"net/http"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/contentDeleteEvent"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupForumModerationContractTest 在共享 harness（setupHTTPContractTest）之上注册
// 6 条论坛审核路由，中间件链与 route4api.go 的生产注册保持一致
// （权限判定在控制器内：CanAccessModeration / CanModerateAnyCategory）。
func setupForumModerationContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&reports.Entity{},
		&moderationLog.Entity{},
		&contentDeleteEvent.Entity{},
	); err != nil {
		t.Fatalf("migrate forum moderation contract tables: %v", err)
	}

	loginAPI := router.Group("/api/forum").Use(middleware.JWTAuthCheck)
	loginAPI.POST("/moderation/topic-status", middleware.CheckWritableAccount, UpButterReq(forum.UpdateModerationTopicStatus))
	loginAPI.POST("/moderation/post-status", middleware.CheckWritableAccount, UpButterReq(forum.UpdateModerationPostStatus))
	loginAPI.POST("/moderation/reports", middleware.NoUpdateUserActivity, UpButterReq(forum.ModerationReportList))
	loginAPI.POST("/moderation/report-status", middleware.CheckWritableAccount, UpButterReq(forum.UpdateModerationReportStatus))
	loginAPI.POST("/moderation/logs", middleware.NoUpdateUserActivity, UpButterReq(forum.ModerationLogList))
	loginAPI.POST("/moderation/view-deleted-content", middleware.CheckWritableAccount, UpButterReq(forum.ViewDeletedContent))
	return conn, router
}

// 确定性 fixture（moderation-* / admin-topic-* / admin-post-*）使用的固定 ID 与时间戳。
// 固定 ID 在多个子测试间复用，prepare 的 cleanup 负责行级还原，避免污染同进程共享的
// sqlite 测试库（例如 user 1024 会被 TestUserCardHTTPContract 以同名同 ID 重建）。
const (
	contractModerationCategoryID  uint64 = 3
	contractModerationTopicID     uint64 = 1201
	contractModerationFirstPostID uint64 = 1301
	contractModerationReplyPostID uint64 = 1302
	contractModerationOperatorID  uint64 = 7
	contractModerationAuthorID    uint64 = 1024
	contractModerationReportID    uint64 = 9012
	contractModerationLogID       uint64 = 701
)

var (
	contractModerationTopicCreated = time.Date(2026, 8, 10, 8, 0, 0, 0, time.UTC)
	contractModerationTopicUpdated = time.Date(2026, 8, 15, 9, 20, 0, 0, time.UTC)
	contractModerationTopicDeleted = time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	contractModerationReportTime   = time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	contractModerationLogTime      = time.Date(2026, 8, 15, 10, 5, 0, 0, time.UTC)
)

// prepareContractModerationTopic 直接写库造确定性分类/用户/话题/首楼/分类索引，
// cleanup 还原固定 ID 行并清分类缓存（hotdataserve 分类缓存 TTL 1min，进程内共享）。
func prepareContractModerationTopic(t *testing.T, conn *gorm.DB) {
	t.Helper()
	createContractAvatarUser(t, conn, contractModerationOperatorID, "mod_user", "")
	createContractAvatarUser(t, conn, contractModerationAuthorID, "tongji_user", "")
	if err := conn.Create(&category.Entity{
		Id:    contractModerationCategoryID,
		Name:  "学习交流",
		Slug:  "study",
		Color: "#3b82f6",
	}).Error; err != nil {
		t.Fatalf("create contract moderation category: %v", err)
	}
	hotdataserve.ClearCategoryCache()
	topic := topics.Entity{
		Id:               contractModerationTopicID,
		Title:            "期中复习资料汇总",
		Excerpt:          "整理了本学期各科目的复习资料，需要的同学自取。",
		UserId:           contractModerationAuthorID,
		Status:           1,
		ProcessStatus:    topics.ProcessStatusNormal,
		PostCount:        1,
		PostSeq:          1,
		FirstPostId:      contractModerationFirstPostID,
		LastPostId:       contractModerationFirstPostID,
		CategoryIds:      []uint64{contractModerationCategoryID},
		ViewCount:        356,
		ReplyCount:       42,
		LikeCount:        18,
		VisibilityStatus: topics.VisibilityActive,
		RetentionStatus:  topics.RetentionNormal,
		CreatedAt:        contractModerationTopicCreated,
		UpdatedAt:        contractModerationTopicUpdated,
	}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create contract moderation topic: %v", err)
	}
	content := "整理了本学期各科目的复习资料，**需要的同学自取**。"
	firstPost := posts.Entity{
		Id:               contractModerationFirstPostID,
		TopicId:          contractModerationTopicID,
		PostNo:           1,
		UserId:           contractModerationAuthorID,
		Content:          content,
		RenderedHTML:     markdown2html.PostMarkdownToHTML(content),
		RenderedVersion:  markdown2html.GetPostVersion(),
		ProcessStatus:    posts.ProcessStatusNormal,
		VisibilityStatus: posts.VisibilityActive,
		RetentionStatus:  posts.RetentionNormal,
		CreatedAt:        contractModerationTopicCreated,
		UpdatedAt:        contractModerationTopicUpdated,
	}
	if err := conn.Create(&firstPost).Error; err != nil {
		t.Fatalf("create contract moderation first post: %v", err)
	}
	if err := conn.Create(&topicCategoryIndex.Entity{
		TopicId:    contractModerationTopicID,
		CategoryId: contractModerationCategoryID,
		Effective:  1,
	}).Error; err != nil {
		t.Fatalf("create contract moderation category index: %v", err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Delete(&topics.Entity{}, contractModerationTopicID)
		conn.Unscoped().Delete(&posts.Entity{}, contractModerationFirstPostID)
		conn.Unscoped().Delete(&posts.Entity{}, contractModerationReplyPostID)
		conn.Where("topic_id = ?", contractModerationTopicID).Delete(&topicCategoryIndex.Entity{})
		// 审核操作会围绕 fixture 话题/回复/举报写 moderation_logs；这些行在话题与分类索引
		// 重建后会重新落入分类作用域查询，必须一并清理，保证 logs 列表 fixture 确定性。
		conn.Where(
			"(subject_type = ? AND subject_id = ?) OR (subject_type = ? AND subject_id IN ?) OR (subject_type = ? AND subject_id = ?)",
			moderationLog.SubjectTopic, contractModerationTopicID,
			moderationLog.SubjectPost, []uint64{contractModerationFirstPostID, contractModerationReplyPostID},
			moderationLog.SubjectReport, contractModerationReportID,
		).Delete(&moderationLog.Entity{})
		conn.Unscoped().Delete(&users.EntityComplete{}, contractModerationOperatorID)
		conn.Unscoped().Delete(&users.EntityComplete{}, contractModerationAuthorID)
		conn.Delete(&category.Entity{}, contractModerationCategoryID)
		hotdataserve.ClearCategoryCache()
	})
}

// markContractModerationTopicDeleted 把 fixture 话题置为治理删除态（含删除元数据），
// 供 view-deleted-content / admin restore 场景使用。
func markContractModerationTopicDeleted(t *testing.T, conn *gorm.DB) {
	t.Helper()
	if err := conn.Unscoped().Model(&topics.Entity{}).
		Where("id = ?", contractModerationTopicID).
		Updates(map[string]any{
			"visibility_status": topics.VisibilityModeratorRemoved,
			"retention_status":  topics.RetentionRecoverable,
			"deleted_at":        contractModerationTopicDeleted,
			"deleted_by":        contractModerationOperatorID,
			"delete_reason":     "违反社区规范",
		}).Error; err != nil {
		t.Fatalf("mark contract moderation topic deleted: %v", err)
	}
}

// grantContractCategoryModerator 授予用户 fixture 分类的版主权限，并失效
// moderationservice 的 1min 快照缓存使授权立即生效。
func grantContractCategoryModerator(t *testing.T, conn *gorm.DB, userID uint64) {
	t.Helper()
	if err := conn.Create(&moderators.Entity{
		UserId:    userID,
		ScopeType: moderators.ScopeCategory,
		ScopeId:   contractModerationCategoryID,
		Status:    moderators.StatusEnabled,
		CreatedBy: userID,
	}).Error; err != nil {
		t.Fatalf("grant contract category moderator: %v", err)
	}
	moderationservice.Invalidate()
}

// createContractCategoryModerator 创建登录用户并授予 fixture 分类的版主权限。
func createContractCategoryModerator(t *testing.T, conn *gorm.DB) *users.EntityComplete {
	t.Helper()
	user := createHTTPContractUser(t, conn, contractTestID())
	grantContractCategoryModerator(t, conn, user.Id)
	return user
}

// createContractModerationReport 写一条针对 fixture 话题的 open 举报（确定性 fixture）。
func createContractModerationReport(t *testing.T, conn *gorm.DB) {
	t.Helper()
	if err := conn.Create(&reports.Entity{
		Id:         contractModerationReportID,
		TargetType: reports.TargetTopic,
		TargetId:   contractModerationTopicID,
		TopicId:    contractModerationTopicID,
		ReporterId: contractModerationAuthorID,
		Reason:     reports.ReasonSpam,
		Note:       "疑似广告引流",
		Status:     reports.StatusOpen,
		CreatedAt:  contractModerationReportTime,
	}).Error; err != nil {
		t.Fatalf("create contract moderation report: %v", err)
	}
	t.Cleanup(func() {
		conn.Delete(&reports.Entity{}, contractModerationReportID)
	})
}

// createContractModerationLog 写一条针对 fixture 话题的审核日志（确定性 fixture）。
func createContractModerationLog(t *testing.T, conn *gorm.DB) {
	t.Helper()
	if err := conn.Create(&moderationLog.Entity{
		Id:          contractModerationLogID,
		ActorUserId: contractModerationOperatorID,
		Action:      moderationLog.ActionTopicBlocked,
		SubjectType: moderationLog.SubjectTopic,
		SubjectId:   contractModerationTopicID,
		Payload: moderationLog.Payload{
			MessageCode: "moderation.log.topic.statusChanged",
			Params: map[string]any{
				"topicId": contractModerationTopicID,
				"title":   "期中复习资料汇总",
				"status":  "blocked",
			},
		},
		CreatedAt: contractModerationLogTime,
	}).Error; err != nil {
		t.Fatalf("create contract moderation log: %v", err)
	}
	t.Cleanup(func() {
		conn.Delete(&moderationLog.Entity{}, contractModerationLogID)
	})
}

// assertModerationPermissionDenied 断言无审核权限的登录用户得到 HTTP 200 + permission.denied
// （论坛审核的权限判定在控制器内，走 FailResponseCode，不是中间件 403）。
func assertModerationPermissionDenied(t *testing.T, conn *gorm.DB, router *gin.Engine, path string, body string, fixture string) {
	t.Helper()
	user := createHTTPContractUser(t, conn, contractTestID())
	recorder := serveJSON(router, path, body, contractSessionToken(t, user))
	if recorder.Code != http.StatusOK {
		t.Fatalf("permission denied status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixture))
}

func TestModerationTopicStatusHTTPContract(t *testing.T) {
	path := "/api/forum/moderation/topic-status"

	t.Run("success", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		prepareContractModerationTopic(t, conn)
		moderator := createContractCategoryModerator(t, conn)
		recorder := serveJSON(router, path, `{"topicId":1201,"action":"ban"}`, contractSessionToken(t, moderator))
		if recorder.Code != http.StatusOK {
			t.Fatalf("topic status status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "moderation-topic-status-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumModerationContractTest(t)
		assertInteractionUnauthenticated(t, router, path, `{}`, "moderation-topic-status-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		assertInteractionForbidden(t, conn, router, path, `{}`, "moderation-topic-status-forbidden.json")
	})

	t.Run("user without moderation scope returns business failure", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		prepareContractModerationTopic(t, conn)
		assertModerationPermissionDenied(t, conn, router, path, `{"topicId":1201,"action":"ban"}`, "moderation-topic-status-permission-denied.json")
	})

	t.Run("unknown topic returns business failure", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, path, `{"topicId":987654321,"action":"ban"}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown topic status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "moderation-topic-status-topic-not-found.json"))
	})

	t.Run("invalid action stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, path, `{"topicId":1,"action":"bogus"}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid params status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "moderation-topic-status-invalid-params.json"))
	})
}

func TestModerationPostStatusHTTPContract(t *testing.T) {
	path := "/api/forum/moderation/post-status"

	t.Run("success", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		prepareContractModerationTopic(t, conn)
		moderator := createContractCategoryModerator(t, conn)
		recorder := serveJSON(router, path, `{"postId":1301,"action":"ban"}`, contractSessionToken(t, moderator))
		if recorder.Code != http.StatusOK {
			t.Fatalf("post status status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "moderation-post-status-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumModerationContractTest(t)
		assertInteractionUnauthenticated(t, router, path, `{}`, "moderation-post-status-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		assertInteractionForbidden(t, conn, router, path, `{}`, "moderation-post-status-forbidden.json")
	})

	t.Run("user without moderation scope returns business failure", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		prepareContractModerationTopic(t, conn)
		assertModerationPermissionDenied(t, conn, router, path, `{"postId":1301,"action":"ban"}`, "moderation-post-status-permission-denied.json")
	})

	t.Run("unknown post returns business failure", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, path, `{"postId":987654321,"action":"ban"}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown post status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "moderation-post-status-post-not-found.json"))
	})
}

func TestModerationReportListHTTPContract(t *testing.T) {
	path := "/api/forum/moderation/reports"

	t.Run("success", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		prepareContractModerationTopic(t, conn)
		createContractModerationReport(t, conn)
		moderator := createContractCategoryModerator(t, conn)
		recorder := serveJSON(router, path, `{}`, contractSessionToken(t, moderator))
		if recorder.Code != http.StatusOK {
			t.Fatalf("report list status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "moderation-reports-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumModerationContractTest(t)
		assertInteractionUnauthenticated(t, router, path, `{}`, "moderation-reports-unauthenticated.json")
	})

	t.Run("user without moderation access returns business failure", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		assertModerationPermissionDenied(t, conn, router, path, `{}`, "moderation-reports-permission-denied.json")
	})

	t.Run("invalid status filter stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, path, `{"status":"bogus"}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid params status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "moderation-reports-invalid-params.json"))
	})
}

func TestModerationReportStatusHTTPContract(t *testing.T) {
	path := "/api/forum/moderation/report-status"

	t.Run("success", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		prepareContractModerationTopic(t, conn)
		createContractModerationReport(t, conn)
		moderator := createContractCategoryModerator(t, conn)
		recorder := serveJSON(router, path, `{"id":9012,"action":"resolve"}`, contractSessionToken(t, moderator))
		if recorder.Code != http.StatusOK {
			t.Fatalf("report status status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "moderation-report-status-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumModerationContractTest(t)
		assertInteractionUnauthenticated(t, router, path, `{}`, "moderation-report-status-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		assertInteractionForbidden(t, conn, router, path, `{}`, "moderation-report-status-forbidden.json")
	})

	t.Run("user without scope over report target returns business failure", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		prepareContractModerationTopic(t, conn)
		createContractModerationReport(t, conn)
		assertModerationPermissionDenied(t, conn, router, path, `{"id":9012,"action":"resolve"}`, "moderation-report-status-permission-denied.json")
	})

	t.Run("unknown report returns business failure", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, path, `{"id":987654321,"action":"resolve"}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown report status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "moderation-report-status-report-not-found.json"))
	})
}

func TestModerationLogListHTTPContract(t *testing.T) {
	path := "/api/forum/moderation/logs"

	t.Run("success", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		prepareContractModerationTopic(t, conn)
		createContractModerationLog(t, conn)
		moderator := createContractCategoryModerator(t, conn)
		recorder := serveJSON(router, path, `{}`, contractSessionToken(t, moderator))
		if recorder.Code != http.StatusOK {
			t.Fatalf("log list status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "moderation-logs-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumModerationContractTest(t)
		assertInteractionUnauthenticated(t, router, path, `{}`, "moderation-logs-unauthenticated.json")
	})

	t.Run("user without moderation access returns business failure", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		assertModerationPermissionDenied(t, conn, router, path, `{}`, "moderation-logs-permission-denied.json")
	})
}

func TestModerationViewDeletedContentHTTPContract(t *testing.T) {
	path := "/api/forum/moderation/view-deleted-content"

	t.Run("success", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		prepareContractModerationTopic(t, conn)
		markContractModerationTopicDeleted(t, conn)
		moderator := createContractCategoryModerator(t, conn)
		body := `{"contentType":"topic","contentId":1201,"reason":"核对删除原因与原文"}`
		recorder := serveJSON(router, path, body, contractSessionToken(t, moderator))
		if recorder.Code != http.StatusOK {
			t.Fatalf("view deleted content status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "moderation-view-deleted-content-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumModerationContractTest(t)
		assertInteractionUnauthenticated(t, router, path, `{}`, "moderation-view-deleted-content-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		assertInteractionForbidden(t, conn, router, path, `{}`, "moderation-view-deleted-content-forbidden.json")
	})

	t.Run("user without moderation access returns business failure", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		body := `{"contentType":"topic","contentId":1201,"reason":"核对删除原因与原文"}`
		assertModerationPermissionDenied(t, conn, router, path, body, "moderation-view-deleted-content-permission-denied.json")
	})

	t.Run("unknown topic returns business failure", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		moderator := createContractCategoryModerator(t, conn)
		body := `{"contentType":"topic","contentId":987654321,"reason":"核对删除原因与原文"}`
		recorder := serveJSON(router, path, body, contractSessionToken(t, moderator))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown topic status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "moderation-view-deleted-content-topic-not-found.json"))
	})

	t.Run("missing reason stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupForumModerationContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, path, `{"contentType":"topic","contentId":1201}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid params status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "moderation-view-deleted-content-invalid-params.json"))
	})
}

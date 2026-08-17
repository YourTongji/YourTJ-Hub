package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/contentDeleteEvent"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupForumInteractionContractTest 在共享 harness（setupHTTPContractTest）之上
// 补齐 14 条论坛互动路由，中间件链与 route4api.go 的生产注册保持一致。
func setupForumInteractionContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&contentDeleteEvent.Entity{},
		&postUserAction.Entity{},
		&userFollow.Entity{},
		&reports.Entity{},
	); err != nil {
		t.Fatalf("migrate forum interaction contract tables: %v", err)
	}

	forumAPI := router.Group("/api/forum")
	// 楼层窗口/版本历史：公开只读，可选 JWT（非 JWTAuthCheck）仅用于 viewer 状态。
	forumAPI.GET("/posts/window", middleware.JWTAuth, middleware.NoUpdateUserActivity, UpQueryReq(forum.PostWindow))
	forumAPI.GET("/posts/revisions", middleware.JWTAuth, middleware.NoUpdateUserActivity, UpQueryReq(forum.PostRevisions))

	loginAPI := forumAPI.Use(middleware.JWTAuthCheck)
	loginAPI.POST("/posts/create", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitPostCreate), UpButterReq(api.CreatePost))
	loginAPI.POST("/posts/update", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitPostUpdate), UpButterReq(api.UpdatePost))
	loginAPI.POST("/posts/delete", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitPostDelete), UpButterReq(api.DeletePost))
	loginAPI.POST("/posts/like", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.LikePost))
	loginAPI.POST("/posts/bookmark", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.BookmarkPost))
	loginAPI.POST("/topics/status", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitTopicStatus), UpButterReq(api.UpdateTopicStatus))
	loginAPI.POST("/topics/delete", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.DeleteTopicByUser))
	loginAPI.POST("/topics/like", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.LikeTopic))
	loginAPI.POST("/topics/bookmark", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.BookmarkTopic))
	loginAPI.POST("/topics/watch", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.WatchTopic))
	loginAPI.POST("/follow-user", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.FollowUser))
	loginAPI.POST("/report", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(forum.CreateReport))
	return conn, router
}

// contractInteractionTime 是确定性 fixture（楼层窗口/版本历史）使用的固定时间戳。
var contractInteractionTime = time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

// createContractPublishedTopic 直接写库造一个已发布、可见的话题及其首楼。
func createContractPublishedTopic(t *testing.T, conn *gorm.DB, topicID, firstPostID, authorID uint64) {
	t.Helper()
	topic := topics.Entity{
		Id:               topicID,
		Title:            "Contract interaction topic",
		UserId:           authorID,
		Status:           1,
		ProcessStatus:    topics.ProcessStatusNormal,
		PostCount:        1,
		PostSeq:          1,
		FirstPostId:      firstPostID,
		LastPostId:       firstPostID,
		VisibilityStatus: topics.VisibilityActive,
		RetentionStatus:  topics.RetentionNormal,
		CreatedAt:        contractInteractionTime,
		UpdatedAt:        contractInteractionTime,
	}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create contract topic: %v", err)
	}
	content := "Topic first post content"
	firstPost := posts.Entity{
		Id:               firstPostID,
		TopicId:          topicID,
		PostNo:           1,
		UserId:           authorID,
		Content:          content,
		RenderedHTML:     markdown2html.PostMarkdownToHTML(content),
		RenderedVersion:  markdown2html.GetPostVersion(),
		ProcessStatus:    posts.ProcessStatusNormal,
		VisibilityStatus: posts.VisibilityActive,
		RetentionStatus:  posts.RetentionNormal,
		CreatedAt:        contractInteractionTime,
		UpdatedAt:        contractInteractionTime,
	}
	if err := conn.Create(&firstPost).Error; err != nil {
		t.Fatalf("create contract first post: %v", err)
	}
}

// createContractReplyPost 直接写库造一条 2 楼回复（PostNo=2，可用于编辑/删除/点赞）。
func createContractReplyPost(t *testing.T, conn *gorm.DB, postID, topicID, authorID uint64) {
	t.Helper()
	content := "Contract reply content"
	reply := posts.Entity{
		Id:               postID,
		TopicId:          topicID,
		PostNo:           2,
		UserId:           authorID,
		Content:          content,
		RenderedHTML:     markdown2html.PostMarkdownToHTML(content),
		RenderedVersion:  markdown2html.GetPostVersion(),
		ProcessStatus:    posts.ProcessStatusNormal,
		VisibilityStatus: posts.VisibilityActive,
		RetentionStatus:  posts.RetentionNormal,
		CreatedAt:        contractInteractionTime,
		UpdatedAt:        contractInteractionTime,
	}
	if err := conn.Create(&reply).Error; err != nil {
		t.Fatalf("create contract reply post: %v", err)
	}
}

// createContractAvatarUser 创建带固定用户名/头像的用户，供确定性 fixture 断言作者卡片。
func createContractAvatarUser(t *testing.T, conn *gorm.DB, id uint64, username string, avatar string) {
	t.Helper()
	user := users.MakeUser(username, "secret123", fmt.Sprintf("%s-%d@example.test", username, id))
	user.Id = id
	user.AvatarUrl = avatar
	user.IsActivated = users.ActivationSuccess
	user.CreatedAt = time.Now().Add(-48 * time.Hour)
	if err := conn.Create(user).Error; err != nil {
		t.Fatalf("create contract avatar user: %v", err)
	}
}

// restrictContractRateLimit 把限流配置收紧到单个 action（60s 窗口 per-IP 5 次），
// 供 rate-limited 子测试在第 6 次请求触发 429。harness 清理会还原原配置。
func restrictContractRateLimit(t *testing.T, conn *gorm.DB, action string) {
	t.Helper()
	persistHTTPContractConfig(t, conn, pageConfig.RateLimitSettings, pageConfig.RateLimitConfig{
		Enabled: true,
		Actions: []pageConfig.RateLimitRule{{Action: action, WindowSeconds: 60, LimitPerIp: 5}},
	})
	hotdataserve.ClearRateLimitConfigCache()
}

// assertInteractionUnauthenticated 断言未携带会话 token 的请求返回 401 及对应 fixture 信封。
func assertInteractionUnauthenticated(t *testing.T, router *gin.Engine, path string, body string, fixture string) {
	t.Helper()
	recorder := serveJSON(router, path, body, "")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401: %s", recorder.Code, recorder.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixture))
}

// assertInteractionForbidden 断言冻结账号的请求被 CheckWritableAccount 拦截为 403。
func assertInteractionForbidden(t *testing.T, conn *gorm.DB, router *gin.Engine, path string, body string, fixture string) {
	t.Helper()
	user := createHTTPContractUser(t, conn, contractTestID())
	if err := conn.Model(user).Update("is_frozen", users.StatusFrozen).Error; err != nil {
		t.Fatalf("freeze contract user: %v", err)
	}
	recorder := serveJSON(router, path, body, contractSessionToken(t, user))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("frozen account status = %d, want 403: %s", recorder.Code, recorder.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixture))
}

// assertInteractionRateLimited 在 60s/5 次配额下连发 5 次（均应为业务 200），
// 第 6 次断言 429 + fixture + Retry-After 元数据。
func assertInteractionRateLimited(t *testing.T, conn *gorm.DB, router *gin.Engine, path string, body string, fixture string, action string) {
	t.Helper()
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	restrictContractRateLimit(t, conn, action)
	for attempt := 0; attempt < 5; attempt++ {
		recorder := serveJSON(router, path, body, token)
		if recorder.Code != http.StatusOK {
			t.Fatalf("attempt %d status = %d, want 200: %s", attempt+1, recorder.Code, recorder.Body.String())
		}
	}
	recorder := serveJSON(router, path, body, token)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limited status = %d, want 429: %s", recorder.Code, recorder.Body.String())
	}
	response := decodeContractEnvelope(t, recorder)
	assertFixtureEnvelope(t, response, contractFixture(t, fixture))
	assertRetryAfter(t, recorder, response, action)
}

func TestCreatePostHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		base := contractTestID()
		topicID, firstPostID := base, base+1
		createContractPublishedTopic(t, conn, topicID, firstPostID, user.Id)
		content := "Contract reply content for the create post scenario."
		body := fmt.Sprintf(`{"topicId":%d,"content":%q}`, topicID, content)
		recorder := serveJSON(router, "/api/forum/posts/create", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("create post status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		response := decodeContractEnvelope(t, recorder)
		if response.Code != 0 {
			t.Fatalf("create post envelope = %#v, want success", response)
		}
		// result 含自增 id（共享测试库中随用例执行变化），逐字段断言确定性字段，
		// 与 agent/course-review 富 payload 成功场景的既有模式一致。
		var created struct {
			Id              uint64 `json:"id"`
			PostNo          uint64 `json:"postNo"`
			RenderedContent string `json:"renderedContent"`
		}
		if err := json.Unmarshal(response.Result, &created); err != nil {
			t.Fatalf("decode create post result %s: %v", response.Result, err)
		}
		if created.Id == 0 || created.PostNo != 2 {
			t.Fatalf("created = %#v, want id>0 postNo=2", created)
		}
		if created.RenderedContent != markdown2html.PostMarkdownToHTML(content) {
			t.Fatalf("renderedContent = %q, want markdown rendering of %q", created.RenderedContent, content)
		}
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/posts/create", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/posts/create", `{}`, "account-frozen.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/posts/create",
			`{"topicId":987654321,"content":"Contract rate limit probe content."}`,
			"agent-post-create-rate-limited.json", middleware.RateLimitPostCreate)
	})

	t.Run("unknown topic returns business failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		body := `{"topicId":987654321,"content":"Contract reply content for the not found scenario."}`
		recorder := serveJSON(router, "/api/forum/posts/create", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown topic status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-topic-categories-edit-topic-not-found.json"))
	})
}

func TestUpdatePostHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		// Windows 时钟粒度下连续调用 contractTestID 可能返回相同值，用 base 偏移保证互异。
		base := contractTestID()
		topicID, firstPostID, replyID := base, base+1, base+2
		createContractPublishedTopic(t, conn, topicID, firstPostID, user.Id)
		createContractReplyPost(t, conn, replyID, topicID, user.Id)
		content := "Contract updated reply content for the update post scenario."
		body := fmt.Sprintf(`{"postId":%d,"content":%q}`, replyID, content)
		recorder := serveJSON(router, "/api/forum/posts/update", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("update post status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		response := decodeContractEnvelope(t, recorder)
		if response.Code != 0 {
			t.Fatalf("update post envelope = %#v, want success", response)
		}
		// updatedAt/lastEditedAt 为请求时刻时间戳，逐字段断言确定性字段。
		var updated struct {
			Id              uint64 `json:"id"`
			PostNo          uint64 `json:"postNo"`
			Content         string `json:"content"`
			RenderedContent string `json:"renderedContent"`
			UpdatedAt       string `json:"updatedAt"`
			LastEditorId    uint64 `json:"lastEditorId"`
			LastEditedAt    string `json:"lastEditedAt"`
			RevisionCount   int64  `json:"revisionCount"`
		}
		if err := json.Unmarshal(response.Result, &updated); err != nil {
			t.Fatalf("decode update post result %s: %v", response.Result, err)
		}
		if updated.Id != replyID || updated.PostNo != 2 || updated.Content != content {
			t.Fatalf("updated = %#v, want id/postNo/content match", updated)
		}
		if updated.RenderedContent != markdown2html.PostMarkdownToHTML(content) {
			t.Fatalf("renderedContent = %q, want markdown rendering of %q", updated.RenderedContent, content)
		}
		if updated.LastEditorId != user.Id {
			t.Fatalf("lastEditorId = %d, want %d", updated.LastEditorId, user.Id)
		}
		// 存量帖子首次编辑惰性播种 v1（旧正文）+ 本次编辑 v2。
		if updated.RevisionCount != 2 {
			t.Fatalf("revisionCount = %d, want 2", updated.RevisionCount)
		}
		if _, err := time.Parse(time.RFC3339, updated.UpdatedAt); err != nil {
			t.Fatalf("updatedAt = %q, want RFC3339: %v", updated.UpdatedAt, err)
		}
		if _, err := time.Parse(time.RFC3339, updated.LastEditedAt); err != nil {
			t.Fatalf("lastEditedAt = %q, want RFC3339: %v", updated.LastEditedAt, err)
		}
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/posts/update", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/posts/update", `{}`, "account-frozen.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/posts/update",
			`{"postId":987654321,"content":"Contract rate limit probe content."}`,
			"post-update-rate-limited.json", middleware.RateLimitPostUpdate)
	})

	t.Run("unknown post returns business failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		body := `{"postId":987654321,"content":"Contract updated content for the not found scenario."}`
		recorder := serveJSON(router, "/api/forum/posts/update", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown post status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-post-delete-post-not-found.json"))
	})
}

func TestDeletePostHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		// Windows 时钟粒度下连续调用 contractTestID 可能返回相同值，用 base 偏移保证互异。
		base := contractTestID()
		topicID, firstPostID, replyID := base, base+1, base+2
		createContractPublishedTopic(t, conn, topicID, firstPostID, user.Id)
		createContractReplyPost(t, conn, replyID, topicID, user.Id)
		body := fmt.Sprintf(`{"postId":%d}`, replyID)
		recorder := serveJSON(router, "/api/forum/posts/delete", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("delete post status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "post-delete-success.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/posts/delete", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/posts/delete", `{}`, "account-frozen.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/posts/delete",
			`{"postId":987654321}`,
			"post-delete-rate-limited.json", middleware.RateLimitPostDelete)
	})

	t.Run("unknown post returns business failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/posts/delete", `{"postId":987654321}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown post status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-post-delete-post-not-found.json"))
	})
}

func TestPostWindowHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		// 固定 id/用户名/头像/时间戳，使楼层窗口响应与确定性 fixture 精确一致。
		createContractAvatarUser(t, conn, 9301, "contract-window-author", "/static/pic/3.webp")
		createContractPublishedTopic(t, conn, 9101, 9201, 9301)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/posts/window?topicId=9101", "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("post window status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "post-window-success.json"))
	})

	t.Run("unknown topic returns business failure", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/posts/window?topicId=987654321", "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown topic status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-topic-categories-edit-topic-not-found.json"))
	})

	t.Run("non-numeric topicId returns strict 400", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/posts/window?topicId=abc", "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("parse failed status = %d, want 400: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "parse-failed.json"))
	})
}

func TestPostRevisionsHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		// 固定 id/编辑者/时间戳 + 直接播种 v1 版本，使响应与确定性 fixture 精确一致。
		createContractAvatarUser(t, conn, 9302, "contract-revision-editor", "/static/pic/3.webp")
		createContractPublishedTopic(t, conn, 9102, 9251, 9302)
		createContractReplyPost(t, conn, 9252, 9102, 9302)
		content := "Original reply content"
		revision := postRevisions.Entity{
			PostId:        9252,
			Version:       1,
			EditorId:      9302,
			Content:       content,
			RenderedHTML:  markdown2html.PostMarkdownToHTML(content),
			ProcessStatus: posts.ProcessStatusNormal,
			CreatedAt:     contractInteractionTime,
		}
		if err := conn.Create(&revision).Error; err != nil {
			t.Fatalf("create contract post revision: %v", err)
		}
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/posts/revisions?postId=9252", "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("post revisions status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "post-revisions-success.json"))
	})

	t.Run("unknown post returns business failure", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/posts/revisions?postId=987654321", "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown post status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-post-delete-post-not-found.json"))
	})

	t.Run("non-numeric postId returns strict 400", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/posts/revisions?postId=abc", "", "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("parse failed status = %d, want 400: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "parse-failed.json"))
	})
}

func TestLikePostHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		// Windows 时钟粒度下连续调用 contractTestID 可能返回相同值，用 base 偏移保证互异。
		base := contractTestID()
		topicID, firstPostID, replyID := base, base+1, base+2
		createContractPublishedTopic(t, conn, topicID, firstPostID, user.Id)
		createContractReplyPost(t, conn, replyID, topicID, user.Id)
		body := fmt.Sprintf(`{"postId":%d,"action":1}`, replyID)
		recorder := serveJSON(router, "/api/forum/posts/like", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("like post status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "result-true.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/posts/like", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/posts/like", `{}`, "account-frozen.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/posts/like",
			`{"postId":987654321,"action":1}`,
			"account-close-rate-limited.json", middleware.RateLimitInteract)
	})

	t.Run("unknown post returns business failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/posts/like", `{"postId":987654321,"action":1}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown post status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-post-delete-post-not-found.json"))
	})

	t.Run("invalid action stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/posts/like", `{"postId":1,"action":3}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid params status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "invalid-params.json"))
	})
}

func TestBookmarkPostHTTPContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		// Windows 时钟粒度下连续调用 contractTestID 可能返回相同值，用 base 偏移保证互异。
		base := contractTestID()
		topicID, firstPostID, replyID := base, base+1, base+2
		createContractPublishedTopic(t, conn, topicID, firstPostID, user.Id)
		createContractReplyPost(t, conn, replyID, topicID, user.Id)
		body := fmt.Sprintf(`{"postId":%d,"action":1}`, replyID)
		recorder := serveJSON(router, "/api/forum/posts/bookmark", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("bookmark post status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "result-true.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupForumInteractionContractTest(t)
		assertInteractionUnauthenticated(t, router, "/api/forum/posts/bookmark", `{}`, "auth-required.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionForbidden(t, conn, router, "/api/forum/posts/bookmark", `{}`, "account-frozen.json")
	})

	t.Run("rate limit returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		assertInteractionRateLimited(t, conn, router, "/api/forum/posts/bookmark",
			`{"postId":987654321,"action":1}`,
			"account-close-rate-limited.json", middleware.RateLimitInteract)
	})

	t.Run("unknown post returns business failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/posts/bookmark", `{"postId":987654321,"action":1}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unknown post status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-post-delete-post-not-found.json"))
	})

	t.Run("invalid action stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupForumInteractionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveJSON(router, "/api/forum/posts/bookmark", `{"postId":1,"action":3}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("invalid params status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "invalid-params.json"))
	})
}

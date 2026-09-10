package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupAnonPostContractTest 在论坛互动契约 harness（setupForumInteractionContractTest，
// 已注册 posts/create + posts/window）之上补齐匿名楼层揭示路由（与 route4api.go 的
// /api/forum/moderation/post-reveal 注册一致），并迁移揭示所需的 opt_record /
// role_permission_rs 表。
func setupAnonPostContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupForumInteractionContractTest(t)
	if err := conn.AutoMigrate(&optRecord.Entity{}, &rolePermissionRs.Entity{}); err != nil {
		t.Fatalf("migrate anon post contract tables: %v", err)
	}
	loginAPI := router.Group("/api/forum").Use(middleware.JWTAuthCheck)
	loginAPI.POST("/moderation/post-reveal", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitPostReveal), UpButterReq(forum.ModerationPostReveal))
	return conn, router
}

// createContractWikiTopic 直接写库造一个 wiki 分站话题（TopicType=wiki）及其首楼，
// 与 createContractPublishedTopic 同构，仅 TopicType 不同。
func createContractWikiTopic(t *testing.T, conn *gorm.DB, topicID, firstPostID, authorID uint64) {
	t.Helper()
	topic := topics.Entity{
		Id:               topicID,
		Title:            "Contract wiki topic",
		UserId:           authorID,
		Status:           1,
		ProcessStatus:    topics.ProcessStatusNormal,
		TopicType:        topics.TopicTypeWiki,
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
		t.Fatalf("create contract wiki topic: %v", err)
	}
	content := "Wiki topic first post content"
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
		t.Fatalf("create contract wiki first post: %v", err)
	}
}

// createContractAnonymousReply 以 user 身份在 wiki 话题下发布一条匿名回复（isAnonymous=true），
// 返回创建的楼层 id。供窗口遮蔽与揭示场景复用。
func createContractAnonymousReply(t *testing.T, router *gin.Engine, token string, topicID, replyToPostID uint64) uint64 {
	t.Helper()
	content := "Contract anonymous reply content."
	body := fmt.Sprintf(`{"topicId":%d,"content":%q,"replyToPostId":%d,"isAnonymous":true}`, topicID, content, replyToPostID)
	recorder := serveJSON(router, "/api/forum/posts/create", body, token)
	if recorder.Code != http.StatusOK {
		t.Fatalf("anonymous create status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	response := decodeContractEnvelope(t, recorder)
	if response.Code != 0 {
		t.Fatalf("anonymous create envelope = %#v, want success", response)
	}
	var created struct {
		Id uint64 `json:"id"`
	}
	if err := json.Unmarshal(response.Result, &created); err != nil || created.Id == 0 {
		t.Fatalf("decode anonymous create result %s: %v", response.Result, err)
	}
	return created.Id
}

// TestAnonPostCreateHTTPContract 锁定匿名发布的准入边界（issue #524）：
// 仅 wiki 话题的回复（postNo>1）允许匿名，其余场景一律 comment.anonymousNotAllowed。
func TestAnonPostCreateHTTPContract(t *testing.T) {
	t.Run("anonymous reply on wiki topic succeeds", func(t *testing.T) {
		conn, router := setupAnonPostContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		base := contractTestID()
		topicID, firstPostID := base, base+1
		createContractWikiTopic(t, conn, topicID, firstPostID, user.Id)
		content := "Contract anonymous reply content."
		body := fmt.Sprintf(`{"topicId":%d,"content":%q,"replyToPostId":%d,"isAnonymous":true}`, topicID, content, firstPostID)
		recorder := serveJSON(router, "/api/forum/posts/create", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("anonymous create status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		response := decodeContractEnvelope(t, recorder)
		if response.Code != 0 {
			t.Fatalf("anonymous create envelope = %#v, want success", response)
		}
		var created struct {
			Id     uint64 `json:"id"`
			PostNo uint64 `json:"postNo"`
		}
		if err := json.Unmarshal(response.Result, &created); err != nil {
			t.Fatalf("decode anonymous create result %s: %v", response.Result, err)
		}
		if created.Id == 0 || created.PostNo != 2 {
			t.Fatalf("created = %#v, want id>0 postNo=2", created)
		}
	})

	t.Run("anonymous create on forum topic is rejected", func(t *testing.T) {
		conn, router := setupAnonPostContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		base := contractTestID()
		topicID, firstPostID := base, base+1
		createContractPublishedTopic(t, conn, topicID, firstPostID, user.Id)
		body := fmt.Sprintf(`{"topicId":%d,"content":"Contract anonymous reply content.","replyToPostId":%d,"isAnonymous":true}`, topicID, firstPostID)
		recorder := serveJSON(router, "/api/forum/posts/create", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("forum anonymous create status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		response := decodeContractEnvelope(t, recorder)
		if response.Code == 0 {
			t.Fatalf("forum anonymous create envelope = %#v, want business failure", response)
		}
		if response.MessageCode != string(component.MessageCommentAnonymousNotAllowed) {
			t.Fatalf("messageCode = %q, want %q", response.MessageCode, component.MessageCommentAnonymousNotAllowed)
		}
	})

	t.Run("anonymous top-level reply on wiki topic is rejected", func(t *testing.T) {
		conn, router := setupAnonPostContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		base := contractTestID()
		topicID, firstPostID := base, base+1
		createContractWikiTopic(t, conn, topicID, firstPostID, user.Id)
		body := fmt.Sprintf(`{"topicId":%d,"content":"Contract anonymous top-level reply content.","isAnonymous":true}`, topicID)
		recorder := serveJSON(router, "/api/forum/posts/create", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("wiki top-level anonymous create status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		response := decodeContractEnvelope(t, recorder)
		if response.Code == 0 {
			t.Fatalf("wiki top-level anonymous create envelope = %#v, want business failure", response)
		}
		if response.MessageCode != string(component.MessageCommentAnonymousNotAllowed) {
			t.Fatalf("messageCode = %q, want %q", response.MessageCode, component.MessageCommentAnonymousNotAllowed)
		}
	})
}

// TestAnonPostWindowMasksAuthorHTTPContract 锁定楼层窗口对匿名作者的遮蔽（issue #524）：
// author 恒为 {id:0, username:"匿名同学", avatarUrl:""}，作者本人视角 isOwnPost=true。
func TestAnonPostWindowMasksAuthorHTTPContract(t *testing.T) {
	conn, router := setupAnonPostContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	base := contractTestID()
	topicID, firstPostID := base, base+1
	createContractWikiTopic(t, conn, topicID, firstPostID, user.Id)
	token := contractSessionToken(t, user)
	anonPostID := createContractAnonymousReply(t, router, token, topicID, firstPostID)

	recorder := serveAuthSecurityJSON(router, http.MethodGet, fmt.Sprintf("/api/forum/posts/window?topicId=%d", topicID), "", token)
	if recorder.Code != http.StatusOK {
		t.Fatalf("post window status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	response := decodeContractEnvelope(t, recorder)
	if response.Code != 0 {
		t.Fatalf("post window envelope = %#v, want success", response)
	}
	var window struct {
		Posts []struct {
			ID          uint64 `json:"id"`
			PostNo      uint64 `json:"postNo"`
			IsAnonymous bool   `json:"isAnonymous"`
			Author      struct {
				ID        uint64 `json:"id"`
				Username  string `json:"username"`
				AvatarURL string `json:"avatarUrl"`
			} `json:"author"`
			IsOwnPost bool `json:"isOwnPost"`
		} `json:"posts"`
	}
	if err := json.Unmarshal(response.Result, &window); err != nil {
		t.Fatalf("decode post window result %s: %v", response.Result, err)
	}
	var anon *struct {
		ID          uint64 `json:"id"`
		PostNo      uint64 `json:"postNo"`
		IsAnonymous bool   `json:"isAnonymous"`
		Author      struct {
			ID        uint64 `json:"id"`
			Username  string `json:"username"`
			AvatarURL string `json:"avatarUrl"`
		} `json:"author"`
		IsOwnPost bool `json:"isOwnPost"`
	}
	for i := range window.Posts {
		if window.Posts[i].ID == anonPostID {
			anon = &window.Posts[i]
			break
		}
	}
	if anon == nil {
		t.Fatalf("anonymous post %d not found in window: %#v", anonPostID, window.Posts)
	}
	if !anon.IsAnonymous {
		t.Fatalf("anonymous post isAnonymous = false, want true")
	}
	if anon.Author.ID != 0 || anon.Author.Username != "匿名同学" || anon.Author.AvatarURL != "" {
		t.Fatalf("anonymous post author = %#v, want {id:0 username:匿名同学 avatarUrl:}", anon.Author)
	}
	if !anon.IsOwnPost {
		t.Fatalf("anonymous post isOwnPost = false, want true for the author")
	}
}

// TestAnonPostRevealHTTPContract 锁定匿名楼层作者揭示的权限边界（issue #524）：
// 仅 Admin 可揭示真实身份，普通用户一律 permission.denied。
func TestAnonPostRevealHTTPContract(t *testing.T) {
	t.Run("non-admin is denied", func(t *testing.T) {
		conn, router := setupAnonPostContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		base := contractTestID()
		topicID, firstPostID := base, base+1
		createContractWikiTopic(t, conn, topicID, firstPostID, user.Id)
		token := contractSessionToken(t, user)
		anonPostID := createContractAnonymousReply(t, router, token, topicID, firstPostID)

		recorder := serveJSON(router, "/api/forum/moderation/post-reveal", fmt.Sprintf(`{"postId":%d,"reason":"取证"}`, anonPostID), token)
		if recorder.Code != http.StatusOK {
			t.Fatalf("non-admin reveal status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "permission-denied.json"))
	})

	t.Run("moderator without admin is denied", func(t *testing.T) {
		conn, router := setupAnonPostContractTest(t)
		author := createHTTPContractUser(t, conn, contractTestID())
		base := contractTestID()
		topicID, firstPostID := base, base+1
		createContractWikiTopic(t, conn, topicID, firstPostID, author.Id)
		authorToken := contractSessionToken(t, author)
		anonPostID := createContractAnonymousReply(t, router, authorToken, topicID, firstPostID)

		// 版主权限（TopicsManager）不自动获得身份揭示权：仅 Admin（安全评审缺口补齐）。
		moderator := createHTTPContractUser(t, conn, contractTestID())
		grantContractPermission(t, conn, moderator.Id, permission.TopicsManager)
		moderatorToken := contractSessionToken(t, moderator)

		recorder := serveJSON(router, "/api/forum/moderation/post-reveal", fmt.Sprintf(`{"postId":%d,"reason":"取证"}`, anonPostID), moderatorToken)
		if recorder.Code != http.StatusOK {
			t.Fatalf("moderator reveal status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "permission-denied.json"))
	})

	t.Run("admin reveals the anonymous author", func(t *testing.T) {
		conn, router := setupAnonPostContractTest(t)
		author := createHTTPContractUser(t, conn, contractTestID())
		base := contractTestID()
		topicID, firstPostID := base, base+1
		createContractWikiTopic(t, conn, topicID, firstPostID, author.Id)
		authorToken := contractSessionToken(t, author)
		anonPostID := createContractAnonymousReply(t, router, authorToken, topicID, firstPostID)

		admin := createHTTPContractUser(t, conn, contractTestID())
		grantContractPermission(t, conn, admin.Id, permission.Admin)
		adminToken := contractSessionToken(t, admin)

		recorder := serveJSON(router, "/api/forum/moderation/post-reveal", fmt.Sprintf(`{"postId":%d,"reason":"取证"}`, anonPostID), adminToken)
		if recorder.Code != http.StatusOK {
			t.Fatalf("admin reveal status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		response := decodeContractEnvelope(t, recorder)
		if response.Code != 0 {
			t.Fatalf("admin reveal envelope = %#v, want success", response)
		}
		var reveal struct {
			PostId       uint64 `json:"postId"`
			AuthorUserId uint64 `json:"authorUserId"`
			Username     string `json:"username"`
			Nickname     string `json:"nickname"`
			IsAnonymous  bool   `json:"isAnonymous"`
		}
		if err := json.Unmarshal(response.Result, &reveal); err != nil {
			t.Fatalf("decode reveal result %s: %v", response.Result, err)
		}
		if reveal.PostId != anonPostID || reveal.AuthorUserId != author.Id || reveal.Username != author.Username || !reveal.IsAnonymous {
			t.Fatalf("reveal = %#v, want postId=%d authorUserId=%d username=%q isAnonymous=true",
				reveal, anonPostID, author.Id, author.Username)
		}
	})
}

// TestAnonPostRevisionsMaskEditorHTTPContract 锁定匿名楼层版本历史的编辑者遮蔽
// （issue #524 安全评审）：posts/revisions 是公开（JWTAuth 可选）接口，匿名楼层
// v1 版本的 EditorId 恒为真实作者，若不遮蔽，一行未登录请求即可完成去匿名化。
// 所有版本的 editor 必须恒为匿名占位；正文本身是公开评论内容，照常返回。
func TestAnonPostRevisionsMaskEditorHTTPContract(t *testing.T) {
	conn, router := setupAnonPostContractTest(t)
	author := createHTTPContractUser(t, conn, contractTestID())
	base := contractTestID()
	topicID, firstPostID := base, base+1
	createContractWikiTopic(t, conn, topicID, firstPostID, author.Id)
	token := contractSessionToken(t, author)
	anonPostID := createContractAnonymousReply(t, router, token, topicID, firstPostID)

	recorder := serveAuthSecurityJSON(router, http.MethodGet, fmt.Sprintf("/api/forum/posts/revisions?postId=%d", anonPostID), "", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("anon revisions status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	response := decodeContractEnvelope(t, recorder)
	if response.Code != 0 {
		t.Fatalf("anon revisions envelope = %#v, want success", response)
	}
	var payload struct {
		Versions []struct {
			Version uint64 `json:"version"`
			Editor  struct {
				ID        uint64 `json:"id"`
				Username  string `json:"username"`
				AvatarURL string `json:"avatarUrl"`
			} `json:"editor"`
			Content string `json:"content"`
		} `json:"versions"`
	}
	if err := json.Unmarshal(response.Result, &payload); err != nil {
		t.Fatalf("decode revisions result %s: %v", response.Result, err)
	}
	if len(payload.Versions) == 0 {
		t.Fatalf("anon revisions versions empty: %#v", payload)
	}
	for _, version := range payload.Versions {
		if version.Editor.ID != 0 || version.Editor.Username != "匿名同学" || version.Editor.AvatarURL != "" {
			t.Fatalf("anonymous revision v%d editor = %#v, want {id:0 username:匿名同学 avatarUrl:}",
				version.Version, version.Editor)
		}
		if version.Content == "" {
			t.Fatalf("anonymous revision v%d content must stay readable", version.Version)
		}
	}
}

package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/ratelimit"
	"github.com/leancodebox/GooseForum/app/models/forum/agentInbox"
	"github.com/leancodebox/GooseForum/app/models/forum/agents"
	"github.com/leancodebox/GooseForum/app/models/forum/category"
	"github.com/leancodebox/GooseForum/app/models/forum/dailyStats"
	"github.com/leancodebox/GooseForum/app/models/forum/fileUsage"
	"github.com/leancodebox/GooseForum/app/models/forum/moderators"
	"github.com/leancodebox/GooseForum/app/models/forum/pointsRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/taskQueue"
	"github.com/leancodebox/GooseForum/app/models/forum/topicCategoryIndex"
	"github.com/leancodebox/GooseForum/app/models/forum/topicUserAction"
	"github.com/leancodebox/GooseForum/app/models/forum/topicUserStat"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/userActivities"
	"github.com/leancodebox/GooseForum/app/models/forum/userBadges"
	"github.com/leancodebox/GooseForum/app/models/forum/userPoints"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"github.com/leancodebox/GooseForum/app/service/agentservice"
	"gorm.io/gorm"
)

func setupAgentForumTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ratelimit.Default().ResetAll()
	conn := db.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&userStatistics.Entity{},
		&agents.Entity{},
		&agentInbox.Entity{},
		&taskQueue.Entity{},
		&topics.Entity{},
		&posts.Entity{},
		&category.Entity{},
		&topicCategoryIndex.Entity{},
		&topicUserAction.Entity{},
		&topicUserStat.Entity{},
		&fileUsage.Entity{},
		&dailyStats.Entity{},
		&userActivities.Entity{},
		&userPoints.Entity{},
		&pointsRecord.Entity{},
		&userBadges.Entity{},
		&moderators.Entity{},
	); err != nil {
		t.Fatalf("migrate agent forum tables: %v", err)
	}
	cleanAgentForumTables(conn)
	hotdataserve.ClearRateLimitConfigCache()
	hotdataserve.ClearSecuritySettingsConfigCache()
	hotdataserve.ClearPostingSettingsConfigCache()
	t.Cleanup(func() {
		ratelimit.Default().ResetAll()
		hotdataserve.ClearRateLimitConfigCache()
		hotdataserve.ClearSecuritySettingsConfigCache()
		hotdataserve.ClearPostingSettingsConfigCache()
	})
	return conn
}

// cleanAgentForumTables removes rows created by other tests in this package:
// all route tests share one in-memory SQLite connection.
func cleanAgentForumTables(conn *gorm.DB) {
	conn.Where("1 = 1").Delete(&agentInbox.Entity{})
	conn.Where("1 = 1").Delete(&taskQueue.Entity{})
	conn.Where("1 = 1").Delete(&posts.Entity{})
	conn.Where("1 = 1").Delete(&topicCategoryIndex.Entity{})
	conn.Where("1 = 1").Delete(&topicUserAction.Entity{})
	conn.Where("1 = 1").Delete(&topicUserStat.Entity{})
	conn.Where("1 = 1").Delete(&fileUsage.Entity{})
	conn.Where("1 = 1").Delete(&dailyStats.Entity{})
	conn.Where("1 = 1").Delete(&userActivities.Entity{})
	conn.Where("1 = 1").Delete(&userPoints.Entity{})
	conn.Where("1 = 1").Delete(&pointsRecord.Entity{})
	conn.Where("1 = 1").Delete(&userBadges.Entity{})
	conn.Where("1 = 1").Delete(&moderators.Entity{})
	conn.Where("1 = 1").Delete(&topics.Entity{})
	conn.Where("1 = 1").Delete(&category.Entity{})
	conn.Where("1 = 1").Delete(&agents.Entity{})
	conn.Where("1 = 1").Delete(&userStatistics.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
}

func createAgentForumAgent(t *testing.T, conn *gorm.DB, username string) (uint64, string) {
	t.Helper()
	result, err := agentservice.Create(agentservice.CreateParams{Username: username})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	return result.Agent.UserId, result.Token
}

func createAgentForumCategory(t *testing.T, conn *gorm.DB, id uint64, name string) {
	t.Helper()
	if err := conn.Create(&category.Entity{Id: id, Name: name, Slug: name}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
}

func agentForumRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiRoute(router)
	return router
}

type agentEnvelope struct {
	Result      json.RawMessage `json:"result"`
	Code        int             `json:"code"`
	MessageCode string          `json:"messageCode"`
	Params      map[string]any  `json:"params"`
}

func serveAgentRequest(router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}
func agentRequest(t *testing.T, router http.Handler, method, path, body, token string) (*httptest.ResponseRecorder, agentEnvelope) {
	t.Helper()
	rec := serveAgentRequest(router, method, path, body, token)
	var envelope agentEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return rec, envelope
}

func TestAgentRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiRoute(router)

	registered := map[string]bool{}
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	want := []string{
		http.MethodGet + " /api/v1/agent/me",
		http.MethodGet + " /api/v1/agent/topics",
		http.MethodPost + " /api/v1/agent/topics",
		http.MethodGet + " /api/v1/agent/topics/:topicId/posts",
		http.MethodPost + " /api/v1/agent/topics/:topicId/posts",
		http.MethodGet + " /api/v1/agent/search",
		http.MethodGet + " /api/v1/agent/inbox",
		http.MethodGet + " /api/v1/agent/inbox/:inboxId",
		http.MethodPost + " /api/v1/agent/inbox/:inboxId/read",
		http.MethodPost + " /api/v1/agent/inbox/read-all",
		http.MethodDelete + " /api/v1/agent/inbox/:inboxId",
		http.MethodDelete + " /api/v1/agent/inbox",
	}
	for _, path := range want {
		if !registered[path] {
			t.Errorf("%s was not registered", path)
		}
	}
}

func TestAgentMeReturnsSafeProfile(t *testing.T) {
	conn := setupAgentForumTestDB(t)
	agentID, token := createAgentForumAgent(t, conn, "me-agent")

	rec, envelope := agentRequest(t, agentForumRouter(), http.MethodGet, "/api/v1/agent/me", "", token)
	if rec.Code != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("me failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var me struct {
		AgentId     uint64 `json:"agentId"`
		Username    string `json:"username"`
		TokenPrefix string `json:"tokenPrefix"`
		Enabled     int8   `json:"enabled"`
	}
	if err := json.Unmarshal(envelope.Result, &me); err != nil {
		t.Fatalf("decode me result %s: %v", envelope.Result, err)
	}
	if me.AgentId != agentID || me.Username != "me-agent" || me.TokenPrefix == "" || me.Enabled != 1 {
		t.Fatalf("me = %#v", me)
	}
	if strings.Contains(rec.Body.String(), token) || strings.Contains(rec.Body.String(), "tokenHash") {
		t.Fatal("me response leaks token or hash material")
	}
}

func TestAgentMeWithoutTokenReturns401(t *testing.T) {
	setupAgentForumTestDB(t)
	rec, envelope := agentRequest(t, agentForumRouter(), http.MethodGet, "/api/v1/agent/me", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if envelope.Code != 1 || envelope.MessageCode != "auth.required" {
		t.Fatalf("envelope = %#v", envelope)
	}
}

func TestAgentTopicListShowsPublishedOnly(t *testing.T) {
	conn := setupAgentForumTestDB(t)
	agentID, token := createAgentForumAgent(t, conn, "list-agent")
	createAgentForumCategory(t, conn, 5001, "general")

	now := time.Now().Add(-time.Hour)
	published := topics.Entity{Id: 6001, Title: "Published", UserId: agentID, Status: 1, ProcessStatus: 0, CategoryIds: []uint64{5001}, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&published).Error; err != nil {
		t.Fatalf("create published topic: %v", err)
	}
	draft := topics.Entity{Id: 6002, Title: "Draft", UserId: agentID, Status: 0, ProcessStatus: 0, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&draft).Error; err != nil {
		t.Fatalf("create draft topic: %v", err)
	}
	pending := topics.Entity{Id: 6003, Title: "Pending", UserId: agentID, Status: 1, ProcessStatus: 2, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&pending).Error; err != nil {
		t.Fatalf("create pending topic: %v", err)
	}

	rec, envelope := agentRequest(t, agentForumRouter(), http.MethodGet, "/api/v1/agent/topics", "", token)
	if rec.Code != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("topics failed: %s", rec.Body.String())
	}
	var list struct {
		List []struct {
			Id            uint64 `json:"id"`
			Title         string `json:"title"`
			Status        int8   `json:"status"`
			ProcessStatus int8   `json:"processStatus"`
		} `json:"list"`
		HasNext bool `json:"hasNext"`
	}
	if err := json.Unmarshal(envelope.Result, &list); err != nil {
		t.Fatalf("decode topics result %s: %v", envelope.Result, err)
	}
	if len(list.List) != 1 || list.List[0].Id != 6001 || list.List[0].Title != "Published" {
		t.Fatalf("list = %#v, want only published topic", list.List)
	}
	if list.HasNext {
		t.Fatal("hasNext should be false for a single topic")
	}
}

func TestAgentWriteTopicCreatesPublishedTopicAndFirstPost(t *testing.T) {
	conn := setupAgentForumTestDB(t)
	agentID, token := createAgentForumAgent(t, conn, "writer-agent")
	createAgentForumCategory(t, conn, 5002, "announce")

	body := fmt.Sprintf(`{"title":"Agent topic","content":"Agent topic content is long enough for default posting rules.","categoryId":[5002]}`)
	rec, envelope := agentRequest(t, agentForumRouter(), http.MethodPost, "/api/v1/agent/topics", body, token)
	if rec.Code != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("write topic failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var topicID uint64
	if err := json.Unmarshal(envelope.Result, &topicID); err != nil || topicID == 0 {
		t.Fatalf("write topic result = %s, want topic id: %v", envelope.Result, err)
	}
	topic := topics.Get(topicID)
	if topic.Id == 0 || topic.UserId != agentID || topic.Status != 1 || topic.PostCount != 1 || topic.PostSeq != 1 {
		t.Fatalf("topic = %#v, want published and owned by agent", topic)
	}
	firstPost := posts.Get(topic.FirstPostId)
	if firstPost.Id == 0 || firstPost.UserId != agentID || firstPost.PostNo != 1 {
		t.Fatalf("first post = %#v", firstPost)
	}
}

func TestAgentCreatePostOwnershipAndPostNo(t *testing.T) {
	conn := setupAgentForumTestDB(t)
	agentID, token := createAgentForumAgent(t, conn, "reply-agent")
	createAgentForumCategory(t, conn, 5003, "meta")

	now := time.Now().Add(-time.Hour)
	topic := topics.Entity{Id: 7001, Title: "Reply target", UserId: agentID, Status: 1, PostCount: 1, PostSeq: 1, CategoryIds: []uint64{5003}, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	firstPost := posts.Entity{Id: 7101, TopicId: topic.Id, PostNo: 1, UserId: agentID, Content: "first", CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&firstPost).Error; err != nil {
		t.Fatalf("create first post: %v", err)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topic.Id).Update("first_post_id", firstPost.Id).Error; err != nil {
		t.Fatalf("set first post: %v", err)
	}

	body := `{"topicId":9999,"content":"A reply with enough content for the posting rules.","replyToPostId":7101}`
	rec, envelope := agentRequest(t, agentForumRouter(), http.MethodPost, "/api/v1/agent/topics/7001/posts", body, token)
	if rec.Code != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("create post failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created struct {
		Id     uint64 `json:"id"`
		PostNo uint64 `json:"postNo"`
	}
	if err := json.Unmarshal(envelope.Result, &created); err != nil {
		t.Fatalf("decode create post result %s: %v", envelope.Result, err)
	}
	if created.Id == 0 || created.PostNo != 2 {
		t.Fatalf("created = %#v, want postNo 2", created)
	}
	post := posts.Get(created.Id)
	if post.Id == 0 || post.UserId != agentID || post.TopicId != 7001 || post.ReplyToPostId != 7101 {
		t.Fatalf("post = %#v", post)
	}
	topic = topics.Get(7001)
	if topic.PostCount != 2 || topic.ReplyCount != 1 || topic.PostSeq != 2 {
		t.Fatalf("topic stats = %#v", topic)
	}
}

func TestAgentCreatePostUnknownTopicBusinessError(t *testing.T) {
	conn := setupAgentForumTestDB(t)
	_, token := createAgentForumAgent(t, conn, "missing-agent")

	body := `{"content":"Reply to a topic that does not exist with enough content."}`
	rec, envelope := agentRequest(t, agentForumRouter(), http.MethodPost, "/api/v1/agent/topics/999999/posts", body, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("unknown topic status = %d, want business 200", rec.Code)
	}
	if envelope.Code != 1 || envelope.MessageCode != "topic.notFound" {
		t.Fatalf("envelope = %#v, want topic.notFound", envelope)
	}
}

func TestAgentStrictBindingRejectsMalformedInput(t *testing.T) {
	setupAgentForumTestDB(t)

	t.Run("malformed json body", func(t *testing.T) {
		_, token := createAgentForumAgent(t, db.Connect(), "strict-json-agent")
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodPost, "/api/v1/agent/topics", "{", token)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		if envelope.Code != 1 || envelope.MessageCode != "common.request.parseFailed" {
			t.Fatalf("envelope = %#v, want common.request.parseFailed", envelope)
		}
	})

	t.Run("malformed json reply body", func(t *testing.T) {
		_, token := createAgentForumAgent(t, db.Connect(), "strict-reply-agent")
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodPost, "/api/v1/agent/topics/1/posts", "{", token)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		if envelope.Code != 1 || envelope.MessageCode != "common.request.parseFailed" {
			t.Fatalf("envelope = %#v, want common.request.parseFailed", envelope)
		}
	})

	t.Run("non numeric path topic id", func(t *testing.T) {
		_, token := createAgentForumAgent(t, db.Connect(), "strict-path-agent")
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodGet, "/api/v1/agent/topics/abc/posts", "", token)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		if envelope.Code != 1 || envelope.MessageCode != "common.request.parseFailed" {
			t.Fatalf("envelope = %#v, want common.request.parseFailed", envelope)
		}
	})

	t.Run("invalid category count", func(t *testing.T) {
		_, token := createAgentForumAgent(t, db.Connect(), "strict-category-agent")
		body := `{"title":"t","content":"c","categoryId":[1,2,3,4]}`
		rec, envelope := agentRequest(t, agentForumRouter(), http.MethodPost, "/api/v1/agent/topics", body, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want business 200", rec.Code)
		}
		if envelope.Code != 1 || envelope.MessageCode != "common.request.invalidParams" {
			t.Fatalf("envelope = %#v, want common.request.invalidParams", envelope)
		}
	})
}

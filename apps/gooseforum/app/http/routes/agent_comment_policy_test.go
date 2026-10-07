package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentcommentservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// seedPolicyTopic creates an Agent-owned topic with a first post, matching the
// layout the Agent reply routes expect.
func seedPolicyTopic(t *testing.T, conn *gorm.DB, agentID, topicID, firstPostID, categoryID uint64) {
	t.Helper()
	now := time.Now().Add(-time.Hour)
	topic := topics.Entity{Id: topicID, Title: "Policy target", UserId: agentID, Status: 1, PostCount: 1, PostSeq: 1, CategoryIds: []uint64{categoryID}, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	first := posts.Entity{Id: firstPostID, TopicId: topicID, PostNo: 1, UserId: agentID, Content: "first", CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&first).Error; err != nil {
		t.Fatalf("create first post: %v", err)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("first_post_id", firstPostID).Error; err != nil {
		t.Fatalf("set first post: %v", err)
	}
}

func TestAgentCreatePostBannedTopicRejected(t *testing.T) {
	conn := setupAgentEventsHTTP(t)
	agentID, token := createAgentForumAgent(t, conn, "policy-agent")
	createAgentForumCategory(t, conn, 5010, "policy")
	seedPolicyTopic(t, conn, agentID, 7100, 7110, 5010)
	if err := topics.UpdateAgentCommentDisabled(7100, true); err != nil {
		t.Fatalf("ban topic: %v", err)
	}

	router := agentForumRouter()
	body := `{"content":"A banned reply attempt with enough content for the posting rules."}`
	code, envelope := keyRequest(t, router, "/api/v1/agent/topics/7100/posts", body, token, "policy-banned-1")
	if code != http.StatusOK || envelope.Code != 1 || envelope.MessageCode != "topic.agentCommentDisabled" {
		t.Fatalf("banned topic envelope = %#v status=%d", envelope, code)
	}
	var count int64
	if err := conn.Model(&posts.Entity{}).Where("topic_id = ?", 7100).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("banned write created a post, count=%d", count)
	}

	if err := topics.UpdateAgentCommentDisabled(7100, false); err != nil {
		t.Fatalf("unban topic: %v", err)
	}
	code, envelope = keyRequest(t, router, "/api/v1/agent/topics/7100/posts", body, token, "policy-banned-1")
	if code != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("allowed retry failed: status=%d envelope=%#v", code, envelope)
	}
}

func TestAgentCreatePostGlobalPolicyBlocksNewWritesButReplays(t *testing.T) {
	conn := setupAgentEventsHTTP(t)
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate page config: %v", err)
	}
	conn.Where("page_type = ?", pageConfig.AgentCommentPolicy).Delete(&pageConfig.Entity{})
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.AgentCommentPolicy).Delete(&pageConfig.Entity{})
		hotdataserve.ClearAgentCommentPolicyConfigCache()
	})
	hotdataserve.ClearAgentCommentPolicyConfigCache()

	agentID, token := createAgentForumAgent(t, conn, "policy-global-agent")
	createAgentForumCategory(t, conn, 5011, "policy-global")
	seedPolicyTopic(t, conn, agentID, 7120, 7130, 5011)

	router := agentForumRouter()
	body := `{"content":"A global policy reply with enough content for the posting rules."}`
	code, envelope := keyRequest(t, router, "/api/v1/agent/topics/7120/posts", body, token, "global-policy-1")
	if code != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("initial write failed: status=%d envelope=%#v", code, envelope)
	}
	var created struct {
		Id uint64 `json:"id"`
	}
	if err := json.Unmarshal(envelope.Result, &created); err != nil || created.Id == 0 {
		t.Fatalf("created = %s %v", envelope.Result, err)
	}

	entity := pageConfig.GetByPageType(pageConfig.AgentCommentPolicy)
	entity.PageType = pageConfig.AgentCommentPolicy
	entity.Config = `{"allowAgentComments":false}`
	if affected := pageConfig.CreateOrSave(&entity); affected == 0 {
		t.Fatal("save global policy: no row written")
	}
	hotdataserve.ClearAgentCommentPolicyConfigCache()

	code, envelope = keyRequest(t, router, "/api/v1/agent/topics/7120/posts", body, token, "global-policy-1")
	if code != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("committed replay blocked by the global switch: status=%d envelope=%#v", code, envelope)
	}
	var replay struct {
		Id uint64 `json:"id"`
	}
	if err := json.Unmarshal(envelope.Result, &replay); err != nil || replay.Id != created.Id {
		t.Fatalf("replay = %s, want post %d", envelope.Result, created.Id)
	}

	code, envelope = keyRequest(t, router, "/api/v1/agent/topics/7120/posts", body, token, "global-policy-2")
	if code != http.StatusOK || envelope.Code != 1 || envelope.MessageCode != "topic.agentCommentDisabled" {
		t.Fatalf("global switch envelope = %#v status=%d", envelope, code)
	}
}

func TestAdminAgentCommentPolicyEndpoints(t *testing.T) {
	setupAgentAdminTestDB(t)
	conn := db.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}, &topics.Entity{}); err != nil {
		t.Fatalf("migrate policy tables: %v", err)
	}
	conn.Where("page_type = ?", pageConfig.AgentCommentPolicy).Delete(&pageConfig.Entity{})
	conn.Where("id = ?", 7150).Delete(&topics.Entity{})
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.AgentCommentPolicy).Delete(&pageConfig.Entity{})
		conn.Where("id = ?", 7150).Delete(&topics.Entity{})
		hotdataserve.ClearAgentCommentPolicyConfigCache()
	})
	hotdataserve.ClearAgentCommentPolicyConfigCache()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("api/admin/agent-comment-policy", UpButterReq(api.GetAgentCommentPolicy))
	router.POST("api/admin/save-agent-comment-policy", UpButterReq(api.SaveAgentCommentPolicy))
	router.POST("api/admin/set-agent-comment-topic-policy", UpButterReq(api.SetAgentCommentTopicPolicy))

	readPolicy := func() bool {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/admin/agent-comment-policy", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var resp struct {
			Code   int `json:"code"`
			Result struct {
				AllowAgentComments bool `json:"allowAgentComments"`
			} `json:"result"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode policy %q: %v", rec.Body.String(), err)
		}
		if resp.Code != 0 {
			t.Fatalf("policy read failed: %s", rec.Body.String())
		}
		return resp.Result.AllowAgentComments
	}

	if !readPolicy() {
		t.Fatal("default policy should allow Agent comments")
	}
	if code, resp := postAgent(t, router, "/api/admin/save-agent-comment-policy", `{"allowAgentComments":false}`); code != http.StatusOK || resp["code"] != float64(0) {
		t.Fatalf("save policy failed: %d %#v", code, resp)
	}
	if readPolicy() {
		t.Fatal("saved policy should reject Agent comments")
	}
	if code, resp := postAgent(t, router, "/api/admin/save-agent-comment-policy", `{}`); code != http.StatusOK || resp["messageCode"] != "common.request.invalidParams" {
		t.Fatalf("missing field should fail validation: %d %#v", code, resp)
	}

	topic := topics.Entity{Id: 7150, Title: "Policy admin topic", UserId: 1, Status: 1}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	code, resp := postAgent(t, router, "/api/admin/set-agent-comment-topic-policy", `{"topicId":7150,"disabled":true}`)
	if code != http.StatusOK || resp["code"] != float64(0) {
		t.Fatalf("set topic policy failed: %d %#v", code, resp)
	}
	if got := topics.Get(7150); !got.AgentCommentDisabled {
		t.Fatalf("topic flag not persisted: %#v", got)
	}
	if code, resp := postAgent(t, router, "/api/admin/set-agent-comment-topic-policy", `{"topicId":999999,"disabled":true}`); code != http.StatusOK || resp["messageCode"] != "topic.notFound" {
		t.Fatalf("unknown topic should fail with topic.notFound: %d %#v", code, resp)
	}
}

func TestAgentCommentPolicyRecheckedAtFirstApproval(t *testing.T) {
	for _, scope := range []string{"topic", "global"} {
		t.Run(scope, func(t *testing.T) {
			setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) { o.TextModeration = true })
			conn := setupAgentEventsHTTP(t)
			agentID, _ := createAgentForumAgent(t, conn, "approval_policy_"+scope)
			author := createHTTPContractUser(t, conn, contractTestID())
			base := contractTestID()
			createContractPublishedTopic(t, conn, base, base+1, author.Id)
			topic := topics.Get(base)
			pending := posts.Entity{TopicId: base, UserId: agentID, Content: "A robot reply queued while comments were allowed."}
			if err := publicationservice.Submit(context.Background(), &topic, &pending); err != nil {
				t.Fatal(err)
			}
			if scope == "topic" {
				if err := topics.UpdateAgentCommentDisabled(base, true); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := agentcommentservice.SaveGlobalPolicy(pageConfig.AgentCommentPolicyConfig{AllowAgentComments: false}); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					_ = agentcommentservice.SaveGlobalPolicy(pageConfig.AgentCommentPolicyConfig{AllowAgentComments: true})
				})
			}
			if err := publicationservice.Review(context.Background(), pending.LatestRevisionId, moderationDecision.ActionAllow, "", author.Id); err != nil {
				t.Fatal(err)
			}
			result := posts.Get(pending.Id)
			revision := postRevisions.Get(pending.LatestRevisionId)
			if result.PublishedRevisionId != 0 || revision.ProcessStatus != posts.ProcessStatusBlocked || revision.ReviewReason == "" {
				t.Fatalf("disabled Agent comment became public: post=%+v revision=%+v", result, revision)
			}
			if err := publicationservice.Review(context.Background(), pending.LatestRevisionId, moderationDecision.ActionAllow, "", author.Id); !errors.Is(err, publicationservice.ErrUnavailable) {
				t.Fatalf("retry revived rejected reply: %v", err)
			}
		})
	}
}

func TestAgentAuthorPolicyBlocksNewWritesButAllowsCommittedReplay(t *testing.T) {
	setupHTTPContractTest(t)
	conn := setupAgentEventsHTTP(t)
	_, token := createAgentForumAgent(t, conn, "author-replay-agent")
	author := createHTTPContractUser(t, conn, contractTestID())
	base := contractTestID()
	createContractPublishedTopic(t, conn, base, base+1, author.Id)
	router := agentForumRouter()
	path := fmt.Sprintf("/api/v1/agent/topics/%d/posts", base)
	body := `{"content":"A committed robot reply long enough to pass forum posting requirements."}`
	code, first := keyRequest(t, router, path, body, token, "author-policy-replay")
	if code != http.StatusOK || first.Code != 0 {
		t.Fatalf("initial write: %d %#v", code, first)
	}
	var created struct {
		Id uint64 `json:"id"`
	}
	if err := json.Unmarshal(first.Result, &created); err != nil || created.Id == 0 {
		t.Fatalf("created: %s %v", first.Result, err)
	}
	toggle := serveJSON(router, "/api/forum/topics/agent-replies", fmt.Sprintf(`{"topicId":%d,"agentRepliesDisabled":true}`, base), contractSessionToken(t, author))
	if toggle.Code != http.StatusOK || decodeContractEnvelope(t, toggle).Code != 0 {
		t.Fatalf("owner toggle: %d %s", toggle.Code, toggle.Body.String())
	}
	code, replay := keyRequest(t, router, path, body, token, "author-policy-replay")
	if code != http.StatusOK || replay.Code != 0 {
		t.Fatalf("committed replay blocked: %d %#v", code, replay)
	}
	var repeated struct {
		Id uint64 `json:"id"`
	}
	if err := json.Unmarshal(replay.Result, &repeated); err != nil || repeated.Id != created.Id {
		t.Fatalf("replay: %s, want %d", replay.Result, created.Id)
	}
	code, rejected := keyRequest(t, router, path, body, token, "author-policy-new")
	if code != http.StatusOK || rejected.Code != 1 || rejected.MessageCode != "topic.agentRepliesDisabled" {
		t.Fatalf("new reply: %d %#v", code, rejected)
	}
	var count int64
	if err := conn.Model(&posts.Entity{}).Where("topic_id = ?", base).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("posts = %d, err=%v", count, err)
	}
}

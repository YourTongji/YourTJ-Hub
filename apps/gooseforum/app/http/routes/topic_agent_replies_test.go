package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/postservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/topicpolicyservice"
)

func TestAgentReplyDisabledHTTP(t *testing.T) {
	conn := setupAgentForumTestDB(t)
	_, token := createAgentForumAgent(t, conn, "reply-policy-agent")
	author := createHTTPContractUser(t, conn, contractTestID())
	base := contractTestID()
	createContractPublishedTopic(t, conn, base, base+1, author.Id)
	if err := conn.Table("topics").Where("id = ?", base).Update("agent_replies_disabled", true).Error; err != nil {
		t.Fatal(err)
	}
	r := serveAgentRequest(agentForumRouter(), http.MethodPost, fmt.Sprintf("/api/v1/agent/topics/%d/posts", base), `{"content":"A sufficiently long robot reply to the topic."}`, token)
	e := decodeContractEnvelope(t, r)
	if e.Code != 1 || e.MessageCode != "topic.agentRepliesDisabled" {
		t.Fatalf("disabled reply = %d %s", r.Code, r.Body.String())
	}
	assertFixtureEnvelope(t, e, contractFixture(t, "topic-agent-replies-disabled.json"))
}

func TestTopicAgentRepliesSettingHTTP(t *testing.T) {
	setupHTTPContractTest(t)
	conn := setupAgentForumTestDB(t)
	author := createHTTPContractUser(t, conn, contractTestID())
	stranger := createHTTPContractUser(t, conn, contractTestID())
	base := contractTestID()
	createContractPublishedTopic(t, conn, base, base+1, author.Id)
	router := agentForumRouter()
	body := fmt.Sprintf(`{"topicId":%d,"agentRepliesDisabled":true}`, base)
	for _, tc := range []struct {
		name, token, body, code string
		status                  int
	}{
		{"guest", "", body, "auth.required", 401},
		{"stranger", contractSessionToken(t, stranger), body, "topic.operationDenied", 200},
		{"missing bool", contractSessionToken(t, author), fmt.Sprintf(`{"topicId":%d}`, base), "common.request.invalidParams", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := serveJSON(router, "/api/forum/topics/agent-replies", tc.body, tc.token)
			if r.Code != tc.status || decodeContractEnvelope(t, r).MessageCode != tc.code {
				t.Fatalf("%d %s", r.Code, r.Body.String())
			}
		})
	}
	authorToken := contractSessionToken(t, author)
	for name, changes := range map[string]map[string]any{
		"wiki":    {"topic_type": topics.TopicTypeWiki},
		"deleted": {"visibility_status": topics.VisibilityUserDeleted},
	} {
		t.Run(name, func(t *testing.T) {
			if err := conn.Model(&topics.Entity{}).Where("id = ?", base).Updates(changes).Error; err != nil {
				t.Fatal(err)
			}
			r := serveJSON(router, "/api/forum/topics/agent-replies", body, authorToken)
			if decodeContractEnvelope(t, r).MessageCode != "topic.operationDenied" {
				t.Fatalf("%s", r.Body.String())
			}
			if err := conn.Model(&topics.Entity{}).Where("id = ?", base).Updates(map[string]any{"topic_type": topics.TopicTypeForum, "visibility_status": topics.VisibilityActive}).Error; err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, disabled := range []bool{true, true, false, true} {
		r := serveJSON(router, "/api/forum/topics/agent-replies", fmt.Sprintf(`{"topicId":%d,"agentRepliesDisabled":%t}`, base, disabled), authorToken)
		assertFixtureEnvelope(t, decodeContractEnvelope(t, r), contractFixture(t, "result-true.json"))
		if topics.Get(base).AgentRepliesDisabled != disabled {
			t.Fatal("setting not saved")
		}
	}
	// Ordinary human replies remain allowed, and content edits must not reset it.
	r := serveJSON(router, "/api/forum/posts/create", fmt.Sprintf(`{"topicId":%d,"content":"A human reply that satisfies the posting length."}`, base), authorToken)
	if decodeContractEnvelope(t, r).Code != 0 {
		t.Fatalf("human reply: %s", r.Body.String())
	}
	r = serveJSON(router, "/api/forum/topics/write", fmt.Sprintf(`{"topicId":%d,"title":"An edited title","content":"A sufficiently long edited topic body.","categoryId":[1],"topicStatus":1,"contentType":3}`, base), authorToken)
	if decodeContractEnvelope(t, r).Code != 0 || !topics.Get(base).AgentRepliesDisabled {
		t.Fatalf("content edit: %s", r.Body.String())
	}
	// New-topic initial choice is stored atomically with the content.
	r = serveJSON(router, "/api/forum/topics/write", `{"title":"A new topic","content":"A sufficiently long new topic body.","categoryId":[1],"topicStatus":1,"contentType":3,"agentRepliesDisabled":true}`, authorToken)
	e := decodeContractEnvelope(t, r)
	var id uint64
	if err := json.Unmarshal(e.Result, &id); err != nil || e.Code != 0 || !topics.Get(id).AgentRepliesDisabled {
		t.Fatalf("creation: %s", r.Body.String())
	}
}

func TestAgentReplyPolicyAtTransactionAndReview(t *testing.T) {
	setupHTTPContractTest(t)
	conn := setupAgentForumTestDB(t)
	if err := conn.AutoMigrate(&moderationDecision.Entity{}, &eventNotification.Entity{}); err != nil {
		t.Fatal(err)
	}
	agentID, _ := createAgentForumAgent(t, conn, "transaction-policy-agent")
	author := createHTTPContractUser(t, conn, contractTestID())
	base := contractTestID()
	createContractPublishedTopic(t, conn, base, base+1, author.Id)
	topic := topics.Get(base)
	published := &posts.Entity{TopicId: base, UserId: agentID, Content: "An existing public robot reply."}
	if err := postservice.CreateTopicPost(published, topic); err != nil {
		t.Fatal(err)
	}
	pending := &posts.Entity{TopicId: base, UserId: agentID, Content: "A pending robot reply."}
	if err := publicationservice.Submit(context.Background(), &topic, pending); err != nil {
		t.Fatal(err)
	}
	if _, err := topicpolicyservice.SetAgentRepliesDisabled(context.Background(), base, author.Id, true); err != nil {
		t.Fatal(err)
	}
	// Passing the old topic snapshot cannot bypass either transaction path.
	for name, write := range map[string]func(*posts.Entity) error{
		"direct":  func(p *posts.Entity) error { return postservice.CreateTopicPost(p, topic) },
		"pending": func(p *posts.Entity) error { return publicationservice.Submit(context.Background(), &topic, p) },
	} {
		t.Run(name, func(t *testing.T) {
			p := &posts.Entity{TopicId: base, UserId: agentID, Content: "A blocked robot reply."}
			if err := write(p); !errors.Is(err, topicpolicyservice.ErrAgentRepliesDisabled) {
				t.Fatalf("write = %v", err)
			}
		})
	}
	if err := publicationservice.Review(context.Background(), pending.LatestRevisionId, moderationDecision.ActionAllow, "", author.Id); err != nil {
		t.Fatal(err)
	}
	// Existing public robot replies may still be edited through delayed review.
	published.Content = "An edited existing public robot reply."
	if err := publicationservice.Submit(context.Background(), &topic, published); err != nil {
		t.Fatal(err)
	}
	if err := publicationservice.Review(context.Background(), published.LatestRevisionId, moderationDecision.ActionAllow, "", author.Id); err != nil {
		t.Fatal(err)
	}
	if posts.Get(published.Id).Content != published.Content {
		t.Fatal("published bot edit was blocked")
	}
	r := postRevisions.Get(pending.LatestRevisionId)
	if r.ProcessStatus != posts.ProcessStatusBlocked || r.ReviewReason == "" || posts.Get(pending.Id).PublishedRevisionId != 0 {
		t.Fatalf("revoked publication: %+v", r)
	}
	if err := publicationservice.Review(context.Background(), pending.LatestRevisionId, moderationDecision.ActionAllow, "", 0); !errors.Is(err, publicationservice.ErrUnavailable) {
		t.Fatalf("retry = %v", err)
	}
}

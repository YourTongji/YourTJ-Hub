package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
	"gorm.io/gorm"
)

func TestAgentAsyncSubmissionReplaysOnePendingWrite(t *testing.T) {
	for _, kind := range []string{"topic", "post", "source_post"} {
		t.Run(kind, func(t *testing.T) {
			setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) { o.TextModeration = true })
			conn := setupAgentEventsHTTP(t)
			agentID, token := createAgentForumAgent(t, conn, "async_writer_"+kind)
			createAgentForumCategory(t, conn, 57001, "async_agent")
			path := "/api/v1/agent/topics"
			targetTopicID := uint64(57010)
			if kind == "source_post" {
				targetTopicID = 57110
			}
			body := `{"title":"Agent pending topic","content":"This is an Agent submission waiting for asynchronous moderation.","categoryId":[57001],"topicStatus":1}`
			if kind != "topic" {
				seedPolicyTopic(t, conn, agentID, targetTopicID, targetTopicID+1, 57001)
				path = fmt.Sprintf("/api/v1/agent/topics/%d/posts", targetTopicID)
				body = `{"content":"This is an Agent reply waiting for asynchronous moderation."}`
			}

			if kind == "source_post" {
				author := createHTTPContractUser(t, conn, contractTestID())
				if err := conn.Model(&topics.Entity{}).Where("id = ?", targetTopicID).Update("user_id", author.Id).Error; err != nil {
					t.Fatal(err)
				}
				if err := conn.Model(&posts.Entity{}).Where("id = ?", targetTopicID+1).Update("user_id", author.Id).Error; err != nil {
					t.Fatal(err)
				}
				now := time.Now().Add(-time.Minute)
				if err := agents.UpdateColumns(conn, agentID, map[string]any{"events_enabled": true, "events_enabled_at": now, "subscription_generation": 1}); err != nil {
					t.Fatal(err)
				}
				event := agentEvents.Entity{ID: "pending-source", InstanceID: "http-agent-events", AgentID: agentID, Seq: 1, SourceIntentID: "source-intent", SubscriptionGeneration: 1, Type: "forum.topic_created", TopicID: targetTopicID, PostID: targetTopicID + 1, PostNo: 1, ActorID: author.Id, ActorType: "human", Reasons: []string{"topic_created"}, ExpiresAt: time.Now().Add(time.Hour)}
				if err := conn.Create(&event).Error; err != nil {
					t.Fatal(err)
				}
				body = `{"content":"This source-linked reply must remain idempotent while pending.","sourceEventId":"pending-source"}`
			}
			router := agentForumRouter()
			var first json.RawMessage
			for n := 0; n < 2; n++ {
				status, e := keyRequest(t, router, path, body, token, "async-pending")
				if status != 200 || e.Code != 0 || e.MessageCode != "content.moderation.checking" {
					t.Fatalf("pending write %d: status=%d %#v", n, status, e)
				}
				if n == 0 {
					first = e.Result
					if kind != "topic" {
						if err := topics.UpdateAgentCommentDisabled(targetTopicID, true); err != nil {
							t.Fatal(err)
						}
					}
					if kind == "source_post" {
						var result struct {
							Id uint64 `json:"id"`
						}
						if err := json.Unmarshal(e.Result, &result); err != nil {
							t.Fatal(err)
						}
						if got := posts.Get(result.Id); got.AgentEventDepth != 1 || got.ProcessStatus != posts.ProcessStatusPending {
							t.Fatalf("pending source metadata: %#v", got)
						}
					}
				} else if string(first) != string(e.Result) {
					t.Fatalf("pending replay created a new resource: %s -> %s", first, e.Result)
				}
			}
		})
	}
}

func TestAsyncApprovalCommitsAgentIntent(t *testing.T) {
	setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) { o.TextModeration = true })
	conn := setupAgentEventsHTTP(t)
	agentID, _ := createAgentForumAgent(t, conn, "async_listener")
	now := time.Now().Add(-time.Minute)
	if err := agents.UpdateColumns(conn, agentID, map[string]any{"events_enabled": true, "events_enabled_at": now, "subscription_generation": 1, "event_types": `["agent.mentioned"]`}); err != nil {
		t.Fatal(err)
	}
	author := createHTTPContractUser(t, conn, contractTestID())
	topic := topics.Entity{UserId: author.Id, Title: "Pending mention", Status: 1, CategoryIds: []uint64{1}}
	post := posts.Entity{UserId: author.Id, Content: "Hello @async_listener, please respond after approval."}
	if err := publicationservice.Submit(context.Background(), &topic, &post); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("injected intent persistence failure")
	const hook = "test:agent-intent-failure"
	if err := conn.Callback().Create().Before("gorm:create").Register(hook, func(tx *gorm.DB) {
		if tx.Statement.Table == "agent_interaction_intents" {
			_ = tx.AddError(failure)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Create().Remove(hook) })
	if err := publicationservice.Review(context.Background(), post.LatestRevisionId, moderationDecision.ActionAllow, "", 0); !errors.Is(err, failure) {
		t.Fatalf("approval ignored Agent persistence failure: %v", err)
	}
	if got := posts.Get(post.Id); got.ProcessStatus != posts.ProcessStatusPending || got.PublishedRevisionId != 0 {
		t.Fatalf("approval escaped transaction rollback: %#v", got)
	}
	if err := conn.Callback().Create().Remove(hook); err != nil {
		t.Fatal(err)
	}
	if err := publicationservice.Review(context.Background(), post.LatestRevisionId, moderationDecision.ActionAllow, "", 0); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := conn.Model(&agentEvents.Intent{}).Where("post_id = ?", post.Id).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("approval committed %d Agent intents, want 1 before effects run", count)
	}
}

func TestAsyncFirstDraftPublicationNotMistakenForPublicBaseline(t *testing.T) {
	for _, wasPublic := range []bool{false, true} {
		t.Run(fmt.Sprintf("previouslyPublic=%t", wasPublic), func(t *testing.T) {
			setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) { o.TextModeration = true })
			conn := setupAgentEventsHTTP(t)
			name := "draft_listener"
			if wasPublic {
				name = "legacy_listener"
			}
			listener, _ := createAgentForumAgent(t, conn, name)
			now := time.Now().Add(-time.Minute)
			if err := agents.UpdateColumns(conn, listener, map[string]any{"events_enabled": true, "events_enabled_at": now, "subscription_generation": 1, "event_types": `["agent.mentioned","forum.topic_created"]`}); err != nil {
				t.Fatal(err)
			}
			author := createHTTPContractUser(t, conn, contractTestID())
			status := int8(0)
			if wasPublic {
				status = 1
			}
			topic := topics.Entity{UserId: author.Id, Title: "Existing draft or public topic", Status: status, CategoryIds: []uint64{1}, PostSeq: 1}
			if err := conn.Create(&topic).Error; err != nil {
				t.Fatal(err)
			}
			post := posts.Entity{TopicId: topic.Id, PostNo: 1, UserId: author.Id, Content: "Original content @" + name}
			if err := conn.Create(&post).Error; err != nil {
				t.Fatal(err)
			}
			if err := conn.Create(&postRevisions.Entity{PostId: post.Id, Version: 1, EditorId: author.Id, Content: post.Content}).Error; err != nil {
				t.Fatal(err)
			}
			topic.FirstPostId = post.Id
			if err := conn.Save(&topic).Error; err != nil {
				t.Fatal(err)
			}
			topic.Status = 1
			post.Content = "Publish the candidate with original mention @" + name
			if err := publicationservice.Submit(context.Background(), &topic, &post); err != nil {
				t.Fatal(err)
			}
			if err := publicationservice.Review(context.Background(), post.LatestRevisionId, moderationDecision.ActionAllow, "", 0); err != nil {
				t.Fatal(err)
			}
			var intents []agentEvents.Intent
			if err := conn.Where("post_id = ?", post.Id).Find(&intents).Error; err != nil {
				t.Fatal(err)
			}
			if wasPublic {
				if len(intents) != 0 {
					t.Fatalf("legacy public edit rebroadcast original occurrence: %#v", intents)
				}
			} else {
				if len(intents) != 1 || len(intents[0].Recipients) != 1 {
					t.Fatalf("first draft publication lost its recipients: %#v", intents)
				}
				reasons := intents[0].Recipients[0].Reasons
				if len(reasons) != 2 || reasons[0] != "mention" || reasons[1] != "topic_created" {
					t.Fatalf("first draft publication reasons: %#v", reasons)
				}
			}
		})
	}
}

package routes

import (
	"fmt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agenteventservice"
	"testing"
	"time"
)

func TestWriteTopicDraftPermanentlyWithdrawsAgentEvent(t *testing.T) {
	setupHTTPContractTest(t)
	conn := setupAgentEventsHTTP(t)
	author := createHTTPContractUser(t, conn, contractTestID())
	agentID, _ := createAgentForumAgent(t, conn, "topic-draft-bot")
	id := contractTestID()
	createContractPublishedTopic(t, conn, id, id+1, author.Id)
	event := agentEvents.Entity{ID: "draft-transition", InstanceID: "http-agent-events", AgentID: agentID, Seq: 1, SourceIntentID: "draft-source", SubscriptionGeneration: 1, Type: "agent.mentioned", TopicID: id, PostID: id + 1, PostNo: 1, ActorID: author.Id, ActorType: "human", Reasons: []string{"mention"}, ExpiresAt: time.Now().Add(time.Hour)}
	if err := conn.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	router := agentForumRouter()
	for _, status := range []int{0, 1} {
		body := fmt.Sprintf(`{"topicId":%d,"title":"A discussion hidden then republished","content":"A sufficiently long edited public discussion.","categoryId":[1],"topicStatus":%d,"contentType":3}`, id, status)
		response := serveJSON(router, "/api/forum/topics/write", body, contractSessionToken(t, author))
		if envelope := decodeContractEnvelope(t, response); envelope.Code != 0 {
			t.Fatalf("write status %d: %s", status, response.Body.String())
		}
	}
	if err := conn.Where("id = ?", event.ID).Take(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.WithdrawnAt == nil || event.ActorID != 0 || event.PostID != 0 {
		t.Fatalf("old event revived after draft/republish: %#v", event)
	}
	var post posts.Entity
	if err := conn.First(&post, id+1).Error; err != nil {
		t.Fatal(err)
	}
	if post.ProcessStatus != posts.ProcessStatusNormal {
		t.Fatal("republished first post did not stay public")
	}
	if envelope := agenteventservice.Envelope(event); envelope.State != "withdrawn" || envelope.Data != nil {
		t.Fatalf("old envelope = %#v", envelope)
	}
}

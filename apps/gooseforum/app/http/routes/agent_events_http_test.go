package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentWrites"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agenteventservice"
	"gorm.io/gorm"
)

func setupAgentEventsHTTP(t *testing.T) *gorm.DB {
	t.Helper()
	conn := setupAgentForumTestDB(t)
	t.Setenv("YOURTJ_AGENT_INSTANCE_ID", "http-agent-events")
	t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "http-epoch-1")
	t.Setenv("YOURTJ_AGENT_API_ENABLED", "true")
	t.Setenv("YOURTJ_AGENT_PRODUCER_ENABLED", "true")
	t.Setenv("YOURTJ_AGENT_WEBHOOK_ENABLED", "false")
	if err := conn.AutoMigrate(&agentEvents.Entity{}, &agentEvents.Intent{}, &agentEvents.Publication{}, &agentEvents.ReplayState{}, &agentWrites.Entry{}, &users.BlockEntity{}, &taskQueue.Entity{}); err != nil {
		t.Fatal(err)
	}
	for _, value := range []any{&agentEvents.Entity{}, &agentEvents.Intent{}, &agentEvents.Publication{}, &agentEvents.ReplayState{}, &agentWrites.Entry{}, &taskQueue.Entity{}, &users.BlockEntity{}} {
		if err := conn.Where("1 = 1").Delete(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, value := range []any{&agentEvents.Entity{}, &agentEvents.Intent{}, &agentEvents.Publication{}, &agentEvents.ReplayState{}, &agentWrites.Entry{}} {
			conn.Where("1 = 1").Delete(value)
		}
	})
	return conn
}
func keyRequest(t *testing.T, router http.Handler, path, body, token, key string) (int, agentEnvelope) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", key)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var envelope agentEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return rec.Code, envelope
}
func TestAgentEventHTTPInboxSourceReplyAndRevocation(t *testing.T) {
	conn := setupAgentEventsHTTP(t)
	agentID, token := createAgentForumAgent(t, conn, "event_http_bot")
	otherID, otherToken := createAgentForumAgent(t, conn, "event_http_other")
	_ = otherID
	now := time.Now().Add(-time.Minute)
	if err := agents.UpdateColumns(conn, agentID, map[string]any{"events_enabled": true, "subscription_generation": 1, "events_enabled_at": now, "event_types": `["agent.mentioned","agent.post_replied","agent.topic_commented"]`}); err != nil {
		t.Fatal(err)
	}
	human := users.EntityComplete{Username: "event_http_human", Email: "event-http-human@example.com"}
	if err := conn.Create(&human).Error; err != nil {
		t.Fatal(err)
	}
	createAgentForumCategory(t, conn, 55001, "events_http")
	topic := topics.Entity{Title: "Human discussion", UserId: human.Id, Status: 1, PostCount: 1, PostSeq: 1, CategoryIds: []uint64{55001}}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	source := posts.Entity{TopicId: topic.Id, UserId: human.Id, PostNo: 1, Content: "Please help @event_http_bot"}
	if err := conn.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&topic).Update("first_post_id", source.Id).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&postRevisions.Entity{PostId: source.Id, Version: 1, Content: source.Content}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { return agenteventservice.CapturePublicTx(tx, &source) }); err != nil {
		t.Fatal(err)
	}
	var intent agentEvents.Intent
	if err := conn.Where("post_id = ?", source.Id).Take(&intent).Error; err != nil {
		t.Fatal(err)
	}
	task, claimed, err := taskQueue.ClaimTask(intent.TaskID)
	if err != nil || !claimed {
		t.Fatalf("claim %t %v", claimed, err)
	}
	if err := agenteventservice.HandleTask(context.Background(), &task); err != nil {
		t.Fatal(err)
	}
	router := agentForumRouter()
	rec, envelope := agentRequest(t, router, "GET", "/api/v1/agent/events?limit=100", "", token)
	if rec.Code != 200 || envelope.Code != 0 {
		t.Fatalf("list %d %#v", rec.Code, envelope)
	}
	var page agenteventservice.Page
	if err := json.Unmarshal(envelope.Result, &page); err != nil || len(page.Events) != 1 {
		t.Fatalf("inbox %s %v", envelope.Result, err)
	}
	event := page.Events[0]
	if event.Data == nil || event.Data.PostID != source.Id || event.AckedAt != nil {
		t.Fatalf("event %#v", event)
	}
	for _, path := range []string{"/api/v1/agent/events/" + event.ID, "/api/v1/agent/events?after=" + page.NextCursor} {
		_, foreign := agentRequest(t, router, "GET", path, "", otherToken)
		if foreign.Code != 1 || foreign.MessageCode != "agent.events.inaccessible" {
			t.Fatalf("foreign access %#v", foreign)
		}
	}
	replyBody := fmt.Sprintf(`{"content":"A response with enough content for normal posting rules.","replyToPostId":%d,"sourceEventId":%q}`, source.Id, event.ID)
	path := fmt.Sprintf("/api/v1/agent/topics/%d/posts", topic.Id)
	status, first := keyRequest(t, router, path, replyBody, token, "reply:"+event.ID)
	if status != 200 || first.Code != 0 {
		t.Fatalf("reply %d %#v", status, first)
	}
	status, repeated := keyRequest(t, router, path, replyBody, token, "reply:"+event.ID)
	if status != 200 || repeated.Code != 0 {
		t.Fatalf("replay %d %#v", status, repeated)
	}
	var one, two struct {
		ID uint64 `json:"id"`
	}
	_ = json.Unmarshal(first.Result, &one)
	_ = json.Unmarshal(repeated.Result, &two)
	if one.ID == 0 || one.ID != two.ID {
		t.Fatalf("repeat created different result: %s / %s", first.Result, repeated.Result)
	}
	var count int64
	conn.Model(&posts.Entity{}).Where("topic_id = ? AND user_id = ?", topic.Id, agentID).Count(&count)
	if count != 1 {
		t.Fatalf("duplicate response rows %d", count)
	}
	_, ack := agentRequest(t, router, "POST", "/api/v1/agent/events/ack", fmt.Sprintf(`{"eventIds":[%q]}`, event.ID), token)
	if ack.Code != 0 {
		t.Fatalf("ACK %#v", ack)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&source).Update("is_anonymous", true).Error; err != nil {
			return err
		}
		return agenteventservice.WithdrawContentTx(tx, topic.Id, source.Id)
	}); err != nil {
		t.Fatal(err)
	}
	_, hidden := agentRequest(t, router, "GET", "/api/v1/agent/events/"+event.ID, "", token)
	var tombstone agenteventservice.Event
	if hidden.Code != 0 {
		t.Fatalf("tombstone failure %#v", hidden)
	}
	if err := json.Unmarshal(hidden.Result, &tombstone); err != nil || tombstone.State != "withdrawn" || tombstone.Data != nil {
		t.Fatalf("tombstone leak %s %v", hidden.Result, err)
	}
	_, denied := keyRequest(t, router, path, replyBody, token, "new-after-withdrawal")
	if denied.Code != 1 || denied.MessageCode != "agent.events.inaccessible" {
		t.Fatalf("withdrawn source reply accepted %#v", denied)
	}
	t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "http-epoch-2")
	_, reset := agentRequest(t, router, "GET", "/api/v1/agent/events?after="+page.NextCursor, "", token)
	if reset.MessageCode != "agent.events.cursor_reset" {
		t.Fatalf("old epoch %#v", reset)
	}
}
func TestAgentWriteHTTPIdempotencyConflictAndIndependentScope(t *testing.T) {
	conn := setupAgentEventsHTTP(t)
	_, token := createAgentForumAgent(t, conn, "write_idempotent_bot")
	_, other := createAgentForumAgent(t, conn, "write_independent_bot")
	createAgentForumCategory(t, conn, 55002, "idempotent_http")
	router := agentForumRouter()
	body := `{"title":"Daily announcement","content":"Daily content with enough text to pass the normal posting rules.","categoryId":[55002]}`
	status, first := keyRequest(t, router, "/api/v1/agent/topics", body, token, "daily:job:2026-10-04")
	if status != 200 || first.Code != 0 {
		t.Fatalf("topic create %d %#v", status, first)
	}
	// Rate limiting remains active for every request; simulate retry after its window.
	ratelimit.Default().ResetAll()
	_, again := keyRequest(t, router, "/api/v1/agent/topics", body, token, "daily:job:2026-10-04")
	if again.Code != 0 || string(first.Result) != string(again.Result) {
		t.Fatalf("topic repeat %#v / %#v", first, again)
	}
	ratelimit.Default().ResetAll()
	status, conflict := keyRequest(t, router, "/api/v1/agent/topics", strings.Replace(body, "Daily announcement", "Different announcement", 1), token, "daily:job:2026-10-04")
	if status != 409 || conflict.MessageCode != "agent.write.idempotencyConflict" {
		t.Fatalf("digest conflict %d %#v", status, conflict)
	}
	ratelimit.Default().ResetAll()
	_, independent := keyRequest(t, router, "/api/v1/agent/topics", body, other, "daily:job:2026-10-04")
	if independent.Code != 0 || string(independent.Result) == string(first.Result) {
		t.Fatalf("cross Agent scope %#v", independent)
	}
}

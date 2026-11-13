package mcpservice

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentWrites"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agenteventservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/topicpolicyservice"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"
)

// Drive the runner boundary through actual MCP transports and registered tools.
func TestMCPInteractionReplyReplayACKAndRevocation(t *testing.T) {
	t.Setenv("YOURTJ_AGENT_INSTANCE_ID", "mcp-loop")
	t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "mcp-loop-epoch")
	t.Setenv("YOURTJ_AGENT_API_ENABLED", "true")
	t.Setenv("YOURTJ_AGENT_PRODUCER_ENABLED", "true")
	t.Setenv("YOURTJ_AGENT_WEBHOOK_ENABLED", "false")
	conn := setupMCPServiceTestDB(t)
	for _, entity := range []any{&agentEvents.Entity{}, &agentEvents.Intent{}, &agentEvents.Publication{}, &agentEvents.ReplayState{}, &agentWrites.Entry{}, &users.BlockEntity{}} {
		if err := conn.AutoMigrate(entity); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Where("1 = 1").Delete(entity) })
	}
	id, _ := createMCPServiceAgent(t, "mcp_loop_bot")
	if err := agents.UpdateColumns(conn, id, map[string]any{"events_enabled": true, "subscription_generation": 1, "events_enabled_at": time.Now().Add(-time.Minute), "event_types": `["agent.mentioned"]`}); err != nil {
		t.Fatal(err)
	}
	human := users.EntityComplete{Username: "mcp_loop_human", Email: "mcp-loop@example.invalid"}
	if err := conn.Create(&human).Error; err != nil {
		t.Fatal(err)
	}
	cat := category.Entity{Id: 95001, Name: "MCP loop", Slug: "mcp-loop"}
	if err := conn.Create(&cat).Error; err != nil {
		t.Fatal(err)
	}
	topic := topics.Entity{Title: "MCP interaction", UserId: human.Id, Status: 1, PostCount: 1, PostSeq: 1, CategoryIds: []uint64{cat.Id}}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	source := posts.Entity{TopicId: topic.Id, UserId: human.Id, PostNo: 1, Content: "Please answer @mcp_loop_bot"}
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
	if err := conn.Where("instance_id = ? AND post_id = ?", "mcp-loop", source.Id).Take(&intent).Error; err != nil {
		t.Fatal(err)
	}
	task, claimed, err := taskQueue.ClaimTask(intent.TaskID)
	if err != nil || !claimed {
		t.Fatalf("claim %t %v", claimed, err)
	}
	if err := agenteventservice.HandleTask(context.Background(), &task); err != nil {
		t.Fatal(err)
	}
	cs := connectMCPServer(t, true, id)
	call := func(name string, args map[string]any) map[string]any {
		t.Helper()
		res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if res.IsError {
			t.Fatalf("%s: %s", name, mustJSON(res.Content))
		}
		out, ok := res.StructuredContent.(map[string]any)
		if !ok {
			t.Fatalf("%s content %T", name, res.StructuredContent)
		}
		return out
	}
	page := call("list_events", map[string]any{"limit": 100})
	events, ok := page["events"].([]any)
	if !ok || len(events) != 1 {
		t.Fatalf("page %s", mustJSON(page))
	}
	event := events[0].(map[string]any)
	eventID := asString(event["id"])
	current := call("get_event", map[string]any{"eventId": eventID})
	if current["state"] != "active" {
		t.Fatalf("current %s", mustJSON(current))
	}
	args := map[string]any{"topicId": topic.Id, "replyToPostId": source.Id, "content": "A deterministic response long enough to pass forum posting requirements.", "idempotencyKey": "reply:" + eventID, "sourceEventId": eventID}
	first := call("create_post", args)
	ratelimit.Default().ResetAll()
	replay := call("create_post", args)
	if asUint(first["id"]) == 0 || asUint(first["id"]) != asUint(replay["id"]) {
		t.Fatalf("reply/replay %s / %s", mustJSON(first), mustJSON(replay))
	}

	// The topic owner can stop new replies without invalidating a committed write.
	if _, err := topicpolicyservice.SetAgentRepliesDisabled(context.Background(), topic.Id, human.Id, true); err != nil {
		t.Fatal(err)
	}
	ratelimit.Default().ResetAll()
	replay = call("create_post", args)
	if asUint(replay["id"]) != asUint(first["id"]) {
		t.Fatalf("owner-policy replay %s", mustJSON(replay))
	}
	blockedArgs := make(map[string]any, len(args))
	for key, value := range args {
		blockedArgs[key] = value
	}
	blockedArgs["idempotencyKey"] = "new-reply:" + eventID
	ratelimit.Default().ResetAll()
	blocked, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "create_post", Arguments: blockedArgs})
	if err != nil || !blocked.IsError || !strings.Contains(mustJSON(blocked.Content), "topic.agentRepliesDisabled") {
		t.Fatalf("new reply after owner policy: %+v %v", blocked, err)
	}
	var count int64
	if err := conn.Model(&posts.Entity{}).Where("topic_id = ? AND user_id = ?", topic.Id, id).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("response rows %d %v", count, err)
	}
	call("ack_events", map[string]any{"eventIds": []any{eventID}})
	var stored agentEvents.Entity
	if err := conn.Where("id = ?", eventID).Take(&stored).Error; err != nil || stored.AckedAt == nil {
		t.Fatalf("ACK missing %v", err)
	}
	if _, err := agentservice.RotateToken(id); err != nil {
		t.Fatal(err)
	}
	revoked, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "get_event", Arguments: map[string]any{"eventId": eventID}})
	if err == nil && !revoked.IsError {
		t.Fatal("open MCP session survived token rotation")
	}
}

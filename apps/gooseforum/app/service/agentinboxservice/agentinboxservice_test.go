package agentinboxservice

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/agentInbox"
	"github.com/leancodebox/GooseForum/app/models/forum/agents"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/taskQueue"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/agentservice"
	"github.com/leancodebox/GooseForum/app/service/agentwebhookservice"
	"gorm.io/gorm"
)

func setupMentionTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&userStatistics.Entity{},
		&agents.Entity{},
		&topics.Entity{},
		&posts.Entity{},
		&agentInbox.Entity{},
		&taskQueue.Entity{},
	); err != nil {
		t.Fatalf("migrate mention tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&agentInbox.Entity{})
	conn.Where("1 = 1").Delete(&taskQueue.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&posts.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&topics.Entity{})
	conn.Where("1 = 1").Delete(&agents.Entity{})
	conn.Where("1 = 1").Delete(&userStatistics.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
	for i := uint64(1); i <= 8; i++ {
		topicID := uint64(100) + i
		postID := uint64(1000) + i
		if err := conn.Create(&topics.Entity{
			Id:            topicID,
			Title:         "published topic",
			Status:        1,
			ProcessStatus: topics.ProcessStatusNormal,
			FirstPostId:   postID,
		}).Error; err != nil {
			t.Fatalf("seed topic %d: %v", topicID, err)
		}
		if err := conn.Create(&posts.Entity{
			Id:            postID,
			TopicId:       topicID,
			PostNo:        1,
			ProcessStatus: posts.ProcessStatusNormal,
		}).Error; err != nil {
			t.Fatalf("seed post %d: %v", postID, err)
		}
	}

}

func createMentionAgent(t *testing.T, username string) uint64 {
	t.Helper()
	result, err := agentservice.Create(agentservice.CreateParams{Username: username})
	if err != nil {
		t.Fatalf("create agent %s: %v", username, err)
	}
	return result.Agent.UserId
}

func createHuman(t *testing.T, username string) uint64 {
	t.Helper()
	user := users.MakeUser(username, "password-123", username+"@example.test")
	user.IsActivated = users.ActivationSuccess
	if err := users.Create(user); err != nil {
		t.Fatalf("create human %s: %v", username, err)
	}
	return user.Id
}

func countTasks(t *testing.T) int64 {
	var count int64
	if err := db.Connect().Model(&taskQueue.Entity{}).Count(&count).Error; err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	return count
}

func TestHandleMentionEventCreatesInboxAndTask(t *testing.T) {
	setupMentionTestDB(t)
	agentID := createMentionAgent(t, "target-bot-1")
	humanID := createHuman(t, "author-one")

	err := HandleMentionEvent(context.Background(), MentionEventParams{
		EventType: agentInbox.EventTypeTopicPublished,
		TopicId:   101,
		PostId:    1001,
		ActorId:   humanID,
		Title:     "Hello @target-bot-1",
		Content:   "Body without mentions",
		Preview:   "Hello @target-bot-1",
	})
	if err != nil {
		t.Fatalf("HandleMentionEvent: %v", err)
	}

	var inboxes []agentInbox.Entity
	if err := db.Connect().Find(&inboxes).Error; err != nil {
		t.Fatalf("list inboxes: %v", err)
	}
	if len(inboxes) != 1 {
		t.Fatalf("inbox rows = %d, want 1", len(inboxes))
	}
	row := inboxes[0]
	if row.AgentId != agentID || row.TopicId != 101 || row.PostId != 1001 {
		t.Fatalf("inbox = %#v, want agent/topic/post ids", row)
	}
	if row.EventType != agentInbox.EventTypeTopicPublished || row.ActorId != humanID {
		t.Fatalf("inbox event/actor = %#v", row)
	}
	if row.Status != agentInbox.StatusUnread || row.DeliveryStatus != agentInbox.DeliveryPending {
		t.Fatalf("inbox state = %#v, want unread+pending", row)
	}
	if row.ContentPreview != "Hello @target-bot-1" {
		t.Fatalf("preview = %q", row.ContentPreview)
	}

	var tasks []taskQueue.Entity
	if err := db.Connect().Find(&tasks).Error; err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("tasks = %d, want 1", len(tasks))
	}
	if tasks[0].Type != agentwebhookservice.TaskTypeAgentWebhook || tasks[0].Status != taskQueue.StatusPending {
		t.Fatalf("task = %#v", tasks[0])
	}
	var taskPayload agentwebhookservice.TaskPayload
	if err := json.Unmarshal([]byte(tasks[0].TaskJson), &taskPayload); err != nil || taskPayload.InboxId != row.Id {
		t.Fatalf("task json = %q, want inboxId %d: %v", tasks[0].TaskJson, row.Id, err)
	}
}

func TestHandleMentionEventIgnoresUnknownHumansAndSelf(t *testing.T) {
	setupMentionTestDB(t)
	agentID := createMentionAgent(t, "target-bot-2")
	humanID := createHuman(t, "author-two")

	// Unknown user, human user, and the actor itself (which is a bot here)
	// must all be ignored.
	err := HandleMentionEvent(context.Background(), MentionEventParams{
		EventType: agentInbox.EventTypePostCreated,
		TopicId:   102,
		PostId:    1002,
		ActorId:   humanID,
		Content:   "Hi @nobody-here @author-two @target-bot-2",
		Preview:   "Hi",
	})
	if err != nil {
		t.Fatalf("HandleMentionEvent: %v", err)
	}
	var inboxes []agentInbox.Entity
	db.Connect().Find(&inboxes)
	if len(inboxes) != 1 || inboxes[0].AgentId != agentID {
		t.Fatalf("inboxes = %#v, want only target-bot-2", inboxes)
	}

	// Actor itself is the bot: self-mention must be ignored.
	if err := HandleMentionEvent(context.Background(), MentionEventParams{
		EventType: agentInbox.EventTypePostCreated,
		TopicId:   103,
		PostId:    1003,
		ActorId:   agentID,
		Content:   "I mention @target-bot-2 myself",
		Preview:   "I",
	}); err != nil {
		t.Fatalf("HandleMentionEvent self: %v", err)
	}
	var after []agentInbox.Entity
	db.Connect().Find(&after)
	if len(after) != 1 {
		t.Fatalf("self mention must be ignored, got %d rows", len(after))
	}
}

func TestHandleMentionEventRequiresExactUsernameCase(t *testing.T) {
	setupMentionTestDB(t)
	createMentionAgent(t, "CaseBot")
	humanID := createHuman(t, "case-author")

	if err := HandleMentionEvent(context.Background(), MentionEventParams{
		EventType: agentInbox.EventTypePostCreated,
		TopicId:   102,
		PostId:    1002,
		ActorId:   humanID,
		Content:   "Wrong case @casebot",
		Preview:   "Wrong case",
	}); err != nil {
		t.Fatalf("HandleMentionEvent wrong case: %v", err)
	}
	if countTasks(t) != 0 {
		t.Fatal("case-mismatched mention must not enqueue a task")
	}
	var inboxCount int64
	if err := db.Connect().Model(&agentInbox.Entity{}).Count(&inboxCount).Error; err != nil {
		t.Fatalf("count inbox: %v", err)
	}
	if inboxCount != 0 {
		t.Fatalf("case-mismatched mention created %d inbox rows", inboxCount)
	}

	if err := HandleMentionEvent(context.Background(), MentionEventParams{
		EventType: agentInbox.EventTypePostCreated,
		TopicId:   102,
		PostId:    1002,
		ActorId:   humanID,
		Content:   "Exact case @CaseBot",
		Preview:   "Exact case",
	}); err != nil {
		t.Fatalf("HandleMentionEvent exact case: %v", err)
	}
	if countTasks(t) != 1 {
		t.Fatal("exact-case mention must enqueue one task")
	}
}

func TestHandleMentionEventUpsertResetsState(t *testing.T) {
	setupMentionTestDB(t)
	agentID := createMentionAgent(t, "target-bot-3")
	humanID := createHuman(t, "author-three")

	params := MentionEventParams{
		EventType: agentInbox.EventTypeTopicPublished,
		TopicId:   104,
		PostId:    1004,
		ActorId:   humanID,
		Title:     "First @target-bot-3",
		Content:   "First body",
		Preview:   "First body",
	}
	if err := HandleMentionEvent(context.Background(), params); err != nil {
		t.Fatalf("first event: %v", err)
	}
	var row agentInbox.Entity
	db.Connect().First(&row)
	rowId := row.Id
	if err := agentInbox.MarkRead(rowId, agentID); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if err := agentInbox.MarkFailed(rowId, "boom"); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	// Replay/edit upserts the same row and resets unread+pending+error.
	params.EventType = agentInbox.EventTypeTopicUpdated
	params.Preview = "Updated body"
	if err := HandleMentionEvent(context.Background(), params); err != nil {
		t.Fatalf("replay event: %v", err)
	}
	var after []agentInbox.Entity
	db.Connect().Find(&after)
	if len(after) != 1 {
		t.Fatalf("replay must upsert one row, got %d", len(after))
	}
	if after[0].Id != rowId {
		t.Fatalf("row id changed: %d -> %d", rowId, after[0].Id)
	}
	if after[0].Status != agentInbox.StatusUnread || after[0].DeliveryStatus != agentInbox.DeliveryPending {
		t.Fatalf("state after replay = %#v, want unread+pending", after[0])
	}
	if after[0].Attempts != 0 || after[0].LastError != "" {
		t.Fatalf("attempts/error after replay = %#v, want reset", after[0])
	}
	if after[0].EventType != agentInbox.EventTypeTopicUpdated || after[0].ContentPreview != "Updated body" {
		t.Fatalf("event/preview after replay = %#v", after[0])
	}
}

func TestHandleMentionEventTruncatesPreviewTo64Runes(t *testing.T) {
	setupMentionTestDB(t)
	createMentionAgent(t, "target-bot-4")
	humanID := createHuman(t, "author-four")

	longPreview := strings.Repeat("长", 100)
	err := HandleMentionEvent(context.Background(), MentionEventParams{
		EventType: agentInbox.EventTypeTopicPublished,
		TopicId:   105,
		PostId:    1005,
		ActorId:   humanID,
		Title:     "Mention @target-bot-4",
		Content:   "x",
		Preview:   longPreview,
	})
	if err != nil {
		t.Fatalf("HandleMentionEvent: %v", err)
	}
	var row agentInbox.Entity
	db.Connect().First(&row)
	if runes := []rune(row.ContentPreview); len(runes) != 64 {
		t.Fatalf("preview runes = %d, want 64", len(runes))
	}
}

func TestHandleMentionEventDeleteThenRecreate(t *testing.T) {
	setupMentionTestDB(t)
	agentID := createMentionAgent(t, "target-bot-5")
	humanID := createHuman(t, "author-five")

	params := MentionEventParams{
		EventType: agentInbox.EventTypeTopicPublished,
		TopicId:   106,
		PostId:    1006,
		ActorId:   humanID,
		Title:     "Hello @target-bot-5",
		Content:   "Body",
		Preview:   "Hello",
	}
	if err := HandleMentionEvent(context.Background(), params); err != nil {
		t.Fatalf("first event: %v", err)
	}
	var row agentInbox.Entity
	db.Connect().First(&row)
	oldId := row.Id
	if err := agentInbox.DeleteOwned(oldId, agentID); err != nil {
		t.Fatalf("delete inbox: %v", err)
	}

	// Same key after delete creates a fresh row (history removal is final).
	if err := HandleMentionEvent(context.Background(), params); err != nil {
		t.Fatalf("recreate event: %v", err)
	}
	var after []agentInbox.Entity
	db.Connect().Find(&after)
	if len(after) != 1 || after[0].Id == oldId {
		t.Fatalf("recreate = %#v, want one fresh row", after)
	}
}

func TestHandleMentionEventNoMentionsNoSideEffects(t *testing.T) {
	setupMentionTestDB(t)
	createMentionAgent(t, "target-bot-6")
	humanID := createHuman(t, "author-six")

	err := HandleMentionEvent(context.Background(), MentionEventParams{
		EventType: agentInbox.EventTypePostCreated,
		TopicId:   107,
		PostId:    1007,
		ActorId:   humanID,
		Content:   "No mentions here",
		Preview:   "No",
	})
	if err != nil {
		t.Fatalf("HandleMentionEvent: %v", err)
	}
	if countTasks(t) != 0 {
		t.Fatal("no mention must not enqueue tasks")
	}
	var inboxes []agentInbox.Entity
	db.Connect().Find(&inboxes)
	if len(inboxes) != 0 {
		t.Fatalf("no mention must not create inbox rows: %#v", inboxes)
	}
}

func TestHandleMentionEventMaxTenDistinct(t *testing.T) {
	setupMentionTestDB(t)
	humanID := createHuman(t, "author-seven")
	// Create 12 agents, mention all: only the first 10 (text order) count.
	ids := make([]uint64, 0, 12)
	for i := 0; i < 12; i++ {
		ids = append(ids, createMentionAgent(t, "target-bot-"+string(rune('a'+i))))
	}
	var builder strings.Builder
	for i := 0; i < 12; i++ {
		builder.WriteString("@target-bot-")
		builder.WriteByte(byte('a' + i))
		builder.WriteString(" ")
	}
	if err := HandleMentionEvent(context.Background(), MentionEventParams{
		EventType: agentInbox.EventTypeTopicPublished,
		TopicId:   108,
		PostId:    1008,
		ActorId:   humanID,
		Title:     builder.String(),
		Content:   "",
		Preview:   "many",
	}); err != nil {
		t.Fatalf("HandleMentionEvent: %v", err)
	}
	var inboxes []agentInbox.Entity
	db.Connect().Find(&inboxes)
	if len(inboxes) != 10 {
		t.Fatalf("inbox rows = %d, want 10", len(inboxes))
	}
	gotFirst := false
	for _, row := range inboxes {
		if row.AgentId == ids[0] {
			gotFirst = true
		}
	}
	if !gotFirst {
		t.Fatal("first mention in text order must win")
	}
	for _, row := range inboxes {
		if row.AgentId == ids[11] {
			t.Fatal("mention beyond the max must be ignored")
		}
	}
}

func TestHandleMentionEventRejectsUnpublishedContent(t *testing.T) {
	setupMentionTestDB(t)
	createMentionAgent(t, "target-bot-x")
	humanID := createHuman(t, "author-eight")

	tests := []struct {
		name      string
		topicID   uint64
		postID    uint64
		mutate    func(t *testing.T)
		eventType string
	}{
		{
			name:      "pending topic",
			topicID:   101,
			postID:    1001,
			eventType: agentInbox.EventTypeTopicPublished,
			mutate: func(t *testing.T) {
				t.Helper()
				if err := topics.UpdateProcessStatus(101, topics.ProcessStatusPending); err != nil {
					t.Fatalf("mark topic pending: %v", err)
				}
			},
		},
		{
			name:      "blocked post",
			topicID:   102,
			postID:    1002,
			eventType: agentInbox.EventTypeTopicUpdated,
			mutate: func(t *testing.T) {
				t.Helper()
				if err := posts.UpdateProcessStatus(1002, posts.ProcessStatusBlocked); err != nil {
					t.Fatalf("mark post blocked: %v", err)
				}
			},
		},
		{
			name:      "draft topic",
			topicID:   103,
			postID:    1003,
			eventType: agentInbox.EventTypeTopicPublished,
			mutate: func(t *testing.T) {
				t.Helper()
				topic := topics.Get(103)
				topic.Status = 0
				if err := topics.Save(&topic); err != nil {
					t.Fatalf("mark topic draft: %v", err)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mutate(t)
			if err := HandleMentionEvent(context.Background(), MentionEventParams{
				EventType: tc.eventType,
				TopicId:   tc.topicID,
				PostId:    tc.postID,
				ActorId:   humanID,
				Title:     "Hidden @target-bot-x",
				Content:   "must not leave the platform",
				Preview:   "hidden preview",
			}); err != nil {
				t.Fatalf("HandleMentionEvent: %v", err)
			}
		})
	}

	var inboxCount int64
	if err := db.Connect().Model(&agentInbox.Entity{}).Count(&inboxCount).Error; err != nil {
		t.Fatalf("count inbox: %v", err)
	}
	if inboxCount != 0 || countTasks(t) != 0 {
		t.Fatalf("unpublished content created inbox/tasks: inbox=%d tasks=%d", inboxCount, countTasks(t))
	}
}

func TestEnqueueMentionTxRollsBackWithContentTransaction(t *testing.T) {
	setupMentionTestDB(t)
	rollbackErr := errors.New("force rollback")
	err := db.Connect().Transaction(func(tx *gorm.DB) error {
		topic := topics.Entity{Id: 9001, Title: "Rollback @target-bot-r", UserId: 91, Status: 1, ProcessStatus: topics.ProcessStatusNormal}
		if err := tx.Create(&topic).Error; err != nil {
			return err
		}
		post := posts.Entity{Id: 9101, TopicId: topic.Id, PostNo: 1, UserId: 91, Content: "rollback", ProcessStatus: posts.ProcessStatusNormal}
		if err := tx.Create(&post).Error; err != nil {
			return err
		}
		if err := EnqueueMentionTx(tx, MentionEventParams{
			EventType: agentInbox.EventTypeTopicPublished,
			TopicId:   topic.Id,
			PostId:    post.Id,
			ActorId:   topic.UserId,
			Title:     topic.Title,
			Content:   post.Content,
			Preview:   post.Content,
		}); err != nil {
			return err
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("transaction error = %v, want rollback sentinel", err)
	}
	var topicCount, postCount, taskCount int64
	if err := db.Connect().Model(&topics.Entity{}).Where("id = ?", 9001).Count(&topicCount).Error; err != nil {
		t.Fatalf("count rolled-back topic: %v", err)
	}
	if err := db.Connect().Model(&posts.Entity{}).Where("id = ?", 9101).Count(&postCount).Error; err != nil {
		t.Fatalf("count rolled-back post: %v", err)
	}
	if err := db.Connect().Model(&taskQueue.Entity{}).Where("type = ?", TaskTypeAgentMention).Count(&taskCount).Error; err != nil {
		t.Fatalf("count rolled-back task: %v", err)
	}
	if topicCount != 0 || postCount != 0 || taskCount != 0 {
		t.Fatalf("rollback left topic=%d post=%d task=%d", topicCount, postCount, taskCount)
	}
}

func TestRunTaskConsumesDurableMentionOutbox(t *testing.T) {
	setupMentionTestDB(t)
	agentID := createMentionAgent(t, "target-bot-r")
	humanID := createHuman(t, "author-nine")
	if err := db.Connect().Transaction(func(tx *gorm.DB) error {
		return EnqueueMentionTx(tx, MentionEventParams{
			EventType: agentInbox.EventTypeTopicPublished,
			TopicId:   101,
			PostId:    1001,
			ActorId:   humanID,
			Title:     "Durable @target-bot-r",
			Content:   "body",
			Preview:   strings.Repeat("长", 80),
		})
	}); err != nil {
		t.Fatalf("enqueue durable mention: %v", err)
	}
	var mentionTask taskQueue.Entity
	if err := db.Connect().Where("type = ?", TaskTypeAgentMention).First(&mentionTask).Error; err != nil {
		t.Fatalf("find durable mention task: %v", err)
	}
	var payload TaskPayload
	if err := json.Unmarshal([]byte(mentionTask.TaskJson), &payload); err != nil {
		t.Fatalf("decode durable task: %v", err)
	}
	if len(payload.Names) != 1 || payload.Names[0] != "target-bot-r" || len([]rune(payload.Preview)) != maxPreviewRunes {
		t.Fatalf("durable payload = %#v", payload)
	}
	if strings.Contains(mentionTask.TaskJson, "body") {
		t.Fatalf("durable task leaked full content: %s", mentionTask.TaskJson)
	}
	if err := RunTask(context.Background(), &mentionTask); err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	var inbox agentInbox.Entity
	if err := db.Connect().Where("agent_id = ? AND topic_id = ? AND post_id = ?", agentID, 101, 1001).First(&inbox).Error; err != nil {
		t.Fatalf("find ingested inbox: %v", err)
	}
	if inbox.ContentPreview != payload.Preview || inbox.DeliveryStatus != agentInbox.DeliveryPending {
		t.Fatalf("inbox = %#v, payload=%#v", inbox, payload)
	}
	var webhookTask taskQueue.Entity
	if err := db.Connect().Where("type = ?", agentwebhookservice.TaskTypeAgentWebhook).First(&webhookTask).Error; err != nil {
		t.Fatalf("find webhook task: %v", err)
	}
	var webhookPayload agentwebhookservice.TaskPayload
	if err := json.Unmarshal([]byte(webhookTask.TaskJson), &webhookPayload); err != nil || webhookPayload.InboxId != inbox.Id {
		t.Fatalf("webhook task = %#v payload=%#v err=%v", webhookTask, webhookPayload, err)
	}
}

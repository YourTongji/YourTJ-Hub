// Package agentinboxservice turns published content writes into durable
// mention-ingestion tasks, Agent inbox rows, and webhook delivery tasks.
// Inbox rows are the source of truth for delivery state; agent.mention tasks
// bridge the content commit to inbox ingestion without a process-crash gap.
package agentinboxservice

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/leancodebox/GooseForum/app/bundles/agentmention"
	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/agentInbox"
	"github.com/leancodebox/GooseForum/app/models/forum/agents"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/taskQueue"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/agentwebhookservice"
	"gorm.io/gorm"
)

const (
	// TaskTypeAgentMention is the durable outbox task for mention ingestion.
	TaskTypeAgentMention = "agent.mention"
	// maxPreviewRunes bounds the stored content preview (inbox column is 255 chars).
	maxPreviewRunes = 64
)

// MentionEventParams is the normalized event data needed to detect mentions.
type MentionEventParams struct {
	EventType string // agentInbox.EventTypeTopicPublished / TopicUpdated / PostCreated
	TopicId   uint64
	PostId    uint64
	ActorId   uint64 // the user who published/edited/created the content
	Title     string
	Content   string
	Preview   string // already content-derived preview; truncated to 64 runes here
}

// TaskPayload is the bounded durable mention outbox payload. It carries only
// stable IDs, at most ten parsed usernames, and a short preview — never full
// topic/post content.
type TaskPayload struct {
	EventType string   `json:"eventType"`
	TopicId   uint64   `json:"topicId"`
	PostId    uint64   `json:"postId"`
	ActorId   uint64   `json:"actorId"`
	Names     []string `json:"names"`
	Preview   string   `json:"preview"`
}

// EnqueueMentionTx parses a published write and stores its bounded ingestion
// task in the caller's content transaction. A process crash after commit can
// therefore delay ingestion, but cannot lose it.
func EnqueueMentionTx(tx *gorm.DB, params MentionEventParams) error {
	names := agentmention.Find(params.Title + "\n" + params.Content)
	if len(names) == 0 {
		return nil
	}
	payload, err := json.Marshal(TaskPayload{
		EventType: params.EventType,
		TopicId:   params.TopicId,
		PostId:    params.PostId,
		ActorId:   params.ActorId,
		Names:     names,
		Preview:   truncatePreview(params.Preview),
	})
	if err != nil {
		return err
	}
	return taskQueue.CreateTx(tx, &taskQueue.Entity{
		Type:     TaskTypeAgentMention,
		Status:   taskQueue.StatusPending,
		TaskJson: string(payload),
	})
}

// RunTask consumes one durable mention-ingestion task. Malformed payloads are
// terminal no-ops; transient database failures are returned for worker retry.
func RunTask(ctx context.Context, task *taskQueue.Entity) error {
	payload, err := decodeTaskPayload(task.TaskJson)
	if err != nil {
		slog.Error("agentinbox: malformed mention task payload", "id", task.Id, "err", err)
		return nil
	}
	return handleMentionNames(ctx, payload.Names, MentionEventParams{
		EventType: payload.EventType,
		TopicId:   payload.TopicId,
		PostId:    payload.PostId,
		ActorId:   payload.ActorId,
		Preview:   payload.Preview,
	})
}

func decodeTaskPayload(taskJSON string) (TaskPayload, error) {
	var payload TaskPayload
	if err := json.Unmarshal([]byte(taskJSON), &payload); err != nil {
		return payload, err
	}
	if payload.TopicId == 0 || payload.PostId == 0 || payload.ActorId == 0 || len(payload.Names) == 0 || len(payload.Names) > 10 {
		return payload, errors.New("agentinbox: invalid mention task payload")
	}
	switch payload.EventType {
	case agentInbox.EventTypeTopicPublished, agentInbox.EventTypeTopicUpdated, agentInbox.EventTypePostCreated:
	default:
		return payload, errors.New("agentinbox: invalid mention event type")
	}
	payload.Preview = truncatePreview(payload.Preview)
	return payload, nil
}

// HandleMentionEvent is the synchronous seam for focused service tests.
// Production write paths must use EnqueueMentionTx so the trigger commits
// atomically with the published content.
func HandleMentionEvent(ctx context.Context, params MentionEventParams) error {
	names := agentmention.Find(params.Title + "\n" + params.Content)
	return handleMentionNames(ctx, names, params)
}

func handleMentionNames(ctx context.Context, names []string, params MentionEventParams) error {
	if len(names) == 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	preview := truncatePreview(params.Preview)
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		published, err := isPublishedContentTx(tx, params.TopicId, params.PostId)
		if err != nil {
			return err
		}
		if !published {
			return nil
		}

		seenAgentIds := make(map[uint64]struct{}, len(names))
		for _, name := range names {
			var user users.EntityComplete
			if err := tx.Where("username = ?", name).First(&user).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return err
			}
			// MySQL commonly compares varchar columns case-insensitively. Recheck
			// in Go so mention matching stays exact across all supported databases.
			if user.Id == 0 || user.Username != name || !user.IsBot() || user.Id == params.ActorId {
				continue
			}
			if _, duplicate := seenAgentIds[user.Id]; duplicate {
				continue
			}
			var agent agents.Entity
			if err := tx.Where("user_id = ?", user.Id).First(&agent).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return err
			}
			seenAgentIds[user.Id] = struct{}{}
			inbox := &agentInbox.Entity{
				AgentId:        agent.UserId,
				TopicId:        params.TopicId,
				PostId:         params.PostId,
				EventType:      params.EventType,
				ActorId:        params.ActorId,
				ContentPreview: preview,
				Status:         agentInbox.StatusUnread,
				DeliveryStatus: agentInbox.DeliveryPending,
			}
			if err := agentInbox.UpsertTx(tx, inbox); err != nil {
				return err
			}
			inboxId, err := agentInbox.GetIdByKeyTx(tx, agent.UserId, params.TopicId, params.PostId)
			if err != nil {
				return err
			}
			taskJSON, err := json.Marshal(agentwebhookservice.TaskPayload{InboxId: inboxId})
			if err != nil {
				return err
			}
			if err := taskQueue.CreateTx(tx, &taskQueue.Entity{
				Type:     agentwebhookservice.TaskTypeAgentWebhook,
				Status:   taskQueue.StatusPending,
				TaskJson: string(taskJSON),
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func isPublishedContentTx(tx *gorm.DB, topicId, postId uint64) (bool, error) {
	var topicCount int64
	if err := tx.Model(&topics.Entity{}).
		Where("id = ? AND status = ? AND process_status = ?", topicId, 1, topics.ProcessStatusNormal).
		Count(&topicCount).Error; err != nil {
		return false, err
	}
	if topicCount == 0 {
		return false, nil
	}
	var postCount int64
	if err := tx.Model(&posts.Entity{}).
		Where("id = ? AND topic_id = ? AND process_status = ?", postId, topicId, posts.ProcessStatusNormal).
		Count(&postCount).Error; err != nil {
		return false, err
	}
	return postCount > 0, nil
}

func truncatePreview(preview string) string {
	runes := []rune(preview)
	if len(runes) > maxPreviewRunes {
		return string(runes[:maxPreviewRunes])
	}
	return preview
}

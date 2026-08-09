// Package agentinboxservice turns published-content events into Agent inbox
// rows and durable webhook tasks. Inbox rows are the single source of truth
// for delivery state; task_queue rows are disposable scheduling entries.
package agentinboxservice

import (
	"context"
	"encoding/json"

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

// maxPreviewRunes bounds the stored content preview (inbox column is 255 chars).
const maxPreviewRunes = 64

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

// HandleMentionEvent scans title+content for @username candidates in text
// order (max 10 distinct), resolves them to existing bot Agents (unknown
// users, humans, non-bot rows and the event actor itself are ignored), and
// transactionally upserts one inbox row per Agent plus one queued
// agent.webhook task per row. Event replay/edit upserts the same row and
// resets unread+pending state, then queues fresh delivery.
func HandleMentionEvent(ctx context.Context, params MentionEventParams) error {
	names := agentmention.Find(params.Title + "\n" + params.Content)
	if len(names) == 0 {
		return nil
	}
	targets := make([]*agents.Entity, 0, len(names))
	for _, name := range names {
		user, err := users.GetByUsername(name)
		if err != nil || user.Id == 0 || !user.IsBot() || user.Id == params.ActorId {
			continue
		}
		agent := agents.GetByUserID(user.Id)
		if agent == nil {
			continue
		}
		targets = append(targets, agent)
	}
	if len(targets) == 0 {
		return nil
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
		for _, agent := range targets {
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

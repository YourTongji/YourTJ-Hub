package publicationservice

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/nativepushservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/realtimeservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/webpushservice"
)

const EffectTaskType = "content-published"

type Effect struct {
	RevisionId         uint64 `json:"revisionId"`
	PreviousRevisionId uint64 `json:"previousRevisionId"`
	NotificationId     uint64 `json:"notificationId,omitempty"`
	NotificationOnly   bool   `json:"notificationOnly,omitempty"`
	// ReviewRequested marks the transfer to human review; moderator approval
	// notifications are published after that commit (issue #1049).
	ReviewRequested bool `json:"reviewRequested,omitempty"`
}

// RunEffectsTask uses the existing event handlers' idempotent rewards/activity
// delivery. Payloads contain IDs only and are rechecked against deletion state.
func RunEffectsTask(ctx context.Context, task *taskQueue.Entity) error {
	var effect Effect
	if err := json.Unmarshal([]byte(task.TaskJson), &effect); err != nil {
		return err
	}
	var revision postRevisions.Entity
	conn := db.ConnectContext(ctx)
	if err := conn.First(&revision, effect.RevisionId).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}
	var post posts.Entity
	if err := conn.First(&post, revision.PostId).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}
	var topic topics.Entity
	if err := conn.First(&topic, post.TopicId).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}
	if !available(topic, post) {
		return nil
	}
	// Publish before the author pushes so a failed publish retries without
	// repeating them; per-approval delivery dedupe absorbs task retries.
	if effect.ReviewRequested && post.LatestRevisionId == revision.Id && revision.ProcessStatus == posts.ProcessStatusPending {
		if err := eventbus.PublishE(ctx, &eventhandlers.ModerationReviewRequestedEvent{RevisionID: revision.Id}); err != nil {
			return err
		}
	}
	realtimeservice.DefaultHub.Publish(post.UserId, realtimeservice.Event{Type: realtimeservice.EventContentChanged})
	userservice.InvalidateUserPublicProfileCache(post.UserId)
	if effect.NotificationId != 0 {
		realtimeservice.DefaultHub.Publish(post.UserId, realtimeservice.Event{Type: realtimeservice.EventNotificationsChanged})
		realtimeservice.DefaultHub.Publish(post.UserId, realtimeservice.Event{Type: realtimeservice.EventUnreadChanged})
		webpushservice.EnqueueNotification(post.UserId, effect.NotificationId)
		nativepushservice.EnqueueNotification(post.UserId, effect.NotificationId)
	}
	if effect.NotificationOnly || revision.ProcessStatus != posts.ProcessStatusNormal {
		return nil
	}
	if (effect.PreviousRevisionId != 0 && post.PublishedRevisionId != revision.Id) || post.ProcessStatus != posts.ProcessStatusNormal || topic.ProcessStatus != topics.ProcessStatusNormal {
		return nil
	}
	if effect.PreviousRevisionId != 0 {
		var previous postRevisions.Entity
		if err := db.ConnectContext(ctx).First(&previous, effect.PreviousRevisionId).Error; err != nil {
			return err
		}
		if post.PostNo == 1 {
			if err := eventbus.PublishE(ctx, &eventhandlers.TopicUpdatedEvent{Topic: &topic, FirstPost: &post}); err != nil {
				return err
			}
		}
		return eventbus.PublishE(ctx, &eventhandlers.PostUpdatedEvent{TopicId: topic.Id, PostId: post.Id, PostNo: post.PostNo, UserId: post.UserId, OldContent: previous.Content, NewContent: post.Content, IsAnonymous: post.IsAnonymous, PersonaUID: post.PersonaUID})
	} else if post.PostNo == 1 {
		return eventbus.PublishE(ctx, &eventhandlers.TopicPublishedEvent{Topic: &topic, FirstPost: &post})
	} else {
		var parent posts.Entity
		if post.ReplyToPostId != 0 {
			if err := conn.First(&parent, post.ReplyToPostId).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		return eventbus.PublishE(ctx, &eventhandlers.CommentCreatedEvent{TopicId: topic.Id, PostId: post.Id, PostNo: post.PostNo, UserId: post.UserId, Content: post.Content, TopicAuthorId: topic.UserId, ReplyToPostId: post.ReplyToPostId, ReplyToPostAuthorId: parent.UserId, IsAnonymous: post.IsAnonymous, PersonaUID: post.PersonaUID})
	}
}

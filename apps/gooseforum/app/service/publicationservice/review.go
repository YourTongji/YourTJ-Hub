package publicationservice

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userActivities"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentcommentservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agenteventservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/llmsservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/postservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/realtimeservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/unreadservice"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RunReviewTask is restart-safe. The task carries only a revision ID, so deleting
// retained content also removes the text an unprocessed job could otherwise use.
func RunReviewTask(ctx context.Context, task *taskQueue.Entity) error {
	var payload Task
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil {
		return err
	}
	var revision postRevisions.Entity
	conn := db.ConnectContext(ctx)
	if err := conn.First(&revision, payload.RevisionId).Error; errors.Is(err, gorm.ErrRecordNotFound) {
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
	if post.LatestRevisionId != revision.Id || revision.ProcessStatus != posts.ProcessStatusPending || revision.ReviewedAt != nil || !available(topic, post) {
		return nil
	}
	subject, id := moderationDecision.SubjectPost, post.Id
	if post.PostNo == 1 {
		subject, id = moderationDecision.SubjectTopic, topic.Id
	}
	action, reason := moderationDecision.ActionAllow, ""
	cfg := hotdataserve.GetSecuritySettingsConfigCache()
	text := revision.Title + "\n" + revision.Content
	if words := moderationservice.FindSensitiveWordsInTextsWithConfig([]string{text, markdown2html.ExtractVisibleText(text)}, cfg); len(words) > 0 {
		action, reason = moderationDecision.ActionBlock, "内容包含不符合发布规则的词语，请修改后重新提交。"
		if cfg.SensitiveAction == "review" {
			action, reason = moderationDecision.ActionReview, "等待人工审核"
		}
	} else if decision := moderationservice.EvaluateSubmission(ctx, moderationservice.AIContentInput{AuthorID: post.UserId, SubjectType: subject, SubjectID: id, Title: revision.Title, Content: revision.Content, Gallery: revision.ImageUrls}); decision != nil {
		decision.RevisionId = revision.Id
		decision.AppliedAction = moderationDecision.ActionReview
		if err := moderationDecision.Create(decision); err != nil {
			return err
		}
		action = decision.FinalAction
		if action == moderationDecision.ActionBlock {
			reason = "内容不符合本站发布规则，请修改后重新提交。"
		}
		if decision.ErrorKind == "external_image_blocked" {
			reason = "请将站外图片上传到本站后重新提交。"
		}
		if action == moderationDecision.ActionReview {
			reason = "等待人工审核"
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	err := Review(ctx, revision.Id, action, reason, 0)
	if errors.Is(err, ErrUnavailable) {
		return nil
	}
	return err
}

func available(topic topics.Entity, post posts.Entity) bool {
	return topic.Id != 0 && post.Id != 0 && topic.Status == 1 && topic.VisibilityStatus == topics.VisibilityActive && post.VisibilityStatus == posts.VisibilityActive && !topic.DeletedAt.Valid && !post.DeletedAt.Valid && (topic.TopicType != topics.TopicTypeWiki || post.PostNo > 1)
}

// Review compares the exact revision under the post lock. Human decisions and
// worker retries share this path; a terminal decision has exactly one notice.
func Review(ctx context.Context, revisionID uint64, action, reason string, actorID uint64) error {
	var topic topics.Entity
	var post posts.Entity
	var oldCategories []uint64
	// Rendering can resolve stickers/mentions via the DB; do it before locking.
	// This also avoids publishing an HTML cache produced by an older renderer.
	var approvedHTML string
	if action == moderationDecision.ActionAllow {
		var candidate postRevisions.Entity
		if err := db.ConnectContext(ctx).First(&candidate, revisionID).Error; err != nil {
			return err
		}
		approvedHTML = postservice.RenderPostHTML(candidate.Content)
	}
	err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		var revision postRevisions.Entity
		if err := tx.First(&revision, revisionID).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&post, revision.PostId).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&topic, post.TopicId).Error; err != nil {
			return err
		}
		oldCategories = append(oldCategories, topic.CategoryIds...)
		// Re-read the revision after acquiring the post lock.
		if err := tx.First(&revision, revisionID).Error; err != nil {
			return err
		}
		if post.LatestRevisionId != revisionID || revision.ProcessStatus != posts.ProcessStatusPending || !available(topic, post) {
			return ErrUnavailable
		}
		if post.ProcessStatus == posts.ProcessStatusBlocked || topic.ProcessStatus == topics.ProcessStatusBlocked {
			return ErrUnavailable
		}
		if post.PostNo > 1 && topic.ProcessStatus != topics.ProcessStatusNormal {
			return ErrUnavailable
		}
		if action == moderationDecision.ActionReview {
			// A pending revision with ReviewedAt has already been routed to a human.
			// The timestamp and notification commit together under the post lock.
			if revision.ReviewedAt != nil {
				return nil
			}
			if strings.TrimSpace(reason) == "" {
				reason = "等待人工审核"
			}
			if err := tx.Model(&revision).Updates(map[string]any{"review_reason": reason, "reviewed_at": time.Now()}).Error; err != nil {
				return err
			}
			return enqueueReviewNotice(tx, revision, topic, post, eventNotification.EventTypeReviewPending, Effect{RevisionId: revisionID, NotificationOnly: true, ReviewRequested: true})
		}
		if actorID == 0 && revision.ReviewedAt != nil {
			return ErrUnavailable
		}
		if action != moderationDecision.ActionAllow && action != moderationDecision.ActionBlock {
			return errors.New("invalid review action")
		}
		// Submission may precede an operator ban. Recheck only first publication
		// of bot replies; an existing public reply remains editable.
		if action == moderationDecision.ActionAllow && post.PostNo > 1 && post.PublishedRevisionId == 0 && post.ProcessStatus != posts.ProcessStatusNormal {
			author, err := users.GetInteractionUserTx(tx, post.UserId)
			if err != nil {
				return err
			}
			if author.IsBot() {
				if err := agentcommentservice.CheckNewCommentTx(tx, topic.Id); errors.Is(err, agentcommentservice.ErrAgentCommentDisabled) {
					action, reason = moderationDecision.ActionBlock, "该话题或站点已禁止机器人回复。"
				} else if err != nil {
					return err
				}
			}
		}
		status := posts.ProcessStatusBlocked
		if action == moderationDecision.ActionAllow {
			status = posts.ProcessStatusNormal
		}
		if action == moderationDecision.ActionBlock && strings.TrimSpace(reason) == "" {
			reason = "内容未通过审核，请修改后重新提交。"
		}
		now := time.Now()
		if err := tx.Model(&revision).Updates(map[string]any{"process_status": status, "review_reason": reason, "review_actor_id": actorID, "reviewed_at": now}).Error; err != nil {
			return err
		}
		previous := post.PublishedRevisionId
		wasPublic := post.ProcessStatus == posts.ProcessStatusNormal && topic.ProcessStatus == topics.ProcessStatusNormal
		if action == moderationDecision.ActionAllow {
			revision.ProcessStatus = status
			revision.RenderedHTML = approvedHTML
			firstPublication := stampFirstPublic(&post, now, wasPublic)
			ApplySnapshot(&topic, &post, revision)
			if post.PostNo == 1 && topic.FirstPublicAt == nil {
				at := *post.FirstPublicAt
				topic.FirstPublicAt = &at
				topic.FirstPublicEstimated = post.FirstPublicEstimated
				if err := tx.Model(&topics.Entity{}).Where("id = ? AND first_public_at IS NULL", topic.Id).UpdateColumns(map[string]any{"first_public_at": at, "first_public_estimated": topic.FirstPublicEstimated}).Error; err != nil {
					return err
				}
			}
			if firstPublication {
				kind := "public_reply"
				if post.PostNo == 1 {
					kind = "public_topic"
				}
				if err := capturePublicContributionTx(tx, post.UserId, topic.Id, post.Id, kind); err != nil {
					return err
				}
			}
			post.PublishedRevisionId = revision.Id
			if err := posts.SaveTx(tx, &post); err != nil {
				return err
			}
			if post.PostNo == 1 {
				if err := topics.UpdateTopicEditableTx(tx, &topic); err != nil {
					return err
				}
				if err := topicCategoryIndex.ReplaceTopicCategoriesTx(tx, topic.Id, topic.CategoryIds); err != nil {
					return err
				}
			}
			if err := fileusageservice.PublishRevisionImagesTx(tx, revision.Id, topic.Id, post.Id, post.PostNo == 1); err != nil {
				return err
			}
			// The public projection and durable Agent intent commit together. Effects
			// may run later or retry without losing or recapturing the publication.
			if err := agenteventservice.CapturePublicTx(tx, &post); err != nil {
				return err
			}
		} else if !wasPublic {
			if err := posts.UpdateProcessStatusTx(tx, post.Id, status); err != nil {
				return err
			}
			if post.PostNo == 1 {
				if err := topics.UpdateProcessStatusTx(tx, topic.Id, status); err != nil {
					return err
				}
			}
		}
		if action == moderationDecision.ActionAllow && previous == 0 {
			activity, subject := userActivities.ActionComment, userActivities.SubjectPost
			id := post.Id
			if post.PostNo == 1 {
				activity, subject, id = userActivities.ActionPost, userActivities.SubjectTopic, topic.Id
			}
			recorded, err := userActivities.HasRecordTx(tx, activity, subject, id)
			if err != nil {
				return err
			}
			if !recorded {
				if err := userStatistics.RecordPublicationTx(tx, post.UserId, post.PostNo == 1); err != nil {
					return err
				}
			}
		}
		if actorID == 0 {
			if err := tx.Model(&moderationDecision.Entity{}).Where("revision_id = ?", revisionID).Update("applied_action", action).Error; err != nil {
				return err
			}
		}

		if err := postservice.RebuildTopicPostStatsTx(tx, topic); err != nil {
			return err
		}
		if err := searchservice.EnqueueTopicSearchTask(tx, topic.Id); err != nil {
			return err
		}
		effect := Effect{RevisionId: revisionID, PreviousRevisionId: previous}
		if actorID != 0 {
			human := moderationDecision.HumanRejected
			if action == moderationDecision.ActionAllow {
				human = moderationDecision.HumanApproved
			}
			if err := tx.Model(&moderationDecision.Entity{}).Where("revision_id = ?", revisionID).Updates(map[string]any{"human_action": human, "human_actor_id": actorID, "human_at": now}).Error; err != nil {
				return err
			}
		}
		if action == moderationDecision.ActionBlock {
			effect.NotificationOnly = true
			return enqueueReviewNotice(tx, revision, topic, post, eventNotification.EventTypeReviewRejected, effect)
		}
		if actorID != 0 {
			return enqueueReviewNotice(tx, revision, topic, post, eventNotification.EventTypeReviewApproved, effect)
		}
		return enqueueEffect(tx, effect)
	})
	if err == nil {
		unreadservice.Invalidate(post.UserId)
		hotdataserve.InvalidateTopicListCacheForCategories(append(oldCategories, topic.CategoryIds...)...)
		llmsservice.ClearCache()
		realtimeservice.DefaultHub.Publish(post.UserId, realtimeservice.Event{Type: realtimeservice.EventContentChanged})
		realtimeservice.DefaultHub.Publish(post.UserId, realtimeservice.Event{Type: realtimeservice.EventNotificationsChanged})
	}
	return err
}

// Notifications contain a safe subject snapshot; rejection reasons and full
// candidates remain in content management, not notification or push payloads.
func enqueueReviewNotice(tx *gorm.DB, revision postRevisions.Entity, topic topics.Entity, post posts.Entity, eventType string, effect Effect) error {
	template := eventNotification.TemplateReviewPending
	if eventType == eventNotification.EventTypeReviewApproved {
		template = eventNotification.TemplateReviewApproved
	}
	subject := strings.TrimSpace(revision.Title)
	if subject == "" {
		subject = revision.Content
	}
	subject = markdown2html.ExtractVisibleText(subject)
	if chars := []rune(subject); eventType != eventNotification.EventTypeReviewRejected && len(chars) > 60 {
		subject = string(chars[:60]) + "…"
	}
	payload := eventNotification.NotificationPayload{TemplateKey: template, TopicTitle: subject, TopicId: topic.Id, PostId: post.Id, PostNo: post.PostNo}
	if eventType == eventNotification.EventTypeReviewRejected {
		payload.TemplateKey = eventNotification.TemplateReviewRejected
		payload = eventNotification.RedactReviewRejectedPayload(payload)
	}
	notice := eventNotification.Entity{UserId: post.UserId, TopicID: topic.Id, EventType: eventType, Payload: payload}
	if err := tx.Create(&notice).Error; err != nil {
		return err
	}
	effect.NotificationId = notice.Id
	return enqueueEffect(tx, effect)
}

func enqueueEffect(tx *gorm.DB, effect Effect) error {
	raw, err := json.Marshal(effect)
	if err != nil {
		return err
	}
	return taskQueue.CreateTx(tx, &taskQueue.Entity{Type: EffectTaskType, TaskJson: string(raw)})
}

// A legacy public projection has an estimated clock; every genuinely pending
// first publication starts its clock at approval, and restore never resets it.
func stampFirstPublic(post *posts.Entity, now time.Time, wasPublic bool) bool {
	if post.FirstPublicAt != nil {
		return false
	}
	at := now
	if wasPublic {
		at = post.CreatedAt
		post.FirstPublicEstimated = true
	}
	post.FirstPublicAt = &at
	return !wasPublic
}

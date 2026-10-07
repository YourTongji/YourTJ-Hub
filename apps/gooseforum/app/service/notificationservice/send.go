package notificationservice

import (
	"log/slog"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/nativepushservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/realtimeservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/unreadservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/webpushservice"
	"github.com/spf13/cast"
)

// SendCommentNotification 发送评论通知
func SendCommentNotification(userId uint64, topicId uint64, commentContent string, commenterId uint64, postId uint64, postNo uint64) error {
	payload := eventNotification.NotificationPayload{
		Content:     commentContent,
		TemplateKey: eventNotification.TemplateComment,
		TemplateParams: eventNotification.NotificationTemplateParams{
			Preview: commentContent,
		},
		ActorId: commenterId,
		TopicId: topicId,
		PostId:  postId,
		PostNo:  postNo,
	}

	notification := &eventNotification.Entity{
		UserId:    userId,
		EventType: eventNotification.EventTypeComment,
		TopicID:   topicId,
		Payload:   payload,
	}

	created, err := persistInteractions(db.Connect(), []*eventNotification.Entity{notification})
	if err == nil && len(created) > 0 {
		notificationCommitted(userId)
		webpushservice.EnqueueNotification(userId, notification.Id)
		nativepushservice.EnqueueNotification(userId, notification.Id)
	}
	return err
}

// SendPostReplyNotification 发送 post 回复通知
func SendPostReplyNotification(userId uint64, postId uint64, postNo uint64, topicId uint64, replyContent string, replierId uint64) error {
	payload := eventNotification.NotificationPayload{
		Content:     replyContent,
		TemplateKey: eventNotification.TemplatePostReply,
		TemplateParams: eventNotification.NotificationTemplateParams{
			Preview: replyContent,
		},
		ActorId: replierId,
		TopicId: topicId,
		PostId:  postId,
		PostNo:  postNo,
	}

	notification := &eventNotification.Entity{
		UserId:    userId,
		EventType: eventNotification.EventTypePostReply,
		TopicID:   topicId,
		Payload:   payload,
	}

	created, err := persistInteractions(db.Connect(), []*eventNotification.Entity{notification})
	if err == nil && len(created) > 0 {
		notificationCommitted(userId)
		webpushservice.EnqueueNotification(userId, notification.Id)
		nativepushservice.EnqueueNotification(userId, notification.Id)
	}
	return err
}

func SendTopicPostNotifications(userIds []uint64, topicId uint64, postId uint64, postNo uint64, commentContent string, commenterId uint64) error {
	if len(userIds) == 0 {
		return nil
	}

	notifications := make([]*eventNotification.Entity, 0, len(userIds))
	for _, userId := range userIds {
		if userId == 0 {
			continue
		}
		notifications = append(notifications, &eventNotification.Entity{
			UserId:    userId,
			EventType: eventNotification.EventTypeTopicPost,
			TopicID:   topicId,
			Payload: eventNotification.NotificationPayload{
				Content:     commentContent,
				TemplateKey: eventNotification.TemplateTopicPost,
				TemplateParams: eventNotification.NotificationTemplateParams{
					Preview: commentContent,
				},
				ActorId: commenterId,
				TopicId: topicId,
				PostId:  postId,
				PostNo:  postNo,
			},
		})
	}
	if len(notifications) == 0 {
		return nil
	}

	created, err := persistInteractions(db.Connect(), notifications)
	if err == nil {
		for _, notification := range created {
			if notification == nil {
				continue
			}
			notificationCommitted(notification.UserId)
			webpushservice.EnqueueNotification(notification.UserId, notification.Id)
			nativepushservice.EnqueueNotification(notification.UserId, notification.Id)
		}
	}
	return err
}

// SendMentionNotifications 批量发送 @mention 通知（issue #563）。
// 调用方已按优先级去重并限制 fan-out 上限，这里只做 0 值过滤。
func SendMentionNotifications(userIds []uint64, topicId uint64, postId uint64, postNo uint64, preview string, mentionerId uint64) error {
	if len(userIds) == 0 {
		return nil
	}

	notifications := make([]*eventNotification.Entity, 0, len(userIds))
	for _, userId := range userIds {
		if userId == 0 {
			continue
		}
		notifications = append(notifications, &eventNotification.Entity{
			UserId:    userId,
			EventType: eventNotification.EventTypeMention,
			TopicID:   topicId,
			Payload: eventNotification.NotificationPayload{
				Content:     preview,
				TemplateKey: eventNotification.TemplateMention,
				TemplateParams: eventNotification.NotificationTemplateParams{
					Preview: preview,
				},
				ActorId: mentionerId,
				TopicId: topicId,
				PostId:  postId,
				PostNo:  postNo,
			},
		})
	}
	if len(notifications) == 0 {
		return nil
	}

	created, err := persistInteractions(db.Connect(), notifications)
	if err == nil {
		for _, notification := range created {
			notificationCommitted(notification.UserId)
		}
	}
	return err
}

func SendBadgeNotification(userId uint64, badgeCode string, badgeName string, badgeIconURL string) error {
	payload := eventNotification.NotificationPayload{
		TemplateKey: eventNotification.TemplateBadge,
		ActorId:     userId,
		Extra: eventNotification.Extra{
			BadgeCode:    badgeCode,
			BadgeName:    badgeName,
			BadgeIconURL: badgeIconURL,
			ProfileURL:   "/u/" + cast.ToString(userId),
		},
	}

	notification := &eventNotification.Entity{
		UserId:    userId,
		EventType: eventNotification.EventTypeBadge,
		Payload:   payload,
	}

	err := eventNotification.Create(notification)
	if err == nil {
		notificationCommitted(userId)
		webpushservice.EnqueueNotification(userId, notification.Id)
		nativepushservice.EnqueueNotification(userId, notification.Id)
	}
	return err
}

// SendSystemAlert 发送无触发者的系统级站内通知（issue #855）。
// 不携带 TemplateKey/ActorId：Web 与移动端按 payload 的 title/content 原样渲染，
// 用于运维告警（如排课同步失败提醒）。通知内容由调用方负责脱敏。
func SendSystemAlert(userID uint64, title string, content string) error {
	payload := eventNotification.NotificationPayload{
		Title:   title,
		Content: content,
	}

	notification := &eventNotification.Entity{
		UserId:    userID,
		EventType: eventNotification.EventTypeSystem,
		Payload:   payload,
	}

	err := eventNotification.Create(notification)
	if err == nil {
		notificationCommitted(userID)
		webpushservice.EnqueueNotification(userID, notification.Id)
		nativepushservice.EnqueueNotification(userID, notification.Id)
	}
	return err
}

// SendReviewResultNotification notifies the author of a human decision.
// Rejected content is recoverable in content management; its subject is masked.
func SendReviewResultNotification(userID uint64, approved bool, topicID uint64, topicTitle string, postID uint64, postNo uint64) error {
	payload := eventNotification.NotificationPayload{
		TemplateKey: eventNotification.TemplateReviewRejected,
		TopicTitle:  topicTitle,
		TopicId:     topicID,
		PostId:      postID,
		PostNo:      postNo,
	}
	eventType := eventNotification.EventTypeReviewRejected
	if approved {
		payload.TemplateKey = eventNotification.TemplateReviewApproved
		eventType = eventNotification.EventTypeReviewApproved
	} else {
		payload = eventNotification.RedactReviewRejectedPayload(payload)
	}
	notification := &eventNotification.Entity{
		UserId:    userID,
		EventType: eventType,
		TopicID:   topicID,
		Payload:   payload,
	}
	err := eventNotification.Create(notification)
	if err == nil {
		notificationCommitted(userID)
		webpushservice.EnqueueNotification(userID, notification.Id)
		nativepushservice.EnqueueNotification(userID, notification.Id)
	}
	return err
}

// SendLikeNotification 发送楼层点赞通知
func SendLikeNotification(userId uint64, topicId uint64, topicTitle string, postId uint64, postNo uint64, likerId uint64) error {
	payload := eventNotification.NotificationPayload{
		TemplateKey: eventNotification.TemplateLike,
		ActorId:     likerId,
		TopicId:     topicId,
		TopicTitle:  topicTitle,
		PostId:      postId,
		PostNo:      postNo,
	}

	notification := &eventNotification.Entity{
		UserId:    userId,
		EventType: eventNotification.EventTypeLike,
		TopicID:   topicId,
		Payload:   payload,
	}

	created, err := persistInteractions(db.Connect(), []*eventNotification.Entity{notification})
	if err == nil && len(created) > 0 {
		notificationCommitted(userId)
		webpushservice.EnqueueNotification(userId, notification.Id)
		nativepushservice.EnqueueNotification(userId, notification.Id)
	}
	return err
}

// SendFollowNotification 发送关注通知
func SendFollowNotification(userId uint64, followerId uint64, followerName string) error {
	payload := eventNotification.NotificationPayload{
		TemplateKey: eventNotification.TemplateFollow,
		ActorId:     followerId,
		Extra:       eventNotification.Extra{FollowerName: followerName},
	}

	notification := &eventNotification.Entity{
		UserId:    userId,
		EventType: eventNotification.EventTypeFollow,
		Payload:   payload,
	}

	created, err := persistInteractions(db.Connect(), []*eventNotification.Entity{notification})
	if err == nil && len(created) > 0 {
		notificationCommitted(userId)
		webpushservice.EnqueueNotification(userId, notification.Id)
		nativepushservice.EnqueueNotification(userId, notification.Id)
	}
	return err
}

// NullifyContentPreviews 内容删除后把相关通知的正文预览置空，避免泄露已删原文。
func NullifyContentPreviews(topicId uint64, postId uint64) {
	if topicId == 0 && postId == 0 {
		return
	}
	recipients, err := eventNotification.ClearPreviewsByTopicWithRecipients(topicId, postId)
	for _, userID := range recipients {
		realtimeservice.PublishNotificationsChanged(userID, "preview-cleared")
	}
	if err != nil {
		slog.Error("clear notification previews failed", "topicId", topicId, "postId", postId, "err", err)
	}
}

func notificationCommitted(userID uint64) {
	unreadservice.Invalidate(userID)
	realtimeservice.PublishNotificationsChanged(userID, "created")
}

// persistInteractions serializes the block check and notification insert with
// block/unblock and chat writes. Callers publish SSE/push only for committed rows.
func persistInteractions(conn *gorm.DB, notifications []*eventNotification.Entity) ([]*eventNotification.Entity, error) {
	if len(notifications) == 0 {
		return nil, nil
	}
	participants := make([]uint64, 0, len(notifications)*2)
	recipients := make(map[uint64][]uint64)
	for _, n := range notifications {
		if n == nil || n.UserId == 0 {
			continue
		}
		participants = append(participants, n.UserId)
		if n.Payload.ActorId != 0 {
			participants = append(participants, n.Payload.ActorId)
		}
		recipients[n.Payload.ActorId] = append(recipients[n.Payload.ActorId], n.UserId)
	}
	var committed []*eventNotification.Entity
	err := conn.Transaction(func(tx *gorm.DB) error {
		if err := users.LockInteractionUserIDs(tx, participants); err != nil {
			return err
		}
		allowed := make(map[uint64]map[uint64]bool)
		for actor, ids := range recipients {
			filtered, err := users.FilterInteractionRecipientsTx(tx, actor, ids)
			if err != nil {
				return err
			}
			allowed[actor] = make(map[uint64]bool, len(filtered))
			for _, id := range filtered {
				allowed[actor][id] = true
			}
		}
		for _, n := range notifications {
			if n != nil && allowed[n.Payload.ActorId][n.UserId] {
				committed = append(committed, n)
			}
		}
		if err := redactAnonymousActors(tx, committed); err != nil {
			return err
		}
		return eventNotification.CreateBatchTx(tx, committed, 100)
	})
	if err != nil {
		return nil, err
	}
	return committed, nil
}

package eventhandlers

import (
	"context"
	"errors"
	"sort"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/notificationservice"
	"gorm.io/gorm"
)

const (
	topicWatchNotifyBatchSize = 500

	// maxMentionFanOut 单篇内容 mention 通知 fan-out 上限（issue #563）：
	// 超过 20 个唯一被提及用户时正文仍可保存/展示，只停止额外通知。
	maxMentionFanOut = 20
)

// 单用户通知优先级：同一新楼层对同一接收用户最多产生 1 条最高优先级通知。
// post_reply > mention > comment > topic_post。
const (
	notificationPriorityPostReply = 1
	notificationPriorityMention   = 2
	notificationPriorityComment   = 3
	notificationPriorityTopicPost = 4
)

type notificationRecipient struct {
	userID    uint64
	eventType string
}

// TakeUpTo64Chars 按字符数截取字符串，最多取 64 个字符
func TakeUpTo64Chars(s string) string {
	return markdown2html.ExtractPreview(s, 64)
}

// CommentCreatedEvent 评论/回复创建事件
type CommentCreatedEvent struct {
	TopicId             uint64
	PostId              uint64 // 新创建的 post ID
	PostNo              uint64 // 新创建的 post 楼层号
	UserId              uint64 // 发表评论者 ID
	Content             string
	TopicAuthorId       uint64 // 主题作者 ID
	ReplyToPostId       uint64 // 被回复的 post ID
	ReplyToPostAuthorId uint64 // 被回复的 post 作者 ID
	IsAnonymous         bool   // 匿名楼层（wiki 评论区，issue #524）
}

// handleCommentCreated 发送评论/回复通知
func handleCommentCreated(ctx context.Context, event *CommentCreatedEvent) error {
	// 匿名楼层不产生任何通知（issue #524）：三个通道的 ActorId 都是匿名作者，
	// ActorName 在读取时按 ActorId 回填真实用户名，会向他人泄露匿名身份。
	// 匿名作者收到「他人回复其楼层」的通知不在此事件内——那是另一条以真实
	// 回复者为 Actor 的 CommentCreatedEvent，走下方非匿名分支正常发送。
	if event == nil || event.IsAnonymous {
		return nil
	}
	visible, err := mentionPostIsPublic(ctx, event.TopicId, event.PostId)
	if err != nil || !visible {
		return err
	}
	contentPreview := TakeUpTo64Chars(event.Content)
	// 收件人/类型在同一流程内计算并去重：被提及者若同时是回复目标/主题作者/
	// 关注者，只保留最高优先级通知（issue #563）。
	mentionUserIDs := resolveMentionUserIDs(event.Content, event.UserId, maxMentionFanOut)
	recipients := priorityRecipients(event, mentionUserIDs)

	for _, recipient := range recipients {
		switch recipient.eventType {
		case eventNotification.EventTypePostReply:
			_ = notificationservice.SendPostReplyNotification(recipient.userID, event.PostId, event.PostNo, event.TopicId, contentPreview, event.UserId)
		case eventNotification.EventTypeComment:
			_ = notificationservice.SendCommentNotification(recipient.userID, event.TopicId, contentPreview, event.UserId, event.PostId, event.PostNo)
		}
	}
	if mentionIDs := collectRecipients(recipients, eventNotification.EventTypeMention); len(mentionIDs) > 0 {
		_ = notificationservice.SendMentionNotifications(mentionIDs, event.TopicId, event.PostId, event.PostNo, contentPreview, event.UserId)
	}
	notifyTopicWatchers(event, contentPreview, mentionUserIDs)
	return nil
}

// resolveMentionUserIDs 从正文提取 mention 并解析为用户 ID：
// 去重、排除作者本人与无效目标；limit>0 时最多返回 limit 个（fan-out 上限）。
func resolveMentionUserIDs(content string, excludeUserID uint64, limit int) []uint64 {
	usernames := markdown2html.ExtractMentions(content)
	if len(usernames) == 0 {
		return nil
	}
	userMap := users.GetMentionTargetIds(usernames)
	userIDs := make([]uint64, 0, len(usernames))
	seen := make(map[uint64]struct{}, len(usernames))
	for _, username := range usernames {
		userID, ok := userMap[username]
		if !ok || userID == 0 || userID == excludeUserID {
			continue
		}
		if _, dup := seen[userID]; dup {
			continue
		}
		seen[userID] = struct{}{}
		userIDs = append(userIDs, userID)
		if limit > 0 && len(userIDs) >= limit {
			break
		}
	}
	return userIDs
}

// priorityRecipients 计算每个接收者应收到的最高优先级通知类型
// （post_reply > mention > comment > topic_post），结果按 userID 升序保证确定性。
func priorityRecipients(event *CommentCreatedEvent, mentionUserIDs []uint64) []notificationRecipient {
	best := make(map[uint64]int, 2+len(mentionUserIDs))
	winner := make(map[uint64]string, 2+len(mentionUserIDs))
	set := func(userID uint64, eventType string, priority int) {
		if userID == 0 {
			return
		}
		if current, ok := best[userID]; ok && current <= priority {
			return
		}
		best[userID] = priority
		winner[userID] = eventType
	}
	if shouldNotifyTopicAuthor(event) {
		set(event.TopicAuthorId, eventNotification.EventTypeComment, notificationPriorityComment)
	}
	if shouldNotifyParentReplyAuthor(event) {
		set(event.ReplyToPostAuthorId, eventNotification.EventTypePostReply, notificationPriorityPostReply)
	}
	for _, userID := range mentionUserIDs {
		set(userID, eventNotification.EventTypeMention, notificationPriorityMention)
	}

	recipients := make([]notificationRecipient, 0, len(winner))
	for userID, eventType := range winner {
		recipients = append(recipients, notificationRecipient{userID: userID, eventType: eventType})
	}
	sort.Slice(recipients, func(i, j int) bool { return recipients[i].userID < recipients[j].userID })
	return recipients
}

func collectRecipients(recipients []notificationRecipient, eventType string) []uint64 {
	userIDs := make([]uint64, 0)
	for _, recipient := range recipients {
		if recipient.eventType == eventType {
			userIDs = append(userIDs, recipient.userID)
		}
	}
	return userIDs
}

func shouldNotifyTopicAuthor(event *CommentCreatedEvent) bool {
	if event.TopicAuthorId == 0 || event.TopicAuthorId == event.UserId {
		return false
	}
	return event.ReplyToPostId == 0 || event.TopicAuthorId != event.ReplyToPostAuthorId
}

func shouldNotifyParentReplyAuthor(event *CommentCreatedEvent) bool {
	return event.ReplyToPostId > 0 && event.ReplyToPostAuthorId > 0 && event.ReplyToPostAuthorId != event.UserId
}

func notifyTopicWatchers(event *CommentCreatedEvent, contentPreview string, mentionUserIDs []uint64) {
	excludeUserIds := commentNotificationExcludeUserIds(event)
	excludeUserIds = append(excludeUserIds, mentionUserIDs...)
	afterUserId := uint64(0)
	for {
		userIds := topicUserAction.ListActiveWatchUserIDsAfter(event.TopicId, afterUserId, excludeUserIds, topicWatchNotifyBatchSize)
		if len(userIds) == 0 {
			return
		}
		_ = notificationservice.SendTopicPostNotifications(userIds, event.TopicId, event.PostId, event.PostNo, contentPreview, event.UserId)
		afterUserId = userIds[len(userIds)-1]
		if len(userIds) < topicWatchNotifyBatchSize {
			return
		}
	}
}

func commentNotificationExcludeUserIds(event *CommentCreatedEvent) []uint64 {
	excludeSet := map[uint64]struct{}{}
	add := func(userId uint64) {
		if userId > 0 {
			excludeSet[userId] = struct{}{}
		}
	}
	add(event.UserId)
	add(event.TopicAuthorId)
	add(event.ReplyToPostAuthorId)

	userIds := make([]uint64, 0, len(excludeSet))
	for userId := range excludeSet {
		userIds = append(userIds, userId)
	}
	return userIds
}

// PostUpdatedEvent 帖子编辑事件（issue #563）：编辑后仅新增 mention 产生通知。
type PostUpdatedEvent struct {
	TopicId     uint64
	PostId      uint64
	PostNo      uint64
	UserId      uint64 // 编辑者
	OldContent  string
	NewContent  string
	IsAnonymous bool // 匿名楼层（wiki 评论区，issue #524）
}

// handlePostUpdated 编辑后发送新增 mention 通知：保留/删除的 mention 与仅文字
// 修改均不重复通知（旧/新内容集合差，issue #563）。
func handlePostUpdated(ctx context.Context, event *PostUpdatedEvent) error {
	// 匿名楼层编辑同样不产生通知（issue #524 边界）。
	if event == nil || event.IsAnonymous {
		return nil
	}
	if event.OldContent == event.NewContent {
		return nil
	}
	visible, err := mentionPostIsPublic(ctx, event.TopicId, event.PostId)
	if err != nil || !visible {
		return err
	}
	addedUserIDs := newMentionUserIDs(event.OldContent, event.NewContent, event.UserId)
	if len(addedUserIDs) == 0 {
		return nil
	}
	return notificationservice.SendMentionNotifications(addedUserIDs, event.TopicId, event.PostId, event.PostNo, TakeUpTo64Chars(event.NewContent), event.UserId)
}

// Recheck current visibility because publication/edit events are asynchronous.
func mentionPostIsPublic(ctx context.Context, topicID, postID uint64) (bool, error) {
	topic, err := topics.GetWithContext(ctx, topicID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if topic.Status != 1 || topic.ProcessStatus != topics.ProcessStatusNormal || topic.VisibilityStatus != topics.VisibilityActive {
		return false, nil
	}
	if topic.FirstPostId != 0 && topic.FirstPostId != postID {
		first, err := posts.GetWithContext(ctx, topic.FirstPostId)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if first.ProcessStatus != posts.ProcessStatusNormal || first.VisibilityStatus != posts.VisibilityActive {
			return false, nil
		}
	}
	post, err := posts.GetWithContext(ctx, postID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return post.TopicId == topicID && !post.IsAnonymous && post.ProcessStatus == posts.ProcessStatusNormal && post.VisibilityStatus == posts.VisibilityActive, nil
}

func handleTopicMentionPublished(ctx context.Context, event *TopicPublishedEvent) error {
	if event == nil || event.Topic == nil || event.FirstPost == nil {
		return nil
	}
	visible, err := mentionPostIsPublic(ctx, event.Topic.Id, event.FirstPost.Id)
	if err != nil || !visible || event.FirstPost.IsAnonymous {
		return err
	}
	targets := resolveMentionUserIDs(event.FirstPost.Content, event.FirstPost.UserId, maxMentionFanOut)
	seen, err := eventNotification.MentionRecipientsForPost(event.Topic.Id, event.FirstPost.Id, targets)
	if err != nil {
		return err
	}
	remaining := make([]uint64, 0, len(targets))
	for _, id := range targets {
		if !seen[id] {
			remaining = append(remaining, id)
		}
	}
	return notificationservice.SendMentionNotifications(remaining, event.Topic.Id, event.FirstPost.Id, event.FirstPost.PostNo, TakeUpTo64Chars(event.FirstPost.Content), event.FirstPost.UserId)

}

// newMentionUserIDs 返回编辑后新增的 mention 用户（最多 maxMentionFanOut 个）。
func newMentionUserIDs(oldContent, newContent string, editorID uint64) []uint64 {
	return mentionDiff(
		resolveMentionUserIDs(oldContent, editorID, 0),
		resolveMentionUserIDs(newContent, editorID, 0),
	)
}

// mentionDiff 返回 newIDs 中不在 oldIDs 里的用户（最多 maxMentionFanOut 个）。
func mentionDiff(oldIDs, newIDs []uint64) []uint64 {
	oldSet := make(map[uint64]struct{}, len(oldIDs))
	for _, id := range oldIDs {
		oldSet[id] = struct{}{}
	}
	added := make([]uint64, 0)
	for _, id := range newIDs {
		if _, kept := oldSet[id]; kept {
			continue
		}
		added = append(added, id)
		if len(added) >= maxMentionFanOut {
			break
		}
	}
	return added
}

// UserFollowedEvent 用户关注事件
type UserFollowedEvent struct {
	UserId       uint64
	FollowerId   uint64
	FollowerName string
}

// handleUserFollowed 发送关注通知
func handleUserFollowed(ctx context.Context, event *UserFollowedEvent) error {
	return notificationservice.SendFollowNotification(event.UserId, event.FollowerId, event.FollowerName)
}

// TopicLikedEvent 主题点赞事件
type TopicLikedEvent struct {
	UserId  uint64
	TopicId uint64
	Title   string
	LikerId uint64
}

// PostLikedEvent 楼层点赞事件
type PostLikedEvent struct {
	UserId     uint64 // 楼层作者
	PostId     uint64
	PostNo     uint64 // 被点赞楼层的楼层号
	TopicId    uint64
	TopicTitle string
	LikerId    uint64
}

// handlePostLiked 发送楼层点赞通知（自己给自己点赞不通知）
func handlePostLiked(ctx context.Context, event *PostLikedEvent) error {
	if event == nil || event.UserId == 0 || event.PostId == 0 || event.LikerId == event.UserId {
		return nil
	}
	return notificationservice.SendLikeNotification(event.UserId, event.TopicId, event.TopicTitle, event.PostId, event.PostNo, event.LikerId)
}

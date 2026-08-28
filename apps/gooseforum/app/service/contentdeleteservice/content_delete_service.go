// Package contentdeleteservice 实现内容删除生命周期（Issue #94）的共享业务逻辑。
//
// 设计要点：
//   - 删除 = 软删标记（GORM DeletedAt）+ 双状态机（visibility_status × retention_status），
//     绝不物理删除正文，避免破坏举报证据与审计链。
//   - 用户删除进入 30 天恢复窗口；管理端删除作者不可自行恢复。
//   - 删除立即生效：通过 ContentDeletedEvent 广播搜索索引 / LLMS 缓存 / 热度缓存 / 通知预览。
package contentdeleteservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/contentDeleteEvent"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPageRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/llmsservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/notificationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/optlogger"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pointservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/postservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"gorm.io/gorm"
)

// RecoveryWindow 是"最近删除"的默认恢复窗口。
const RecoveryWindow = 30 * 24 * time.Hour

// EvidenceSnapshotRetention 已结案举报证据快照的默认保留期（Issue #94 合规）。
// Legal/Evidence Hold 目标在清理时被跳过，可覆盖本 TTL。
const EvidenceSnapshotRetention = 180 * 24 * time.Hour

const deleteConfirmWindow = 10 * time.Minute
const deleteConfirmLimit = int64(20)

// CheckDeleteRate centralizes the confirmation gate for all destructive entry points.
func CheckDeleteRate(userID uint64, count int, force bool, password string) error {
	recent, err := contentDeleteEvent.CountRecentByActorEvents(userID, []string{
		string(contentDeleteEvent.EventDeleted), string(contentDeleteEvent.EventPrivacyDelete),
	}, time.Now().Add(-deleteConfirmWindow))
	if err != nil {
		return component.NewMessageError(component.MessageOperationFailed, "删除频率检查失败", nil)
	}
	if recent+int64(count) <= deleteConfirmLimit {
		return nil
	}
	if !force {
		return component.NewMessageError(component.MessageContentBatchConfirmRequired, "短时间内删除过多，需要二次确认", component.MessageParams{"count": recent + int64(count)})
	}
	user, err := users.Get(userID)
	if err != nil || user.Id == 0 {
		return component.NewMessageError(component.MessageUserFetchFailed, "用户不存在", nil)
	}
	if _, err := users.Verify(user.Username, password); err != nil {
		return component.NewMessageError(component.MessageAuthInvalidCredentials, "密码错误", nil)
	}
	return nil
}

// ContentType 表示删除目标类型。
type ContentType string

// DeletePostResult describes the server-side deletion shape so clients do not
// have to infer tombstone behavior from the currently loaded reply window.
type DeletePostResult struct {
	HasChildren bool
}

const (
	ContentTypeTopic ContentType = "topic"
	ContentTypePost  ContentType = "post"
)

// DeleteTopicByUser 删除自己的话题（R1）。
//   - 无任何回复：软删话题 + 级联软删全部回复 + 递减统计，全渠道消失。
//   - 已有回复：仅标记 USER_DELETED/RECOVERABLE（deleted_at 置位），
//     保留其他用户的回复可见，详情页渲染"原帖已由作者删除"占位。
func DeleteTopicByUser(userID uint64, topicID uint64) error {
	topic := topics.Get(topicID)
	if topic.Id == 0 {
		return component.NewMessageError(component.MessageTopicNotFound, "话题不存在", nil)
	}
	if topic.UserId != userID {
		return component.NewMessageError(component.MessageTopicOwnerMismatch, "不能删除他人的话题", nil)
	}
	return DeleteTopicAs(topic, userID, topics.VisibilityUserDeleted, "")
}

// DeleteTopicAs 执行话题删除（用户/管理员共用），visibility 区分删除来源。
func DeleteTopicAs(topic topics.Entity, operatorID uint64, visibility string, reason string) error {
	cascadeReason := reason
	if visibility == topics.VisibilityUserDeleted {
		cascadeReason = topicDeleteCascadeReason(topic.Id)
	}
	// 是否存在回复（post_no > 1 且未删除）
	replies := posts.GetByTopicPostNoAfter(topic.Id, 1, 1)
	hasReplies := len(replies) > 0
	var activePosts []*posts.Entity
	if !hasReplies {
		if err := posts.ListByTopicID(topic.Id, &activePosts); err != nil {
			return component.NewMessageError(component.MessageContentDeleteFailed, "删除话题失败", component.MessageParams{"error": err.Error()})
		}
	}

	if err := markTopicDeleted(topic.Id, visibility, operatorID, reason); err != nil {
		return component.NewMessageError(component.MessageContentDeleteFailed, "删除话题失败", component.MessageParams{"error": err.Error()})
	}

	// wiki 分站页面话题：同步清理 wiki_pages 及其修订，避免删除话题后残留
	// 孤儿页面继续出现在公开导航树/首页（与 DeleteAllUserContent 级联一致）。
	if topic.TopicType == topics.TopicTypeWiki {
		if page := wikiPages.GetByTopicIdUnscoped(topic.Id); page.Id != 0 {
			pageID := page.Id
			if err := wikiPageRevisions.DeleteByPage(page.Id); err != nil {
				return component.NewMessageError(component.MessageContentDeleteFailed, "删除话题失败", component.MessageParams{"error": err.Error()})
			}
			if err := wikiPages.Delete(page.Id); err != nil {
				return component.NewMessageError(component.MessageContentDeleteFailed, "删除话题失败", component.MessageParams{"error": err.Error()})
			}
			deleteWikiPageSearchIndex(pageID)
		}
	}

	// 无回复时级联软删全部回复并递减统计；有回复时保留他人讨论。
	if !hasReplies {
		// 按删除前收集到的回复 ID 精确级联，避免并发新回复在读取与写入之间
		// 被 TOCTOU 竞态一并软删并重复递减统计。
		activePostIDs := make([]uint64, 0, len(activePosts))
		for _, post := range activePosts {
			if post != nil {
				activePostIDs = append(activePostIDs, post.Id)
			}
		}
		deletedCount := posts.SoftDeleteByIDs(activePostIDs, operatorID, cascadeReason, visibility)
		for _, post := range activePosts {
			if post == nil || post.UserId == 0 {
				continue
			}
			postservice.SyncTopicPostStats(topic, *post, true)
			fileusageservice.HardenTargetFiles(postsTarget(post.Id), time.Now().Add(RecoveryWindow))
		}
		slog.Info("topic deleted without replies, cascade posts", "topicId", topic.Id, "posts", deletedCount)
	} else {
		// 首楼正文随话题隐藏；统计保留，评论区继续展示他人回复。
		firstPost := posts.Get(topic.FirstPostId)
		if firstPost.Id > 0 {
			if err := posts.MarkDeletedKeepVisible(firstPost.Id, visibility, operatorID, cascadeReason); err != nil {
				return component.NewMessageError(component.MessageContentDeleteFailed, "删除话题首帖失败", component.MessageParams{"error": err.Error()})
			}
			fileusageservice.HardenTargetFiles(postsTarget(firstPost.Id), time.Now().Add(RecoveryWindow))
		}
	}

	// 删除立即生效：全渠道清除 + 通知预览置空 + 附件转入受限恢复态。
	clearTopicCaches(topic.Id)
	notificationservice.NullifyContentPreviews(topic.Id, 0)
	fileusageservice.HardenTargetFiles(topicsTarget(topic.Id), time.Now().Add(RecoveryWindow))
	eventbus.Publish(context.Background(), &eventhandlers.ContentDeletedEvent{
		ContentType:  string(ContentTypeTopic),
		TopicId:      topic.Id,
		PostId:       0,
		DeletedBy:    operatorID,
		DeleteReason: reason,
	})

	// 审计：moderation_log + 操作日志。
	moderationservice.TopicDeleted(operatorID, moderationservice.TopicDeletedSnapshot{
		TopicId:       topic.Id,
		TopicTitle:    topic.Title,
		DeletedBy:     operatorID,
		DeletedByUser: "",
		Reason:        reason,
	})
	optlogger.UserOptCode(operatorID, optlogger.EditTopic, topic.Id, "admin.opt.topic.deleted", optlogger.MessageParams{
		"title":  topic.Title,
		"reason": reason,
	})
	recordEvent(contentDeleteEvent.EventDeleted, ContentTypeTopic, topic.Id, topic.Id, operatorID)
	return nil
}

func markTopicDeleted(topicID uint64, visibility string, operatorID uint64, reason string) error {
	// 删除时作废待审状态：被删话题不应继续停留在管理审核队列（PRD R1）。
	// 无论作者删除还是管理端删除，一律把 process_status 复位为正常，
	// 避免"已删除"与"待审"语义叠加导致审核队列出现幽灵项。
	if err := topics.ResetPendingReview(topicID); err != nil {
		return err
	}
	switch visibility {
	case topics.VisibilityModeratorRemoved:
		return topics.MarkModeratorRemoved(topicID, operatorID, reason)
	default:
		return topics.MarkUserDeleted(topicID, operatorID, reason)
	}
}

func clearTopicCaches(topicID uint64) {
	hotdataserve.ClearTopicListCache()
	llmsservice.ClearCache()
	slog.Debug("topic delete caches cleared", "topicId", topicID)
}

// DeletePostByUser 删除自己的回复（R2）。
//   - 无子回复：软删，直接消失。
//   - 有子回复：墓碑态（保留行可见，正文由前端渲染为占位），讨论树完整。
func DeletePostByUser(userID uint64, postID uint64) (DeletePostResult, error) {
	var post posts.Entity
	var result DeletePostResult
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		loaded, err := posts.GetUnscopedTx(tx, postID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return component.NewMessageError(component.MessagePostNotFound, "回复不存在", nil)
		}
		if err != nil {
			return component.NewMessageError(component.MessageContentDeleteFailed, "删除回复失败", component.MessageParams{"error": err.Error()})
		}
		if loaded.PostNo <= 1 {
			return component.NewMessageError(component.MessagePostNotFound, "回复不存在", nil)
		}
		post = loaded
		if post.UserId != userID {
			return component.NewMessageError(component.MessageTopicOperationDenied, "不能删除他人的回复", nil)
		}

		hasChildren, err := posts.HasChildrenTx(tx, postID)
		if err != nil {
			return component.NewMessageError(component.MessageContentDeleteFailed, "删除回复失败", component.MessageParams{"error": err.Error()})
		}
		result.HasChildren = hasChildren
		if post.VisibilityStatus != posts.VisibilityActive || post.RetentionStatus != posts.RetentionNormal {
			if post.VisibilityStatus == posts.VisibilityUserDeleted && post.DeletedBy == userID {
				return nil
			}
			return component.NewMessageError(component.MessageTopicOperationDenied, "该回复已被处理", nil)
		}

		// Reward reversal and content deletion must commit or roll back together.
		// ReversePostRewardTx is idempotent through the deletion tombstone key.
		if err := pointservice.ReversePostRewardTx(tx, post.UserId, postID); err != nil {
			return component.NewMessageError(component.MessageContentDeleteFailed, "删除回复失败", component.MessageParams{"error": err.Error()})
		}
		if hasChildren {
			err = posts.MarkUserDeletedKeepVisibleTx(tx, postID, userID, "")
		} else {
			err = posts.MarkUserDeletedTx(tx, postID, userID, "")
		}
		if err != nil {
			return component.NewMessageError(component.MessageContentDeleteFailed, "删除回复失败", component.MessageParams{"error": err.Error()})
		}
		// 作废待审状态：被删回复不应继续停留在管理审核队列（PRD R1）。
		if err := posts.ResetPendingReviewTx(tx, postID); err != nil {
			return component.NewMessageError(component.MessageContentDeleteFailed, "删除回复失败", component.MessageParams{"error": err.Error()})
		}
		return nil
	})
	if err != nil {
		return DeletePostResult{}, err
	}

	topicEntity := topics.GetSimple(post.TopicId)
	if topicEntity.Id > 0 {
		postservice.SyncTopicPostStats(topicEntity, post, true)
		hotdataserve.ClearTopicListCache()
		llmsservice.ClearCache()
	}
	fileusageservice.HardenTargetFiles(postsTarget(postID), time.Now().Add(RecoveryWindow))
	notificationservice.NullifyContentPreviews(post.TopicId, postID)
	eventbus.Publish(context.Background(), &eventhandlers.ContentDeletedEvent{
		ContentType: string(ContentTypePost),
		TopicId:     post.TopicId,
		PostId:      postID,
		DeletedBy:   userID,
	})
	moderationservice.PostDeleted(userID, postDeletionSnapshot(post), "", userID)
	recordEvent(contentDeleteEvent.EventDeleted, ContentTypePost, postID, post.TopicId, userID)
	return result, nil
}

// DeletePostAsModerator 管理员删除回复（治理删除），作者不可自行恢复。
func DeletePostAsModerator(moderatorID uint64, postID uint64, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return component.NewMessageError(component.MessageRequestInvalidParams, "管理员删除回复必须填写原因", nil)
	}
	var post posts.Entity
	if err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		loaded, err := posts.GetUnscopedTx(tx, postID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return component.NewMessageError(component.MessagePostNotFound, "回复不存在", nil)
		}
		if err != nil {
			return component.NewMessageError(component.MessageContentDeleteFailed, "删除回复失败", component.MessageParams{"error": err.Error()})
		}
		post = loaded
		// 首楼守卫：治理删除话题应走 admin/topics/delete，不允许通过回复删除端点
		// 删除话题首楼（否则话题渲染/搜索索引/统计错乱，review H3）。
		if post.PostNo <= 1 {
			return component.NewMessageError(component.MessageRequestInvalidParams, "不能删除话题首楼，请使用话题删除", nil)
		}
		// 幂等：已 PURGED 或隐私擦除（ACCOUNT_ANONYMIZED）的内容不再处理。
		if post.RetentionStatus == posts.RetentionPurged {
			return nil
		}
		// 幂等：已是治理删除态，直接成功（不覆盖原删除原因）。
		if post.VisibilityStatus == posts.VisibilityModeratorRemoved {
			return nil
		}

		// Reward reversal and moderator deletion must commit or roll back together.
		if err := pointservice.ReversePostRewardTx(tx, post.UserId, postID); err != nil {
			return component.NewMessageError(component.MessageContentDeleteFailed, "删除回复失败", component.MessageParams{"error": err.Error()})
		}
		if err := posts.MarkModeratorRemovedTx(tx, postID, moderatorID, reason); err != nil {
			return component.NewMessageError(component.MessageContentDeleteFailed, "删除回复失败", component.MessageParams{"error": err.Error()})
		}
		// 作废待审状态：被删回复不应继续停留在管理审核队列（PRD R1）。
		if err := posts.ResetPendingReviewTx(tx, postID); err != nil {
			return component.NewMessageError(component.MessageContentDeleteFailed, "删除回复失败", component.MessageParams{"error": err.Error()})
		}
		return nil
	}); err != nil {
		return err
	}
	fileusageservice.HardenTargetFiles(postsTarget(postID), time.Now().Add(RecoveryWindow))
	topicEntity := topics.GetSimple(post.TopicId)
	if topicEntity.Id > 0 {
		postservice.SyncTopicPostStats(topicEntity, post, true)
		hotdataserve.ClearTopicListCache()
		llmsservice.ClearCache()
	}
	notificationservice.NullifyContentPreviews(post.TopicId, postID)
	eventbus.Publish(context.Background(), &eventhandlers.ContentDeletedEvent{
		ContentType:  string(ContentTypePost),
		TopicId:      post.TopicId,
		PostId:       postID,
		DeletedBy:    moderatorID,
		DeleteReason: reason,
	})
	moderationservice.PostDeleted(moderatorID, postDeletionSnapshot(post), reason, moderatorID)
	recordEvent(contentDeleteEvent.EventDeleted, ContentTypePost, postID, post.TopicId, moderatorID)
	return nil
}

// RestoreContent 恢复已删除内容（R3）。仅作者本人、恢复窗口内、非管理端删除可恢复。
func RestoreContent(userID uint64, contentType ContentType, contentID uint64) error {
	switch contentType {
	case ContentTypeTopic:
		topic := topics.UnscopedGet(contentID)
		if topic.Id == 0 || topic.UserId != userID {
			return component.NewMessageError(component.MessageTopicNotFound, "话题不存在", nil)
		}
		if err := checkRestorable(topic.VisibilityStatus, topic.RetentionStatus, topic.DeletedAt.Time); err != nil {
			return err
		}
		if err := topics.Restore(contentID); err != nil {
			return component.NewMessageError(component.MessageContentRestoreFailed, "恢复话题失败", component.MessageParams{"error": err.Error()})
		}
		restoreTopicPosts(topic.Id, userID)
		restoreWikiTopicPages(topic)
		rebuildTopicSearchIndex(contentID)
		fileusageservice.RecoverTargetFiles(topicsTarget(contentID))
		hotdataserve.ClearTopicListCache()
		llmsservice.ClearCache()
		eventbus.Publish(context.Background(), &eventhandlers.ContentRestoredEvent{
			ContentType: string(ContentTypeTopic),
			TopicId:     topic.Id,
		})
		moderationservice.ContentRestored(userID, "topic", topic.Id, topic.Title)
		recordEvent(contentDeleteEvent.EventRestored, ContentTypeTopic, topic.Id, topic.Id, userID)
		return nil
	case ContentTypePost:
		post := posts.UnscopedGet(contentID)
		if post.Id == 0 || post.UserId != userID {
			return component.NewMessageError(component.MessagePostNotFound, "回复不存在", nil)
		}
		if post.PostNo <= 1 {
			return component.NewMessageError(component.MessageContentNotRecoverable, "话题首楼不能单独恢复", nil)
		}
		topic := topics.UnscopedGet(post.TopicId)
		if topic.VisibilityStatus != topics.VisibilityActive || topic.RetentionStatus != topics.RetentionNormal {
			return component.NewMessageError(component.MessageContentNotRecoverable, "话题仍处于删除状态，不能单独恢复首楼", nil)
		}
		// 墓碑态行 deleted_at 为空，以 updated_at 作为删除时刻的近似。
		deletedAt := post.DeletedAt.Time
		if !post.DeletedAt.Valid {
			deletedAt = post.UpdatedAt
		}
		if err := checkRestorable(post.VisibilityStatus, post.RetentionStatus, deletedAt); err != nil {
			return err
		}
		if err := posts.Restore(contentID); err != nil {
			return component.NewMessageError(component.MessageContentRestoreFailed, "恢复回复失败", component.MessageParams{"error": err.Error()})
		}
		// 恢复回补创建奖励（先清除删除回滚墓碑，否则积分永久丢失）。
		reapplyPostReward(post.UserId, post.Id)
		topicEntity := topics.GetSimple(post.TopicId)
		if topicEntity.Id > 0 {
			var activePosts []*posts.Entity
			if err := posts.ListByTopicID(topicEntity.Id, &activePosts); err == nil {
				_ = postservice.RebuildTopicPostStats(topicEntity, activePosts)
			}
			hotdataserve.ClearTopicListCache()
			llmsservice.ClearCache()
		}
		fileusageservice.RecoverTargetFiles(postsTarget(post.Id))
		moderationservice.ContentRestored(userID, "post", post.Id, "")
		recordEvent(contentDeleteEvent.EventRestored, ContentTypePost, post.Id, post.TopicId, userID)
		return nil
	default:
		return component.NewMessageError(component.MessageRequestInvalidParams, "无效的内容类型", nil)
	}
}

func checkRestorable(visibility string, retention string, deletedAt time.Time) error {
	if visibility == topics.VisibilityModeratorRemoved {
		return component.NewMessageError(component.MessageContentNotRecoverable, "该内容不可由作者恢复", nil)
	}
	if retention == topics.RetentionPurged {
		return component.NewMessageError(component.MessageContentNotRecoverable, "该内容已永久删除", nil)
	}
	if retention != topics.RetentionRecoverable {
		return component.NewMessageError(component.MessageContentNotRecoverable, "该内容不在恢复期", nil)
	}
	if time.Since(deletedAt) > RecoveryWindow {
		return component.NewMessageError(component.MessageContentRecoveryExpired, "已超出恢复窗口", nil)
	}
	return nil
}

func restoreTopicPosts(topicID uint64, operatorID uint64) {
	// 话题删除时级联软删的回复一并恢复；墓碑态（deleted_at 为空）行不受影响。
	posts.RestoreCascadeDeletedByTopicID(topicID, operatorID, topicDeleteCascadeReason(topicID))
	topic := topics.Get(topicID)
	if topic.Id == 0 {
		return
	}
	firstPost := posts.UnscopedGet(topic.FirstPostId)
	if firstPost.Id > 0 && firstPost.VisibilityStatus == posts.VisibilityUserDeleted &&
		firstPost.RetentionStatus == posts.RetentionRecoverable && firstPost.DeletedBy == operatorID &&
		firstPost.DeleteReason == topicDeleteCascadeReason(topicID) {
		if err := posts.Restore(firstPost.Id); err != nil {
			slog.Error("failed to restore topic first post", "topicId", topicID, "postId", firstPost.Id, "error", err)
		}
	}
	var activePosts []*posts.Entity
	if err := posts.ListUnscopedByTopicID(topicID, &activePosts); err != nil {
		slog.Error("failed to load posts for topic stats rebuild", "topicId", topicID, "error", err)
		return
	}
	if err := postservice.RebuildTopicPostStats(topic, activePosts); err != nil {
		slog.Error("failed to rebuild topic stats", "topicId", topicID, "error", err)
	}
	for _, post := range activePosts {
		if post != nil && post.VisibilityStatus == posts.VisibilityActive {
			fileusageservice.RecoverTargetFiles(postsTarget(post.Id))
		}
	}
}

// restoreWikiTopicPages 恢复 wiki 分站页面话题时一并恢复其 wiki_pages 与全部修订
// （清除 deleted_at）。删除时页面/修订随话题软删；只恢复 topics 会出现
// "话题可见、页面消失"的幽灵状态（review wiki soft-delete alignment）。
func restoreWikiTopicPages(topic topics.Entity) {
	if topic.TopicType != topics.TopicTypeWiki {
		return
	}
	var page wikiPages.Entity
	if err := dbconnect.Connect().Unscoped().Table("wiki_pages").
		Where("topic_id = ?", topic.Id).First(&page).Error; err != nil || page.Id == 0 {
		return
	}
	if err := dbconnect.Connect().Unscoped().Table("wiki_pages").
		Where("id = ?", page.Id).Update("deleted_at", gorm.Expr("NULL")).Error; err != nil {
		slog.Error("failed to restore wiki page", "pageId", page.Id, "error", err)
	}
	if err := dbconnect.Connect().Unscoped().Table("wiki_page_revisions").
		Where("page_id = ?", page.Id).Update("deleted_at", gorm.Expr("NULL")).Error; err != nil {
		slog.Error("failed to restore wiki page revisions", "pageId", page.Id, "error", err)
	}
	if err := searchservice.IndexWikiPageDocuments(page.Id); err != nil {
		slog.Warn("failed to restore wiki page search index", "pageId", page.Id, "error", err)
	}
}

// deleteWikiPageSearchIndex 清理 Wiki 页面删除后留下的段落索引。
// 搜索索引是可重建投影，清理失败不应阻断内容删除；下一次全量重建或页面同步会修复它。
func deleteWikiPageSearchIndex(pageID uint64) {
	if pageID == 0 {
		return
	}
	if err := searchservice.DeleteWikiPageDocuments(pageID); err != nil {
		slog.Warn("failed to delete wiki page search index", "pageId", pageID, "error", err)
	}
}

// RestoreTopicAsModerator 管理端恢复被治理删除的话题（PRD R7 / review MEDIUM-2）。
// 仅可恢复 MODERATOR_REMOVED 的话题：作者不可自行恢复管理端删除，管理端是
// 唯一的恢复通道。恢复级联删除的回复、重建搜索索引、恢复附件可见性并写审计。
func RestoreTopicAsModerator(moderatorID uint64, topicID uint64) error {
	topic := topics.UnscopedGet(topicID)
	if topic.Id == 0 {
		return component.NewMessageError(component.MessageTopicNotFound, "话题不存在", nil)
	}
	if topic.VisibilityStatus != topics.VisibilityModeratorRemoved || topic.RetentionStatus != topics.RetentionRecoverable {
		return component.NewMessageError(component.MessageContentNotRecoverable, "该话题不可由管理端恢复", nil)
	}
	if err := topics.Restore(topicID); err != nil {
		return component.NewMessageError(component.MessageContentRestoreFailed, "恢复话题失败", component.MessageParams{"error": err.Error()})
	}
	restoreModeratorRemovedTopicPosts(topic, moderatorID)
	restoreWikiTopicPages(topic)
	rebuildTopicSearchIndex(topicID)
	fileusageservice.RecoverTargetFiles(topicsTarget(topicID))
	hotdataserve.ClearTopicListCache()
	llmsservice.ClearCache()
	eventbus.Publish(context.Background(), &eventhandlers.ContentRestoredEvent{
		ContentType: string(ContentTypeTopic),
		TopicId:     topicID,
	})
	moderationservice.ContentRestored(moderatorID, "topic", topic.Id, topic.Title)
	recordEvent(contentDeleteEvent.EventRestored, ContentTypeTopic, topicID, topicID, moderatorID)
	return nil
}

// restoreModeratorRemovedTopicPosts 恢复管理端治理删除话题时级联软删的回复。
// 只恢复本次删除操作标记的 MODERATOR_REMOVED 行，不误恢复独立删除的回复。
func restoreModeratorRemovedTopicPosts(topic topics.Entity, moderatorID uint64) {
	posts.RestoreCascadeDeletedByTopicIDWithVisibility(topic.Id, moderatorID, topic.DeleteReason, posts.VisibilityModeratorRemoved)
	firstPost := posts.UnscopedGet(topic.FirstPostId)
	if firstPost.Id > 0 && firstPost.VisibilityStatus == posts.VisibilityModeratorRemoved &&
		firstPost.RetentionStatus == posts.RetentionRecoverable && firstPost.DeletedBy == moderatorID &&
		firstPost.DeleteReason == topic.DeleteReason {
		if err := posts.Restore(firstPost.Id); err != nil {
			slog.Error("failed to restore moderator-removed topic first post", "topicId", topic.Id, "postId", firstPost.Id, "error", err)
		}
	}
	var activePosts []*posts.Entity
	if err := posts.ListUnscopedByTopicID(topic.Id, &activePosts); err != nil {
		slog.Error("failed to load posts for moderator restore stats rebuild", "topicId", topic.Id, "error", err)
		return
	}
	if err := postservice.RebuildTopicPostStats(topic, activePosts); err != nil {
		slog.Error("failed to rebuild topic stats after moderator restore", "topicId", topic.Id, "error", err)
	}
	for _, post := range activePosts {
		if post != nil && post.VisibilityStatus == posts.VisibilityActive {
			fileusageservice.RecoverTargetFiles(postsTarget(post.Id))
		}
	}
}

func topicDeleteCascadeReason(topicID uint64) string {
	return "topic_delete:" + fmt.Sprint(topicID)
}

func rebuildTopicSearchIndex(topicID uint64) {
	topic := topics.Get(topicID)
	if topic.Id == 0 {
		return
	}
	firstPost := posts.Get(topic.FirstPostId)
	if _, err := searchservice.BuildSingleTopicSearchDocument(&topic, &firstPost); err != nil {
		slog.Error("failed to rebuild topic search document after restore", "topicId", topicID, "err", err)
	}
}

// PurgeContent 永久删除（R4）：置 PURGED、清理附件引用、通知预览置空。
// 治理证据与审计日志独立保留，不受永久删除影响。
func PurgeContent(userID uint64, contentType ContentType, contentID uint64, reason string) error {
	switch contentType {
	case ContentTypeTopic:
		topic := topics.UnscopedGet(contentID)
		if topic.Id == 0 || topic.UserId != userID {
			return component.NewMessageError(component.MessageTopicNotFound, "话题不存在", nil)
		}
		if topic.VisibilityStatus == topics.VisibilityModeratorRemoved {
			return component.NewMessageError(component.MessageContentNotRecoverable, "治理删除内容不能通过隐私删除绕过审核", nil)
		}
		if err := checkPurgeable(topic.VisibilityStatus, topic.RetentionStatus); err != nil {
			return err
		}
		if err := topics.MarkPurged(contentID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) && topic.RetentionStatus == topics.RetentionPurged {
				return nil
			}
			return component.NewMessageError(component.MessageContentPurgeFailed, "永久删除失败", component.MessageParams{"error": err.Error()})
		}
		purgeTopicPosts(contentID, topic.UserId)
		fileusageservice.PurgeTargetFiles(topicsTarget(contentID))
		notificationservice.NullifyContentPreviews(contentID, 0)
		clearTopicCaches(contentID)
		eventbus.Publish(context.Background(), &eventhandlers.ContentDeletedEvent{
			ContentType:  string(ContentTypeTopic),
			TopicId:      contentID,
			DeletedBy:    userID,
			DeleteReason: reason,
		})
		moderationservice.ContentPurged(userID, "topic", topic.Id, topic.Title, reason)
		recordEvent(contentDeleteEvent.EventPermanentDelete, ContentTypeTopic, contentID, contentID, userID)
		return nil
	case ContentTypePost:
		post := posts.UnscopedGet(contentID)
		if post.Id == 0 || post.UserId != userID {
			return component.NewMessageError(component.MessagePostNotFound, "回复不存在", nil)
		}
		if post.VisibilityStatus == posts.VisibilityModeratorRemoved {
			return component.NewMessageError(component.MessageContentNotRecoverable, "治理删除内容不能通过隐私删除绕过审核", nil)
		}
		if err := checkPurgeable(post.VisibilityStatus, post.RetentionStatus); err != nil {
			return err
		}
		if err := posts.MarkPurged(contentID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) && post.RetentionStatus == posts.RetentionPurged {
				return nil
			}
			return component.NewMessageError(component.MessageContentPurgeFailed, "永久删除失败", component.MessageParams{"error": err.Error()})
		}
		fileusageservice.PurgeTargetFiles(postsTarget(contentID))
		notificationservice.NullifyContentPreviews(post.TopicId, contentID)
		topicEntity := topics.GetSimple(post.TopicId)
		if topicEntity.Id > 0 {
			hotdataserve.ClearTopicListCache()
			llmsservice.ClearCache()
		}
		eventbus.Publish(context.Background(), &eventhandlers.ContentDeletedEvent{
			ContentType:  string(ContentTypePost),
			TopicId:      post.TopicId,
			PostId:       contentID,
			DeletedBy:    userID,
			DeleteReason: reason,
		})
		moderationservice.ContentPurged(userID, "post", post.Id, "", reason)
		recordEvent(contentDeleteEvent.EventPermanentDelete, ContentTypePost, contentID, post.TopicId, userID)
		return nil
	default:
		return component.NewMessageError(component.MessageRequestInvalidParams, "无效的内容类型", nil)
	}
}

func checkPurgeable(visibility string, retention string) error {
	if visibility != topics.VisibilityUserDeleted || retention != topics.RetentionRecoverable {
		return component.NewMessageError(component.MessageContentNotRecoverable, "该内容不允许由作者永久删除", nil)
	}
	return nil
}

// PrivacyEraseContent is the explicit privacy path. Unlike ordinary purge it
// can process active user-owned content, but it always makes the row hidden,
// unrecoverable, and clears reply body fields before downstream cleanup.
func PrivacyEraseContent(userID uint64, contentType ContentType, contentID uint64) error {
	const reason = "privacy_erase"
	switch contentType {
	case ContentTypeTopic:
		topic := topics.UnscopedGet(contentID)
		if topic.Id == 0 || topic.UserId != userID {
			return component.NewMessageError(component.MessageTopicNotFound, "话题不存在", nil)
		}
		if topic.VisibilityStatus == topics.VisibilityModeratorRemoved {
			return component.NewMessageError(component.MessageContentNotRecoverable, "治理删除内容不能通过隐私删除绕过审核", nil)
		}
		// 级联前预检：话题内同作者回复若存在治理删除（MODERATOR_REMOVED），
		// 整体拒绝——否则作者可保留 ACTIVE 话题、让版主治理删除其一条自回复后，
		// 再对父话题隐私擦除，级联路径会把治理删除回复改写为
		// ACCOUNT_ANONYMIZED/PURGED 并清空正文，绕过治理证据留存（review）。
		var topicPosts []*posts.Entity
		if err := posts.ListUnscopedByTopicID(contentID, &topicPosts); err != nil {
			return component.NewMessageError(component.MessageContentPurgeFailed, "隐私删除失败", component.MessageParams{"error": err.Error()})
		}
		for _, post := range topicPosts {
			if post == nil || post.UserId != userID {
				continue
			}
			if post.VisibilityStatus == posts.VisibilityModeratorRemoved {
				return component.NewMessageError(component.MessageContentNotRecoverable, "话题内存在治理删除的回复，不能通过隐私删除绕过审核", nil)
			}
		}
		if err := topics.MarkPrivacyErased(contentID, userID, reason); err != nil {
			return component.NewMessageError(component.MessageContentPurgeFailed, "隐私删除失败", component.MessageParams{"error": err.Error()})
		}
		for _, post := range topicPosts {
			if post == nil || post.UserId != userID {
				continue
			}
			_ = posts.MarkPrivacyErased(post.Id, userID, reason)
			fileusageservice.PurgeTargetFiles(postsTarget(post.Id))
		}
		fileusageservice.PurgeTargetFiles(topicsTarget(contentID))
		notificationservice.NullifyContentPreviews(contentID, 0)
		clearTopicCaches(contentID)
		eventbus.Publish(context.Background(), &eventhandlers.ContentDeletedEvent{ContentType: string(ContentTypeTopic), TopicId: contentID, DeletedBy: userID, DeleteReason: reason})
		moderationservice.ContentPurged(userID, "topic", contentID, topic.Title, reason)
		recordEvent(contentDeleteEvent.EventPrivacyDelete, ContentTypeTopic, contentID, contentID, userID)
		return nil
	case ContentTypePost:
		post := posts.UnscopedGet(contentID)
		if post.Id == 0 || post.UserId != userID {
			return component.NewMessageError(component.MessagePostNotFound, "回复不存在", nil)
		}
		if post.VisibilityStatus == posts.VisibilityModeratorRemoved {
			return component.NewMessageError(component.MessageContentNotRecoverable, "治理删除内容不能通过隐私删除绕过审核", nil)
		}
		if err := posts.MarkPrivacyErased(contentID, userID, reason); err != nil {
			return component.NewMessageError(component.MessageContentPurgeFailed, "隐私删除失败", component.MessageParams{"error": err.Error()})
		}
		fileusageservice.PurgeTargetFiles(postsTarget(contentID))
		notificationservice.NullifyContentPreviews(post.TopicId, contentID)
		clearTopicCaches(post.TopicId)
		eventbus.Publish(context.Background(), &eventhandlers.ContentDeletedEvent{ContentType: string(ContentTypePost), TopicId: post.TopicId, PostId: contentID, DeletedBy: userID, DeleteReason: reason})
		moderationservice.ContentPurged(userID, "post", contentID, "", reason)
		recordEvent(contentDeleteEvent.EventPrivacyDelete, ContentTypePost, contentID, post.TopicId, userID)
		return nil
	default:
		return component.NewMessageError(component.MessageRequestInvalidParams, "无效的内容类型", nil)
	}
}

func purgeTopicPosts(topicID uint64, ownerID uint64) {
	var topicPosts []*posts.Entity
	if err := posts.ListUnscopedByTopicID(topicID, &topicPosts); err != nil {
		slog.Error("failed to load topic posts for purge", "topicId", topicID, "error", err)
		return
	}
	for _, post := range topicPosts {
		if post == nil {
			continue
		}
		// 话题作者本人的回复（含 ACTIVE 与已进入生命周期）随话题永久删除一起清空：
		// 作者对本人内容的永久删除应彻底生效（PRD R4/R12），否则自回帖的正文与
		// 附件会永远留在库中且附件仍可公开下载（review H2）。
		if post.UserId == ownerID {
			if err := posts.MarkPurgedOwned(post.Id, ownerID); err != nil {
				slog.Error("failed to purge owner topic post", "topicId", topicID, "postId", post.Id, "err", err)
				continue
			}
			fileusageservice.PurgeTargetFiles(postsTarget(post.Id))
			continue
		}
		// 其他用户已进入删除生命周期（级联软删/墓碑/独立软删）的回复：清空。
		if post.VisibilityStatus != posts.VisibilityActive {
			if err := posts.MarkPurged(post.Id); err != nil {
				slog.Error("failed to purge topic post", "topicId", topicID, "postId", post.Id, "err", err)
				continue
			}
			fileusageservice.PurgeTargetFiles(postsTarget(post.Id))
			continue
		}
		// 其他用户仍 ACTIVE 的回复属于他人内容，正文保留（PRD Out of Scope：
		// 不允许删除他人内容），但话题已永久删除、回复不可达，其附件不得再
		// 公开下载——转入 RECOVERING，由 retention 定时任务清理。
		fileusageservice.HardenTargetFiles(postsTarget(post.Id), time.Now().Add(RecoveryWindow))
	}
}

func postDeletionSnapshot(post posts.Entity) moderationservice.PostSnapshot {
	topic := topics.UnscopedGet(post.TopicId)
	authorName := ""
	if author, err := users.Get(post.UserId); err == nil && author.Id > 0 {
		authorName = author.Username
	}
	return moderationservice.PostSnapshot{
		PostId:       post.Id,
		TopicId:      post.TopicId,
		TopicTitle:   topic.Title,
		PostNo:       post.PostNo,
		PostAuthorId: post.UserId,
		PostAuthor:   authorName,
		Excerpt:      excerpt(post.Content),
	}
}

func excerpt(content string) string {
	runes := []rune(content)
	if len(runes) > 160 {
		return string(runes[:160])
	}
	return content
}

// ExpireRecoverableBatch 供 retention scheduler 调用：将超过恢复窗口的 RECOVERABLE 内容置为 PURGED 并清理。
func ExpireRecoverableBatch(limit int) error {
	if limit <= 0 {
		limit = 200
	}
	before := time.Now().Add(-RecoveryWindow)

	expiredTopics := topics.ExpireRecoverable(before, limit)
	for _, topic := range expiredTopics {
		if reports.HasOpenForTopic(topic.Id) {
			continue
		}
		_ = topics.MarkPurged(topic.Id)
		purgeTopicPosts(topic.Id, topic.UserId)
		fileusageservice.PurgeTargetFiles(topicsTarget(topic.Id))
		notificationservice.NullifyContentPreviews(topic.Id, 0)
		hotdataserve.ClearTopicListCache()
		llmsservice.ClearCache()
		slog.Info("retention: topic purged after recovery window", "topicId", topic.Id)
	}

	expiredPosts := posts.ExpireRecoverable(before, limit)
	for _, post := range expiredPosts {
		if reports.HasOpenForTopic(post.TopicId) {
			continue
		}
		_ = posts.MarkPurged(post.Id)
		fileusageservice.PurgeTargetFiles(postsTarget(post.Id))
		notificationservice.NullifyContentPreviews(post.TopicId, post.Id)
		slog.Info("retention: post purged after recovery window", "postId", post.Id)
	}
	return nil
}

// ExpireEvidenceSnapshotsBatch 清理超过保留期的已结案举报证据快照。
// LEGAL_HOLD / EVIDENCE_HOLD 话题上的举报由 ClearExpiredEvidenceSnapshots 自行跳过。
func ExpireEvidenceSnapshotsBatch(limit int) error {
	if limit <= 0 {
		limit = 200
	}
	cleared, err := reports.ClearExpiredEvidenceSnapshots(time.Now().Add(-EvidenceSnapshotRetention), limit)
	if err != nil {
		return err
	}
	slog.Info("retention: expired evidence snapshots cleared", "count", cleared)
	return nil
}

func topicsTarget(topicID uint64) fileusageservice.TargetRef {
	return fileusageservice.TargetRef{TargetType: "topic", TargetID: topicID}
}

func postsTarget(postID uint64) fileusageservice.TargetRef {
	return fileusageservice.TargetRef{TargetType: "post", TargetID: postID}
}

// reapplyPostReward 恢复回复时回补创建奖励。先清除删除回滚墓碑（post-deleted:ID），
// 否则 applyPointsTx 对已回滚过的 sourceKey 会跳过加分，导致"删除→恢复"积分永久丢失。
func reapplyPostReward(userId, postID uint64) {
	if userId == 0 || postID == 0 {
		return
	}
	if err := pointservice.ClearPostDeletedTombstone(postID); err != nil {
		slog.Error("clear post-deleted tombstone on restore failed", "userId", userId, "postId", postID, "err", err)
	}
	if err := pointservice.RewardPoints(userId, pointservice.PostCreatedReward, pointservice.PointsActionPostCreated,
		fmt.Sprintf("post:%d", postID)); err != nil {
		slog.Error("reapply post reward on restore failed", "userId", userId, "postId", postID, "err", err)
	}
}

// DeleteAllUserContent 注销账号时删除该用户全部话题与回复（PRD R10 mode=delete）。
// 删除自己话题不会删除他人回复；删除回复只删自己的回复，他人内容不受影响。
// 用 id 递减游标推进分页：即使个别删除持续失败（话题/回复保持 ACTIVE），
// 游标也逐批推进，避免"从头再查 + 恒失败"导致的无限循环挂死（review M2）。
// 失败项记录日志并跳过（注销尽力而为，不因单条失败阻断整个注销）。
func DeleteAllUserContent(userID uint64) error {
	const batchSize = 100
	// 先删话题（级联删除该话题下所有回复，含他人回复——话题删除语义即整体移除讨论）。
	var topicCursor uint64
	for {
		activeTopics := topics.GetActiveByUserPage(userID, topicCursor, batchSize)
		if len(activeTopics) == 0 {
			break
		}
		for _, topic := range activeTopics {
			if err := DeleteTopicByUser(userID, topic.Id); err != nil {
				slog.Warn("delete user topic on account close failed", "userId", userID, "topicId", topic.Id, "err", err)
			}
		}
		topicCursor = activeTopics[len(activeTopics)-1].Id
		if topicCursor == 0 {
			break
		}
	}
	// 再删 wiki 分站页面话题（review P1：GetActiveByUserPage 的 topic_type=forum
	// 过滤会导致 wiki 页面漏删）。先删页面与修订行，再走话题删除生命周期，
	// 避免 topic 删除后残留孤儿 wiki_pages/wiki_page_revisions。
	var wikiCursor uint64
	for {
		activeWikiTopics := topics.GetActiveWikiByUserPage(userID, wikiCursor, batchSize)
		if len(activeWikiTopics) == 0 {
			break
		}
		for _, topic := range activeWikiTopics {
			if err := DeleteTopicByUser(userID, topic.Id); err != nil {
				slog.Warn("delete user wiki topic on account close failed", "userId", userID, "topicId", topic.Id, "err", err)
			}
		}
		wikiCursor = activeWikiTopics[len(activeWikiTopics)-1].Id
		if wikiCursor == 0 {
			break
		}
	}
	// 他人 wiki 页面上 editor_id=注销者 的修订一并清理（对齐"删除全部本人内容"
	// 策略）。此刻本人页面及其修订已在上方 DeleteByPage 清理，DeleteByEditor
	// 只命中他人页面残留的本人修订，幂等无副作用。
	if err := wikiPageRevisions.DeleteByEditor(userID); err != nil {
		slog.Warn("delete wiki revisions by editor on account close failed", "userId", userID, "err", err)
	}
	// 再删剩余未删除的本人回复（他人话题下的回复）。
	var postCursor uint64
	for {
		activePosts := posts.GetActiveByUserPage(userID, postCursor, batchSize)
		if len(activePosts) == 0 {
			break
		}
		for _, post := range activePosts {
			if _, err := DeletePostByUser(userID, post.Id); err != nil {
				slog.Warn("delete user post on account close failed", "userId", userID, "postId", post.Id, "err", err)
			}
		}
		postCursor = activePosts[len(activePosts)-1].Id
		if postCursor == 0 {
			break
		}
	}
	return nil
}

// recordEvent 记录删除生命周期埋点事件（PRD R14）。
// 事件在状态变更成功后记录，失败仅记日志，不影响删除主流程。
func recordEvent(eventType contentDeleteEvent.EventType, contentType ContentType, contentID uint64, topicID uint64, actorID uint64) {
	if err := contentDeleteEvent.Record(contentDeleteEvent.Entity{
		EventType:   string(eventType),
		ContentType: string(contentType),
		ContentID:   contentID,
		ActorID:     actorID,
		TopicID:     topicID,
	}); err != nil {
		slog.Error("record content delete event failed", "eventType", eventType, "contentType", contentType, "contentId", contentID, "err", err)
	}
}

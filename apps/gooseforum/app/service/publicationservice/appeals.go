package publicationservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationAppeals"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/realtimeservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/unreadservice"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrAppealPermission = errors.New("appeal not found or permission denied")
	ErrAppealReason     = errors.New("appeal explanation is required and must be at most 512 characters")
	// 以下细分错误仍满足 errors.Is(ErrUnavailable / ErrAppealPermission)，只为让客户端给出具体文案。
	ErrAppealFrozen       = fmt.Errorf("%w: frozen accounts cannot appeal", ErrAppealPermission)
	ErrAppealDuplicate    = fmt.Errorf("%w: this revision was already appealed", ErrUnavailable)
	ErrAppealIneligible   = fmt.Errorf("%w: this revision cannot be appealed", ErrUnavailable)
	ErrAppealProcessed    = fmt.Errorf("%w: appeal already decided", ErrUnavailable)
	ErrAppealStale        = fmt.Errorf("%w: appeal target changed before the decision", ErrUnavailable)
	ErrAppealAuthorFrozen = fmt.Errorf("%w: frozen authors cannot be restored", ErrUnavailable)
)

func appealable(topic topics.Entity, post posts.Entity, revision postRevisions.Entity) bool {
	return available(topic, post) && (topic.TopicType != topics.TopicTypeWiki || post.PostNo > 1) && post.RetentionStatus != posts.RetentionPurged && topic.RetentionStatus != topics.RetentionPurged && (post.PublishedRevisionId == 0 || (post.ProcessStatus != posts.ProcessStatusBlocked && topic.ProcessStatus != topics.ProcessStatusBlocked)) && post.LatestRevisionId == revision.Id && revision.PostId == post.Id && revision.ProcessStatus == posts.ProcessStatusBlocked && (post.PostNo == 1 || topic.ProcessStatus == topics.ProcessStatusNormal)
}

// SubmitAppeal fences ownership and the rejected version under the same content
// locks as edits. A database constraint guarantees one submission per revision.
func SubmitAppeal(ctx context.Context, userID uint64, kind string, targetID, revisionID uint64, reason string) (AppealItem, error) {
	item := AppealItem{}
	appeal := moderationAppeals.Entity{}
	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > 512 {
		return item, ErrAppealReason
	}
	if userID == 0 || (kind != "topic" && kind != "post") || revisionID == 0 {
		return item, ErrAppealPermission
	}
	err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		var revision postRevisions.Entity
		if err := tx.First(&revision, revisionID).Error; err != nil {
			return err
		}
		var post posts.Entity
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&post, revision.PostId).Error; err != nil {
			return err
		}
		var topic topics.Entity
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&topic, post.TopicId).Error; err != nil {
			return err
		}
		matches := kind == "topic" && post.PostNo == 1 && topic.Id == targetID || kind == "post" && post.PostNo > 1 && post.Id == targetID
		if !matches || post.UserId != userID {
			return ErrAppealPermission
		}
		var existing int64
		if err := tx.Model(&moderationAppeals.Entity{}).Where("revision_id = ?", revisionID).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return ErrAppealDuplicate
		}
		if !appealable(topic, post, revision) {
			return ErrAppealIneligible
		}
		if post.PostNo > 1 {
			if err := appealParentPublicTx(tx, topic); errors.Is(err, ErrUnavailable) {
				return ErrAppealIneligible
			} else if err != nil {
				return err
			}
		}
		author, err := users.GetInteractionUserTx(tx, userID)
		if err != nil {
			return err
		}
		if author.IsFrozen == users.StatusFrozen {
			return ErrAppealFrozen
		}
		appeal = moderationAppeals.Entity{PostId: post.Id, RevisionId: revisionID, Reason: reason, Status: moderationAppeals.Pending, OriginalReviewReason: revision.ReviewReason, OriginalReviewActorId: revision.ReviewActorId, OriginalReviewedAt: revision.ReviewedAt}
		if err := tx.Create(&appeal).Error; err != nil {
			return err
		}
		payload := eventNotification.RedactReviewRejectedPayload(eventNotification.NotificationPayload{})
		payload.TemplateKey = eventNotification.TemplateAppealSubmitted
		payload.TopicId = topic.Id
		payload.Extra.ProfileURL = "/settings?tab=content"
		notice := eventNotification.Entity{UserId: userID, TopicID: topic.Id, EventType: eventNotification.EventTypeReviewPending, Payload: payload}
		if err := tx.Create(&notice).Error; err != nil {
			return err
		}
		item = AppealItem{Id: appeal.Id, Kind: kind, TargetId: targetID, RevisionId: revisionID, TopicId: topic.Id, PostNo: post.PostNo, Title: revision.Title, Content: revision.Content, Images: revision.ImageUrls, Reason: reason, OriginalReviewReason: revision.ReviewReason, Status: appeal.Status, CreatedAt: appeal.CreatedAt}
		if post.PostNo > 1 {
			item.Title = topic.Title
		}
		if item.Images == nil {
			item.Images = []string{}
		}
		return enqueueEffect(tx, Effect{RevisionId: revisionID, NotificationId: notice.Id, NotificationOnly: true, AppealId: appeal.Id})
	})
	if err == nil {
		unreadservice.Invalidate(userID)
		realtimeservice.DefaultHub.Publish(userID, realtimeservice.Event{Type: realtimeservice.EventNotificationsChanged})
		realtimeservice.DefaultHub.Publish(userID, realtimeservice.Event{Type: realtimeservice.EventContentChanged})
	}
	return item, err
}

func canReviewAppealTx(tx *gorm.DB, actorID uint64, topic topics.Entity, post posts.Entity, revision postRevisions.Entity) (bool, error) {
	if actorID == 0 || actorID == post.UserId {
		return false, nil
	}
	global, ids, err := moderationservice.ScopeForUserTx(tx, actorID)
	if err != nil {
		return false, err
	}
	if global {
		return true, nil
	}
	matches := func(categories []uint64) bool {
		for _, id := range ids {
			if slices.Contains(categories, id) {
				return true
			}
		}
		return false
	}
	return matches(topic.CategoryIds) && (post.PostNo > 1 || matches(revision.CategoryIds)), nil
}

func decideAppealTx(tx *gorm.DB, appealID, actorID uint64, approved bool, reason string) error {
	status := moderationAppeals.Rejected
	if approved {
		status = moderationAppeals.Approved
	}
	return tx.Model(&moderationAppeals.Entity{}).Where("id = ? AND status = ?", appealID, moderationAppeals.Pending).Updates(map[string]any{"status": status, "review_reason": strings.TrimSpace(reason), "review_actor_id": actorID, "reviewed_at": time.Now()}).Error
}

// ReviewAppeal reuses Review's transaction and publication effects without
// reopening rejected content or exposing a pending window to the worker.
func ReviewAppeal(ctx context.Context, appealID, actorID uint64, approve bool, reason string) error {
	reason = strings.TrimSpace(reason)
	if (!approve && reason == "") || utf8.RuneCountInString(reason) > 512 {
		return ErrAppealReason
	}
	var appeal moderationAppeals.Entity
	if err := db.ConnectContext(ctx).First(&appeal, appealID).Error; err != nil {
		return err
	}
	action := moderationDecision.ActionBlock
	if approve {
		action = moderationDecision.ActionAllow
	}
	return review(ctx, appeal.RevisionId, action, reason, actorID, appealID)
}

// AppealItem deliberately omits author/actor identities, including anonymous owners.
type AppealItem struct {
	Id         uint64   `json:"id"`
	Kind       string   `json:"kind"`
	TargetId   uint64   `json:"targetId"`
	RevisionId uint64   `json:"revisionId"`
	TopicId    uint64   `json:"topicId"`
	PostNo     uint64   `json:"postNo"`
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Images     []string `gorm:"serializer:json" json:"images"`
	Reason     string   `json:"reason"`
	// OriginalReviewReason is the rejection being appealed, so reviewers judge both sides.
	OriginalReviewReason string     `json:"originalReviewReason"`
	Status               string     `json:"status"`
	ReviewReason         string     `json:"reviewReason"`
	CreatedAt            time.Time  `json:"createdAt"`
	ReviewedAt           *time.Time `json:"reviewedAt"`
}

const appealStaleSQL = `(a.status = 'pending' AND (p.latest_revision_id <> a.revision_id OR r.process_status <> 1 OR (p.published_revision_id <> 0 AND (p.process_status = 1 OR t.process_status = 1)) OR (p.post_no > 1 AND t.process_status <> 0)))`

func appealQuery(tx *gorm.DB) *gorm.DB {
	return tx.Table("moderation_appeals AS a").Joins("JOIN post_revisions AS r ON r.id = a.revision_id AND r.post_id = a.post_id").Joins("JOIN posts AS p ON p.id = a.post_id").Joins("JOIN topics AS t ON t.id = p.topic_id").Joins("LEFT JOIN posts AS fp ON fp.id = t.first_post_id").Where("(p.post_no = 1 OR (fp.deleted_at IS NULL AND fp.visibility_status = ? AND fp.process_status = ? AND fp.retention_status <> ?))", posts.VisibilityActive, posts.ProcessStatusNormal, posts.RetentionPurged).Where("p.deleted_at IS NULL AND t.deleted_at IS NULL AND p.visibility_status = ? AND t.visibility_status = ? AND t.status = 1 AND (t.topic_type <> ? OR p.post_no > 1) AND p.retention_status <> ? AND t.retention_status <> ?", posts.VisibilityActive, topics.VisibilityActive, topics.TopicTypeWiki, posts.RetentionPurged, topics.RetentionPurged)
}

func appealItems(query *gorm.DB, items *[]AppealItem) error {
	return query.Select(`a.id, CASE WHEN p.post_no = 1 THEN 'topic' ELSE 'post' END AS kind, CASE WHEN p.post_no = 1 THEN t.id ELSE p.id END AS target_id, a.revision_id, t.id AS topic_id, p.post_no, CASE WHEN p.post_no = 1 THEN r.title ELSE t.title END AS title, r.content, r.image_urls AS images, a.reason, a.original_review_reason, CASE WHEN ` + appealStaleSQL + ` THEN 'stale' ELSE a.status END AS status, a.review_reason, a.created_at, a.reviewed_at`).Order("a.id DESC").Scan(items).Error
}

// OwnerAppeals returns only active retained content belonging to the caller.
func OwnerAppeals(ctx context.Context, userID uint64, kind string, targetID uint64) ([]AppealItem, error) {
	items := []AppealItem{}
	if userID == 0 || (kind != "topic" && kind != "post") {
		return items, ErrAppealPermission
	}
	err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		var post posts.Entity
		q := tx.Where("user_id = ?", userID)
		if kind == "topic" {
			q = q.Where("topic_id = ? AND post_no = 1", targetID)
		} else {
			q = q.Where("id = ? AND post_no > 1", targetID)
		}
		if err := q.Take(&post).Error; err != nil {
			return err
		}
		var topic topics.Entity
		if err := tx.First(&topic, post.TopicId).Error; err != nil {
			return err
		}
		if !available(topic, post) || (topic.TopicType == topics.TopicTypeWiki && post.PostNo == 1) || post.RetentionStatus == posts.RetentionPurged || topic.RetentionStatus == topics.RetentionPurged {
			return ErrAppealPermission
		}
		return appealItems(appealQuery(tx).Where("p.id = ? AND p.user_id = ?", post.Id, userID).Limit(50), &items)
	})
	for i := range items {
		if items[i].Images == nil {
			items[i].Images = []string{}
		}
	}
	return items, err
}

// AppealForReviewer returns one appeal under the same scope rules as decisions.
// ErrAppealPermission hides whether an out-of-scope appeal exists; ErrAppealStale
// means its content is no longer reviewable.
func AppealForReviewer(ctx context.Context, reviewerID, appealID uint64) (AppealItem, error) {
	item := AppealItem{}
	err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		var appeal moderationAppeals.Entity
		if err := tx.First(&appeal, appealID).Error; err != nil {
			return err
		}
		var revision postRevisions.Entity
		if err := tx.First(&revision, appeal.RevisionId).Error; err != nil {
			return err
		}
		var post posts.Entity
		if err := tx.First(&post, appeal.PostId).Error; err != nil {
			return err
		}
		var topic topics.Entity
		if err := tx.First(&topic, post.TopicId).Error; err != nil {
			return err
		}
		allowed, err := canReviewAppealTx(tx, reviewerID, topic, post, revision)
		if err != nil {
			return err
		}
		if !allowed {
			return ErrAppealPermission
		}
		items := []AppealItem{}
		if err := appealItems(appealQuery(tx).Where("a.id = ?", appealID), &items); err != nil {
			return err
		}
		if len(items) == 0 {
			return ErrAppealStale
		}
		item = items[0]
		return nil
	})
	if item.Images == nil {
		item.Images = []string{}
	}
	return item, err
}

// ModeratorAppeals filters both the live topic and immutable first-post categories
// in SQL before count/pagination, so hidden categories cannot leak through totals.
func ModeratorAppeals(ctx context.Context, userID uint64, page, pageSize int, status string) ([]AppealItem, int64, error) {
	items := []AppealItem{}
	var total int64
	err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		global, ids, err := moderationservice.ScopeForUserTx(tx, userID)
		if err != nil {
			return err
		}
		if !global && len(ids) == 0 {
			return ErrAppealPermission
		}
		query := appealQuery(tx).Where("p.user_id <> ?", userID)
		if !global {
			query = query.Where("EXISTS (SELECT 1 FROM topic_category_index AS ci WHERE ci.topic_id = t.id AND ci.category_id IN ? AND ci.effective = ?)", ids, 1)
			// 'null' 是 nil 切片经 serializer:json 后的字面量；jsonb_array_elements_text 收到标量会报错。
			candidate := "EXISTS (SELECT 1 FROM json_each(COALESCE(NULLIF(NULLIF(r.category_ids, ''), 'null'), '[]')) AS rc WHERE rc.value IN ?)"
			if tx.Name() == "postgres" {
				candidate = "EXISTS (SELECT 1 FROM jsonb_array_elements_text(COALESCE(NULLIF(NULLIF(r.category_ids, ''), 'null'), '[]')::jsonb) AS rc(value) WHERE rc.value::bigint IN ?)"
			}
			query = query.Where("(p.post_no > 1 OR "+candidate+")", ids)
		}
		if status == moderationAppeals.Stale {
			query = query.Where("(a.status = ? OR "+appealStaleSQL+")", moderationAppeals.Stale)
		} else {
			query = query.Where("a.status = ?", status)
			if status == moderationAppeals.Pending {
				query = query.Where("NOT " + appealStaleSQL)
			}
		}
		if err := query.Count(&total).Error; err != nil {
			return err
		}
		return appealItems(query.Offset((page-1)*pageSize).Limit(pageSize), &items)
	})
	for i := range items {
		if items[i].Images == nil {
			items[i].Images = []string{}
		}
	}
	return items, total, err
}

func appealParentPublicTx(tx *gorm.DB, topic topics.Entity) error {
	var first posts.Entity
	if err := tx.First(&first, topic.FirstPostId).Error; err != nil {
		return err
	}
	if first.ProcessStatus != posts.ProcessStatusNormal || first.VisibilityStatus != posts.VisibilityActive || first.RetentionStatus == posts.RetentionPurged {
		return ErrUnavailable
	}
	return nil
}

// OwnerAppeal is the appeal on a content item's latest revision, as its owner sees it.
type OwnerAppeal struct {
	Status       string     `json:"status"`
	Reason       string     `json:"reason"`
	ReviewReason string     `json:"reviewReason"`
	CreatedAt    time.Time  `json:"createdAt"`
	ReviewedAt   *time.Time `json:"reviewedAt"`
}

// OwnerAppealState drives the content-management entry: whether the latest
// rejected revision can still be appealed, and the appeal already filed for it.
type OwnerAppealState struct {
	CanAppeal bool
	Appeal    *OwnerAppeal
}

// OwnerAppealStateFor evaluates one owned topic/reply with the same rules as
// SubmitAppeal, so the entry is hidden exactly when submission would fail.
// Pass the stored entities, before OwnerSnapshot projects the candidate onto them.
func OwnerAppealStateFor(ctx context.Context, userID uint64, topic topics.Entity, post posts.Entity) OwnerAppealState {
	state := OwnerAppealState{}
	if userID == 0 || post.UserId != userID || post.LatestRevisionId == 0 || post.LatestRevisionId == post.PublishedRevisionId {
		return state
	}
	conn := db.ConnectContext(ctx)
	var revision postRevisions.Entity
	if err := conn.First(&revision, post.LatestRevisionId).Error; err != nil {
		return state
	}
	eligible := appealable(topic, post, revision)
	if eligible && post.PostNo > 1 {
		eligible = appealParentPublicTx(conn, topic) == nil
	}
	var appeal moderationAppeals.Entity
	if err := conn.Where("revision_id = ?", revision.Id).Take(&appeal).Error; err == nil {
		status := appeal.Status
		if status == moderationAppeals.Pending && !eligible {
			status = moderationAppeals.Stale
		}
		state.Appeal = &OwnerAppeal{Status: status, Reason: appeal.Reason, ReviewReason: appeal.ReviewReason, CreatedAt: appeal.CreatedAt, ReviewedAt: appeal.ReviewedAt}
		return state
	}
	if !eligible {
		return state
	}
	author, err := users.GetInteractionUserTx(conn, userID)
	state.CanAppeal = err == nil && author.IsFrozen != users.StatusFrozen
	return state
}

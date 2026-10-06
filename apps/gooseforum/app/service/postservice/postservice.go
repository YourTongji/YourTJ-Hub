package postservice

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserStat"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pointservice"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const topicSequenceLockShards = 256

var topicSequenceLocks [topicSequenceLockShards]sync.Mutex

var ErrPostNotFound = errors.New("post not found")

func CreateTopicPost(entity *posts.Entity, topicEntity topics.Entity) error {
	lock := &topicSequenceLocks[entity.TopicId%topicSequenceLockShards]
	lock.Lock()
	defer lock.Unlock()

	// Sequence reservation locks the topic before the post becomes visible. All
	// derived writes share that transaction, so rebuilds see either side in full.
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		if err := identity.ValidateWriterTx(tx, entity.UserId, entity.PersonaUID); err != nil {
			return err
		}
		postNo, err := topics.ReservePostSequenceTx(tx, entity.TopicId)
		if err != nil {
			return err
		}
		entity.PostNo = postNo
		if err := posts.CreateTx(tx, entity); err != nil {
			return err
		}
		if err := SeedPostRevision(tx, entity); err != nil {
			return err
		}
		if !entity.IsAnonymous {
			if err := topicUserStat.IncrementUserPostTx(tx, entity.TopicId, entity.UserId); err != nil {
				return err
			}
		}
		ids, err := topicUserStat.SyncTopicPostersTx(tx, entity.TopicId, publicTopicOwner(topicEntity))
		if err != nil {
			return err
		}
		posters := []topics.Poster{}
		if topicEntity.PersonaUID == "" {
			posters = append(posters, topics.Poster{UserID: topicEntity.UserId})
		}
		for _, id := range ids {
			posters = append(posters, topics.Poster{UserID: id})
		}
		if err := topics.IncrementPostFastTx(tx, entity.TopicId, posters, entity.Id, entity.CreatedAt); err != nil {
			return err
		}
		if entity.ProcessStatus == posts.ProcessStatusNormal && entity.VisibilityStatus == posts.VisibilityActive {
			// Fence already queued rank work before it can acknowledge a new
			// reply's watermark using the old participant projection.
			return feed.MarkProjectionTx(tx, entity.TopicId)
		}
		return nil
	})
}

func DeleteTopicPost(postID, userID uint64) (posts.Entity, error) {
	return deleteTopicPost(db.Connect(), postID, userID)
}

func deleteTopicPost(conn *gorm.DB, postID, userID uint64) (postEntity posts.Entity, err error) {
	err = conn.Transaction(func(tx *gorm.DB) error {
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ? AND post_no > 1", postID, userID).
			Take(&postEntity)
		if query.Error != nil {
			if errors.Is(query.Error, gorm.ErrRecordNotFound) {
				return ErrPostNotFound
			}
			return query.Error
		}
		if err := pointservice.ReversePostRewardTx(tx, postEntity.UserId, postEntity.Id); err != nil {
			return err
		}
		result := tx.Delete(&postEntity)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrPostNotFound
		}
		return nil
	})
	return postEntity, err
}

// SyncTopicPostStats refreshes projections after committed lifecycle changes.
// An absolute rebuild is idempotent when another writer already included the
// deletion or restoration; a delayed decrement would corrupt those counters.
func SyncTopicPostStats(topicEntity topics.Entity) {
	if err := RebuildTopicPostStats(topicEntity); err != nil {
		slog.Error("failed to rebuild topic post stats", "topicId", topicEntity.Id, "err", err)
	}
}

// RebuildTopicPostStats recalculates derived topic and participant counters
// from the current active post set inside its own transaction. It is
// intentionally absolute rather than incremental so restoring a topic cannot
// double-count existing replies.
func RebuildTopicPostStats(topicEntity topics.Entity) error {
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		return RebuildTopicPostStatsTx(tx, topicEntity)
	})
}

// RebuildTopicPostStatsTx is the transactional variant: the delete, the
// aggregated stat writes and the absolute topic replace commit or roll back
// together, so a failed rebuild can no longer leave a half-erased
// topic_user_stat behind a rolled-back migration version (PR #575 review
// P1-atomicity). The topics row is locked (FOR UPDATE; SQLite serializes
// writers so the clause is accepted and inert) BEFORE the posts snapshot is
// taken and the lock is held through the replace: a concurrent
// CreateTopicPost holds the same lock from ReservePostSequenceTx through commit,
// so its post insert and follow-up stats sync cannot interleave with the
// snapshot→replace window (PR #575 review P1-concurrency). Aggregation is
// set-based — bounded multi-row upserts instead of one upsert per reply
// — and last_reply_at comes from the replies' own MAX(created_at); replaying
// the live increment API would stamp deployment time over the historical
// participant ordering (PR #575 review P2).
func RebuildTopicPostStatsTx(tx *gorm.DB, topicEntity topics.Entity) error {
	var locked topics.Entity
	// Unscoped：回填/恢复路径会处理软删话题，锁读不得被软删作用域过滤掉。
	if err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", topicEntity.Id).
		Take(&locked).Error; err != nil {
		return err
	}
	// 锁内行数据为准：调用方可能只传 Id（回填批扫/测试），作者 ID 等
	// 派生口径一律取锁定行的权威值，不信任事务外的快照字段。
	topicEntity = locked

	// 与 contentdeleteservice 的重建载入一致：Unscoped + post_no/id 升序，
	// 仅 ACTIVE 楼层进入派生统计。
	var activePosts []*posts.Entity
	if err := tx.Unscoped().
		Where("topic_id = ?", topicEntity.Id).
		Select("id,topic_id,post_no,user_id,visibility_status,process_status,deleted_at,is_anonymous,created_at,first_public_at").
		Order("post_no asc").
		Order("id asc").
		Find(&activePosts).Error; err != nil {
		return err
	}

	postCount := uint64(0)
	replyCount := uint64(0)
	var lastPost *posts.Entity
	replierIndex := map[uint64]*topicUserStat.ReplierStat{}
	rankIndex := map[uint64]*topicUserStat.ReplierStat{}
	for _, post := range activePosts {
		if post == nil || post.DeletedAt.Valid || post.VisibilityStatus != posts.VisibilityActive || post.ProcessStatus != posts.ProcessStatusNormal {
			continue
		}
		postCount++
		if post.PostNo > 1 {
			replyCount++
			r := rankIndex[post.UserId]
			if r == nil {
				r = &topicUserStat.ReplierStat{UserID: post.UserId}
				rankIndex[post.UserId] = r
			}
			r.RankReplyCount++
			at := post.CreatedAt
			if post.FirstPublicAt != nil {
				at = *post.FirstPublicAt
			}
			if r.LastPublicReplyAt == nil || at.After(*r.LastPublicReplyAt) {
				r.LastPublicReplyAt = &at
			}
			if !post.IsAnonymous {
				replier, ok := replierIndex[post.UserId]
				if !ok {
					replier = r
					replierIndex[post.UserId] = replier
				}
				replier.ReplyCount++
				if post.CreatedAt.After(replier.LastReplyAt) {
					replier.LastReplyAt = post.CreatedAt
				}

			}

		}
		if lastPost == nil || lastPost.CreatedAt.Before(post.CreatedAt) ||
			(lastPost.CreatedAt.Equal(post.CreatedAt) && lastPost.Id < post.Id) {
			lastPost = post
		}
	}

	repliers := make([]topicUserStat.ReplierStat, 0, len(replierIndex))
	for _, replier := range replierIndex {
		repliers = append(repliers, *replier)
	}

	lastPostID := uint64(0)
	lastPostedAt := time.Time{}
	if lastPost != nil {
		lastPostID = lastPost.Id
		lastPostedAt = lastPost.CreatedAt
	}

	if err := topicUserStat.DeleteByTopicIDTx(tx, topicEntity.Id); err != nil {
		return err
	}
	if err := topicUserStat.BulkUpsertRepliersTx(tx, topicEntity.Id, repliers); err != nil {
		return err
	}

	activePosterIDs, err := topicUserStat.SyncTopicPostersTx(tx, topicEntity.Id, publicTopicOwner(topicEntity))
	if err != nil {
		return err
	}
	posterIDs := make([]topics.Poster, 0, len(activePosterIDs)+1)
	if topicEntity.PersonaUID == "" {
		posterIDs = append(posterIDs, topics.Poster{UserID: topicEntity.UserId})
	}
	for _, userID := range activePosterIDs {
		posterIDs = append(posterIDs, topics.Poster{UserID: userID})
	}
	if err := topics.ReplacePostStatsTx(tx, topicEntity.Id, postCount, replyCount, posterIDs, lastPostID, lastPostedAt); err != nil {
		return err
	}
	var lastPublic *time.Time
	for _, r := range rankIndex {
		if r.UserID != topicEntity.UserId && r.LastPublicReplyAt != nil && (lastPublic == nil || r.LastPublicReplyAt.After(*lastPublic)) {
			at := *r.LastPublicReplyAt
			lastPublic = &at
		}
	}
	if err := tx.Model(&topics.Entity{}).Where("id = ?", topicEntity.Id).UpdateColumn("last_public_reply_at", lastPublic).Error; err != nil {
		return err
	}
	if feedconfig.Current().Ranking {
		ranks := make([]topicUserStat.ReplierStat, 0, len(rankIndex))
		for _, r := range rankIndex {
			ranks = append(ranks, *r)
		}
		if err := topicUserStat.BackfillRankProjectionTx(tx, topicEntity.Id, ranks); err != nil {
			return err
		}
	}
	return feed.MarkTx(tx, topicEntity.Id)
}

func publicTopicOwner(topic topics.Entity) uint64 {
	if topic.PersonaUID != "" {
		return 0
	}
	return topic.UserId
}

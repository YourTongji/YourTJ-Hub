package postservice

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserStat"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pointservice"
	"github.com/samber/lo"
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

	postNo, err := topics.ReservePostSequence(entity.TopicId)
	if err != nil {
		return err
	}

	entity.PostNo = postNo
	// 帖子行 + 版本 v1（editor = 作者）同事务：播种失败则帖子不落库，
	// 避免出现无初始版本的楼层。
	if err := db.Connect().Transaction(func(tx *gorm.DB) error {
		if err := posts.CreateTx(tx, entity); err != nil {
			return err
		}
		return SeedPostRevision(tx, entity)
	}); err != nil {
		return err
	}
	SyncTopicPostStats(topicEntity, *entity, false)
	return nil
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

func SyncTopicPostStats(topicEntity topics.Entity, postEntity posts.Entity, isDelete bool) {
	userId := postEntity.UserId
	// 匿名楼层（wiki 评论区，issue #524）不进入 topic_user_stat：
	// 避免匿名作者出现在「参与过的话题」/ Posters 等身份表面，泄露匿名身份。
	if !postEntity.IsAnonymous {
		if isDelete {
			if err := topicUserStat.DecrementUserPost(topicEntity.Id, userId); err != nil {
				slog.Error("failed to decrement topic user post stat", "topicId", topicEntity.Id, "userId", userId, "err", err)
			}
		} else {
			if err := topicUserStat.IncrementUserPost(topicEntity.Id, userId); err != nil {
				slog.Error("failed to increment topic user post stat", "topicId", topicEntity.Id, "userId", userId, "err", err)
			}
		}
	}

	// 作者在 SQL 内先排除再取 top-3（第二参数）：楼主高频回复不得挤占
	// 他人参与人名额（PR #575 review P2）。
	list := topicUserStat.SyncTopicPosters(topicEntity.Id, topicEntity.UserId)
	finalList := append([]uint64{topicEntity.UserId}, list...)

	pList := lo.Map(finalList, func(userID uint64, _ int) topics.Poster {
		return topics.Poster{
			UserID: userID,
		}
	})

	if isDelete {
		lastPost, _ := posts.GetLastByTopicID(topicEntity.Id)
		if err := topics.DecrementPostFast(topicEntity.Id, pList, lastPost.Id, lastPost.CreatedAt); err != nil {
			slog.Error("failed to decrement topic post count", "topicId", topicEntity.Id, "err", err)
		}
	} else {
		if err := topics.IncrementPostFast(topicEntity.Id, pList, postEntity.Id, postEntity.CreatedAt); err != nil {
			slog.Error("failed to increment topic post count", "topicId", topicEntity.Id, "err", err)
		}
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
// CreateTopicPost blocks at ReservePostSequence (it updates the same row),
// so its post insert and follow-up stats sync cannot interleave with the
// snapshot→replace window (PR #575 review P1-concurrency). Aggregation is
// set-based — one multi-row upsert per topic instead of one upsert per reply
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
		Order("post_no asc").
		Order("id asc").
		Find(&activePosts).Error; err != nil {
		return err
	}

	postCount := uint64(0)
	replyCount := uint64(0)
	var lastPost *posts.Entity
	replierIndex := map[uint64]*topicUserStat.ReplierStat{}
	for _, post := range activePosts {
		if post == nil || post.VisibilityStatus != posts.VisibilityActive {
			continue
		}
		postCount++
		if post.PostNo > 1 {
			replyCount++
			// 匿名楼层（issue #524）不进入 topic_user_stat，与增量路径口径一致。
			if !post.IsAnonymous {
				replier, ok := replierIndex[post.UserId]
				if !ok {
					replier = &topicUserStat.ReplierStat{UserID: post.UserId}
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

	activePosterIDs, err := topicUserStat.SyncTopicPostersTx(tx, topicEntity.Id, topicEntity.UserId)
	if err != nil {
		return err
	}
	posterIDs := make([]topics.Poster, 0, len(activePosterIDs)+1)
	posterIDs = append(posterIDs, topics.Poster{UserID: topicEntity.UserId})
	for _, userID := range activePosterIDs {
		posterIDs = append(posterIDs, topics.Poster{UserID: userID})
	}
	return topics.ReplacePostStatsTx(tx, topicEntity.Id, postCount, replyCount, posterIDs, lastPostID, lastPostedAt)
}

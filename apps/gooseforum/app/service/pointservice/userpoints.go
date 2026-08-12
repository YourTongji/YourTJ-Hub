package pointservice

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	TopicPublishedReward int64 = 10
	PostCreatedReward    int64 = 2
)

type PointsAction int

const (
	PointsActionUnknown PointsAction = iota
	PointsActionInit
	PointsActionTopicPublished
	PointsActionPostCreated
	PointsActionPostDeleted
)

func (action PointsAction) Code() string {
	switch action {
	case PointsActionInit:
		return "init"
	case PointsActionTopicPublished:
		return "topic_published"
	case PointsActionPostCreated:
		return "post_created"
	case PointsActionPostDeleted:
		return "post_deleted"
	default:
		return "unknown"
	}
}

func RewardPoints(userId uint64, points int64, action PointsAction, sourceKey string) error {
	return applyPoints(userId, points, action, sourceKey, "")
}

func ReversePostRewardTx(tx *gorm.DB, userId, postID uint64) error {
	return applyPointsTx(tx, userId, -PostCreatedReward, PointsActionPostDeleted,
		fmt.Sprintf("post-deleted:%d", postID), fmt.Sprintf("post:%d", postID))
}

func applyPoints(userId uint64, points int64, action PointsAction, sourceKey, originalSourceKey string) error {
	if userId == 0 || sourceKey == "" {
		return nil
	}
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		return applyPointsTx(tx, userId, points, action, sourceKey, originalSourceKey)
	})
}

func applyPointsTx(tx *gorm.DB, userId uint64, points int64, action PointsAction, sourceKey, originalSourceKey string) error {
	if userId == 0 || sourceKey == "" {
		return nil
	}
	// Bot personas do not participate in the points ledger: they have no user_points
	// row (agentservice.Create intentionally skips InitUserPointsTx) and the forum
	// reward model is scoped to humans. Skipping here covers RewardPoints from both
	// topic/reply event handlers and ReversePostRewardTx from the delete path, so
	// bot content neither earns nor reverses points and never errors into the
	// eventbus retry/drop loop.
	var actorType int8
	if err := tx.Table("users").Select("actor_type").Where("id = ?", userId).Take(&actorType).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if actorType == users.ActorTypeBot {
		return nil
	}
	locking := clause.Locking{Strength: "UPDATE"}
	// Publication handlers receive events only after the write path has accepted the
	// content. Rechecking mutable moderation state here would make rewards depend on
	// asynchronous scheduling; deletion is handled deterministically by its tombstone.
	if strings.HasPrefix(sourceKey, "topic:") {
		topicID, err := strconv.ParseUint(strings.TrimPrefix(sourceKey, "topic:"), 10, 64)
		if err != nil {
			return err
		}
		var topic struct{ UserId uint64 }
		if err := tx.Table("topics").Clauses(locking).Where("id = ?", topicID).
			Take(&topic).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if topic.UserId != userId {
			return nil
		}
	}
	if action == PointsActionPostCreated {
		if !strings.HasPrefix(sourceKey, "post:") {
			return fmt.Errorf("invalid post reward source key %q", sourceKey)
		}
		postID, err := strconv.ParseUint(strings.TrimPrefix(sourceKey, "post:"), 10, 64)
		if err != nil {
			return err
		}
		var post struct {
			Id     uint64
			UserId uint64
		}
		if err := tx.Table("posts").Clauses(locking).Where("id = ?", postID).Take(&post).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if post.UserId != userId {
			return nil
		}
		var deletedCount int64
		if err := tx.Table("points_record").Where("source_key = ?", "post-deleted:"+sourceKey[len("post:"):]).Count(&deletedCount).Error; err != nil {
			return err
		}
		if deletedCount > 0 {
			return nil
		}
	}
	applyBalance := true
	if originalSourceKey != "" {
		var original pointsRecord.Entity
		if err := tx.Clauses(locking).Where("source_key = ?", originalSourceKey).Take(&original).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				applyBalance = false
			} else {
				return err
			}
		}
		if original.Id != 0 && (original.UserId != userId || original.PointsChange != -points) {
			return fmt.Errorf("original points record %q does not match user %d and points %d", originalSourceKey, userId, -points)
		}
	}
	key := sourceKey
	recordPoints := points
	if !applyBalance {
		recordPoints = 0
	}
	const insertSavepoint = "before_points_record_insert"
	if err := tx.SavePoint(insertSavepoint).Error; err != nil {
		return err
	}
	result := tx.Create(&pointsRecord.Entity{
		UserId:       userId,
		Action:       action.Code(),
		PointsChange: recordPoints,
		SourceKey:    &key,
		CreatedAt:    time.Now(),
	})
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			if err := tx.RollbackTo(insertSavepoint).Error; err != nil {
				return err
			}
			return nil
		}
		return result.Error
	}
	if !applyBalance {
		return nil
	}
	result = tx.Table("user_points").Where("user_id = ?", userId).
		Update("current_points", gorm.Expr("current_points + ?", points))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 && points > 0 {
		// Legacy user whose `users` row predates the points feature, or was imported
		// without InitUserPointsTx, has no user_points row. BackfillMissingUserPoints
		// (migration v14) is the primary repair; this lazy create keeps the in-flight
		// reward from being lost before the backfill runs. Reversals (points < 0)
		// skip this path: the balance was already lost, the ledger tombstone records
		// the reversal, and backfill reconstructs from the ledger SUM. A concurrent
		// creator wins the insert and the loser retries the increment.
		createResult := tx.Create(&userPoints.Entity{UserId: userId, CurrentPoints: points})
		if createResult.Error != nil {
			if !errors.Is(createResult.Error, gorm.ErrDuplicatedKey) {
				return createResult.Error
			}
			retry := tx.Table("user_points").Where("user_id = ?", userId).
				Update("current_points", gorm.Expr("current_points + ?", points))
			if retry.Error != nil {
				return retry.Error
			}
		}
	}
	if err := tx.Table("users").Where("id = ?", userId).
		Update("prestige", gorm.Expr("prestige + ?", points)).Error; err != nil {
		return err
	}
	return nil
}

func InitUserPointsTx(tx *gorm.DB, userId uint64, points int64) error {
	var userPoint userPoints.Entity
	if err := tx.Where("user_id = ?", userId).Take(&userPoint).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if userPoint.UserId > 0 {
		return nil
	}
	userPoint.UserId = userId
	userPoint.CurrentPoints += points
	if err := tx.Create(&userPoint).Error; err != nil {
		return fmt.Errorf("create user points failed for user %d: %w", userId, err)
	}

	pointsRecordEntity := pointsRecord.Entity{
		UserId:       userId,
		Action:       PointsActionInit.Code(),
		PointsChange: points,
		CreatedAt:    time.Now(),
	}
	return tx.Create(&pointsRecordEntity).Error
}

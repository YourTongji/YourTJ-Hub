package pointservice

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/pointsRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/userPoints"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func ReversePoints(userId uint64, points int64, action PointsAction, sourceKey, originalSourceKey string) error {
	return applyPoints(userId, -points, action, sourceKey, originalSourceKey)
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
	locking := clause.Locking{Strength: "UPDATE"}
	if strings.HasPrefix(sourceKey, "topic:") {
		topicID, err := strconv.ParseUint(strings.TrimPrefix(sourceKey, "topic:"), 10, 64)
		if err != nil {
			return err
		}
		var status int8
		if err := tx.Table("topics").Clauses(locking).Where("id = ? AND deleted_at IS NULL", topicID).
			Pluck("status", &status).Error; err != nil {
			return err
		}
		if status != 1 {
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
			Id      uint64
			TopicId uint64
		}
		if err := tx.Table("posts").Unscoped().Clauses(locking).
			Where("id = ? AND deleted_at IS NULL", postID).First(&post).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		var topicStatus int8
		if err := tx.Table("topics").Clauses(locking).Where("id = ? AND deleted_at IS NULL", post.TopicId).
			Pluck("status", &topicStatus).Error; err != nil {
			return err
		}
		if topicStatus != 1 {
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
		var count int64
		if err := tx.Table("points_record").Where("source_key = ?", originalSourceKey).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			applyBalance = false
		}
	}
	key := sourceKey
	recordPoints := points
	if !applyBalance {
		recordPoints = 0
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&pointsRecord.Entity{
		UserId:       userId,
		Action:       action.Code(),
		PointsChange: recordPoints,
		SourceKey:    &key,
		CreatedAt:    time.Now(),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}
	if !applyBalance {
		return nil
	}
	result = tx.Table("user_points").Where("user_id = ?", userId).
		Update("current_points", gorm.Expr("current_points + ?", points))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("user points row missing for user %d", userId)
	}
	if err := tx.Table("users").Where("id = ?", userId).
		Update("prestige", gorm.Expr("prestige + ?", points)).Error; err != nil {
		return err
	}
	return nil
}

func InitUserPoints(userId uint64, points int64) error {
	userPoint := userPoints.Get(userId)
	if userPoint.UserId > 0 {
		return nil
	}
	userPoint.UserId = userId
	userPoint.CurrentPoints += points
	if err := userPoints.CreateError(&userPoint); err != nil {
		return fmt.Errorf("create user points failed for user %d: %w", userId, err)
	}

	pointsRecordEntity := pointsRecord.Entity{
		UserId:       userId,
		Action:       PointsActionInit.Code(),
		PointsChange: points,
		CreatedAt:    time.Now(),
	}
	return pointsRecord.SaveError(&pointsRecordEntity)
}

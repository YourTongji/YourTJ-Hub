package stickerservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"gorm.io/gorm"
)

const TaskTypeReview = "sticker-review"

var ErrReviewProcessed = errors.New("sticker review already processed")

type reviewTask struct {
	StickerID uint64 `json:"stickerId"`
}

func EnqueueReviewTx(tx *gorm.DB, stickerID uint64) error {
	payload, err := json.Marshal(reviewTask{StickerID: stickerID})
	if err != nil {
		return err
	}
	return taskQueue.CreateTx(tx, &taskQueue.Entity{Type: TaskTypeReview, Status: taskQueue.StatusPending, TaskJson: string(payload)})
}

// RunReviewTask returns nil without enabling the sticker when moderation is
// disabled/shadow or asks for a person to review the image.
func RunReviewTask(ctx context.Context, task *taskQueue.Entity) error {
	if task == nil || task.Type != TaskTypeReview {
		return errors.New("invalid sticker review task")
	}
	var payload reviewTask
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil || payload.StickerID == 0 {
		return errors.New("invalid sticker review payload")
	}
	conn := db.ConnectContext(ctx)
	var entity sticker.Entity
	if err := conn.First(&entity, payload.StickerID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}
	if entity.IsOfficial || entity.ReviewStatus != sticker.ReviewStatusPending {
		return nil
	}
	var prior int64
	if err := conn.Model(&moderationDecision.Entity{}).
		Where("subject_type = ? AND subject_id = ?", moderationDecision.SubjectSticker, entity.Id).
		Count(&prior).Error; err != nil {
		return err
	}
	if prior > 0 {
		return nil
	}
	decision := moderationservice.EvaluateSubmission(ctx, moderationservice.AIContentInput{
		AuthorID: entity.CreatedBy, SubjectType: moderationDecision.SubjectSticker,
		SubjectID: entity.Id, Gallery: []string{ResolveURLFor(entity)},
	})
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if decision == nil {
		return nil
	}
	return applyAIReview(ctx, entity.Id, decision)
}

func applyAIReview(ctx context.Context, stickerID uint64, decision *moderationDecision.Entity) error {
	if decision.FinalAction != moderationDecision.ActionAllow && decision.FinalAction != moderationDecision.ActionReview && decision.FinalAction != moderationDecision.ActionBlock {
		return fmt.Errorf("invalid sticker moderation action %q", decision.FinalAction)
	}
	return db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		return applyAIReviewTx(tx, stickerID, decision)
	})
}

func applyAIReviewTx(tx *gorm.DB, stickerID uint64, decision *moderationDecision.Entity) error {
	entity, err := sticker.GetByIDTx(tx, stickerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if entity.IsOfficial || entity.ReviewStatus != sticker.ReviewStatusPending {
		return nil
	}
	var prior int64
	if err := tx.Model(&moderationDecision.Entity{}).
		Where("subject_type = ? AND subject_id = ?", moderationDecision.SubjectSticker, entity.Id).
		Count(&prior).Error; err != nil {
		return err
	}
	if prior > 0 {
		return nil
	}
	decision.SubjectType = moderationDecision.SubjectSticker
	decision.SubjectId = entity.Id
	decision.AuthorId = entity.CreatedBy
	decision.AppliedAction = decision.FinalAction
	if err := tx.Create(decision).Error; err != nil {
		return err
	}
	switch decision.FinalAction {
	case moderationDecision.ActionAllow:
		entity.ReviewStatus = sticker.ReviewStatusApproved
		entity.IsEnabled = true
	case moderationDecision.ActionBlock:
		entity.ReviewStatus = sticker.ReviewStatusRejected
		entity.IsEnabled = false
	case moderationDecision.ActionReview:
		return nil
	}
	return sticker.SaveTx(tx, &entity)
}

func PendingReview(ctx context.Context, page, pageSize int) ([]sticker.Entity, int64, error) {
	items := make([]sticker.Entity, 0, pageSize)
	query := db.ConnectContext(ctx).Model(&sticker.Entity{}).Where("is_official = ? AND review_status = ?", false, sticker.ReviewStatusPending)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("created_at ASC, id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	return items, total, err
}

func ReviewPersonalSticker(ctx context.Context, stickerID uint64, approve bool, actorID uint64) error {
	return db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		return reviewPersonalStickerTx(tx, stickerID, approve, actorID)
	})
}

func reviewPersonalStickerTx(tx *gorm.DB, stickerID uint64, approve bool, actorID uint64) error {
	entity, err := sticker.GetByIDTx(tx, stickerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if entity.IsOfficial || entity.CreatedBy == 0 {
		return ErrNotFound
	}
	if entity.ReviewStatus != sticker.ReviewStatusPending {
		return ErrReviewProcessed
	}
	entity.ReviewStatus = sticker.ReviewStatusRejected
	entity.IsEnabled = false
	if approve {
		entity.ReviewStatus = sticker.ReviewStatusApproved
		entity.IsEnabled = true
	}
	if err := sticker.SaveTx(tx, &entity); err != nil {
		return err
	}
	var decision moderationDecision.Entity
	if err := tx.Where("subject_type = ? AND subject_id = ?", moderationDecision.SubjectSticker, entity.Id).
		Order("id DESC").First(&decision).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if decision.Id != 0 {
		action := moderationDecision.HumanRejected
		if approve {
			action = moderationDecision.HumanApproved
		}
		return tx.Model(&decision).Updates(map[string]any{
			"human_action": action, "human_actor_id": actorID, "human_at": time.Now(),
		}).Error
	}
	return nil
}

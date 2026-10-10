package stickerservice

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func stickerReviewDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := conn.AutoMigrate(&sticker.Entity{}, &moderationDecision.Entity{}, &taskQueue.Entity{}); err != nil {
		t.Fatal(err)
	}
	return conn
}

func TestStickerReviewTaskIsTransactionalAndAIOutcomesAreIdempotent(t *testing.T) {
	conn := stickerReviewDB(t)
	var taskID uint64
	var pending sticker.Entity
	if err := conn.Transaction(func(tx *gorm.DB) error {
		pending = sticker.Entity{Name: "u_pending", FileName: "sticker.png", CreatedBy: 5, IsOfficial: false, IsEnabled: false, ReviewStatus: sticker.ReviewStatusPending}
		if err := sticker.InsertPersonalTx(tx, &pending); err != nil {
			return err
		}
		return EnqueueReviewTx(tx, pending.Id)
	}); err != nil {
		t.Fatal(err)
	}
	var task taskQueue.Entity
	if err := conn.First(&task).Error; err != nil {
		t.Fatal(err)
	}
	taskID = task.Id
	if task.Type != TaskTypeReview || task.Status != taskQueue.StatusPending {
		t.Fatalf("task = %#v", task)
	}
	var payload reviewTask
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil || payload.StickerID != pending.Id {
		t.Fatalf("payload = %#v, err=%v", payload, err)
	}

	for _, test := range []struct {
		action string
		status string
		enable bool
	}{
		{action: moderationDecision.ActionAllow, status: sticker.ReviewStatusApproved, enable: true},
		{action: moderationDecision.ActionBlock, status: sticker.ReviewStatusRejected, enable: false},
		{action: moderationDecision.ActionReview, status: sticker.ReviewStatusPending, enable: false},
	} {
		t.Run(test.action, func(t *testing.T) {
			entity := sticker.Entity{Name: "u_" + test.action, FileName: "sticker.png", CreatedBy: 5, IsOfficial: false, IsEnabled: false, ReviewStatus: sticker.ReviewStatusPending}
			if err := conn.Transaction(func(tx *gorm.DB) error { return sticker.InsertPersonalTx(tx, &entity) }); err != nil {
				t.Fatal(err)
			}
			decision := &moderationDecision.Entity{FinalAction: test.action}
			for range 2 {
				if err := conn.Transaction(func(tx *gorm.DB) error {
					return applyAIReviewTx(tx, entity.Id, decision)
				}); err != nil {
					t.Fatal(err)
				}
			}
			var reviewed sticker.Entity
			if err := conn.First(&reviewed, entity.Id).Error; err != nil {
				t.Fatal(err)
			}
			if reviewed.ReviewStatus != test.status || reviewed.IsEnabled != test.enable {
				t.Fatalf("reviewed = %#v, want status=%s enabled=%t", reviewed, test.status, test.enable)
			}
			var count int64
			if err := conn.Model(&moderationDecision.Entity{}).Where("subject_type = ? AND subject_id = ?", moderationDecision.SubjectSticker, entity.Id).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 1 {
				t.Fatalf("decision count = %d, want 1", count)
			}
		})
	}
	if taskID == 0 {
		t.Fatal("review task was not persisted")
	}
}

func TestManualStickerDecisionUpdatesStatusAndAIReviewRecord(t *testing.T) {
	conn := stickerReviewDB(t)
	entity := sticker.Entity{Name: "u_manual", FileName: "sticker.png", CreatedBy: 7, IsOfficial: false, IsEnabled: false, ReviewStatus: sticker.ReviewStatusPending}
	if err := conn.Transaction(func(tx *gorm.DB) error { return sticker.InsertPersonalTx(tx, &entity) }); err != nil {
		t.Fatal(err)
	}
	decision := moderationDecision.Entity{SubjectType: moderationDecision.SubjectSticker, SubjectId: entity.Id, FinalAction: moderationDecision.ActionReview}
	if err := conn.Create(&decision).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error {
		return reviewPersonalStickerTx(tx, entity.Id, true, 42)
	}); err != nil {
		t.Fatal(err)
	}
	var reviewed sticker.Entity
	if err := conn.First(&reviewed, entity.Id).Error; err != nil {
		t.Fatal(err)
	}
	if reviewed.ReviewStatus != sticker.ReviewStatusApproved || !reviewed.IsEnabled {
		t.Fatalf("reviewed = %#v", reviewed)
	}
	var recorded moderationDecision.Entity
	if err := conn.First(&recorded, decision.Id).Error; err != nil {
		t.Fatal(err)
	}
	if recorded.HumanAction != moderationDecision.HumanApproved || recorded.HumanActorId != 42 || recorded.HumanAt == nil {
		t.Fatalf("human review = %#v", recorded)
	}
	err := conn.Transaction(func(tx *gorm.DB) error { return reviewPersonalStickerTx(tx, entity.Id, false, 43) })
	if !errors.Is(err, ErrReviewProcessed) {
		t.Fatalf("second decision err = %v, want ErrReviewProcessed", err)
	}
}

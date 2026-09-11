package posts

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
)

// 回归（review nit，MADR-0021）：PURGED 是终态——posts 侧删除转移函数在 repo 层
// 拒绝把终态行改写回 RECOVERABLE，与 topics 侧守卫对齐。调用方
// （DeletePostByUser/DeletePostAsModerator）的幂等分支在其上层先行处理，
// 本守卫仅防御未来新调用方漏判。
func TestDeletionTransitionsRefusePurgedRows(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("migrate posts: %v", err)
	}
	const postID = uint64(9401)
	now := time.Now()
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", postID).Delete(&Entity{})
	})
	if err := conn.Create(&Entity{
		Id: postID, TopicId: 90000 + postID, PostNo: 2, UserId: 1,
		Content:          "purged body",
		VisibilityStatus: VisibilityActive,
		RetentionStatus:  RetentionNormal,
		CreatedAt:        now,
		UpdatedAt:        now,
	}).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := conn.Model(&Entity{}).Unscoped().Where("id = ?", postID).
		Update("retention_status", RetentionPurged).Error; err != nil {
		t.Fatalf("flip retention: %v", err)
	}

	if err := MarkUserDeleted(postID, 99, "retry"); err == nil {
		t.Fatal("MarkUserDeleted must refuse PURGED row (MADR-0021)")
	}
	if err := MarkModeratorRemoved(postID, 99, "retry"); err == nil {
		t.Fatal("MarkModeratorRemoved must refuse PURGED row (MADR-0021)")
	}
	var txErrs []error
	if err := conn.Transaction(func(tx *gorm.DB) error {
		txErrs = append(txErrs,
			MarkUserDeletedTx(tx, postID, 99, "retry"),
			MarkModeratorRemovedTx(tx, postID, 99, "retry"),
			MarkUserDeletedKeepVisibleTx(tx, postID, 99, "retry"),
		)
		return nil
	}); err != nil {
		t.Fatalf("tx: %v", err)
	}
	for i, err := range txErrs {
		if err == nil {
			t.Fatalf("Tx deletion transition #%d must refuse PURGED row (MADR-0021)", i)
		}
	}

	got := UnscopedGet(postID)
	if got.VisibilityStatus != VisibilityActive || got.RetentionStatus != RetentionPurged {
		t.Fatalf("purged row mutated: %s/%s", got.VisibilityStatus, got.RetentionStatus)
	}
}

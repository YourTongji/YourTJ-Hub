package topicUserStat

import (
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestIncrementUserPostExistingRowPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL topic user stat test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	if err := db.AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("AutoMigrate topic user stat on postgres: %v", err)
	}

	const topicID uint64 = 99588001
	const userID uint64 = 99588002
	if err := db.Where("topic_id = ? AND user_id = ?", topicID, userID).Delete(&Entity{}).Error; err != nil {
		t.Fatalf("clean topic user stat before test: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Where("topic_id = ? AND user_id = ?", topicID, userID).Delete(&Entity{}).Error
	})

	if err := IncrementUserPostTx(db, topicID, userID); err != nil {
		t.Fatalf("first increment: %v", err)
	}
	if err := IncrementUserPostTx(db, topicID, userID); err != nil {
		t.Fatalf("second increment: %v", err)
	}

	var got Entity
	if err := db.Where("topic_id = ? AND user_id = ?", topicID, userID).First(&got).Error; err != nil {
		t.Fatalf("load topic user stat: %v", err)
	}
	if got.ReplyCount != 2 {
		t.Fatalf("reply_count = %d, want 2", got.ReplyCount)
	}
}

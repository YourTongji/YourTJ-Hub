package topicCategoryIndex

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
)

func TestTopicCategoryIndexRepositoryParity(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("migrate topic category index: %v", err)
	}
	conn.Where("1 = 1").Delete(&Entity{})

	conn.Create(&Entity{TopicId: 10, CategoryId: 3, Effective: 1})
	conn.Create(&Entity{TopicId: 20, CategoryId: 3, Effective: 0})
	conn.Create(&Entity{TopicId: 10, CategoryId: 4, Effective: 1})

	rows := GetByTopicId(10)
	if len(rows) != 2 {
		t.Fatalf("GetByTopicId() len=%d, want 2", len(rows))
	}
	if got := GetOneByCategoryId(3); got.TopicId != 10 {
		t.Fatalf("GetOneByCategoryId(3).TopicId=%d, want 10", got.TopicId)
	}
}

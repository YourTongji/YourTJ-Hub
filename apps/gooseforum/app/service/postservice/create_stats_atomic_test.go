package postservice

import (
	"errors"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserStat"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"gorm.io/gorm"
	"testing"
)

func TestCreateTopicPostStatsFailureRollsBackWholeWrite(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}, &postRevisions.Entity{}, &topicUserStat.Entity{}); err != nil {
		t.Fatal(err)
	}
	topic := topics.Entity{Id: 991234, UserId: 1, PostSeq: 1, PostCount: 1, Status: 1}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", topic.Id).Delete(&topics.Entity{})
		conn.Unscoped().Where("topic_id = ?", topic.Id).Delete(&posts.Entity{})
		conn.Where("topic_id = ?", topic.Id).Delete(&topicUserStat.Entity{})
	})
	failure := errors.New("stats unavailable")
	if err := conn.Callback().Create().Before("gorm:create").Register("review:fail_stats", func(tx *gorm.DB) {
		if tx.Statement.Table == "topic_user_stat" {
			_ = tx.AddError(failure)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Create().Remove("review:fail_stats") })
	post := posts.Entity{TopicId: topic.Id, UserId: 2, Content: "reply", VisibilityStatus: posts.VisibilityActive}
	err := CreateTopicPost(&post, topic)
	if !errors.Is(err, failure) {
		t.Fatalf("expected stats error, got %v", err)
	}
	var count int64
	conn.Model(&posts.Entity{}).Where("topic_id = ?", topic.Id).Count(&count)
	if count != 0 {
		t.Fatalf("post committed without stats: %d", count)
	}
	conn.First(&topic, topic.Id)
	if topic.PostSeq != 1 || topic.PostCount != 1 {
		t.Fatalf("topic changed after failure: %#v", topic)
	}
}

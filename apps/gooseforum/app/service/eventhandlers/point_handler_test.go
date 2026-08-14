package eventhandlers

import (
	"context"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

// TestHandlePointTopicPublishedSkipsWiki wiki 分站页面不发放论坛发帖积分
// （review #219：wiki 创建/编辑走独立激励，PointsActionTopicPublished 仅限
// 论坛话题；此前 wiki topic 触发 TopicPublishedEvent 会错误发 10 分）。
func TestHandlePointTopicPublishedSkipsWiki(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &topics.Entity{}, &pointsRecord.Entity{}, &userPoints.Entity{}); err != nil {
		t.Fatalf("migrate point tables: %v", err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Delete(&users.EntityComplete{}, "username LIKE 'pointtest%'")
		conn.Unscoped().Delete(&pointsRecord.Entity{}, "1 = 1")
		conn.Unscoped().Delete(&userPoints.Entity{}, "1 = 1")
		conn.Unscoped().Delete(&topics.Entity{}, "1 = 1")
	})

	ctx := context.Background()

	// wiki 话题（TopicTypeWiki）：不得产生任何积分记录。
	wikiTopic := &topics.Entity{Id: 91001, UserId: 91002, Title: "wiki page", TopicType: topics.TopicTypeWiki, Status: 1}
	if err := conn.Create(wikiTopic).Error; err != nil {
		t.Fatalf("create wiki topic: %v", err)
	}
	userID := uint64(91002)
	if err := conn.Create(&users.EntityComplete{Id: userID, Username: "pointtestwiki", Email: "pointtestwiki@example.test", IsActivated: users.ActivationSuccess}).Error; err != nil {
		t.Fatalf("create wiki user: %v", err)
	}
	if err := handlePointTopicPublished(ctx, &TopicPublishedEvent{Topic: wikiTopic}); err != nil {
		t.Fatalf("handlePointTopicPublished(wiki) error: %v", err)
	}
	var count int64
	if err := conn.Table("points_record").Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count wiki points: %v", err)
	}
	if count != 0 {
		t.Fatalf("wiki topic earned %d point records, want 0", count)
	}

	// 论坛话题（TopicTypeForum）：正常发放发帖积分。
	forumTopic := &topics.Entity{Id: 91003, UserId: 91002, Title: "forum topic", TopicType: topics.TopicTypeForum, Status: 1}
	if err := conn.Create(forumTopic).Error; err != nil {
		t.Fatalf("create forum topic: %v", err)
	}
	if err := handlePointTopicPublished(ctx, &TopicPublishedEvent{Topic: forumTopic}); err != nil {
		t.Fatalf("handlePointTopicPublished(forum) error: %v", err)
	}
	if err := conn.Table("points_record").Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count forum points: %v", err)
	}
	if count != 1 {
		t.Fatalf("forum topic earned %d point records, want 1", count)
	}
}

// TestHandlePointTopicPublishedNilSafe 事件为 nil / topic 缺失时安全返回。
func TestHandlePointTopicPublishedNilSafe(t *testing.T) {
	ctx := context.Background()
	if err := handlePointTopicPublished(ctx, nil); err != nil {
		t.Fatalf("handlePointTopicPublished(nil) error: %v", err)
	}
	if err := handlePointTopicPublished(ctx, &TopicPublishedEvent{}); err != nil {
		t.Fatalf("handlePointTopicPublished(empty) error: %v", err)
	}
}

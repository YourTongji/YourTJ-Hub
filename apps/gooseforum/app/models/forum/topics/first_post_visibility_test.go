package topics

import (
	"context"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"testing"
)

func TestPublicListRejectsLifecycleDeletedFirstPost(t *testing.T) {
	db := dbconnect.Connect()
	if err := db.AutoMigrate(&Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	topic := Entity{Title: "hidden first post", Status: 1, UserId: 981231}
	if err := db.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	post := posts.Entity{TopicId: topic.Id, PostNo: 1, UserId: topic.UserId, VisibilityStatus: posts.VisibilityUserDeleted}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&topic).Update("first_post_id", post.Id).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(&post); db.Unscoped().Delete(&topic) })
	page := Page(PageQuery{FilterStatus: true, UserId: topic.UserId, Page: 1, PageSize: 20})
	if len(page.Data) != 0 {
		t.Fatal("public list exposed a first post in the deletion lifecycle")
	}
}

func TestFiniteRecallIDsCannotBypassAuthoritativeVisibility(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}, &posts.Entity{}, &users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	author := users.EntityComplete{Id: 981232, Username: "finite-author"}
	if err := conn.Create(&author).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&author) })
	topic := Entity{Title: "finite hidden candidate", Status: 1, UserId: author.Id}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	post := posts.Entity{TopicId: topic.Id, PostNo: 1, UserId: topic.UserId, VisibilityStatus: posts.VisibilityUserDeleted}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&topic).UpdateColumn("first_post_id", post.Id).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&post); conn.Unscoped().Delete(&topic) })
	ids, err := RecallRankIDs(context.Background(), Recall{Source: "author", Authors: []uint64{topic.UserId}, Limit: 20})
	if err != nil || len(ids) != 1 || ids[0] != topic.Id {
		t.Fatalf("bounded ID candidate was not recalled: %v %v", ids, err)
	}
	rows, err := RankTopics(context.Background(), ids)
	if err != nil || len(rows) != 0 {
		t.Fatalf("hidden first post reached authoritative metadata: %v %v", rows, err)
	}
}

func TestPublicRankRejectsFrozenAndClosedAuthorsImmediately(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}, &posts.Entity{}, &users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	author := users.EntityComplete{Id: 981233, Username: "rank-eligible-author"}
	if err := conn.Create(&author).Error; err != nil {
		t.Fatal(err)
	}
	topic := Entity{Title: "rank eligible", Status: 1, UserId: author.Id}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	post := posts.Entity{TopicId: topic.Id, PostNo: 1, UserId: author.Id}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&topic).UpdateColumn("first_post_id", post.Id).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&post); conn.Unscoped().Delete(&topic); conn.Unscoped().Delete(&author) })
	if _, err := RankTopic(context.Background(), topic.Id); err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&author).UpdateColumn("is_frozen", 1).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := RankTopic(context.Background(), topic.Id); err == nil {
		t.Fatal("frozen author retained ranking before worker invalidation")
	}
	if err := conn.Model(&author).UpdateColumn("is_frozen", 0).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := RankTopic(context.Background(), topic.Id); err != nil {
		t.Fatal("unfrozen author was not restored", err)
	}
	if err := conn.Delete(&author).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := RankTopic(context.Background(), topic.Id); err == nil {
		t.Fatal("closed author retained ranking")
	}
}

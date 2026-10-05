package topics

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
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

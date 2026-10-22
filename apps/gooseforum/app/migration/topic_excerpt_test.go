package migration

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
)

func TestVersionedMigrationRepairsEmptyShortExcerpt(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}, &topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	topic := topics.Entity{Id: 13601360, FirstPostId: 13601360, Status: 1, ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityActive}
	post := posts.Entity{Id: topic.FirstPostId, TopicId: topic.Id, PostNo: 1, Content: "失眠咯", ProcessStatus: posts.ProcessStatusNormal, VisibilityStatus: posts.VisibilityActive}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Delete(&topic); conn.Delete(&post) })
	if err := pageConfig.SyncMigrationVersion(33); err != nil {
		t.Fatal(err)
	}
	if err := runVersionedDataMigrations(); err != nil {
		t.Fatal(err)
	}
	var got topics.Entity
	if err := conn.First(&got, topic.Id).Error; err != nil {
		t.Fatal(err)
	}
	if got.Excerpt != "失眠咯" {
		t.Fatalf("excerpt=%q, want 失眠咯", got.Excerpt)
	}
}

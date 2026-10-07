package postservice

import (
	"fmt"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"gorm.io/gorm"
	"testing"
)

func TestDirectEditBaselineUsesPublishedRevision(t *testing.T) {
	for _, withPointer := range []bool{true, false} {
		t.Run(fmt.Sprintf("publishedPointer=%t", withPointer), func(t *testing.T) {
			conn := db.Connect()
			t.Setenv("YOURTJ_AGENT_INSTANCE_ID", "direct-baseline")
			t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "epoch")
			if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}, &postRevisions.Entity{}, &agentEvents.Publication{}); err != nil {
				t.Fatal(err)
			}
			topic := topics.Entity{Id: 991920, UserId: 1, Status: 1}
			post := posts.Entity{Id: 991921, TopicId: topic.Id, UserId: 1, PostNo: 1, Content: "The original published body"}
			if err := conn.Create(&topic).Error; err != nil {
				t.Fatal(err)
			}
			if err := conn.Create(&post).Error; err != nil {
				t.Fatal(err)
			}
			published := postRevisions.Entity{PostId: post.Id, Version: 1, Content: post.Content}
			private := postRevisions.Entity{PostId: post.Id, Version: 2, Content: "A private candidate mentions @some-bot", ProcessStatus: posts.ProcessStatusPending}
			for _, row := range []*postRevisions.Entity{&published, &private} {
				if err := conn.Create(row).Error; err != nil {
					t.Fatal(err)
				}
			}
			post.LatestRevisionId = private.Id
			if withPointer {
				post.PublishedRevisionId = published.Id
			}
			post.Content = "The first public mention of @some-bot"
			t.Cleanup(func() {
				conn.Where("post_id = ?", post.Id).Delete(&agentEvents.Publication{})
				conn.Where("post_id = ?", post.Id).Delete(&postRevisions.Entity{})
				conn.Unscoped().Delete(&post)
				conn.Unscoped().Delete(&topic)
			})
			if err := conn.Transaction(func(tx *gorm.DB) error {
				if err := tx.Save(&post).Error; err != nil {
					return err
				}
				return AppendPostRevisionWithOld(tx, &post, 1, posts.ProcessStatusNormal, published.Content, posts.ProcessStatusNormal)
			}); err != nil {
				t.Fatal(err)
			}
			state, err := agentEvents.PublicationTx(conn, "direct-baseline", post.Id)
			if err != nil {
				t.Fatal(err)
			}
			if state.Version != published.Version {
				t.Fatalf("public baseline = %d, want published version %d (private candidate %d)", state.Version, published.Version, private.Version)
			}
		})
	}
}

package publicationservice_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/migration"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pointservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
	"gorm.io/gorm"
)

func TestReviewPointsAtomicPublication(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(migration.SchemaModels()...); err != nil {
		t.Fatal(err)
	}
	checkReviewPointsAtomicPublication(t, conn)
}

func TestPostgreSQLReviewPointsAtomicPublication(t *testing.T) {
	conn := setupAgentPolicyPostgreSQL(t)
	// Match the production connection's duplicate-key translation.
	conn.TranslateError = true
	checkReviewPointsAtomicPublication(t, conn)
}

func checkReviewPointsAtomicPublication(t *testing.T, conn *gorm.DB) {
	t.Helper()
	for _, first := range []bool{true, false} {
		t.Run(fmt.Sprintf("first-post-%t", first), func(t *testing.T) {
			owner := users.EntityComplete{Username: fmt.Sprintf("reward-%d", time.Now().UnixNano())}
			if err := conn.Create(&owner).Error; err != nil {
				t.Fatal(err)
			}
			topic := topics.Entity{UserId: owner.Id, Status: 1, Title: "pending reward"}
			if first {
				topic.ProcessStatus = topics.ProcessStatusPending
			}
			if err := conn.Create(&topic).Error; err != nil {
				t.Fatal(err)
			}
			post := posts.Entity{TopicId: topic.Id, UserId: owner.Id, PostNo: 2, ProcessStatus: posts.ProcessStatusPending}
			if first {
				post.PostNo = 1
			}
			if err := conn.Create(&post).Error; err != nil {
				t.Fatal(err)
			}
			revision := postRevisions.Entity{PostId: post.Id, Version: 1, Title: topic.Title, Content: "approved body", ProcessStatus: posts.ProcessStatusPending}
			if err := conn.Create(&revision).Error; err != nil {
				t.Fatal(err)
			}
			if err := conn.Model(&post).Update("latest_revision_id", revision.Id).Error; err != nil {
				t.Fatal(err)
			}
			if first {
				if err := conn.Model(&topic).Update("first_post_id", post.Id).Error; err != nil {
					t.Fatal(err)
				}
			}
			// Pending content must never receive the publication reward.
			if err := conn.Transaction(func(tx *gorm.DB) error { return pointservice.RewardPublicationTx(tx, topic.Id, post.Id) }); err != nil {
				t.Fatal(err)
			}
			var count int64
			if err := conn.Model(&pointsRecord.Entity{}).Where("user_id = ?", owner.Id).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatal("pending content earned points")
			}
			injected := errors.New("ledger unavailable")
			name := "review_points_failure"
			if err := conn.Callback().Create().Before("gorm:create").Register(name, func(tx *gorm.DB) {
				if tx.Statement.Table == "points_record" {
					_ = tx.AddError(injected)
				}
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = conn.Callback().Create().Remove(name) })
			err := publicationservice.Review(context.Background(), revision.Id, moderationDecision.ActionAllow, "", 0)
			if !errors.Is(err, injected) {
				t.Fatalf("approval error = %v", err)
			}
			if err := conn.First(&post, post.Id).Error; err != nil {
				t.Fatal(err)
			}
			if err := conn.First(&revision, revision.Id).Error; err != nil {
				t.Fatal(err)
			}
			if post.ProcessStatus != posts.ProcessStatusPending || post.PublishedRevisionId != 0 || revision.ReviewedAt != nil {
				t.Fatal("failed ledger write committed the approval")
			}
			if err := conn.Callback().Create().Remove(name); err != nil {
				t.Fatal(err)
			}
			// Retry succeeds without running the event or publication-effects workers.
			if err := publicationservice.Review(context.Background(), revision.Id, moderationDecision.ActionAllow, "", 0); err != nil {
				t.Fatal(err)
			}
			if err := conn.First(&post, post.Id).Error; err != nil {
				t.Fatal(err)
			}
			if post.PublishedRevisionId != revision.Id || post.ProcessStatus != posts.ProcessStatusNormal {
				t.Fatal("approval did not commit")
			}
			// A duplicate reward must leave the surrounding transaction usable on PG.
			if err := conn.Transaction(func(tx *gorm.DB) error {
				if err := pointservice.RewardPublicationTx(tx, topic.Id, post.Id); err != nil {
					return err
				}
				return tx.Model(&topic).Update("title", "replay committed").Error
			}); err != nil {
				t.Fatal(err)
			}
			var record pointsRecord.Entity
			if err := conn.Where("user_id = ?", owner.Id).Take(&record).Error; err != nil {
				t.Fatal(err)
			}
			amount := pointservice.PostCreatedReward
			if first {
				amount = pointservice.TopicPublishedReward
			}
			var balance int64
			if err := conn.Table("user_points").Where("user_id = ?", owner.Id).Select("current_points").Scan(&balance).Error; err != nil {
				t.Fatal(err)
			}
			if record.PointsChange != amount || balance != amount {
				t.Fatalf("ledger=%d balance=%d want=%d", record.PointsChange, balance, amount)
			}
		})
	}
}

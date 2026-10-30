package publicationservice_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
	"gorm.io/gorm"
)

func TestReviewRejectReasonGuard(t *testing.T) {
	for _, reason := range []string{"", " \n\u2003 ", strings.Repeat("😀", 513)} {
		if err := publicationservice.Review(context.Background(), 0, moderationDecision.ActionBlock, reason, 42); !errors.Is(err, publicationservice.ErrReviewReason) {
			t.Fatalf("missing reason guard: %v", err)
		}
	}
}

func TestPostgreSQLLegacyRejectionSnapshotRollbackAndFencing(t *testing.T) {
	conn := setupAgentPolicyPostgreSQL(t)
	topic := topics.Entity{UserId: 960991, Title: "Pending legacy topic", Status: 1, ProcessStatus: topics.ProcessStatusPending}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	post := posts.Entity{UserId: topic.UserId, TopicId: topic.Id, PostNo: 1, Content: "Original legacy content", ProcessStatus: posts.ProcessStatusPending}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&topic).Update("first_post_id", post.Id).Error; err != nil {
		t.Fatal(err)
	}
	synthetic := errors.New("synthetic status failure")
	reason := "Privacy concern. Please remove the private address."
	err := conn.Transaction(func(tx *gorm.DB) error {
		if _, err := publicationservice.RecordLegacyRejectionTx(tx, post.Id, 42, reason); err != nil {
			return err
		}
		return synthetic
	})
	if !errors.Is(err, synthetic) {
		t.Fatal(err)
	}
	var stored posts.Entity
	if err := conn.First(&stored, post.Id).Error; err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := conn.Model(&postRevisions.Entity{}).Where("post_id = ?", post.Id).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if stored.LatestRevisionId != 0 || stored.ProcessStatus != posts.ProcessStatusPending || count != 0 {
		t.Fatal("rollback left a decision snapshot")
	}
	var revisionID uint64
	if err := conn.Transaction(func(tx *gorm.DB) error {
		var err error
		revisionID, err = publicationservice.RecordLegacyRejectionTx(tx, post.Id, 42, "  "+reason+"  ")
		if err != nil {
			return err
		}
		return tx.Model(&post).Update("process_status", posts.ProcessStatusBlocked).Error
	}); err != nil {
		t.Fatal(err)
	}
	var revision postRevisions.Entity
	if err := conn.First(&revision, revisionID).Error; err != nil {
		t.Fatal(err)
	}
	if revision.ReviewReason != reason || revision.ReviewActorId != 42 || revision.ReviewedAt == nil || revision.Content != post.Content {
		t.Fatalf("decision lost: %+v", revision)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error {
		_, err := publicationservice.RecordLegacyRejectionTx(tx, post.Id, 43, "Different reason")
		return err
	}); !errors.Is(err, publicationservice.ErrUnavailable) {
		t.Fatalf("replay overwrote reason: %v", err)
	}
	if err := conn.First(&revision, revisionID).Error; err != nil {
		t.Fatal(err)
	}
	if revision.ReviewReason != reason {
		t.Fatal("later reviewer changed the decision")
	}
}

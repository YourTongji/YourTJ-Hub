package publicationservice_test

import (
	"context"
	"errors"
	"fmt"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/migration"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationAppeals"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
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

func TestLegacyRejectionAppealPreservesImageAccess(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(migration.SchemaModels()...); err != nil {
		t.Fatal(err)
	}
	for _, first := range []bool{true, false} {
		t.Run(fmt.Sprint(first), func(t *testing.T) { testLegacyRejectionImages(t, conn, first) })
	}
}
func TestPostgreSQLLegacyRejectionAppealPreservesImageAccess(t *testing.T) {
	for _, first := range []bool{true, false} {
		t.Run(fmt.Sprint(first), func(t *testing.T) { testLegacyRejectionImages(t, setupAgentPolicyPostgreSQL(t), first) })
	}
}
func testLegacyRejectionImages(t *testing.T, conn *gorm.DB, first bool) {
	t.Helper()
	ownerID := uint64(960995)
	if !first {
		ownerID += 2
	}
	owner := users.EntityComplete{Id: ownerID, Username: "legacy-image-owner"}
	reviewer := users.EntityComplete{Id: ownerID + 1, Username: "legacy-image-reviewer"}
	grant := moderators.Entity{UserId: reviewer.Id, ScopeType: moderators.ScopeGlobal, Status: 1}
	for _, row := range []any{&owner, &reviewer, &grant} {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	topic := topics.Entity{UserId: owner.Id, Title: "legacy images", Status: 1, ProcessStatus: topics.ProcessStatusNormal}
	if first {
		topic.ProcessStatus = topics.ProcessStatusPending
	}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	postNo := uint64(2)
	var firstPost posts.Entity
	if first {
		postNo = 1
	} else {
		firstPost = posts.Entity{TopicId: topic.Id, UserId: owner.Id, PostNo: 1, Content: "public baseline", ProcessStatus: posts.ProcessStatusNormal}
		if err := conn.Create(&firstPost).Error; err != nil {
			t.Fatal(err)
		}
		if err := conn.Model(&topic).Update("first_post_id", firstPost.Id).Error; err != nil {
			t.Fatal(err)
		}
	}
	name := fmt.Sprintf("legacy-review/%d-%v.webp", topic.Id, first)
	gallery := fmt.Sprintf("legacy-review/%d-gallery.webp", topic.Id)
	post := posts.Entity{TopicId: topic.Id, UserId: owner.Id, PostNo: postNo, Content: "![inline](/file/img/" + name + ")", ProcessStatus: posts.ProcessStatusPending}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	target, id := fileUsage.TargetPost, post.Id
	if first {
		target, id = fileUsage.TargetTopic, topic.Id
		topic.FirstPostId = post.Id
		topic.ImageUrls = []string{"/file/img/" + gallery}
		if err := conn.Model(&topic).Select("first_post_id", "image_urls").Updates(&topic).Error; err != nil {
			t.Fatal(err)
		}
	}
	names := []string{name}
	if first {
		names = append(names, gallery)
	}
	for _, file := range names {
		if err := conn.Create(&fileUsage.Entity{FileName: file, TargetType: target, TargetId: id, UsageType: fileUsage.UsageInlineImage, UserId: owner.Id, Status: fileUsage.UsageStatusPending}).Error; err != nil {
			t.Fatal(err)
		}
	}
	// Some old first-post writers registered the same inline image on both targets.
	if first {
		if err := conn.Create(&fileUsage.Entity{FileName: name, TargetType: fileUsage.TargetPost, TargetId: post.Id, UsageType: fileUsage.UsageInlineImage, UserId: owner.Id, Status: fileUsage.UsageStatusPending}).Error; err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		conn.Where("post_id = ?", post.Id).Delete(&moderationAppeals.Entity{})
		conn.Where("post_id = ?", post.Id).Delete(&postRevisions.Entity{})
		conn.Where("file_name IN ?", names).Delete(&fileUsage.Entity{})
		conn.Unscoped().Where("topic_id = ?", topic.Id).Delete(&posts.Entity{})
		conn.Unscoped().Delete(&topic)
		conn.Delete(&grant)
		conn.Unscoped().Delete(&owner)
		conn.Unscoped().Delete(&reviewer)
	})
	rolledBack := errors.New("synthetic failure after image snapshot")
	if err := conn.Transaction(func(tx *gorm.DB) error {
		if _, err := publicationservice.RecordLegacyRejectionTx(tx, post.Id, reviewer.Id, "temporary decision"); err != nil {
			return err
		}
		return rolledBack
	}); !errors.Is(err, rolledBack) {
		t.Fatal(err)
	}
	var leaked int64
	if err := conn.Model(&fileUsage.Entity{}).Where("target_type = ? AND file_name IN ?", fileUsage.TargetPostRevision, names).Count(&leaked).Error; err != nil || leaked != 0 {
		t.Fatalf("rollback left %d private image refs: %v", leaked, err)
	}
	var revisionID uint64
	if err := conn.Transaction(func(tx *gorm.DB) error {
		var err error
		revisionID, err = publicationservice.RecordLegacyRejectionTx(tx, post.Id, reviewer.Id, "Please revise the explanation.")
		if err != nil {
			return err
		}
		if err = tx.Model(&post).Update("process_status", posts.ProcessStatusBlocked).Error; err != nil {
			return err
		}
		if first {
			return tx.Model(&topic).Update("process_status", topics.ProcessStatusBlocked).Error
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, file := range names {
		access, err := fileusageservice.ResolveReadAccess(file)
		if err != nil || access.Public {
			t.Fatalf("rejected image became public: %+v %v", access, err)
		}
	}
	appeal, err := publicationservice.SubmitAppeal(context.Background(), owner.Id, map[bool]string{true: "topic", false: "post"}[first], id, revisionID, "Please review the original image.")
	if err != nil {
		t.Fatal(err)
	}
	if err := publicationservice.ReviewAppeal(context.Background(), appeal.Id, reviewer.Id, true, ""); err != nil {
		t.Fatal(err)
	}
	for _, file := range names {
		access, err := fileusageservice.ResolveReadAccess(file)
		if err != nil || !access.Public {
			t.Fatalf("approved legacy image lost read access: %s %+v %v", file, access, err)
		}
	}
	var count int64
	if err := conn.Model(&fileUsage.Entity{}).Where("target_type = ? AND target_id = ? AND status = ?", fileUsage.TargetPostRevision, revisionID, fileUsage.UsageStatusPending).Count(&count).Error; err != nil || count != int64(len(names)) {
		t.Fatalf("private snapshot image references = %d, want %d: %v", count, len(names), err)
	}
}

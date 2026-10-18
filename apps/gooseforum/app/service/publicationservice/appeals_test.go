package publicationservice_test

import (
	"context"
	"errors"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/migration"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationAppeals"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
)

func TestModerationAppealExactVersionAndScope(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(migration.SchemaModels()...); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	const owner, mod, outsider uint64 = 961401, 961402, 961403
	for _, row := range []users.EntityComplete{{Id: owner, Username: "appeal-owner"}, {Id: mod, Username: "appeal-moderator"}, {Id: outsider, Username: "appeal-outsider"}} {
		if err := conn.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	grant := moderators.Entity{UserId: mod, ScopeType: moderators.ScopeCategory, ScopeId: 9614, Status: 1}
	if err := conn.Create(&grant).Error; err != nil {
		t.Fatal(err)
	}
	topic := topics.Entity{UserId: owner, Title: "original", Status: 1, ProcessStatus: topics.ProcessStatusBlocked, CategoryIds: []uint64{9614}}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	post := posts.Entity{TopicId: topic.Id, UserId: owner, PostNo: 1, Content: "blocked content", ProcessStatus: posts.ProcessStatusBlocked}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	at := time.Now()
	revision := postRevisions.Entity{PostId: post.Id, Version: 1, Title: "candidate", Content: "immutable rejected body", CategoryIds: []uint64{9614}, ProcessStatus: posts.ProcessStatusBlocked, ReviewReason: "original decision", ReviewActorId: 91, ReviewedAt: &at}
	if err := conn.Create(&revision).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&post).Update("latest_revision_id", revision.Id).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&topic).Update("first_post_id", post.Id).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&topicCategoryIndex.Entity{TopicId: topic.Id, CategoryId: 9614, Effective: 1}).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Where("post_id = ?", post.Id).Delete(&moderationAppeals.Entity{})
		conn.Where("post_id = ?", post.Id).Delete(&postRevisions.Entity{})
		conn.Where("topic_id = ?", topic.Id).Delete(&topicCategoryIndex.Entity{})
		conn.Unscoped().Delete(&post)
		conn.Unscoped().Delete(&topic)
		conn.Where("user_id = ?", owner).Delete(&eventNotification.Entity{})
		conn.Delete(&grant)
		conn.Unscoped().Where("id IN ?", []uint64{owner, mod, outsider}).Delete(&users.EntityComplete{})
	})
	if _, err := publicationservice.SubmitAppeal(ctx, outsider, "topic", topic.Id, revision.Id, "explanation"); !errors.Is(err, publicationservice.ErrAppealPermission) {
		t.Fatalf("nonowner submit: %v", err)
	}
	if _, err := publicationservice.SubmitAppeal(ctx, owner, "topic", topic.Id, revision.Id, "  "); !errors.Is(err, publicationservice.ErrAppealReason) {
		t.Fatalf("blank explanation: %v", err)
	}
	if state := publicationservice.OwnerAppealStateFor(ctx, owner, topic, post); !state.CanAppeal || state.Appeal != nil {
		t.Fatalf("fresh rejection should be appealable: %+v", state)
	}
	if state := publicationservice.OwnerAppealStateFor(ctx, outsider, topic, post); state.CanAppeal || state.Appeal != nil {
		t.Fatalf("nonowner state leaked: %+v", state)
	}
	appeal, err := publicationservice.SubmitAppeal(ctx, owner, "topic", topic.Id, revision.Id, " Please review ")
	if err != nil {
		t.Fatal(err)
	}
	var audit moderationAppeals.Entity
	conn.First(&audit, appeal.Id)
	if audit.OriginalReviewReason != revision.ReviewReason || audit.OriginalReviewActorId != 91 || audit.OriginalReviewedAt == nil {
		t.Fatal("lost initial decision audit")
	}
	if _, err := publicationservice.SubmitAppeal(ctx, owner, "topic", topic.Id, revision.Id, "again"); !errors.Is(err, publicationservice.ErrAppealDuplicate) {
		t.Fatalf("duplicate appeal: %v", err)
	}
	if state := publicationservice.OwnerAppealStateFor(ctx, owner, topic, post); state.CanAppeal || state.Appeal == nil || state.Appeal.Status != moderationAppeals.Pending || state.Appeal.Reason != "Please review" {
		t.Fatalf("submitted appeal state: %+v", state)
	}
	var receipt eventNotification.Entity
	if err := conn.Where("user_id = ? AND event_type = ?", owner, eventNotification.EventTypeReviewPending).Order("id DESC").Take(&receipt).Error; err != nil || receipt.Payload.TemplateKey != eventNotification.TemplateAppealSubmitted {
		t.Fatalf("receipt template %q %v", receipt.Payload.TemplateKey, err)
	}
	items, total, err := publicationservice.ModeratorAppeals(ctx, mod, 1, 20, "pending")
	if err != nil || total != 1 || len(items) != 1 || items[0].Content != revision.Content || items[0].Reason != "Please review" || items[0].OriginalReviewReason != "original decision" {
		t.Fatalf("queue: %+v %d %v", items, total, err)
	}
	// 话题移出分类时索引行只是软切换（effective=0），旧分类版主不得再看到私有候选。
	if err := conn.Model(&topicCategoryIndex.Entity{}).Where("topic_id = ?", topic.Id).Update("effective", 0).Error; err != nil {
		t.Fatal(err)
	}
	if items, total, err = publicationservice.ModeratorAppeals(ctx, mod, 1, 20, "pending"); err != nil || total != 0 || len(items) != 0 {
		t.Fatalf("inactive category index leaked %d %v", total, err)
	}
	if err := conn.Model(&topicCategoryIndex.Entity{}).Where("topic_id = ?", topic.Id).Update("effective", 1).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&revision).Update("category_ids", "[9615]").Error; err != nil {
		t.Fatal(err)
	}
	items, total, err = publicationservice.ModeratorAppeals(ctx, mod, 1, 20, "pending")
	if err != nil || total != 0 || len(items) != 0 {
		t.Fatalf("candidate scope leak %d %v", total, err)
	}
	if err := publicationservice.ReviewAppeal(ctx, appeal.Id, mod, true, ""); !errors.Is(err, publicationservice.ErrAppealPermission) {
		t.Fatalf("out-of-scope action: %v", err)
	}
	if err := conn.Model(&revision).Update("category_ids", "[9614]").Error; err != nil {
		t.Fatal(err)
	}
	if err := publicationservice.ReviewAppeal(ctx, appeal.Id, owner, true, ""); !errors.Is(err, publicationservice.ErrAppealPermission) {
		t.Fatalf("self review: %v", err)
	}
	if err := publicationservice.ReviewAppeal(ctx, appeal.Id, mod, false, ""); !errors.Is(err, publicationservice.ErrAppealReason) {
		t.Fatalf("reasonless denial: %v", err)
	}
	// Simulate an edit committed while the appeal is pending. Reviewing cannot publish it.
	newer := postRevisions.Entity{PostId: post.Id, Version: 2, Content: "newer", ProcessStatus: posts.ProcessStatusPending}
	if err := conn.Create(&newer).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&post).Update("latest_revision_id", newer.Id).Error; err != nil {
		t.Fatal(err)
	}
	if err := publicationservice.ReviewAppeal(ctx, appeal.Id, mod, true, ""); !errors.Is(err, publicationservice.ErrAppealStale) {
		t.Fatalf("stale approval: %v", err)
	}
	var staleNotice eventNotification.Entity
	if err := conn.Where("user_id = ? AND event_type = ?", owner, eventNotification.EventTypeReviewRejected).Order("id DESC").Take(&staleNotice).Error; err != nil || staleNotice.Payload.TemplateKey != eventNotification.TemplateAppealStale {
		t.Fatalf("stale notice template %q %v", staleNotice.Payload.TemplateKey, err)
	}
	if err := conn.Table("moderation_appeals").Select("id,status,review_reason,reviewed_at").Take(&appeal, appeal.Id).Error; err != nil {
		t.Fatal(err)
	}
	if appeal.Status != moderationAppeals.Stale {
		t.Fatal("stale decision not durable")
	}
	if err := conn.First(&post, post.Id).Error; err != nil {
		t.Fatal(err)
	}
	if post.PublishedRevisionId != 0 || post.Content != "blocked content" {
		t.Fatal("stale appeal changed public projection")
	}
}

func TestModerationAppealApprovalAndRejection(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(migration.SchemaModels()...); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	owner := users.EntityComplete{Id: 961411, Username: "appeal-decision-owner"}
	reviewer := users.EntityComplete{Id: 961412, Username: "appeal-decision-reviewer"}
	for _, row := range []*users.EntityComplete{&owner, &reviewer} {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	grant := moderators.Entity{UserId: reviewer.Id, ScopeType: moderators.ScopeGlobal, Status: 1}
	if err := conn.Create(&grant).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Delete(&grant); conn.Unscoped().Delete(&owner); conn.Unscoped().Delete(&reviewer) })
	for _, approve := range []bool{false, true} {
		t.Run(map[bool]string{false: "reject", true: "approve"}[approve], func(t *testing.T) {
			topic := topics.Entity{UserId: owner.Id, Title: "rejected topic", Status: 1, ProcessStatus: topics.ProcessStatusBlocked}
			if err := conn.Create(&topic).Error; err != nil {
				t.Fatal(err)
			}
			post := posts.Entity{TopicId: topic.Id, PostNo: 1, UserId: owner.Id, Content: "body", ProcessStatus: posts.ProcessStatusBlocked}
			if err := conn.Create(&post).Error; err != nil {
				t.Fatal(err)
			}
			rev := postRevisions.Entity{PostId: post.Id, Version: 1, Title: "approved candidate", Content: "candidate body", ProcessStatus: posts.ProcessStatusBlocked, ReviewReason: "original rejection"}
			if err := conn.Create(&rev).Error; err != nil {
				t.Fatal(err)
			}
			conn.Model(&post).Update("latest_revision_id", rev.Id)
			conn.Model(&topic).Update("first_post_id", post.Id)
			t.Cleanup(func() {
				conn.Where("post_id = ?", post.Id).Delete(&moderationAppeals.Entity{})
				conn.Where("post_id = ?", post.Id).Delete(&postRevisions.Entity{})
				conn.Unscoped().Delete(&post)
				conn.Unscoped().Delete(&topic)
			})
			appeal, err := publicationservice.SubmitAppeal(ctx, owner.Id, "topic", topic.Id, rev.Id, "explanation")
			if err != nil {
				t.Fatal(err)
			}
			if err := publicationservice.ReviewAppeal(ctx, appeal.Id, reviewer.Id, approve, "decision explanation"); err != nil {
				t.Fatal(err)
			}
			conn.Table("moderation_appeals").Select("id,status,review_reason,reviewed_at").Take(&appeal, appeal.Id)
			conn.First(&post, post.Id)
			conn.First(&topic, topic.Id)
			expected := moderationAppeals.Rejected
			if approve {
				expected = moderationAppeals.Approved
			}
			if appeal.Status != expected || appeal.ReviewedAt == nil || appeal.ReviewReason != "decision explanation" {
				t.Fatalf("decision %+v", appeal)
			}
			if approve && (post.PublishedRevisionId != rev.Id || topic.ProcessStatus != topics.ProcessStatusNormal || post.Content != rev.Content) {
				t.Fatal("approval did not publish exact candidate")
			}
			if !approve && (post.PublishedRevisionId != 0 || topic.ProcessStatus != topics.ProcessStatusBlocked) {
				t.Fatal("denial published candidate")
			}
			if err := publicationservice.ReviewAppeal(ctx, appeal.Id, reviewer.Id, approve, "repeat"); !errors.Is(err, publicationservice.ErrAppealProcessed) {
				t.Fatalf("duplicate decision %v", err)
			}
			var notices []eventNotification.Entity
			conn.Where("user_id = ? AND topic_id = ?", owner.Id, topic.Id).Find(&notices)
			if len(notices) != 2 {
				t.Fatalf("expected receipt and result, got %d", len(notices))
			}
			result := map[bool]string{false: eventNotification.TemplateAppealRejected, true: eventNotification.TemplateAppealApproved}[approve]
			templates := map[string]bool{}
			for _, notice := range notices {
				if notice.Payload.Content != "" || notice.Payload.ActorId != 0 {
					t.Fatal("unsafe notification")
				}
				templates[notice.Payload.TemplateKey] = true
			}
			if !templates[eventNotification.TemplateAppealSubmitted] || !templates[result] {
				t.Fatalf("appeal notices must use appeal templates, got %v", templates)
			}
			if state := publicationservice.OwnerAppealStateFor(ctx, owner.Id, topic, post); !approve && (state.CanAppeal || state.Appeal == nil || state.Appeal.Status != moderationAppeals.Rejected || state.Appeal.ReviewReason != "decision explanation") {
				t.Fatalf("rejected appeal state: %+v", state)
			}
		})
	}
}

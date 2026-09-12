package posts

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
)

// setupRevisionPurgeDB 迁移 posts + post_revisions 两张表，
// 供永久删除/隐私擦除终态（数据保留，MADR-0021）的测试使用。
func setupRevisionPurgeDB(t *testing.T) {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}, &postRevisions.Entity{}); err != nil {
		t.Fatalf("migrate purge tables: %v", err)
	}
}

// createPostWithRevisions 直接插入一条帖子（含可见性/保留状态）与两个版本
// （v1 正常、v2 待审），返回 postID。
func createPostWithRevisions(t *testing.T, postID uint64, visibility, retention string) uint64 {
	t.Helper()
	conn := dbconnect.Connect()
	post := Entity{
		Id: postID, TopicId: 90000 + postID, PostNo: 1, UserId: 1,
		Content: "body before purge", RenderedHTML: "<p>body before purge</p>",
		ProcessStatus:    ProcessStatusNormal,
		VisibilityStatus: visibility, RetentionStatus: retention,
		CreatedAt: time.Now().Add(-time.Hour), UpdatedAt: time.Now().Add(-time.Hour),
	}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	for _, version := range []uint64{1, 2} {
		rev := postRevisions.Entity{
			PostId: postID, Version: version, EditorId: 1,
			Content: "revision body", RenderedHTML: "<p>revision body</p>",
			ProcessStatus: ProcessStatusNormal,
		}
		if version == 2 {
			rev.ProcessStatus = ProcessStatusPending
		}
		if err := conn.Create(&rev).Error; err != nil {
			t.Fatalf("create revision v%d: %v", version, err)
		}
	}
	return postID
}

// assertRevisionContentRetained 断言某帖全部版本正文保留
// （删除终态为数据保留，MADR-0021；取证经 view-deleted-content 审计通道）。
func assertRevisionContentRetained(t *testing.T, postID uint64) {
	t.Helper()
	versions := postRevisions.ListByPostId(postID)
	if len(versions) != 2 {
		t.Fatalf("revision count = %d, want 2", len(versions))
	}
	for _, v := range versions {
		if v.Content == "" || v.RenderedHTML == "" {
			t.Fatalf("revision v%d content must be retained (MADR-0021) = %q / %q", v.Version, v.Content, v.RenderedHTML)
		}
	}
}

// TestMarkPrivacyErasedRetainsRevisionContent 验证隐私擦除（#492 无楼话题
// 联动下架）只做状态翻转：正文与全部版本保留（数据保留终态 MADR-0021），
// visibility 置 ACCOUNT_ANONYMIZED 且不可恢复。
func TestMarkPrivacyErasedRetainsRevisionContent(t *testing.T) {
	setupRevisionPurgeDB(t)
	postID := createPostWithRevisions(t, 9301, VisibilityActive, RetentionNormal)

	if err := MarkPrivacyErased(postID, 999, "privacy erase test"); err != nil {
		t.Fatalf("MarkPrivacyErased() err=%v", err)
	}
	post := UnscopedGet(postID)
	if post.VisibilityStatus != VisibilityAccountAnonymized {
		t.Fatalf("visibility_status = %q, want ACCOUNT_ANONYMIZED", post.VisibilityStatus)
	}
	if post.RetentionStatus != RetentionPurged {
		t.Fatalf("retention_status = %q, want PURGED", post.RetentionStatus)
	}
	if post.Content != "body before purge" || post.RenderedHTML != "<p>body before purge</p>" {
		t.Fatalf("post content must be retained (MADR-0021) = %q / %q", post.Content, post.RenderedHTML)
	}
	assertRevisionContentRetained(t, postID)
}

// TestMarkPurgedRetainsRevisionContent 验证永久删除（purge）只做状态翻转：
// 正文与全部版本保留（数据保留终态 MADR-0021），用户不可恢复。
func TestMarkPurgedRetainsRevisionContent(t *testing.T) {
	setupRevisionPurgeDB(t)
	// MarkPurged 只处理已进入删除生命周期的回复（USER_DELETED/MODERATOR_REMOVED + RECOVERABLE）
	postID := createPostWithRevisions(t, 9302, VisibilityUserDeleted, RetentionRecoverable)

	if err := MarkPurged(postID); err != nil {
		t.Fatalf("MarkPurged() err=%v", err)
	}
	post := UnscopedGet(postID)
	if post.Content != "body before purge" || post.RenderedHTML != "<p>body before purge</p>" {
		t.Fatalf("post content must be retained (MADR-0021) = %q / %q", post.Content, post.RenderedHTML)
	}
	if post.RetentionStatus != RetentionPurged {
		t.Fatalf("retention_status = %q, want PURGED", post.RetentionStatus)
	}
	if !post.DeletedAt.Valid {
		t.Fatal("deleted_at must be set after purge")
	}
	assertRevisionContentRetained(t, postID)
}

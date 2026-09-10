package posts

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
)

// setupRevisionPurgeDB 迁移 posts + post_revisions 两张表，
// 供永久删除/隐私擦除联动清空版本正文的测试使用。
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

// assertRevisionContentBlanked 断言某帖全部版本正文均已清空。
func assertRevisionContentBlanked(t *testing.T, postID uint64) {
	t.Helper()
	versions := postRevisions.ListByPostId(postID)
	if len(versions) != 2 {
		t.Fatalf("revision count = %d, want 2", len(versions))
	}
	for _, v := range versions {
		if v.Content != "" || v.RenderedHTML != "" {
			t.Fatalf("revision v%d content not blanked = %q / %q", v.Version, v.Content, v.RenderedHTML)
		}
	}
}

// TestMarkPrivacyErasedBlanksRevisionContent 验证隐私擦除（账号注销联动）后
// 该帖全部版本正文被清空，原文不得经版本历史留存（行为 5）。
func TestMarkPrivacyErasedBlanksRevisionContent(t *testing.T) {
	setupRevisionPurgeDB(t)
	postID := createPostWithRevisions(t, 9301, VisibilityActive, RetentionNormal)

	if err := MarkPrivacyErased(postID, 999, "privacy erase test"); err != nil {
		t.Fatalf("MarkPrivacyErased() err=%v", err)
	}
	post := Get(postID)
	if post.Content != "" || post.RenderedHTML != "" {
		t.Fatalf("post content not blanked = %q / %q", post.Content, post.RenderedHTML)
	}
	assertRevisionContentBlanked(t, postID)
}

// TestMarkPurgedBlanksRevisionContent 验证永久删除（purge）后该帖全部版本
// 正文被清空（行为 5）。
func TestMarkPurgedBlanksRevisionContent(t *testing.T) {
	setupRevisionPurgeDB(t)
	// MarkPurged 只处理已进入删除生命周期的回复（USER_DELETED/MODERATOR_REMOVED + RECOVERABLE）
	postID := createPostWithRevisions(t, 9302, VisibilityUserDeleted, RetentionRecoverable)

	if err := MarkPurged(postID); err != nil {
		t.Fatalf("MarkPurged() err=%v", err)
	}
	post := UnscopedGet(postID)
	if post.Content != "" || post.RenderedHTML != "" {
		t.Fatalf("post content not blanked = %q / %q", post.Content, post.RenderedHTML)
	}
	if post.RetentionStatus != RetentionPurged {
		t.Fatalf("retention_status = %q, want PURGED", post.RetentionStatus)
	}
	assertRevisionContentBlanked(t, postID)
}

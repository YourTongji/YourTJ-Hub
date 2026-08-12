package contentdeleteservice

import (
	"testing"
	"time"

	"github.com/leancodebox/GooseForum/app/models/forum/posts"
)

// 回归 H1：作者先自删墓碑态回复（有子回复），版主治理删除后应升级为
// MODERATOR_REMOVED，作者不可再恢复（修复前版主删除 silent no-op → 逃罚）。
func TestModeratorDeleteUpgradesTombstonePost(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const topicID = uint64(944100)
	authorID, replyAuthorID := seedTopicWithOptionalReply(t, conn, topicID, true)
	const postID = topicID + 200
	const childID = topicID + 500

	// 给 reply 添加子回复，使其自删时进入墓碑态
	if err := conn.Create(&posts.Entity{
		Id: childID, TopicId: topicID, PostNo: 3, UserId: authorID, Content: "child",
		VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
		ReplyToPostId: postID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}
	// 作者自删（墓碑态）
	if _, err := DeletePostByUser(replyAuthorID, postID); err != nil {
		t.Fatalf("DeletePostByUser: %v", err)
	}

	// 版主治理删除：必须升级为 MODERATOR_REMOVED
	if err := DeletePostAsModerator(topicID+99, postID, "policy violation"); err != nil {
		t.Fatalf("DeletePostAsModerator: %v", err)
	}
	p := posts.UnscopedGet(postID)
	if p.VisibilityStatus != posts.VisibilityModeratorRemoved {
		t.Fatalf("post vis = %s, want MODERATOR_REMOVED (逃罚未堵住)", p.VisibilityStatus)
	}

	// 作者尝试恢复：必须被拒绝（MODERATOR_REMOVED 不可由作者恢复）
	if err := RestoreContent(replyAuthorID, ContentTypePost, postID); err == nil {
		t.Fatal("author restored post after moderator governance delete (逃罚)")
	}
}

// 回归 H1：作者自删软删态回复（无子回复，deleted_at 置位），版主治理删除
// 应升级为 MODERATOR_REMOVED（修复前 UnscopedGet 缺失导致"回复不存在"）。
func TestModeratorDeleteUpgradesSoftDeletedPost(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const topicID = uint64(944800)
	_, replyAuthorID := seedTopicWithOptionalReply(t, conn, topicID, true)
	const postID = topicID + 200

	if _, err := DeletePostByUser(replyAuthorID, postID); err != nil {
		t.Fatalf("DeletePostByUser: %v", err)
	}
	if err := DeletePostAsModerator(topicID+99, postID, "spam"); err != nil {
		t.Fatalf("DeletePostAsModerator: %v", err)
	}
	p := posts.UnscopedGet(postID)
	if p.VisibilityStatus != posts.VisibilityModeratorRemoved {
		t.Fatalf("post vis = %s, want MODERATOR_REMOVED", p.VisibilityStatus)
	}
}

// 回归 H3：管理端删除回复端点必须拒绝话题首楼（post_no<=1）。
func TestModeratorDeleteRejectsFirstPost(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const topicID = uint64(944600)
	authorID, _ := seedTopicWithOptionalReply(t, conn, topicID, false)
	const firstPostID = topicID + 100
	_ = authorID

	if err := DeletePostAsModerator(topicID+99, firstPostID, "remove first post"); err == nil {
		t.Fatal("expected first-post deletion to be rejected (H3 首楼守卫)")
	}
	// 首楼不应被改动
	p := posts.UnscopedGet(firstPostID)
	if p.VisibilityStatus != posts.VisibilityActive {
		t.Fatalf("first post state changed: %s", p.VisibilityStatus)
	}
}

// 回归：版主重复治理删除（MODERATOR_REMOVED 态）应幂等成功。
func TestModeratorDeleteIdempotent(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const topicID = uint64(946100)
	_, replyAuthorID := seedTopicWithOptionalReply(t, conn, topicID, true)
	const postID = topicID + 200

	if err := DeletePostAsModerator(topicID+99, postID, "first"); err != nil {
		t.Fatalf("DeletePostAsModerator: %v", err)
	}
	if err := DeletePostAsModerator(topicID+99, postID, "second"); err != nil {
		t.Fatalf("DeletePostAsModerator (idempotent): %v", err)
	}
	p := posts.UnscopedGet(postID)
	if p.VisibilityStatus != posts.VisibilityModeratorRemoved || p.DeleteReason != "first" {
		t.Fatalf("idempotent delete overwrote reason: %#v", p)
	}
	_ = replyAuthorID
}

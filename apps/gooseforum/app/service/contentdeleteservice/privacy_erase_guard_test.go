package contentdeleteservice

import (
	"testing"

	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"gorm.io/gorm"
)

func zzCleanupContent(t *testing.T, conn *gorm.DB, ids []uint64) {
	t.Helper()
	t.Cleanup(func() {
		conn.Unscoped().Where("id IN ?", ids).Delete(&posts.Entity{})
		conn.Unscoped().Where("id IN ?", ids).Delete(&topics.Entity{})
	})
}

// 回归 MEDIUM-3：作者不得通过 privacy-erase 擦除治理删除（MODERATOR_REMOVED）
// 的内容 —— 治理证据与审计链必须保留（review 发现：修复前 privacy-erase
// 无状态前置条件，作者可对治理删除内容清空标题/正文并置 PURGED，管理端
// 恢复通道随之失效）。
func TestPrivacyEraseRejectsModeratorRemovedTopic(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const topicID = uint64(948100)
	authorID, _ := seedTopicWithOptionalReply(t, conn, topicID, true)
	zzCleanupContent(t, conn, []uint64{topicID, topicID + 100, topicID + 200, topicID + 300})

	// 版主治理删除该话题
	topicBefore := topics.UnscopedGet(topicID)
	if err := DeleteTopicAs(topicBefore, topicID+99, topics.VisibilityModeratorRemoved, "policy violation"); err != nil {
		t.Fatalf("DeleteTopicAs(moderator): %v", err)
	}
	topic := topics.UnscopedGet(topicID)
	if topic.VisibilityStatus != topics.VisibilityModeratorRemoved {
		t.Fatalf("precondition: topic vis = %s, want MODERATOR_REMOVED", topic.VisibilityStatus)
	}

	// 作者尝试隐私擦除：必须被拒绝，且治理证据（标题/正文）保留
	if err := PrivacyEraseContent(authorID, ContentTypeTopic, topicID); err == nil {
		t.Fatal("privacy-erase succeeded on MODERATOR_REMOVED topic (治理证据被擦除)")
	}
	after := topics.UnscopedGet(topicID)
	if after.VisibilityStatus != topics.VisibilityModeratorRemoved || after.RetentionStatus != topics.RetentionRecoverable {
		t.Fatalf("topic state mutated by rejected privacy-erase: %s/%s", after.VisibilityStatus, after.RetentionStatus)
	}
	if after.Title == "" {
		t.Fatal("topic title wiped despite rejected privacy-erase")
	}
}

// 回归 MEDIUM-3（post 分支）：治理删除的回复同样不可被作者隐私擦除。
func TestPrivacyEraseRejectsModeratorRemovedPost(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const topicID = uint64(948300)
	authorID, replyAuthorID := seedTopicWithOptionalReply(t, conn, topicID, true)
	const postID = topicID + 200
	zzCleanupContent(t, conn, []uint64{topicID, topicID + 100, topicID + 200, topicID + 300})

	if err := DeletePostAsModerator(topicID+99, postID, "spam"); err != nil {
		t.Fatalf("DeletePostAsModerator: %v", err)
	}
	p := posts.UnscopedGet(postID)
	if p.VisibilityStatus != posts.VisibilityModeratorRemoved {
		t.Fatalf("precondition: post vis = %s, want MODERATOR_REMOVED", p.VisibilityStatus)
	}

	if err := PrivacyEraseContent(replyAuthorID, ContentTypePost, postID); err == nil {
		t.Fatal("privacy-erase succeeded on MODERATOR_REMOVED post")
	}
	after := posts.UnscopedGet(postID)
	if after.VisibilityStatus != posts.VisibilityModeratorRemoved {
		t.Fatalf("post state mutated by rejected privacy-erase: %s", after.VisibilityStatus)
	}
	if after.Content == "" {
		t.Fatal("post content wiped despite rejected privacy-erase")
	}
	// 正常用户删除的回复（USER_DELETED）仍可隐私擦除 —— 不破坏 R8 主路径
	if err := PrivacyEraseContent(authorID, ContentTypePost, topicID+100); err != nil {
		t.Fatalf("privacy-erase on own USER_DELETED content should still work: %v", err)
	}
}

// 回归 R8 主路径：普通已删话题仍可被作者隐私擦除（守卫不得误伤）。
func TestPrivacyEraseStillWorksForUserDeletedTopic(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const topicID = uint64(948500)
	authorID, _ := seedTopicWithOptionalReply(t, conn, topicID, false)
	zzCleanupContent(t, conn, []uint64{topicID, topicID + 100})

	if err := DeleteTopicByUser(authorID, topicID); err != nil {
		t.Fatalf("DeleteTopicByUser: %v", err)
	}
	topic := topics.UnscopedGet(topicID)
	if topic.VisibilityStatus != topics.VisibilityUserDeleted {
		t.Fatalf("precondition: topic vis = %s, want USER_DELETED", topic.VisibilityStatus)
	}

	if err := PrivacyEraseContent(authorID, ContentTypeTopic, topicID); err != nil {
		t.Fatalf("privacy-erase on own USER_DELETED topic should still work: %v", err)
	}
	after := topics.UnscopedGet(topicID)
	if after.RetentionStatus != topics.RetentionPurged {
		t.Fatalf("topic retention after privacy-erase = %s, want PURGED", after.RetentionStatus)
	}
	if after.Title != "" {
		t.Fatalf("topic title not wiped after privacy-erase: %q", after.Title)
	}
	// 幂等：重复隐私擦除不报错
	if err := PrivacyEraseContent(authorID, ContentTypeTopic, topicID); err != nil {
		t.Fatalf("repeated privacy-erase should be idempotent: %v", err)
	}
}

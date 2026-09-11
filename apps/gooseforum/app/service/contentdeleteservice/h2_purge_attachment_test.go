package contentdeleteservice

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
)

// 回归 H2（MADR-0021 后语义）：永久删除话题后，作者本人的回复（含 ACTIVE 自回帖）
// 置 PURGED 但正文保留（数据保留终态，取证经 view-deleted-content）；附件引用
// 退役（不再 ACTIVE，切断公开下载），附件本体保留。他人仍 ACTIVE 的回复正文
// 保留（PRD 不删他人内容），其附件引用同样不得再公开访问。
func TestPurgeTopicCleansOwnerPostsAndHardensOtherAttachments(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	if err := conn.AutoMigrate(&fileUsage.Entity{}); err != nil {
		t.Fatalf("migrate fileUsage: %v", err)
	}
	const topicID = uint64(945100)
	authorID, replyAuthorID := seedTopicWithOptionalReply(t, conn, topicID, true)
	const ownerPostID = topicID + 500

	// 作者自回帖（ACTIVE）带附件
	if err := conn.Create(&posts.Entity{
		Id: ownerPostID, TopicId: topicID, PostNo: 3, UserId: authorID, Content: "author own reply",
		VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}).Error; err != nil {
		t.Fatalf("create owner post: %v", err)
	}
	if err := conn.Create(&fileUsage.Entity{FileName: "owner-img.png", TargetType: "post", TargetId: ownerPostID,
		UsageType: "inline_image", UserId: authorID, Status: fileUsage.UsageStatusActive}).Error; err != nil {
		t.Fatalf("create owner file usage: %v", err)
	}
	// 他人 reply 带附件
	if err := conn.Create(&fileUsage.Entity{FileName: "other-img.png", TargetType: "post", TargetId: topicID + 200,
		UsageType: "inline_image", UserId: replyAuthorID, Status: fileUsage.UsageStatusActive}).Error; err != nil {
		t.Fatalf("create other file usage: %v", err)
	}

	// 作者删除话题（有回复 → 首楼墓碑、回复保留）再永久删除
	if err := DeleteTopicByUser(authorID, topicID); err != nil {
		t.Fatalf("DeleteTopicByUser: %v", err)
	}
	if err := PurgeContent(authorID, ContentTypeTopic, topicID, "user_purge"); err != nil {
		t.Fatalf("PurgeContent: %v", err)
	}

	topic := topics.UnscopedGet(topicID)
	if topic.RetentionStatus != topics.RetentionPurged {
		t.Fatalf("topic not purged: %s", topic.RetentionStatus)
	}

	// 作者首楼（墓碑）→ PURGED，正文保留（数据保留，取证可还原）
	first := posts.UnscopedGet(topicID + 100)
	if first.RetentionStatus != posts.RetentionPurged {
		t.Fatalf("owner first post not purged: %s", first.RetentionStatus)
	}
	if first.Content != "first post body" {
		t.Fatalf("purged first post content must be retained (MADR-0021): %q", first.Content)
	}
	// 作者自回帖（原 ACTIVE）→ PURGED，正文保留，附件引用不再 ACTIVE
	owner := posts.UnscopedGet(ownerPostID)
	if owner.RetentionStatus != posts.RetentionPurged {
		t.Fatalf("owner post not purged: %s", owner.RetentionStatus)
	}
	if owner.VisibilityStatus != posts.VisibilityUserDeleted {
		t.Fatalf("purged ACTIVE self-reply must leave ACTIVE visibility paths (review P2): %s", owner.VisibilityStatus)
	}
	if owner.Content != "author own reply" {
		t.Fatalf("purged owner post content must be retained (MADR-0021): %q", owner.Content)
	}
	var ownerUsage fileUsage.Entity
	if err := conn.Where("file_name = ?", "owner-img.png").First(&ownerUsage).Error; err != nil {
		t.Fatalf("owner usage: %v", err)
	}
	if ownerUsage.Status == fileUsage.UsageStatusActive {
		t.Fatalf("owner attachment still ACTIVE after purge (可公开下载)")
	}

	// 他人 reply（原 ACTIVE）→ 正文保留，但附件不得再 ACTIVE
	other := posts.UnscopedGet(topicID + 200)
	if other.VisibilityStatus != posts.VisibilityActive || other.Content != "reply body" {
		t.Fatalf("other user reply changed: %s content=%q", other.VisibilityStatus, other.Content)
	}
	var otherUsage fileUsage.Entity
	if err := conn.Where("file_name = ?", "other-img.png").First(&otherUsage).Error; err != nil {
		t.Fatalf("other usage: %v", err)
	}
	if otherUsage.Status == fileUsage.UsageStatusActive {
		t.Fatalf("other user attachment still ACTIVE after topic purge (泄露)")
	}
}

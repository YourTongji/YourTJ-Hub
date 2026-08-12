package contentdeleteservice

import (
	"testing"
	"time"

	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
)

// 回归：墓碑态回复（有子回复）删除后，30 天窗口内应立即可恢复（修复前按旧 updated_at
// 判定为"已超出恢复窗口"）。
func TestRestoreTombstonePostWithinWindow(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const topicID = uint64(942100)
	authorID, replyAuthorID := seedTopicWithOptionalReply(t, conn, topicID, true) // topic + 首楼 + reply(post_no=2)

	// 给 reply 添加一个子回复，使其删除时进入墓碑态
	childPostID := uint64(topicID + 400)
	if err := conn.Create(&posts.Entity{
		Id: childPostID, TopicId: topicID, PostNo: 3, UserId: authorID, Content: "child",
		VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
		ReplyToPostId: topicID + 200, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}

	// reply 作者删除自己的 reply（有子回复 → 墓碑态，deleted_at 不置位）
	result, err := DeletePostByUser(replyAuthorID, topicID+200)
	if err != nil {
		t.Fatalf("DeletePostByUser: %v", err)
	}
	if !result.HasChildren {
		t.Fatal("expected tombstone branch (HasChildren=true)")
	}
	tomb := posts.UnscopedGet(topicID + 200)
	if tomb.DeletedAt.Valid {
		t.Fatalf("tombstone must not set deleted_at: %#v", tomb)
	}
	// 墓碑行 updated_at 应被刷新为删除时刻
	if time.Since(tomb.UpdatedAt) > time.Minute {
		t.Fatalf("tombstone updated_at not refreshed to delete time: %v", tomb.UpdatedAt)
	}

	// 立即恢复：修复前这里返回"已超出恢复窗口"
	if err := RestoreContent(replyAuthorID, ContentTypePost, topicID+200); err != nil {
		t.Fatalf("RestoreContent tombstone within window should succeed: %v", err)
	}
	restored := posts.UnscopedGet(topicID + 200)
	if restored.VisibilityStatus != posts.VisibilityActive || restored.RetentionStatus != posts.RetentionNormal {
		t.Fatalf("restored state = %s/%s, want ACTIVE/NORMAL", restored.VisibilityStatus, restored.RetentionStatus)
	}
	if restored.Content != "reply body" {
		t.Fatalf("restored content = %q, want %q", restored.Content, "reply body")
	}
}

// 回归：删除有回复的话题后，长寿命首楼的墓碑态不应被 retention cron 按旧 updated_at
// 误判为过期清空；30 天窗口内恢复话题，首楼正文必须完整保留（修复前会永久丢失）。
func TestTombstoneFirstPostSurvivesExpiryCron(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	authorID, _ := seedTopicWithOptionalReply(t, conn, 943200, true) // topic + 首楼 + reply

	// 模拟长寿命话题：首楼 updated_at 在 40 天前（修复前此值被当作删除时刻导致立即过期）
	old := time.Now().Add(-40 * 24 * time.Hour)
	if err := conn.Model(&posts.Entity{}).Where("id = ?", 943200+100).
		Update("updated_at", old).Error; err != nil {
		t.Fatalf("age first post: %v", err)
	}

	if err := DeleteTopicByUser(authorID, 943200); err != nil {
		t.Fatalf("DeleteTopicByUser: %v", err)
	}
	tomb := posts.UnscopedGet(943200 + 100)
	if tomb.DeletedAt.Valid || tomb.VisibilityStatus != posts.VisibilityUserDeleted {
		t.Fatalf("first post not tombstone: %#v", tomb)
	}
	// 墓碑首楼 updated_at 应被刷新为删除时刻，不再停留在 40 天前
	if time.Since(tomb.UpdatedAt) > time.Minute {
		t.Fatalf("tombstone first post updated_at not refreshed: %v", tomb.UpdatedAt)
	}

	// 模拟 retention cron：以 30 天窗口起点判定，墓碑首楼（删除时刻=今天）不应被选中
	expired := posts.ExpireRecoverable(time.Now().Add(-RecoveryWindow), 50)
	for _, p := range expired {
		if p.Id == 943200+100 {
			t.Fatalf("tombstone first post wrongly selected for expiry (updated_at=%v)", p.UpdatedAt)
		}
	}

	// 30 天窗口内恢复话题，首楼正文必须保留
	if err := RestoreContent(authorID, ContentTypeTopic, 943200); err != nil {
		t.Fatalf("RestoreContent: %v", err)
	}
	restoredFirst := posts.UnscopedGet(943200 + 100)
	if restoredFirst.VisibilityStatus != posts.VisibilityActive || restoredFirst.RetentionStatus != posts.RetentionNormal {
		t.Fatalf("restored first post state = %s/%s", restoredFirst.VisibilityStatus, restoredFirst.RetentionStatus)
	}
	if restoredFirst.Content != "first post body" {
		t.Fatalf("DATA-LOSS: restored first post content = %q, want %q", restoredFirst.Content, "first post body")
	}
	topic := topics.UnscopedGet(943200)
	if topic.VisibilityStatus != topics.VisibilityActive {
		t.Fatalf("restored topic state = %s", topic.VisibilityStatus)
	}
}

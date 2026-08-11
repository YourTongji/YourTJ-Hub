package datamigration

import (
	"testing"
	"time"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
)

// D-2：历史已删行回填。deleted_at 非空但状态仍为默认 ACTIVE/NORMAL 的
// topics/posts 应被回填为 USER_DELETED + RECOVERABLE，进入 30 天清理管线。
func TestBackfillDeleteLifecycleFillsLegacyDeletedRows(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	now := time.Now()
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", 9_800_000_001).Delete(&topics.Entity{})
		conn.Unscoped().Where("id = ?", 9_800_000_002).Delete(&posts.Entity{})
	})

	// 历史已删行：deleted_at 非空、visibility/retention 为默认值。
	if err := conn.Create(&topics.Entity{
		Id:               9_800_000_001,
		Title:            "legacy deleted topic",
		UserId:           9_800_000_099,
		VisibilityStatus: topics.VisibilityActive,
		RetentionStatus:  topics.RetentionNormal,
	}).Error; err != nil {
		t.Fatalf("create legacy topic: %v", err)
	}
	conn.Unscoped().Model(&topics.Entity{}).Where("id = ?", 9_800_000_001).
		Update("deleted_at", now)

	if err := conn.Create(&posts.Entity{
		Id:               9_800_000_002,
		TopicId:          9_800_000_001,
		PostNo:           1,
		UserId:           9_800_000_099,
		Content:          "legacy deleted post",
		VisibilityStatus: posts.VisibilityActive,
		RetentionStatus:  posts.RetentionNormal,
	}).Error; err != nil {
		t.Fatalf("create legacy post: %v", err)
	}
	conn.Unscoped().Model(&posts.Entity{}).Where("id = ?", 9_800_000_002).
		Update("deleted_at", now)

	result := BackfillDeleteLifecycleWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("BackfillDeleteLifecycleWithDB() failed = %d last=%s", result.Failed, result.LastFailed)
	}
	if result.TopicsBackfilled != 1 || result.PostsBackfilled != 1 {
		t.Fatalf("backfill counts topics=%d posts=%d, want 1/1", result.TopicsBackfilled, result.PostsBackfilled)
	}

	topic := topics.UnscopedGet(9_800_000_001)
	if topic.VisibilityStatus != topics.VisibilityUserDeleted || topic.RetentionStatus != topics.RetentionRecoverable {
		t.Fatalf("topic state after backfill = %s/%s", topic.VisibilityStatus, topic.RetentionStatus)
	}
	post := posts.UnscopedGet(9_800_000_002)
	if post.VisibilityStatus != posts.VisibilityUserDeleted || post.RetentionStatus != posts.RetentionRecoverable {
		t.Fatalf("post state after backfill = %s/%s", post.VisibilityStatus, post.RetentionStatus)
	}

	// 幂等：再次执行不再处理已回填行。
	again := BackfillDeleteLifecycleWithDB(conn)
	if again.TopicsBackfilled != 0 || again.PostsBackfilled != 0 {
		t.Fatalf("second backfill counts topics=%d posts=%d, want 0/0", again.TopicsBackfilled, again.PostsBackfilled)
	}
}

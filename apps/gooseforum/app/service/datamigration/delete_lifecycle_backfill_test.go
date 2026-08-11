package datamigration

import (
	"testing"
	"time"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/fileUsage"
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

// D-2/P2：历史已删内容的附件引用应在回填时转入 RECOVERING，
// 使公开下载门禁对存量数据立即生效，而不是等到每日 retention 定时任务。
func TestBackfillDeleteLifecycleHardensLegacyFileUsages(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}, &fileUsage.Entity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	now := time.Now()
	const topicID = uint64(9_800_001_001)
	const postID = uint64(9_800_001_002)
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", topicID).Delete(&topics.Entity{})
		conn.Unscoped().Where("id = ?", postID).Delete(&posts.Entity{})
		conn.Unscoped().Where("file_name IN ?", []string{"legacy-topic.png", "legacy-post.png"}).Delete(&fileUsage.Entity{})
	})

	if err := conn.Create(&topics.Entity{
		Id:               topicID,
		Title:            "legacy topic with attachment",
		UserId:           9_800_001_099,
		VisibilityStatus: topics.VisibilityActive,
		RetentionStatus:  topics.RetentionNormal,
	}).Error; err != nil {
		t.Fatalf("create legacy topic: %v", err)
	}
	conn.Unscoped().Model(&topics.Entity{}).Where("id = ?", topicID).Update("deleted_at", now)

	if err := conn.Create(&posts.Entity{
		Id:               postID,
		TopicId:          topicID,
		PostNo:           1,
		UserId:           9_800_001_099,
		Content:          "legacy post with attachment",
		VisibilityStatus: posts.VisibilityActive,
		RetentionStatus:  posts.RetentionNormal,
	}).Error; err != nil {
		t.Fatalf("create legacy post: %v", err)
	}
	conn.Unscoped().Model(&posts.Entity{}).Where("id = ?", postID).Update("deleted_at", now)

	for _, usage := range []fileUsage.Entity{
		{FileName: "legacy-topic.png", TargetType: fileUsage.TargetTopic, TargetId: topicID, UsageType: fileUsage.UsageInlineImage, Status: fileUsage.UsageStatusActive},
		{FileName: "legacy-post.png", TargetType: fileUsage.TargetPost, TargetId: postID, UsageType: fileUsage.UsageInlineImage, Status: fileUsage.UsageStatusActive},
	} {
		if err := conn.Create(&usage).Error; err != nil {
			t.Fatalf("create file usage: %v", err)
		}
	}

	result := BackfillDeleteLifecycleWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("BackfillDeleteLifecycleWithDB() failed = %d last=%s", result.Failed, result.LastFailed)
	}

	var topicUsage fileUsage.Entity
	if err := conn.Where("target_type = ? AND target_id = ?", fileUsage.TargetTopic, topicID).First(&topicUsage).Error; err != nil {
		t.Fatalf("read topic usage: %v", err)
	}
	if topicUsage.Status != fileUsage.UsageStatusRecovering || topicUsage.ExpiresAt == nil {
		t.Fatalf("topic usage after backfill = %s expires=%v, want RECOVERING with expiry", topicUsage.Status, topicUsage.ExpiresAt)
	}
	if topicUsage.ExpiresAt.Before(time.Now().Add(29 * 24 * time.Hour)) {
		t.Fatalf("topic usage expiry too early: %v", topicUsage.ExpiresAt)
	}

	var postUsage fileUsage.Entity
	if err := conn.Where("target_type = ? AND target_id = ?", fileUsage.TargetPost, postID).First(&postUsage).Error; err != nil {
		t.Fatalf("read post usage: %v", err)
	}
	if postUsage.Status != fileUsage.UsageStatusRecovering || postUsage.ExpiresAt == nil {
		t.Fatalf("post usage after backfill = %s expires=%v, want RECOVERING with expiry", postUsage.Status, postUsage.ExpiresAt)
	}

	if fileUsage.HasActiveReferences("legacy-topic.png") || fileUsage.HasActiveReferences("legacy-post.png") {
		t.Fatal("legacy attachment should no longer be publicly downloadable after backfill")
	}
	if !fileUsage.HasAnyReferences("legacy-topic.png") {
		t.Fatal("legacy attachment should remain tracked (RECOVERING) after backfill")
	}
}

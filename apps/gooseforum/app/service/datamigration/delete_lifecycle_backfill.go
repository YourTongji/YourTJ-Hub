package datamigration

import (
	"log/slog"
	"time"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"gorm.io/gorm"
)

// DeleteLifecycleBackfillResult 汇总历史删除行回填结果（PRD G15/D-2）。
// 回填前只能以 deleted_at 是否为空判断删除状态，visibility/retention 均为
// 默认值 ACTIVE/NORMAL；回填后这些行进入双状态机语义，可被 retention
// scheduler 的 30 天清理管线接管，不再成为不归管的孤儿行。
type DeleteLifecycleBackfillResult struct {
	TopicsBackfilled int
	PostsBackfilled  int
	Failed           int
	LastFailed       string
}

// BackfillDeleteLifecycle 为历史已删除行回填 visibility_status/retention_status。
//
// 策略（与 PRD §5.2 迁移建议一致）：
//   - topics/posts 中 deleted_at 非空、且 visibility_status 仍为默认 ACTIVE 的行，
//     视为历史作者删除（当时无版主/匿名化状态机），回填为
//     USER_DELETED + RECOVERABLE，让历史数据进入 30 天恢复与清理管线；
//   - 已具备非 ACTIVE 状态的行跳过，避免覆盖既有语义；
//   - 幂等：重复执行只处理仍未回填的行。
func BackfillDeleteLifecycle() DeleteLifecycleBackfillResult {
	return BackfillDeleteLifecycleWithDB(db.Connect())
}

// BackfillDeleteLifecycleWithDB 使用指定连接执行回填，便于测试注入。
func BackfillDeleteLifecycleWithDB(conn *gorm.DB) DeleteLifecycleBackfillResult {
	result := DeleteLifecycleBackfillResult{}

	topicRows := conn.Model(&topics.Entity{}).
		Unscoped().
		Where("deleted_at IS NOT NULL").
		Where("visibility_status = ?", topics.VisibilityActive).
		Where("retention_status = ?", topics.RetentionNormal).
		UpdateColumns(map[string]any{
			"visibility_status": topics.VisibilityUserDeleted,
			"retention_status":  topics.RetentionRecoverable,
		})
	if topicRows.Error != nil {
		result.Failed++
		result.LastFailed = "topics_backfill:" + topicRows.Error.Error()
		slog.Error("backfill topic delete lifecycle failed", "err", topicRows.Error)
	} else {
		result.TopicsBackfilled = int(topicRows.RowsAffected)
	}

	postRows := conn.Model(&posts.Entity{}).
		Unscoped().
		Where("deleted_at IS NOT NULL").
		Where("visibility_status = ?", posts.VisibilityActive).
		Where("retention_status = ?", posts.RetentionNormal).
		UpdateColumns(map[string]any{
			"visibility_status": posts.VisibilityUserDeleted,
			"retention_status":  posts.RetentionRecoverable,
		})
	if postRows.Error != nil {
		result.Failed++
		result.LastFailed = "posts_backfill:" + postRows.Error.Error()
		slog.Error("backfill post delete lifecycle failed", "err", postRows.Error)
	} else {
		result.PostsBackfilled = int(postRows.RowsAffected)
	}

	if result.Failed == 0 {
		slog.Info("backfill delete lifecycle done",
			"topics", result.TopicsBackfilled,
			"posts", result.PostsBackfilled,
			"at", time.Now().Format(time.RFC3339))
	}
	return result
}

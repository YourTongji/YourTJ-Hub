package courseservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
)

// TaskTypeCourseReviewCleanup 是 course-review-cleanup worker 的任务类型前缀
// （issue #175 B3 隐私合规：删除隔离窗口后的课评正文/作者关联脱敏）。
const TaskTypeCourseReviewCleanup = "course-review-cleanup"

// ReviewCleanupRetentionDays 删除隔离窗口：status=deleted 行在此窗口内保留
// 完整正文与作者关联（允许恢复重写），窗口超期后由清理 job 脱敏。
const ReviewCleanupRetentionDays = 30

// reviewCleanupBatchSize 单次清理批次上限（吸收 #202：worker 循环分批直到
// 本批不足 batchSize；已清理行 author_user_id 置 NULL，批次间不会重复选中）。
const reviewCleanupBatchSize = 500

// CleanupExpiredReviewsBatch 清理一批删除隔离窗口超期的课程评价（issue #175
// B3，吸收 #202 的 NULL 设计）：
//   - 清空 content（正文不再留存）
//   - author_user_id 置 NULL（断开作者关联；NULL 在唯一索引
//     uniq_course_review_offering_author 中彼此不冲突，SQLite/PostgreSQL/
//     MySQL 一致——同 offering 多条已清理行可共存，同用户可重新写评新建行，
//     唯一冲突问题从根源消失）
//   - 行保留（status 仍为 deleted，可审计）；deleted_at 保持原始删除时刻
//     不变（清理不改锚点）
//
// 窗口判定：deleted_at 显式锚点（MarkReviewDeletedFromTx 写入），存量行
// （本功能上线前删除）回退 updated_at 近似（COALESCE，与 topics/posts
// 墓碑口径一致）。
// 已清理行的唯一标记是 author_user_id IS NULL：扫描条件带该谓词保证幂等。
// 统计投影无需修正：删除时已扣减 stats，清理不改 status（仍非 visible）。
// 更新带 status 谓词：防止与 ReactivateReviewTx（恢复重写）并发竞态。
func CleanupExpiredReviewsBatch(limit int, cutoff time.Time) (int, error) {
	if limit <= 0 {
		return 0, nil
	}
	conn := dbconnect.Connect()
	table := (&course.ReviewEntity{}).TableName()

	var rows []course.ReviewEntity
	if err := conn.Table(table).
		Where("status = ? AND author_user_id IS NOT NULL AND COALESCE(deleted_at, updated_at) < ?",
			course.ReviewStatusDeleted, cutoff).
		Limit(limit).
		Find(&rows).Error; err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	ids := make([]uint64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.Id)
	}
	res := conn.Table(table).
		Where(queryopt.In("id", ids)).
		Where("status = ?", course.ReviewStatusDeleted).
		Updates(map[string]any{
			"content":        "",
			"author_user_id": nil,
		})
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
}

// CleanupDeletedReviews 全量清理（CLI 手动触发）：循环分批直到无剩余过期行。
// cutoff 为隔离窗口时长（如 30 天），真实生效（spec 复审 N2：RetentionDays
// 覆盖链修复）；0 或负值时回退默认窗口。返回累计清理行数，失败返回 error。
func CleanupDeletedReviews(cutoff time.Duration) (int64, error) {
	if cutoff <= 0 {
		cutoff = time.Duration(ReviewCleanupRetentionDays) * 24 * time.Hour
	}
	cutoffTime := time.Now().Add(-cutoff)
	total := 0
	for {
		n, err := CleanupExpiredReviewsBatch(reviewCleanupBatchSize, cutoffTime)
		if err != nil {
			return int64(total), fmt.Errorf("cleanup expired deleted reviews: %w", err)
		}
		total += n
		if n < reviewCleanupBatchSize {
			return int64(total), nil
		}
	}
}

// CleanupReviewTask 是 course-review-cleanup worker 的任务负载。
type CleanupReviewTask struct {
	// RetentionDays 覆盖默认隔离窗口（0 表示用 ReviewCleanupRetentionDays）。
	RetentionDays int `json:"retentionDays,omitempty"`
}

// EnqueueCleanupTask 入队一次课评清理任务（cron 每日调用，或 CLI 手动触发）。
// 幂等：已有 pending/retrying 的清理任务时跳过，避免 cron 与手动触发叠加。
func EnqueueCleanupTask() error {
	payload, err := json.Marshal(CleanupReviewTask{})
	if err != nil {
		return err
	}
	var count int64
	if err := dbconnect.Connect().Table((&taskQueue.Entity{}).TableName()).
		Where("type LIKE ?", TaskTypeCourseReviewCleanup+"%").
		Where("status IN ?", []int{taskQueue.StatusPending, taskQueue.StatusRetrying}).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil // 已有待处理清理任务
	}
	return taskQueue.Create(&taskQueue.Entity{
		Type:     TaskTypeCourseReviewCleanup + ".run",
		TaskJson: string(payload),
	})
}

// RunCleanupTask 是 course-review-cleanup worker 的 handler：执行一次清理。
// 复用 backgroundservice.RunWorker 的领取/重试框架（issue #175 验收 3：
// 失败 retrying 至多 3 次后 failed，LastError 记录错误，日志可查）。
func RunCleanupTask(ctx context.Context, task *taskQueue.Entity) error {
	var payload CleanupReviewTask
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil {
		return fmt.Errorf("decode cleanup task: %w", err)
	}
	retentionDays := ReviewCleanupRetentionDays
	if payload.RetentionDays > 0 {
		// 兜底（security F5）：窗口 1..365 天，防 task JSON 异常值
		// 导致窗口塌缩（0/负 → 清全部）或长到永不清理。
		retentionDays = payload.RetentionDays
		if retentionDays < 1 {
			retentionDays = 1
		}
		if retentionDays > 365 {
			retentionDays = 365
		}
	}
	cleaned, err := CleanupDeletedReviews(time.Duration(retentionDays) * 24 * time.Hour)
	if err != nil {
		return err
	}
	slog.Info("course review cleanup finished",
		"taskId", task.Id,
		"retentionDays", retentionDays,
		"cleaned", cleaned,
	)
	return nil
}

// RecoverStaleTasks 启动时恢复 course-review-cleanup worker 崩溃遗留的
// Running 任务（与其余 worker 共用同一恢复逻辑）。
func RecoverStaleTasks() error {
	return taskQueue.RecoverStaleRunning(TaskTypeCourseReviewCleanup, 10*time.Minute)
}

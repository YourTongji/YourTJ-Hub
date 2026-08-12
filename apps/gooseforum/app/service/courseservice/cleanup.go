package courseservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"gorm.io/gorm"
)

// TaskTypeCourseReviewCleanup 是 course-review-cleanup worker 的任务类型前缀
// （issue #175 B3 隐私合规：删除隔离窗口后的课评正文/作者关联脱敏）。
const TaskTypeCourseReviewCleanup = "course-review-cleanup"

// ReviewCleanupRetentionDays 删除隔离窗口：status=deleted 行在此窗口内保留
// 完整正文与作者关联（允许恢复重写），窗口超期后由清理 job 脱敏。
const ReviewCleanupRetentionDays = 30

// CleanupDeletedReviews 清理删除隔离窗口超期的课程评价（issue #175 B3）：
// 清空 content、断开 author_user_id（释放 (offering_id, author_user_id) 唯一
// 约束占位，同 offering 同用户可重新写评）、行保留可审计。
// 返回本次清理的行数。失败时返回 error，由 taskQueue worker 按重试语义处理
// （retrying 至多 3 次后 failed，并有日志）。
func CleanupDeletedReviews(cutoff time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-cutoff)
	var cleaned int64
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		var err error
		cleaned, err = course.CleanupExpiredDeletedReviewsTx(tx, cutoffTime)
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("cleanup expired deleted reviews: %w", err)
	}
	return cleaned, nil
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

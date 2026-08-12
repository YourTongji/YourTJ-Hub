package courseservice

import (
	"context"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
)

// ReviewCleanupWindow 课评删除隔离窗口：status=deleted 的正文与作者关联保留
// 该时长后才清理（隐私合规 B3）。窗口起点是删除时刻写入的 deleted_at；
// 存量 deleted 行（本功能上线前删除）没有 deleted_at，回退用 updated_at 近似
// （与 topics/posts 墓碑行口径一致）。
const ReviewCleanupWindow = 30 * 24 * time.Hour

// reviewCleanupBatchSize 单次清理批次上限（批量模式与 CLI 共用）。
const reviewCleanupBatchSize = 200

// TaskTypeCourseReviewCleanup 是课评清理 outbox worker 的任务类型前缀。
const TaskTypeCourseReviewCleanup = "course-review-cleanup."

// EnqueueCourseReviewCleanupTask 入队一次课评隔离窗口清理（每日 cron 调用）。
// 去重：已存在 pending/retrying 的清理任务时跳过，避免堆积重复任务
// （worker 消费慢时每日一任务足以，无需排队）。
func EnqueueCourseReviewCleanupTask() error {
	conn := db.Connect()
	var count int64
	if err := conn.Table((&taskQueue.Entity{}).TableName()).
		Where("type LIKE ?", TaskTypeCourseReviewCleanup+"%").
		Where("status IN ?", []int{taskQueue.StatusPending, taskQueue.StatusRetrying}).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil // 已有待处理清理任务，合并
	}
	return taskQueue.Create(&taskQueue.Entity{
		Type:     TaskTypeCourseReviewCleanup + "sweep",
		TaskJson: `{}`,
	})
}

// RunCourseReviewCleanupTask worker 处理：分批清理直到本批不足 batchSize
// （已清理行 author_user_id 置 NULL，批次间不会重复选中同一行）。
// 任一批次失败即返回错误，由 backgroundservice 框架重试（至多 3 次后 failed）。
func RunCourseReviewCleanupTask(ctx context.Context, _ *taskQueue.Entity) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		cleaned, err := CleanupExpiredReviewsBatch(reviewCleanupBatchSize)
		if err != nil {
			return err
		}
		if cleaned < reviewCleanupBatchSize {
			return nil
		}
	}
}

// CleanupExpiredReviewsBatch 清理一批隔离窗口已过的 deleted 课评（隐私合规 B3）：
//   - 清空 content（正文不再留存）
//   - author_user_id 置 NULL（断开作者关联；NULL 在唯一索引
//     uniq_course_review_offering_author 中彼此不冲突，同 offering 多条已清理行可共存，
//     且同用户可重新写评新建行）
//   - 行保留（status 仍为 deleted，可审计）；deleted_at 保持原始删除时刻不变
//
// 已清理行的唯一标记是 author_user_id IS NULL：扫描条件带该谓词保证幂等
// （已清理行不再被选中），且 deleted_at 语义不被清理动作污染（仍为删除时刻）。
// 统计投影无需修正：删除时已扣减 stats，清理不改 status（仍非 visible）。
func CleanupExpiredReviewsBatch(limit int) (int, error) {
	if limit <= 0 {
		return 0, nil
	}
	cutoff := time.Now().Add(-ReviewCleanupWindow)
	conn := db.Connect()
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
	// 更新带 status 条件：防止与 ReactivateReviewTx（恢复重写）并发竞态——
	// 行被选中后若被并发恢复为 visible，本更新不得把已恢复的评价清空/断开作者。
	// 不改写 deleted_at/updated_at：保留删除时刻供审计（幂等由 author_user_id
	// IS NULL 谓词保证，已清理行不会被再次选中）。
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

// RecoverCourseReviewCleanupStaleTasks 启动时恢复课评清理 worker 类型前缀下
// 崩溃遗留的 Running 任务（与其它 worker 的 RecoverStaleTasks 语义一致）。
func RecoverCourseReviewCleanupStaleTasks() error {
	return taskQueue.RecoverStaleRunning(TaskTypeCourseReviewCleanup, 10*time.Minute)
}

// CleanupExpiredReviewsAll 全量清理（CLI 手动触发）：循环分批直到无剩余过期行。
func CleanupExpiredReviewsAll() (int, error) {
	total := 0
	for {
		n, err := CleanupExpiredReviewsBatch(reviewCleanupBatchSize)
		if err != nil {
			return total, err
		}
		total += n
		if n < reviewCleanupBatchSize {
			return total, nil
		}
	}
}

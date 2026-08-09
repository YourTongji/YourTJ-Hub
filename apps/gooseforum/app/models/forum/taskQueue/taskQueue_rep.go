package taskQueue

import (
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/queryopt"
	"gorm.io/gorm"
)

func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

// CreateTx 在调用方事务内创建任务行（与 inbox upsert 同事务入队）。
func CreateTx(tx *gorm.DB, entity *Entity) error {
	return tx.Create(entity).Error
}

// UpdateRunAt 更新任务的最早可执行时间（延迟重试调度）。
func UpdateRunAt(id uint64, runAt time.Time) error {
	return builder().Where("id = ?", id).Update("run_at", runAt).Error
}

// dueFilter 只取已到期的任务：run_at 为空（存量任务立即执行）或 run_at <= now。
func dueFilter(query *gorm.DB) *gorm.DB {
	return query.Where("run_at IS NULL OR run_at <= ?", time.Now())
}

// GetPendingTasks 获取已到期的待处理任务
func GetPendingTasks(limit int) (tasks []*Entity) {
	dueFilter(builder().Where(queryopt.In("status", []int{StatusPending, StatusRetrying}))).
		Order("id asc").
		Limit(limit).
		Find(&tasks)
	return
}

// GetPendingTasksByType 获取指定类型前缀、已到期的待处理任务（按 type 前缀隔离 worker）。
func GetPendingTasksByType(typePrefix string, limit int) (tasks []*Entity) {
	query := builder()
	if typePrefix != "" {
		query = query.Where("type LIKE ?", typePrefix+"%")
	}
	dueFilter(query.Where(queryopt.In("status", []int{StatusPending, StatusRetrying}))).
		Order("id asc").
		Limit(limit).
		Find(&tasks)
	return
}

// GetPendingEmailTasks 获取邮件 worker 专属的、已到期的待处理任务。
// 新任务 type 带 "email." 前缀；存量任务仅 activation/reset_password 两种
// （历史邮件任务），显式白名单避免 export/file-migrate 等无前缀任务被误消费。
func GetPendingEmailTasks(limit int) (tasks []*Entity) {
	dueFilter(builder().
		Where("(type LIKE 'email.%' OR type IN (?, ?))", "activation", "reset_password").
		Where(queryopt.In("status", []int{StatusPending, StatusRetrying}))).
		Order("id asc").
		Limit(limit).
		Find(&tasks)
	return
}

// RequeueStaleRunningByType 将指定类型前缀下超时未完成的 running 任务恢复为 retrying。
// run_at 保持不变，使已安排的延迟重试仍按原计划执行。
func RequeueStaleRunningByType(typePrefix string, cutoff time.Time) (int64, error) {
	query := builder().
		Where("status = ? AND processed_at <= ?", StatusRunning, cutoff)
	if typePrefix != "" {
		query = query.Where("type LIKE ?", typePrefix+"%")
	}
	result := query.Update("status", StatusRetrying)
	return result.RowsAffected, result.Error
}

// UpdateStatus 更新任务状态
func UpdateStatus(id uint64, status uint8, err error) error {
	updates := map[string]any{
		"status":       status,
		"processed_at": time.Now(),
	}
	if err != nil {
		updates["last_error"] = err.Error()
	}
	return builder().Where("id = ?", id).Updates(updates).Error
}

// IncrementRetryCount 增加重试次数
func IncrementRetryCount(id uint64) error {
	return builder().Exec("UPDATE task_queue SET retry_count = retry_count + 1 where id = ?", id).Error
}

// QueryByTypeDesc returns tasks of the given type prefix, newest first.
func QueryByTypeDesc(typePrefix string, limit int) *gorm.DB {
	return builder().
		Where("type LIKE ?", typePrefix+"%").
		Order("id desc").
		Limit(limit)
}

// UpdateTaskJson updates a task's payload without touching its status.
func UpdateTaskJson(id uint64, taskJSON string) error {
	return builder().Where("id = ?", id).Update("task_json", taskJSON).Error
}

// GetByID returns a single task by id.
func GetByID(id any) (entity Entity, err error) {
	err = builder().Where("id = ?", id).First(&entity).Error
	return
}

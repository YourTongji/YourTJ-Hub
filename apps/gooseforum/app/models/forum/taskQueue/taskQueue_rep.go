package taskQueue

import (
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/queryopt"
	"gorm.io/gorm"
)

func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

// CreateTx 在业务事务内入队（transaction-bound outbox）：
// 任务行与业务写入同事务提交，崩溃前不提交 ⇒ 不产生任务；提交后 ⇒ worker 可消费。
func CreateTx(tx *gorm.DB, entity *Entity) error {
	return tx.Table(tableName).Create(entity).Error
}

// GetPendingTasks 获取待处理的任务
func GetPendingTasks(limit int) (tasks []*Entity) {
	builder().Where(queryopt.In("status", []int{StatusPending, StatusRetrying})).
		Order("id asc").
		Limit(limit).
		Find(&tasks)
	return
}

// GetPendingTasksByType 获取指定类型前缀的待处理任务（按 type 前缀隔离 worker）。
func GetPendingTasksByType(typePrefix string, limit int) (tasks []*Entity) {
	query := builder()
	if typePrefix != "" {
		query = query.Where("type LIKE ?", typePrefix+"%")
	}
	query.Where(queryopt.In("status", []int{StatusPending, StatusRetrying})).
		Order("id asc").
		Limit(limit).
		Find(&tasks)
	return
}

// GetPendingEmailTasks 获取邮件 worker 专属的待处理任务。
// 新任务 type 带 "email." 前缀；存量任务仅 activation/reset_password 两种
// （历史邮件任务），显式白名单避免 export/file-migrate 等无前缀任务被误消费。
func GetPendingEmailTasks(limit int) (tasks []*Entity) {
	builder().
		Where("(type LIKE 'email.%' OR type IN (?, ?))", "activation", "reset_password").
		Where(queryopt.In("status", []int{StatusPending, StatusRetrying})).
		Order("id asc").
		Limit(limit).
		Find(&tasks)
	return
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

// RecoverStaleRunning 将指定类型前缀下长时间停留在 Running 的任务恢复为 Pending
// （processed_at 超过 staleAfter 视为进程崩溃残留；正常运行的 worker 会持续
// 更新 processed_at）。启动时调用一次即可安全回收崩溃遗留任务，
// 避免任务在 Running 与终态之间永久不可见导致投影更新丢失。
func RecoverStaleRunning(typePrefix string, staleAfter time.Duration) error {
	cutoff := time.Now().Add(-staleAfter)
	return builder().
		Where("type LIKE ?", typePrefix+"%").
		Where("status = ?", StatusRunning).
		Where("processed_at < ?", cutoff).
		Updates(map[string]any{
			"status":       StatusPending,
			"last_error":   "recovered stale running task at startup",
			"processed_at": time.Now(),
		}).Error
}

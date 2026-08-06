package taskQueue

import (
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/queryopt"
)

func Create(entity *Entity) error {
	return builder().Create(entity).Error
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
// 新任务 type 带 "email." 前缀；存量任务（无 "." 前缀）仍按邮件任务处理，
// 避免历史邮件丢失。
func GetPendingEmailTasks(limit int) (tasks []*Entity) {
	builder().
		Where("(type LIKE 'email.%' OR type NOT LIKE '%.%')").
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

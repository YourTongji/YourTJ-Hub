package taskQueue

import (
	"time"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/queryopt"
	"gorm.io/gorm"
)

func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

// DeleteOwned 在仍持有租约（status=Running 且 processed_at=lease）时删除任务行。
// 仅用于 noop 等时化 dummy 任务（账号枚举防护 #124）消费后清理，避免 task_queue
// 无界增长；真实邮件任务仍保留为 Success 以留存审计痕迹。租约已被回收时静默跳过，
// 防止删除被其他 worker 重新领取的任务。
func DeleteOwned(id uint64, lease time.Time) error {
	return builder().
		Where("id = ? AND status = ? AND processed_at = ?", id, StatusRunning, lease).
		Delete(&Entity{}).Error
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

// emailTypeQuery 返回邮件 worker 的类型匹配条件（email.* 前缀 + 存量
// activation/reset_password 白名单）。GetPendingEmailTasks 与
// RecoverStaleEmailTasks 共用，保证"领取"与"回收"两侧谓词一致：存量历史
// 邮件行崩溃后同样能被回收，不会因类型不匹配而永久卡在 Running。
func emailTypeQuery() (clause string, args []any) {
	return "(type LIKE ? OR type IN (?, ?))", []any{"email.%", "activation", "reset_password"}
}

// GetPendingEmailTasks 获取邮件 worker 专属的待处理任务。
// 新任务 type 带 "email." 前缀；存量任务仅 activation/reset_password 两种
// （历史邮件任务），显式白名单避免 export/file-migrate 等无前缀任务被误消费。
func GetPendingEmailTasks(limit int) (tasks []*Entity) {
	clause, args := emailTypeQuery()
	builder().
		Where(clause, args...).
		Where(queryopt.In("status", []int{StatusPending, StatusRetrying})).
		Order("id asc").
		Limit(limit).
		Find(&tasks)
	return
}

// ClaimTask 原子领取任务（issue #138）：仅当任务仍处于 Pending/Retrying 时
// 将其置为 Running，并以 processed_at 记录租约起点（返回领取后的行，其
// ProcessedAt 为 DB 规范化后的租约值，供 RenewLease 的 CAS 续租使用）。
// 并发 worker 同时领取同一任务时只有一个成功（RowsAffected=1），其余返回
// claimed=false —— 取代原先"查询 pending + 无守卫更新 running"的两步分离，
// 消除多 worker 重复领取与重复外部副作用。
func ClaimTask(id uint64) (entity Entity, claimed bool, err error) {
	err = db.Connect().Transaction(func(tx *gorm.DB) error {
		res := tx.Table(tableName).
			Where("id = ? AND status IN (?, ?)", id, StatusPending, StatusRetrying).
			Updates(map[string]any{
				"status":       StatusRunning,
				"processed_at": time.Now(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return nil // 已被其他 worker 领取，或状态不可领取
		}
		claimed = true
		return tx.Table(tableName).Where("id = ?", id).First(&entity).Error
	})
	return
}

// RenewLease 续租：仅当任务仍由同一 worker 持有（status=Running 且 processed_at
// 仍为 expectLease，即上次成功写入的租约值）时推进租约，并返回续租后的租约值
// （DB 规范化读回，供下一次续租与终态写入的 CAS 使用）。任务若已被回收并被其他
// worker 重新领取，processed_at 已变化，续租失败（ok=false）—— 调用方应取消
// 处理并放弃后续状态写入，避免与新持有者重复执行外部副作用（租约 fencing）。
func RenewLease(id uint64, expectLease time.Time) (ok bool, lease time.Time, err error) {
	err = db.Connect().Transaction(func(tx *gorm.DB) error {
		res := tx.Table(tableName).
			Where("id = ? AND status = ? AND processed_at = ?", id, StatusRunning, expectLease).
			Updates(map[string]any{"processed_at": time.Now()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return nil // 租约已被回收或状态已变化，不再持有
		}
		ok = true
		var row struct {
			ProcessedAt time.Time
		}
		if err := tx.Table(tableName).Select("processed_at").Where("id = ?", id).Scan(&row).Error; err != nil {
			return err
		}
		lease = row.ProcessedAt
		return nil
	})
	return
}

// UpdateStatusOwned 在仍持有租约（status=Running 且 processed_at=lease）时更新
// 任务状态，用于成功/失败/重试终态写入；租约已被回收时静默跳过，避免覆盖新持有者。
func UpdateStatusOwned(id uint64, status uint8, lease time.Time, err error) error {
	updates := map[string]any{
		"status":       status,
		"processed_at": time.Now(),
	}
	if err != nil {
		updates["last_error"] = err.Error()
	}
	return builder().
		Where("id = ? AND status = ? AND processed_at = ?", id, StatusRunning, lease).
		Updates(updates).Error
}

// IncrementRetryCountOwned 在仍持有租约时递增重试次数。
func IncrementRetryCountOwned(id uint64, lease time.Time) error {
	return builder().Exec(
		"UPDATE task_queue SET retry_count = retry_count + 1 WHERE id = ? AND status = ? AND processed_at = ?",
		id, StatusRunning, lease,
	).Error
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

// RecoverStaleRunning 将指定类型前缀下租约过期的 Running 任务恢复为 Pending
// （processed_at 超过 staleAfter 视为进程崩溃残留；正常运行的 worker 会通过
// RenewLease 持续刷新 processed_at，因此不会被误回收）。worker 的每个轮询周期
// 都会调用一次，崩溃遗留任务在租约过期后即可被重新领取，
// 避免任务在 Running 与终态之间永久不可见导致投影更新丢失。
func RecoverStaleRunning(typePrefix string, staleAfter time.Duration) error {
	cutoff := time.Now().Add(-staleAfter)
	return builder().
		Where("type LIKE ?", typePrefix+"%").
		Where("status = ?", StatusRunning).
		Where("processed_at < ?", cutoff).
		Updates(map[string]any{
			"status":       StatusPending,
			"last_error":   "recovered stale running task (lease expired)",
			"processed_at": time.Now(),
		}).Error
}

// RecoverStaleEmailTasks 将邮件 worker 名下租约过期的 Running 任务恢复为
// Pending。与 GetPendingEmailTasks 使用同一类型谓词（email.* 前缀 +
// activation/reset_password 白名单），因此存量历史邮件行崩溃后同样能被
// 回收重领，不会像仅按前缀回收那样永久卡在 Running。
func RecoverStaleEmailTasks(staleAfter time.Duration) error {
	clause, args := emailTypeQuery()
	cutoff := time.Now().Add(-staleAfter)
	return builder().
		Where(clause, args...).
		Where("status = ?", StatusRunning).
		Where("processed_at < ?", cutoff).
		Updates(map[string]any{
			"status":       StatusPending,
			"last_error":   "recovered stale running task (lease expired)",
			"processed_at": time.Now(),
		}).Error
}

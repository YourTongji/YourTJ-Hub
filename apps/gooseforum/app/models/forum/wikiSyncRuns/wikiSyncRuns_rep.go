package wikiSyncRuns

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
)

func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

func CreateTx(tx *gorm.DB, entity *Entity) error {
	return tx.Table(tableName).Create(entity).Error
}

// MarkFinishedTx 事务内结束一次同步运行（状态 + head SHA + 计数 + 完成时间）。
func MarkFinishedTx(tx *gorm.DB, id uint64, status int8, headSha string, added, updated, deleted int, errMsg string) error {
	updates := map[string]any{
		"status":        status,
		"head_sha":      headSha,
		"pages_added":   added,
		"pages_updated": updated,
		"pages_deleted": deleted,
		"finished_at":   gorm.Expr("CURRENT_TIMESTAMP"),
	}
	if errMsg != "" {
		updates["error"] = errMsg
	}
	return tx.Table(tableName).Where("id = ?", id).Updates(updates).Error
}

// GetById 返回指定 id 的同步运行。
func GetById(id uint64) (entity Entity) {
	builder().Where("id = ?", id).First(&entity)
	return
}

// Latest 返回最近一次同步运行（同步面板状态展示）。
// 显式返回查询错误：状态面板必须区分 DB 故障与从未同步过（issue #287）。
func Latest() (entity Entity, err error) {
	err = builder().Order(queryopt.Desc("id")).First(&entity).Error
	return
}

// MarkAllRunningAbandoned 把全部 status=running 的运行标记为 failed（issue #290
// 崩溃恢复）：进程重启/被杀后旧 run 不可能继续执行，统一回收避免 UI 因
// lastRun.status=running 永久禁用手动同步。返回受影响行数。
func MarkAllRunningAbandoned(errMsg string) (int64, error) {
	res := builder().Where("status = ?", StatusRunning).Updates(map[string]any{
		"status":      StatusFailed,
		"error":       errMsg,
		"finished_at": gorm.Expr("CURRENT_TIMESTAMP"),
	})
	return res.RowsAffected, res.Error
}

// ListRecent 返回最近 N 次同步运行（倒序）。
// 显式返回查询错误：状态面板/运行日志必须区分 DB 故障与空列表（issue #287）。
func ListRecent(limit int) (entities []Entity, err error) {
	err = builder().Order(queryopt.Desc("id")).Limit(limit).Find(&entities).Error
	return
}

// MarkFinished 结束一次同步运行（非事务版；同步器提交后收尾用）。
func MarkFinished(id uint64, status int8, headSha string, added, updated, deleted int, errMsg string) error {
	updates := map[string]any{
		"status":        status,
		"head_sha":      headSha,
		"pages_added":   added,
		"pages_updated": updated,
		"pages_deleted": deleted,
		"finished_at":   gorm.Expr("CURRENT_TIMESTAMP"),
	}
	if errMsg != "" {
		updates["error"] = errMsg
	}
	return builder().Where("id = ?", id).Updates(updates).Error
}

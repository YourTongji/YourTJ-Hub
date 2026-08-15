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

// Latest 返回最近一次同步运行（同步面板状态展示）。
func Latest() (entity Entity) {
	builder().Order(queryopt.Desc("id")).First(&entity)
	return
}

// ListRecent 返回最近 N 次同步运行（倒序）。
func ListRecent(limit int) (entities []Entity) {
	builder().Order(queryopt.Desc("id")).Limit(limit).Find(&entities)
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

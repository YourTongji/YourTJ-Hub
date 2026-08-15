package wikiSyncRuns

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
)

func Get(id uint64) (entity Entity) {
	builder().First(&entity, id)
	return
}

// GetLatest 返回最近一次同步运行（按创建时间降序首条；无则零值）。
// 管理端「同步状态」面板用：head_sha / 上次同步时间 / 最近一次结果。
func GetLatest() (entity Entity) {
	builder().Order(queryopt.Desc("id")).First(&entity)
	return
}

// ListRecent 分页返回同步运行日志（降序；pageSize+1 探测 hasNext，与
// wikiPageRevisions.ListRecent 同模式，避免额外 COUNT）。
func ListRecent(page, pageSize int) ([]*Entity, bool) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	var entities []*Entity
	builder().
		Order(queryopt.Desc("id")).
		Offset((page - 1) * pageSize).
		Limit(pageSize + 1).
		Find(&entities)
	hasNext := len(entities) > pageSize
	if hasNext {
		entities = entities[:pageSize]
	}
	return entities, hasNext
}

func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

func CreateTx(tx *gorm.DB, entity *Entity) error {
	return tx.Table(tableName).Create(entity).Error
}

func Save(entity *Entity) error {
	return builder().Save(entity).Error
}

func SaveTx(tx *gorm.DB, entity *Entity) error {
	return tx.Table(tableName).Save(entity).Error
}

// DeleteOlderThan 清理某时间点之前的运行日志（保留窗口，防止日志无限增长）。
func DeleteOlderThan(before string) error {
	return builder().Where("created_at < ?", before).Delete(&Entity{}).Error
}

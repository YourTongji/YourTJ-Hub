package wikiPages

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
)

func Get(id uint64) (entity Entity) {
	builder().First(&entity, id)
	return
}

// GetTx 事务内按 id 获取页面（避免单连接测试库下事务内走全局连接死锁）。
func GetTx(tx *gorm.DB, id uint64) (entity Entity) {
	tx.Table(tableName).First(&entity, id)
	return
}

func GetByPath(path string) (entity Entity) {
	builder().Where(queryopt.Eq("path", path)).First(&entity)
	return
}

func GetByTopicId(topicId uint64) (entity Entity) {
	builder().Where(queryopt.Eq("topic_id", topicId)).First(&entity)
	return
}

func ListByNamespace(namespace string) []*Entity {
	var entities []*Entity
	builder().
		Where(queryopt.Eq("namespace", namespace)).
		Order(queryopt.Asc("sort_order")).
		Order(queryopt.Asc("id")).
		Find(&entities)
	return entities
}

func ListAll() []*Entity {
	var entities []*Entity
	builder().
		Order(queryopt.Asc("namespace")).
		Order(queryopt.Asc("sort_order")).
		Order(queryopt.Asc("id")).
		Find(&entities)
	return entities
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

func Delete(id uint64) error {
	return builder().Where(queryopt.Eq("id", id)).Delete(&Entity{}).Error
}

// PathExists 判断 path 是否已被占用（排除指定 id）。
func PathExists(path string, excludeID uint64) bool {
	var count int64
	q := builder().Where(queryopt.Eq("path", path))
	if excludeID != 0 {
		q = q.Where(queryopt.Ne("id", excludeID))
	}
	q.Count(&count)
	return count > 0
}

// CountChildren 统计某页面的直接子页面数。
func CountChildren(parentID uint64) int64 {
	var count int64
	builder().Where(queryopt.Eq("parent_id", parentID)).Count(&count)
	return count
}

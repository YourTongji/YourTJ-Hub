package wikiNamespaces

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
)

func List() []*Entity {
	var entities []*Entity
	builder().
		Order(queryopt.Asc("sort_order")).
		Order(queryopt.Asc("id")).
		Find(&entities)
	return entities
}

func GetByName(name string) (entity Entity) {
	builder().Where(queryopt.Eq("name", name)).First(&entity)
	return
}

// GetBySlug 按 slug 取命名空间（slug 为空时返回零值；用于 URL 路由按 slug 解析）。
func GetBySlug(slug string) (entity Entity) {
	if slug == "" {
		return
	}
	builder().Where(queryopt.Eq("slug", slug)).First(&entity)
	return
}

// SlugExists 判断 slug 是否已被占用（排除指定 id；空 slug 恒返回 false）。
func SlugExists(slug string, excludeID uint64) bool {
	if slug == "" {
		return false
	}
	var count int64
	q := builder().Where(queryopt.Eq("slug", slug))
	if excludeID != 0 {
		q = q.Where(queryopt.Ne("id", excludeID))
	}
	q.Count(&count)
	return count > 0
}

func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

func CreateTx(tx *gorm.DB, entity *Entity) error {
	return tx.Table(tableName).Create(entity).Error
}

func ListTx(tx *gorm.DB) (entities []*Entity, err error) {
	err = tx.Table(tableName).Order(queryopt.Asc("id")).Find(&entities).Error
	return
}

func Save(entity *Entity) error {
	return builder().Save(entity).Error
}

func DeleteByName(name string) error {
	return builder().Where(queryopt.Eq("name", name)).Delete(&Entity{}).Error
}

func Exists(name string) bool {
	var count int64
	builder().Where(queryopt.Eq("name", name)).Count(&count)
	return count > 0
}

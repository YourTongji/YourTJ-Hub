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

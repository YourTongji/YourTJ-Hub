package wikiNamespaces

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
)

// List 返回全部命名空间（按 sort_order/id 升序）。
// 显式返回查询错误：公开读必须区分 DB 故障与真实空数据，不能吞错（issue #287）。
func List() ([]*Entity, error) {
	var entities []*Entity
	err := builder().
		Order(queryopt.Asc("sort_order")).
		Order(queryopt.Asc("id")).
		Find(&entities).Error
	return entities, err
}

func GetByName(name string) (entity Entity) {
	builder().Where(queryopt.Eq("name", name)).First(&entity)
	return
}

func Create(entity *Entity) error {
	return builder().Create(entity).Error
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

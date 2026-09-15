package sticker

// 排序约定：sort_order 升序、id 升序兜底，保证列表与选择器顺序稳定。
import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
)

func All() []Entity {
	entities := make([]Entity, 0)
	builder().Order("sort_order ASC").Order("id ASC").Find(&entities)
	return entities
}

func AllEnabled() []Entity {
	entities := make([]Entity, 0)
	builder().
		Where(queryopt.Eq("is_enabled", true)).
		Order("sort_order ASC").Order("id ASC").
		Find(&entities)
	return entities
}

func GetByName(name string) Entity {
	var entity Entity
	builder().Where(queryopt.Eq("name", name)).First(&entity)
	return entity
}

func GetById(id uint64) Entity {
	var entity Entity
	builder().Where(queryopt.Eq("id", id)).First(&entity)
	return entity
}

func Save(entity *Entity) error {
	return builder().Save(entity).Error
}

func DeleteById(id uint64) error {
	return builder().Where(queryopt.Eq("id", id)).Delete(&Entity{}).Error
}

func Count() int64 {
	var count int64
	builder().Count(&count)
	return count
}

package sticker

// 排序约定：sort_order 升序、id 升序兜底，保证列表与选择器顺序稳定。
import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func All() []Entity {
	entities := make([]Entity, 0)
	builder().Order("sort_order ASC").Order("id ASC").Find(&entities)
	return entities
}

func AllEnabled() ([]Entity, error) {
	entities := make([]Entity, 0)
	result := builder().
		Where(queryopt.Eq("is_enabled", true)).
		Order("sort_order ASC").Order("id ASC").
		Find(&entities)
	return entities, result.Error
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

func EnabledByNames(names []string) ([]Entity, error) {
	entities := make([]Entity, 0)
	if len(names) == 0 {
		return entities, nil
	}
	err := builder().Where("name IN ? AND is_enabled = ?", names, true).Find(&entities).Error
	return entities, err
}

// InsertIfNameAvailable leaves a PostgreSQL transaction usable on a name race.
func InsertIfNameAvailable(tx *gorm.DB, entity *Entity) (bool, error) {
	result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "name"}}, DoNothing: true}).Create(entity)
	return result.RowsAffected > 0, result.Error
}
func SaveTx(tx *gorm.DB, entity *Entity) error { return tx.Save(entity).Error }
func GetByIDTx(tx *gorm.DB, id uint64) (Entity, error) {
	var row Entity
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error
	return row, err
}
func DeleteTx(tx *gorm.DB, id uint64) error { return tx.Delete(&Entity{}, id).Error }

func GetByNameTx(tx *gorm.DB, name string) (Entity, error) {
	var row Entity
	err := tx.Where("name = ?", name).First(&row).Error
	return row, err
}

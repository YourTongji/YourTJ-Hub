package badges

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
)

func All() (entities []*Entity) {
	builder().
		Order("sort_order ASC, id ASC").
		Find(&entities)
	return
}

func GetByCode(code string) (entity Entity) {
	builder().Where(queryopt.Eq("code", code)).First(&entity)
	return
}

func Save(entity *Entity) error {
	return builder().Save(entity).Error
}

func DeleteByCode(code string) error {
	return builder().Where(queryopt.Eq("code", code)).Delete(&Entity{}).Error
}

// GetByCodeTx observes administrator overrides in the caller transaction.
func GetByCodeTx(tx *gorm.DB, code string) (Entity, error) {
	var row Entity
	err := tx.Where("code = ?", code).First(&row).Error
	return row, err
}

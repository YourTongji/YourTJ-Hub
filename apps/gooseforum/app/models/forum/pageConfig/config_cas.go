package pageConfig

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ReadConfig preserves storage errors for settings that must not silently fall back.
func ReadConfig(ctx context.Context, pageType string) (string, error) {
	var entity Entity
	err := builder().WithContext(ctx).Where("page_type = ?", pageType).First(&entity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	return entity.Config, err
}

// CompareAndSwapConfig avoids overwriting settings changed since an admin loaded them.
// Empty expected means absent. Config consumers own validation and version presentation.
func CompareAndSwapConfig(ctx context.Context, pageType, expected, next string) (bool, error) {
	if expected == "" {
		result := builder().WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&Entity{PageType: pageType, Config: next})
		return result.RowsAffected == 1, result.Error
	}
	result := builder().WithContext(ctx).Where("page_type = ? AND config = ?", pageType, expected).Updates(map[string]any{"config": next, "updated_at": gorm.Expr("CURRENT_TIMESTAMP")})
	return result.RowsAffected == 1, result.Error
}

package course

import "context"

func GetSearchProjection(ctx context.Context, id uint64) (Entity, error) {
	var row Entity
	err := courseBuilder().WithContext(ctx).Where("id = ?", id).First(&row).Error
	return row, err
}

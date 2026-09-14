package taskQueue

import (
	"context"

	"gorm.io/gorm/clause"
)

// SearchMaintenanceType has one active slot across every managed index. The
// partial unique index on Entity.Type also arbitrates concurrent processes.
const SearchMaintenanceType = "search-maintenance"

func CreateSearchMaintenance(ctx context.Context, payload string) (Entity, bool, error) {
	row := Entity{Type: SearchMaintenanceType, TaskJson: payload}
	result := builder().WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	if result.Error != nil {
		return row, false, result.Error
	}
	if result.RowsAffected == 1 {
		return row, true, nil
	}
	err := builder().WithContext(ctx).Where("type = ? AND status IN ?", SearchMaintenanceType,
		[]int{StatusPending, StatusRunning, StatusRetrying}).First(&row).Error
	return row, false, err
}

func ListSearchMaintenance(ctx context.Context) ([]Entity, error) {
	rows := make([]Entity, 0)
	err := builder().WithContext(ctx).Where("type = ?", SearchMaintenanceType).Order("id DESC").Limit(20).Find(&rows).Error
	return rows, err
}

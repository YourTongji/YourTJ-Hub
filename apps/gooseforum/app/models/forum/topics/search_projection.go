package topics

import "context"

// ListSearchProjectionPage reads source rows by primary key. Hidden rows still
// advance the cursor; visibility belongs to the projection builder.
func ListSearchProjectionPage(ctx context.Context, afterID uint64, limit int) ([]Entity, error) {
	rows := make([]Entity, 0)
	err := builder().WithContext(ctx).Select("id", "title", "category_id", "first_post_id", "status", "process_status", "topic_type", "visibility_status", "deleted_at", "created_at", "updated_at").Where("id > ?", afterID).Order("id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

func GetSearchTopics(ctx context.Context, ids []uint64) (map[uint64]Entity, error) {
	result := make(map[uint64]Entity, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var rows []Entity
	err := builder().WithContext(ctx).Select("id", "status", "process_status", "visibility_status", "deleted_at").Where("id IN ?", ids).Find(&rows).Error
	for _, row := range rows {
		result[row.Id] = row
	}
	return result, err
}

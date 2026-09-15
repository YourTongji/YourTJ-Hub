package posts

import "context"

// GetFirstSearchPost preserves the legacy first-post fallback and propagates DB failures.
func GetFirstSearchPost(ctx context.Context, topicID uint64) (Entity, error) {
	var row Entity
	err := builder().WithContext(ctx).Where("topic_id = ? AND post_no >= ?", topicID, 1).Order("post_no ASC").First(&row).Error
	return row, err
}

func GetSearchPosts(ctx context.Context, ids []uint64) (map[uint64]Entity, error) {
	result := make(map[uint64]Entity, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var rows []Entity
	err := builder().WithContext(ctx).Select("id", "topic_id", "content", "process_status", "visibility_status", "deleted_at").Where("id IN ?", ids).Find(&rows).Error
	for _, row := range rows {
		result[row.Id] = row
	}
	return result, err
}

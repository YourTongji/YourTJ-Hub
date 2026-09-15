package wikiPages

import "context"

// ListSearchProjectionPage reads source rows by primary key. Hidden rows still
// advance the cursor; visibility belongs to the projection builder.
func ListSearchProjectionPage(ctx context.Context, afterID uint64, limit int) ([]Entity, error) {
	rows := make([]Entity, 0)
	err := builder().WithContext(ctx).Select("id", "topic_id", "namespace", "path", "title", "para_anchors").Where("id > ?", afterID).Order("id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

func GetWithContext(ctx context.Context, id uint64) (Entity, error) {
	var row Entity
	err := builder().WithContext(ctx).Select("id", "topic_id", "namespace", "path", "title", "para_anchors").Where("id = ?", id).First(&row).Error
	return row, err
}

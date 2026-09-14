package users

import "context"

// ListSearchProjectionPage reads source rows by primary key. Hidden rows still
// advance the cursor; visibility belongs to the projection builder.
func ListSearchProjectionPage(ctx context.Context, afterID uint64, limit int) ([]EntityComplete, error) {
	rows := make([]EntityComplete, 0)
	err := builder().WithContext(ctx).Select("id", "username", "nickname", "bio", "actor_type", "deleted_at").Where("id > ?", afterID).Order("id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

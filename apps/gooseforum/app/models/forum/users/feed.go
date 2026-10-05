package users

import (
	"context"
	"fmt"
	"gorm.io/gorm"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
)

// FilterEligibleFeedAuthors rechecks current account and both block directions.
// The returned map never discloses which direction caused an exclusion.
func FilterEligibleFeedAuthors(ctx context.Context, viewer uint64, ids []uint64) (map[uint64]bool, error) {
	allowed := map[uint64]bool{}
	if len(ids) == 0 {
		return allowed, nil
	}
	if len(ids) > 300 {
		return nil, fmt.Errorf("feed author batch exceeds bound")
	}
	var active []uint64
	err := db.ConnectContext(ctx).Model(&EntityComplete{}).Select("users.id").Where("users.id IN ? AND users.is_frozen = ?", ids, StatusNormal).Where("NOT EXISTS (SELECT 1 FROM user_blocks b WHERE (b.owner_id = ? AND b.target_user_id = users.id) OR (b.target_user_id = ? AND b.owner_id = users.id))", viewer, viewer).Find(&active).Error
	if err != nil {
		return nil, err
	}
	for _, id := range active {
		if id != viewer {
			allowed[id] = true
		}
	}
	return allowed, nil
}

// EligibleIDsQuery is the identity owner's scalar query for internal content
// projections. It contains no profile, credentials or relationship details.
func EligibleIDsQuery(ctx context.Context) *gorm.DB {
	return db.ConnectContext(ctx).Model(&EntityComplete{}).Select("id").Where("is_frozen = ?", StatusNormal)
}

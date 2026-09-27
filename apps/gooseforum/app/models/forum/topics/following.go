package topics

import (
	"context"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/pageutil"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
)

// CreatedCursor is an exclusive position in chronological topic order. The
// timestamp travels with the ID so deleting the boundary topic cannot move it.
type CreatedCursor struct {
	CreatedAt time.Time
	ID        uint64
}

// FollowingPage applies the same public-list visibility as Page(FilterStatus),
// plus a live, viewer-scoped follow subquery. It never uses the public feed cache.
func FollowingPage(ctx context.Context, viewerID uint64, before *CreatedCursor, limit int) ([]Entity, bool, error) {
	if viewerID == 0 {
		return []Entity{}, false, nil
	}
	limit = pageutil.BoundPageSize(limit)
	b := builder().WithContext(ctx).
		Where("user_id IN (?)", userFollow.ActiveFollowedIDsQuery(ctx, viewerID)).
		Where("status = ? AND process_status = ? AND visibility_status = ?", 1, ProcessStatusNormal, VisibilityActive).
		Where("topic_type = ?", TopicTypeForum).
		Where(firstPostVisibleSQL, ProcessStatusNormal)
	if before != nil {
		b = b.Where("created_at < ? OR (created_at = ? AND id < ?)", before.CreatedAt, before.CreatedAt, before.ID)
	}
	var rows []Entity
	if err := b.Order("created_at DESC").Order("id DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, false, err
	}
	hasNext := len(rows) > limit
	if hasNext {
		rows = rows[:limit]
	}
	return rows, hasNext, nil
}

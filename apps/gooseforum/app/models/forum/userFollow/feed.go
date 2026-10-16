package userFollow

import "context"

func FollowingAmong(ctx context.Context, viewer uint64, authors []uint64) (map[uint64]bool, error) {
	out := map[uint64]bool{}
	if len(authors) == 0 {
		return out, nil
	}
	var rows []Entity
	err := builder().WithContext(ctx).Select("follow_user_id").Where("user_id = ? AND follow_user_id IN ? AND status = 1", viewer, authors).Find(&rows).Error
	for _, r := range rows {
		out[r.FollowUserId] = true
	}
	return out, err
}

// MutualState reads both active follow edges between two users in one query.
func MutualState(viewer, owner uint64) (viewerFollows, ownerFollows bool, err error) {
	if viewer == 0 || owner == 0 || viewer == owner {
		return false, false, nil
	}
	var rows []Entity
	err = builder().Select("user_id", "follow_user_id").
		Where("status = 1 AND ((user_id = ? AND follow_user_id = ?) OR (user_id = ? AND follow_user_id = ?))", viewer, owner, owner, viewer).
		Find(&rows).Error
	for _, r := range rows {
		if r.UserId == viewer {
			viewerFollows = true
		} else {
			ownerFollows = true
		}
	}
	return viewerFollows, ownerFollows, err
}

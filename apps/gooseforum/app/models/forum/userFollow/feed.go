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

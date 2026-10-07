package topicUserAction

import (
	"context"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"time"
)

func RankLikers(ctx context.Context, topicID uint64) ([]uint64, error) {
	var ids []uint64
	err := builder().WithContext(ctx).Select("user_id").Where("topic_id = ? AND liked_at IS NOT NULL", topicID).Where("user_id IN (?)", users.EligibleIDsQuery(ctx)).Limit(50001).Find(&ids).Error
	return ids, err
}

type ProfileAction struct {
	TopicID uint64
	At      time.Time
}

func ProfileActions(ctx context.Context, uid uint64, kind string, after time.Time) ([]ProfileAction, error) {
	field := "liked_at"
	if kind == "bookmark" {
		field = "bookmarked_at"
	}
	var rows []ProfileAction
	err := builder().WithContext(ctx).Select("topic_id,"+field+" AS at").Where("user_id = ? AND "+field+" >= ?", uid, after).Order(field + " DESC").Limit(50).Scan(&rows).Error
	return rows, err
}

// ActorTopicIDs returns a bounded batch for identity-driven rank invalidation.
func ActorTopicIDs(ctx context.Context, uid, after uint64) ([]uint64, error) {
	var ids []uint64
	err := builder().WithContext(ctx).Select("topic_id").Where("user_id = ? AND topic_id > ?", uid, after).Order("topic_id").Limit(200).Find(&ids).Error
	return ids, err
}

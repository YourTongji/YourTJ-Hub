package contentDeleteEvent

import (
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/queryopt"
)

func Record(entity Entity) error {
	return builder().Create(&entity).Error
}

// CountRecentByActor 统计某用户在时间窗内的删除事件数（PRD R9 频率控制）。
func CountRecentByActor(actorID uint64, eventType string, since time.Time) (int64, error) {
	var count int64
	err := builder().
		Where(queryopt.Eq("actor_id", actorID)).
		Where(queryopt.Eq("event_type", eventType)).
		Where("created_at >= ?", since).
		Count(&count).Error
	return count, err
}

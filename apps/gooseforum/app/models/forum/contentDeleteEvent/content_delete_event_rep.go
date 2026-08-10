package contentDeleteEvent

import (
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/queryopt"
)

func Record(entity Entity) error {
	return builder().Create(&entity).Error
}

// CountByType 供埋点指标使用：统计某类型事件在指定时间窗内的数量。
func CountByType(eventType string, since time.Time) (int64, error) {
	var count int64
	err := builder().
		Where(queryopt.Eq("event_type", eventType)).
		Where("created_at >= ?", since).
		Count(&count).Error
	return count, err
}

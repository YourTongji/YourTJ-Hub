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

// CountRecentByActorEvents 统计某用户在时间窗内属于给定事件集合的删除事件数。
// 批量删除频率窗口把隐私紧急删除一并计入（PRD R9 注释与实现一致）：
// 无论用户走"删除→永久删除"还是"隐私紧急删除"，短时间内的删除动作
// 都计入 20 条/10 分钟的二次确认阈值，避免绕过限速清空内容。
func CountRecentByActorEvents(actorID uint64, eventTypes []string, since time.Time) (int64, error) {
	var count int64
	err := builder().
		Where(queryopt.Eq("actor_id", actorID)).
		Where(queryopt.In("event_type", eventTypes)).
		Where("created_at >= ?", since).
		Count(&count).Error
	return count, err
}

package eventNotification

import (
	"encoding/json"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
)

// Create 创建通知
func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

func CreateBatch(entities []*Entity, batchSize int) error {
	if len(entities) == 0 {
		return nil
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return builder().CreateInBatches(entities, batchSize).Error
}

// GetLatestByTopicAndType 返回某话题某类型的最新一条通知（wiki 通知节流用：
// 窗口内已有通知则跳过本次编辑的 fan-out）。
func GetLatestByTopicAndType(topicId uint64, eventType string) (entity Entity) {
	builder().
		Where(queryopt.Eq("topic_id", topicId)).
		Where(queryopt.Eq("event_type", eventType)).
		Order(queryopt.Desc("id")).
		First(&entity)
	return
}

// QueryByUserId 获取用户的通知列表
func QueryByUserId(userId uint64, limit int, startId uint64, unreadOnly bool) (notifications []*Entity, err error) {
	db := builder().Where(queryopt.Eq("user_id", userId))
	if startId != 0 {
		db = db.Where(queryopt.Lt("id", startId))
	}
	if unreadOnly {
		db = db.Where(queryopt.Eq("is_read", false))
	}
	err = db.Order(queryopt.Desc(`id`)).
		Limit(limit).
		Find(&notifications).Error
	return
}

// GetByID 返回指定通知行（不存在时返回零值实体，调用方用 Id==0 判断）。
// 推送 worker 用：读取通知行作为推送事件源（类型/标题/已读状态）。
func GetByID(id uint64) (entity Entity) {
	builder().
		Where(queryopt.Eq("id", id)).
		First(&entity)
	return
}

// GetLastUnread 获取用户未读通知数量
func GetLastUnread(userId uint64) (entity Entity) {
	builder().
		Where(queryopt.Eq("user_id", userId)).
		Where(queryopt.Eq("is_read", false)).
		Order("id DESC").
		First(&entity)
	return
}

// GetUnreadCount 获取用户未读通知数量
func GetUnreadCount(userId uint64) (count int64, err error) {
	err = builder().
		Where(queryopt.Eq("user_id", userId)).
		Where(queryopt.Eq("is_read", false)).
		Count(&count).Error
	return
}

// MarkAsRead 标记通知为已读
func MarkAsRead(notificationId uint64, userId uint64) error {
	now := time.Now()
	return builder().
		Where(queryopt.Eq("id", notificationId)).
		Where(queryopt.Eq("user_id", userId)). // 确保只能标记自己的通知
		Updates(map[string]any{
			"is_read": true,
			"read_at": now,
		}).Error
}

// MarkAllAsRead 标记用户所有通知为已读
func MarkAllAsRead(userId uint64) error {
	now := time.Now()
	return builder().
		Where(queryopt.Eq("user_id", userId)).
		Where(queryopt.Eq("is_read", false)).
		Updates(map[string]any{
			"is_read": true,
			"read_at": now,
		}).Error
}

// ClearPreviewsByTopic 将某话题/回复相关通知的正文预览置空（内容删除后联动）。
// 通过冗余 topic_id 列（SQL 过滤）定位目标话题的通知，避免对全表做 JSON 解析；
// 命中后再按 postId（0=话题级）过滤，并逐行写回（payload 各行不同，无法合并为
// 单条 UPDATE）。游标分批处理，确保高通知量话题不会被固定上限截断。
func ClearPreviewsByTopic(topicId uint64, postId uint64) error {
	if topicId == 0 {
		return nil
	}
	const batchSize = 500
	var cursorID uint64
	for {
		query := builder().Where(queryopt.Eq("topic_id", topicId)).Order(queryopt.Asc("id")).Limit(batchSize)
		if cursorID != 0 {
			query = query.Where(queryopt.Gt("id", cursorID))
		}
		var notifications []Entity
		if err := query.Find(&notifications).Error; err != nil {
			return err
		}
		if len(notifications) == 0 {
			return nil
		}
		for _, item := range notifications {
			cursorID = item.Id
			if postId != 0 && item.Payload.PostId != postId {
				continue
			}
			item.Payload.TemplateParams.Preview = ""
			item.Payload.Content = ""
			// 标题/话题标题同样置空：通知列表不应再展示被删内容的原文标题
			// （PRD R11「该内容已被删除」），仅保留可用于定位的 topicId/postId。
			item.Payload.Title = ""
			item.Payload.TopicTitle = ""
			// 显式序列化为 JSON 字节再写入：GORM 的 Updates(map[...]) 不会对值应用
			// serializer:json，直接传结构体在两库驱动下都会报错。先 marshal 保证
			// SQLite/PostgreSQL 的 JSON 列都能正常写入。
			payloadBytes, err := json.Marshal(item.Payload)
			if err != nil {
				return err
			}
			if err := builder().Model(&Entity{}).Where(queryopt.Eq("id", item.Id)).Updates(map[string]any{
				"payload": payloadBytes,
			}).Error; err != nil {
				return err
			}
		}
	}
}

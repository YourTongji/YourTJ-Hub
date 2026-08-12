package contentDeleteEvent

import "time"

const tableName = "content_delete_events"

// EventType 删除生命周期埋点事件（PRD R14）。
// 前端点击/确认类事件由站点上报，后端删除/恢复/永久删除/隐私删除在状态
// 变更成功后记录，管理端证据查看由 R7 记录。
type EventType string

const (
	EventDeleteClicked    EventType = "content_delete_clicked"            // 前端：打开删除确认弹窗
	EventDeleteConfirmed  EventType = "content_delete_confirmed"          // 前端：确认删除
	EventDeleted          EventType = "content_deleted"                   // 后端：删除完成
	EventRestored         EventType = "content_restored"                  // 后端：恢复完成
	EventPermanentDelete  EventType = "content_permanent_delete"          // 后端：永久删除完成
	EventPrivacyDelete    EventType = "privacy_delete_requested"          // 后端：隐私紧急删除
	EventModerationViewed EventType = "moderation_deleted_content_viewed" // 管理端：查看已删内容
)

type Entity struct {
	Id          uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	EventType   string    `gorm:"column:event_type;type:varchar(64);not null;default:'';index:idx_content_delete_events_type,priority:1;" json:"eventType"`
	ContentType string    `gorm:"column:content_type;type:varchar(32);not null;default:'';" json:"contentType"`
	ContentID   uint64    `gorm:"column:content_id;not null;default:0;" json:"contentId"`
	ActorID     uint64    `gorm:"column:actor_id;not null;default:0;index:idx_content_delete_events_actor,priority:1;" json:"actorId"`
	TopicID     uint64    `gorm:"column:topic_id;not null;default:0;" json:"topicId"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime;<-:create;index:idx_content_delete_events_type,priority:2;" json:"createdAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}

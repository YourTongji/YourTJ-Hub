package agentInbox

import (
	"time"
)

const tableName = "agent_inbox"

// 收件箱状态：unread=0 / read=1
const (
	StatusUnread uint8 = 0
	StatusRead   uint8 = 1
)

// 投递状态：pending=0 / delivered=1 / failed=2 / skipped=3
const (
	DeliveryPending   uint8 = 0
	DeliveryDelivered uint8 = 1
	DeliveryFailed    uint8 = 2
	DeliverySkipped   uint8 = 3
)

// 入库事件类型（事件总线侧）
const (
	EventTypeTopicPublished = "topic.published"
	EventTypeTopicUpdated   = "topic.updated"
	EventTypePostCreated    = "post.created"
)

const (
	fieldAgentId  = "agent_id"
	fieldTopicId  = "topic_id"
	fieldPostId   = "post_id"
	fieldStatus   = "status"
	fieldReadAt   = "read_at"
	fieldAttempts = "attempts"
)

// Entity 是 Agent 提及收件箱行，也是投递状态的唯一事实源。
// 唯一键 (agent_id, topic_id, post_id)：同一内容的事件重放/编辑只会 upsert 一行。
type Entity struct {
	Id             uint64     `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	AgentId        uint64     `gorm:"column:agent_id;not null;default:0;uniqueIndex:uk_agent_inbox_key,priority:1;index:idx_agent_inbox_agent_status,priority:1;" json:"agentId"`
	TopicId        uint64     `gorm:"column:topic_id;not null;default:0;uniqueIndex:uk_agent_inbox_key,priority:2;" json:"topicId"`
	PostId         uint64     `gorm:"column:post_id;not null;default:0;uniqueIndex:uk_agent_inbox_key,priority:3;" json:"postId"`
	EventType      string     `gorm:"column:event_type;type:varchar(64);not null;default:'';" json:"eventType"`
	ActorId        uint64     `gorm:"column:actor_id;not null;default:0;" json:"actorId"`
	ContentPreview string     `gorm:"column:content_preview;type:varchar(255);not null;default:'';" json:"contentPreview"`
	Status         uint8      `gorm:"column:status;not null;default:0;index:idx_agent_inbox_agent_status,priority:2;" json:"status"`
	DeliveryStatus uint8      `gorm:"column:delivery_status;not null;default:0;" json:"deliveryStatus"`
	Attempts       uint8      `gorm:"column:attempts;not null;default:0;" json:"attempts"`
	LastError      string     `gorm:"column:last_error;type:varchar(512);not null;default:'';" json:"lastError"`
	ReadAt         *time.Time `gorm:"column:read_at;" json:"readAt"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}

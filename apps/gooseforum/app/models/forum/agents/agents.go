package agents

import (
	"time"
)

const tableName = "agents"

// fieldUserId 关联机器人用户id（即 Agent ID）
const fieldUserId = "user_id"

// fieldTokenPrefix 令牌非敏感标识前缀
const fieldTokenPrefix = "token_prefix"

// fieldTokenHash 令牌哈希
const fieldTokenHash = "token_hash"

// fieldWebhookEndpoint 可配置的 webhook 端点（零或一个）
const fieldWebhookEndpoint = "webhook_endpoint"

// fieldEnabled 是否启用
const fieldEnabled = "enabled"

// fieldCreatedBy 创建人用户id
const fieldCreatedBy = "created_by"

// fieldLastUsedAt 最近一次令牌使用时间
const fieldLastUsedAt = "last_used_at"

// fieldCreatedAt 创建时间
const fieldCreatedAt = "created_at"

// fieldUpdatedAt 更新时间
const fieldUpdatedAt = "updated_at"

const (
	StatusDisabled = 0
	StatusEnabled  = 1
)

type Entity struct {
	UserId          uint64     `gorm:"primaryKey;column:user_id;not null;" json:"userId"`
	TokenPrefix     string     `gorm:"column:token_prefix;type:varchar(16);not null;default:'';uniqueIndex" json:"tokenPrefix"`
	TokenHash       string     `gorm:"column:token_hash;type:varchar(128);not null;default:'';" json:"-"`
	WebhookEndpoint string     `gorm:"column:webhook_endpoint;type:varchar(512);not null;default:'';" json:"webhookEndpoint"`
	Enabled         int8       `gorm:"column:enabled;not null;default:1;" json:"enabled"`
	CreatedBy       uint64     `gorm:"column:created_by;not null;default:0;" json:"createdBy"`
	LastUsedAt      *time.Time `gorm:"column:last_used_at;" json:"lastUsedAt"`
	CreatedAt       time.Time  `gorm:"column:created_at;index;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}

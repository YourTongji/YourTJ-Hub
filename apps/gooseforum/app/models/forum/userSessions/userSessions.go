package userSessions

import (
	"time"
)

const tableName = "user_sessions"

// pid id
const pid = "id"

// fieldUserId 关联用户id
const fieldUserId = "user_id"

// fieldJti 会话唯一ID（JWT jti claim）
const fieldJti = "jti"

// fieldUserAgent 用户代理
const fieldUserAgent = "user_agent"

// fieldIp 登录IP
const fieldIp = "ip"

// fieldExpiresAt 会话过期时间
const fieldExpiresAt = "expires_at"

// fieldCreatedAt 创建时间
const fieldCreatedAt = "created_at"

// fieldUpdatedAt 更新时间
const fieldUpdatedAt = "updated_at"

type Entity struct {
	Id        uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`                    // id
	UserId    uint64    `gorm:"column:user_id;not null;default:0;index" json:"userId"`                     // 关联用户id
	Jti       string    `gorm:"column:jti;type:varchar(64);not null;default:'';uniqueIndex" json:"jti"`    // 会话唯一ID（JWT jti claim）
	UserAgent string    `gorm:"column:user_agent;type:varchar(512);not null;default:'';" json:"userAgent"` // 用户代理
	Ip        string    `gorm:"column:ip;type:varchar(64);not null;default:'';" json:"ip"`                 // 登录IP
	ExpiresAt time.Time `gorm:"column:expires_at;index;" json:"expiresAt"`                                 // 会话过期时间
	CreatedAt time.Time `gorm:"column:created_at;index;autoCreateTime;<-:create;" json:"createdAt"`        //
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`                        //
}

func (itself *Entity) TableName() string {
	return tableName
}

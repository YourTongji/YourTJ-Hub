package userTotpRecoveryCodes

import (
	"time"
)

const tableName = "user_totp_recovery_codes"

// pid id
const pid = "id"

// fieldUserId 关联用户id
const fieldUserId = "user_id"

// fieldCodeHash 恢复码SHA-256哈希
const fieldCodeHash = "code_hash"

// fieldUsedAt 使用时间，为空表示未使用
const fieldUsedAt = "used_at"

// fieldCreatedAt 创建时间
const fieldCreatedAt = "created_at"

type Entity struct {
	Id        uint64     `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`                           // id
	UserId    uint64     `gorm:"column:user_id;type:bigint unsigned;not null;default:0;index" json:"userId"`       // 关联用户id
	CodeHash  string     `gorm:"column:code_hash;type:varchar(128);not null;default:'';uniqueIndex" json:"-"`      // 恢复码SHA-256哈希
	UsedAt    *time.Time `gorm:"column:used_at;type:datetime;" json:"usedAt"`                                      // 使用时间，为空表示未使用
	CreatedAt time.Time  `gorm:"column:created_at;type:datetime;index;autoCreateTime;<-:create;" json:"createdAt"` //
}

func (itself *Entity) TableName() string {
	return tableName
}

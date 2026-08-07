package userTotp

import (
	"time"
)

const tableName = "user_totp"

// pid id
const pid = "id"

// fieldUserId 关联用户id
const fieldUserId = "user_id"

// fieldSecretEncrypted 加密后的TOTP密钥
const fieldSecretEncrypted = "secret_encrypted"

// fieldEnabled 是否已启用
const fieldEnabled = "enabled"

// fieldCreatedAt 创建时间
const fieldCreatedAt = "created_at"

// fieldUpdatedAt 更新时间
const fieldUpdatedAt = "updated_at"

type Entity struct {
	Id              uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`                  // id
	UserId          uint64    `gorm:"column:user_id;not null;default:0;uniqueIndex" json:"userId"`             // 关联用户id
	SecretEncrypted string    `gorm:"column:secret_encrypted;type:varchar(512);not null;default:'';" json:"-"` // 加密后的TOTP密钥
	Enabled         int8      `gorm:"column:enabled;not null;default:0;" json:"enabled"`                       // 是否已启用
	CreatedAt       time.Time `gorm:"column:created_at;index;autoCreateTime;<-:create;" json:"createdAt"`      //
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`                      //
}

func (itself *Entity) TableName() string {
	return tableName
}

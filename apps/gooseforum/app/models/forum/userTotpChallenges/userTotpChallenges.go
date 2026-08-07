package userTotpChallenges

import (
	"time"
)

const tableName = "user_totp_challenges"

const pid = "id"
const fieldUserId = "user_id"
const fieldJti = "jti"
const fieldConsumedAt = "consumed_at"
const fieldExpiresAt = "expires_at"
const fieldCreatedAt = "created_at"
const fieldUpdatedAt = "updated_at"

type Entity struct {
	Id         uint64     `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	UserId     uint64     `gorm:"column:user_id;not null;default:0;index" json:"userId"`
	Jti        string     `gorm:"column:jti;type:varchar(64);not null;default:'';uniqueIndex" json:"jti"`
	ConsumedAt *time.Time `gorm:"column:consumed_at;index" json:"consumedAt"`
	ExpiresAt  time.Time  `gorm:"column:expires_at;index" json:"expiresAt"`
	CreatedAt  time.Time  `gorm:"column:created_at;index;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt  time.Time  `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}

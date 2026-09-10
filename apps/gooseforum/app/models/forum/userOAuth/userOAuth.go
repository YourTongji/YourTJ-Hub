package userOAuth

import (
	"time"
)

const tableName = "user_o_auth"

// pid id
const pid = "id"

// fieldUserId 关联用户id
const fieldUserId = "user_id"

// fieldProvider 平台标识(github/twitter)
const fieldProvider = "provider"

// fieldProviderUid 第三方用户唯一ID
const fieldProviderUid = "provider_uid"

// fieldCreatedAt 创建时间
const fieldCreatedAt = "created_at"

// fieldUpdatedAt 更新时间
const fieldUpdatedAt = "updated_at"

type Entity struct {
	Id          uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`                                                                                                                         // id
	UserId      uint64    `gorm:"column:user_id;not null;default:0;index;index:idx_user_provider,priority:1" json:"userId"`                                                                                       // 关联用户id
	Provider    string    `gorm:"column:provider;type:varchar(32);default:0;index:idx_provider_uid,priority:1;index:idx_user_provider,priority:2;uniqueIndex:idx_provider_uid_unique,priority:1" json:"provider"` // 平台标识(github/twitter)
	ProviderUid string    `gorm:"column:provider_uid;type:varchar(255);not null;default:'';index;index:idx_provider_uid,priority:2;uniqueIndex:idx_provider_uid_unique,priority:2" json:"providerUid"`            // 第三方用户唯一ID
	CreatedAt   time.Time `gorm:"column:created_at;index;autoCreateTime;<-:create;" json:"createdAt"`                                                                                                             //
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

// func (itself *Entity) BeforeSave(tx *gorm.DB) (err error) {}
// func (itself *Entity) BeforeCreate(tx *gorm.DB) (err error) {}
// func (itself *Entity) AfterCreate(tx *gorm.DB) (err error) {}
// func (itself *Entity) BeforeUpdate(tx *gorm.DB) (err error) {}
// func (itself *Entity) AfterUpdate(tx *gorm.DB) (err error) {}
// func (itself *Entity) AfterSave(tx *gorm.DB) (err error) {}
// func (itself *Entity) BeforeDelete(tx *gorm.DB) (err error) {}
// func (itself *Entity) AfterDelete(tx *gorm.DB) (err error) {}
// func (itself *Entity) AfterFind(tx *gorm.DB) (err error) {}

func (itself *Entity) TableName() string {
	return tableName
}

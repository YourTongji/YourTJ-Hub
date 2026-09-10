// Package pushSubscription 持久化 Web Push 浏览器订阅。
// 一个用户可以拥有多条订阅（每浏览器/每设备一条）；endpoint 全局唯一
// （同一浏览器重新授权会得到同 endpoint，订阅冲突时按 endpoint 收敛归属）。
package pushSubscription

import "time"

const tableName = "push_subscriptions"

// Entity 一条 Web Push 订阅。endpoint/p256dh/auth 是推送服务长期凭据，
// 等同用户会话凭据对待：仅本人可管理（端点 user_id 锚定）、账号注销即删、
// 不写入日志明文。
type Entity struct {
	Id        uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	UserId    uint64    `gorm:"column:user_id;not null;default:0;index" json:"userId"`
	Endpoint  string    `gorm:"column:endpoint;type:text;not null;uniqueIndex" json:"endpoint"`
	P256dh    string    `gorm:"column:p256dh;type:text;not null;" json:"p256dh"`
	Auth      string    `gorm:"column:auth;type:text;not null;" json:"auth"`
	Lang      string    `gorm:"column:lang;type:varchar(8);not null;default:'';" json:"lang"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	// LastActiveAt 最近活跃时间（成功发送推送后刷新；用于运维侧判断僵尸订阅）。
	// 复用 updated_at 会随任意归属迁移刷新，无法区分真实活跃，故单独维护。
	LastActiveAt time.Time `gorm:"column:last_active_at;index;" json:"lastActiveAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}

// Package pushDevice 持久化移动端原生推送设备注册（iOS APNs / Android JPush 或 FCM，
// mobile Route A）。一个用户可注册多台设备；token 全局唯一——同一设备
// 重复登录/换账号注册时按 token 收敛归属到当前用户（与 Web Push endpoint
// 的收敛语义一致）。
package pushDevice

import "time"

const tableName = "push_device"

// 平台取值：ios 走 APNs、android 由 provider 选择 JPush 或 FCM（与注册接口契约枚举一致）。
const (
	PlatformIOS     = "ios"
	PlatformAndroid = "android"
)

// Entity 一台已注册的原生推送设备。token 是推送服务长期凭据，等同用户会话
// 凭据对待：仅本人可管理（user_id 锚定）、账号注销即删、不写入日志明文。
type Entity struct {
	Id        uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	UserId    uint64    `gorm:"column:user_id;not null;default:0;index" json:"userId"`
	Platform  string    `gorm:"column:platform;type:varchar(16);not null;default:'';" json:"platform"`
	Provider  string    `gorm:"column:provider;type:varchar(16);not null;default:'';" json:"provider"`
	Token     string    `gorm:"column:token;type:varchar(512);not null;uniqueIndex" json:"-"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	// LastRegisteredAt 最近注册时间：每次重新注册（Upsert 命中冲突）刷新，
	// 用于运维侧判断僵尸注册；created_at 保持首次注册时间不变。
	LastRegisteredAt time.Time `gorm:"column:last_registered_at;index;" json:"lastRegisteredAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}

// DeliveryProvider preserves pre-provider registrations (iOS APNs / Android FCM).
func (itself *Entity) DeliveryProvider() string {
	if itself.Provider != "" {
		return itself.Provider
	}
	if itself.Platform == PlatformIOS {
		return "apns"
	}
	return "fcm"
}

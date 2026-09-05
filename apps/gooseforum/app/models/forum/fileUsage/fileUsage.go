package fileUsage

import "time"

const tableName = "file_usages"

const (
	TargetTopic       = "topic"
	TargetPost        = "post"
	TargetUser        = "user"
	TargetAdminUpload = "admin_upload"
	TargetUploadOwner = "upload_owner"
)

const (
	UsageInlineImage = "inline_image"
	UsageAvatar      = "avatar"
	UsageAdminUpload = "admin_upload"
	UsageUploadOwner = "upload_owner"
)

// 附件生命周期状态（Issue #94）：删除时转 RECOVERING，恢复时回 ACTIVE，永久删除置 PURGED。
const (
	UsageStatusActive     = "ACTIVE"
	UsageStatusRecovering = "RECOVERING"
	UsageStatusPurged     = "PURGED"
)

type Entity struct {
	Id         uint64     `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	FileName   string     `gorm:"column:file_name;type:varchar(512);not null;default:'';index:idx_file_usage_file_name;uniqueIndex:idx_file_usage_target_file,priority:4;" json:"fileName"` // 公开图片读取路径按 file_name 检查附件引用（/file/img/*），独立索引避免全表扫描
	TargetType string     `gorm:"column:target_type;type:varchar(32);not null;default:'';uniqueIndex:idx_file_usage_target_file,priority:1;" json:"targetType"`
	TargetId   uint64     `gorm:"column:target_id;not null;default:0;uniqueIndex:idx_file_usage_target_file,priority:2;" json:"targetId"`
	UsageType  string     `gorm:"column:usage_type;type:varchar(32);not null;default:'';uniqueIndex:idx_file_usage_target_file,priority:3;" json:"usageType"`
	UserId     uint64     `gorm:"column:user_id;not null;default:0;" json:"userId"`
	Status     string     `gorm:"column:status;type:varchar(32);not null;default:'ACTIVE';index:idx_file_usage_status,priority:1;" json:"status"`
	ExpiresAt  *time.Time `gorm:"column:expires_at;null;" json:"expiresAt"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt  time.Time  `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}

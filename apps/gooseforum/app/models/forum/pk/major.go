package pk

import (
	"time"

	"gorm.io/gorm"
)

const majorTableName = "pk_major"

// MajorEntity 一系统专业字典：name 唯一（"2025(03074 土木工程(国际班))" 形如 parseMajorString 解析结果）。
type MajorEntity struct {
	Id            uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	Code          string         `gorm:"column:code;type:varchar(32);not null;default:'';index:idx_pk_major_code;" json:"code"`
	Grade         *int           `gorm:"column:grade;index:idx_pk_major_grade;" json:"grade"`
	Name          string         `gorm:"column:name;type:varchar(255);not null;default:'';uniqueIndex:uniq_pk_major_name;" json:"name"`
	CalendarId    uint64         `gorm:"column:calendar_id;not null;default:0;" json:"calendarId"`
	SchemaVersion string         `gorm:"column:schema_version;type:varchar(64);not null;default:'';" json:"-"`
	SyncedAt      *time.Time     `gorm:"column:synced_at;" json:"-"`
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `json:"-"`
}

func (itself *MajorEntity) TableName() string {
	return majorTableName
}

package pk

import (
	"time"

	"gorm.io/gorm"
)

const calendarTableName = "pk_calendar"

// CalendarEntity 一系统学期（calendar）维度：calendar_id 是一系统侧的学期 ID，
// calendar_id_i18n 是人类可读学期标记（如 "2025-2026-1"），course-pk-sync 用它做 term↔calendar 解析。
type CalendarEntity struct {
	CalendarId     uint64 `gorm:"primaryKey;column:calendar_id;not null;" json:"calendarId"`
	CalendarIdI18n string `gorm:"column:calendar_id_i18n;type:varchar(64);not null;default:'';index:idx_pk_calendar_i18n;" json:"calendarIdI18n"`
	// 学期起止日期（可选，纯日期）：一系统 manualArrange 数据不含学期日期，
	// 由部署 config [pk.semester_dates] 维护，course-pk-sync 命中 calendar_id_i18n 时写入。
	// 排课器「当前周次」定位与学期日期条展示依赖这两列；未配置为 NULL。
	StartDate     *time.Time     `gorm:"column:start_date;type:date;" json:"-"`
	EndDate       *time.Time     `gorm:"column:end_date;type:date;" json:"-"`
	SchemaVersion string         `gorm:"column:schema_version;type:varchar(64);not null;default:'';" json:"-"`
	SyncedAt      *time.Time     `gorm:"column:synced_at;" json:"-"`
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `json:"-"`
}

func (itself *CalendarEntity) TableName() string {
	return calendarTableName
}

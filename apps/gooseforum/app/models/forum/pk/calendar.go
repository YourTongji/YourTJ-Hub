package pk

import (
	"time"

	"gorm.io/gorm"
)

const calendarTableName = "pk_calendar"

// CalendarEntity 一系统学期（calendar）维度：calendar_id 是一系统侧的学期 ID，
// calendar_id_i18n 是人类可读学期标记（如 "2025-2026-1"），course-pk-sync 用它做 term↔calendar 解析。
type CalendarEntity struct {
	CalendarId     uint64         `gorm:"primaryKey;column:calendar_id;not null;" json:"calendarId"`
	CalendarIdI18n string         `gorm:"column:calendar_id_i18n;type:varchar(64);not null;default:'';index:idx_pk_calendar_i18n;" json:"calendarIdI18n"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `json:"-"`
}

func (itself *CalendarEntity) TableName() string {
	return calendarTableName
}

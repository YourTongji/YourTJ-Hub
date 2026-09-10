package pk

import (
	"time"

	"gorm.io/gorm"
)

const languageTableName = "pk_language"

// LanguageEntity 一系统教学语言字典（由排课抓取数据幂等 upsert）。
type LanguageEntity struct {
	TeachingLanguage     string         `gorm:"primaryKey;column:teaching_language;type:varchar(64);not null;" json:"teachingLanguage"`
	TeachingLanguageI18n string         `gorm:"column:teaching_language_i18n;type:varchar(128);not null;default:'';" json:"teachingLanguageI18n"`
	CalendarId           uint64         `gorm:"column:calendar_id;not null;default:0;" json:"calendarId"`
	SchemaVersion        string         `gorm:"column:schema_version;type:varchar(64);not null;default:'';" json:"-"`
	SyncedAt             *time.Time     `gorm:"column:synced_at;" json:"-"`
	CreatedAt            time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt            time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt            gorm.DeletedAt `json:"-"`
}

func (itself *LanguageEntity) TableName() string {
	return languageTableName
}

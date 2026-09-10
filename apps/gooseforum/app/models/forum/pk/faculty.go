package pk

import (
	"time"

	"gorm.io/gorm"
)

const facultyTableName = "pk_faculty"

// FacultyEntity 一系统院系字典（faculty → I18n）。
type FacultyEntity struct {
	Faculty       string         `gorm:"primaryKey;column:faculty;type:varchar(255);not null;" json:"faculty"`
	FacultyI18n   string         `gorm:"column:faculty_i18n;type:varchar(255);not null;default:'';" json:"facultyI18n"`
	CalendarId    uint64         `gorm:"column:calendar_id;not null;default:0;" json:"calendarId"`
	SchemaVersion string         `gorm:"column:schema_version;type:varchar(64);not null;default:'';" json:"-"`
	SyncedAt      *time.Time     `gorm:"column:synced_at;" json:"-"`
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `json:"-"`
}

func (itself *FacultyEntity) TableName() string {
	return facultyTableName
}

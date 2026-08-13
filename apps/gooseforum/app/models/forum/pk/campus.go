package pk

import (
	"time"

	"gorm.io/gorm"
)

const campusTableName = "pk_campus"

// CampusEntity 一系统校区字典（campus → I18n）。
type CampusEntity struct {
	Campus     string         `gorm:"primaryKey;column:campus;type:varchar(128);not null;" json:"campus"`
	CampusI18n string         `gorm:"column:campus_i18n;type:varchar(128);not null;default:'';" json:"campusI18n"`
	CalendarId uint64         `gorm:"column:calendar_id;not null;default:0;" json:"calendarId"`
	CreatedAt  time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `json:"-"`
}

func (itself *CampusEntity) TableName() string {
	return campusTableName
}

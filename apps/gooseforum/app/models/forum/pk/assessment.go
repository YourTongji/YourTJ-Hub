package pk

import (
	"time"

	"gorm.io/gorm"
)

const assessmentTableName = "pk_assessment"

// AssessmentEntity 一系统考核方式字典（assessmentMode → I18n）。
type AssessmentEntity struct {
	AssessmentMode     string         `gorm:"primaryKey;column:assessment_mode;type:varchar(64);not null;" json:"assessmentMode"`
	AssessmentModeI18n string         `gorm:"column:assessment_mode_i18n;type:varchar(128);not null;default:'';" json:"assessmentModeI18n"`
	CalendarId         uint64         `gorm:"column:calendar_id;not null;default:0;" json:"calendarId"`
	CreatedAt          time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt          gorm.DeletedAt `json:"-"`
}

func (itself *AssessmentEntity) TableName() string {
	return assessmentTableName
}

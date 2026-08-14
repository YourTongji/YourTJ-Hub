package course

import (
	"time"

	"gorm.io/gorm"
)

const termTableName = "course_term"

// Entity 学期：code 形如 "2025-2026-1"，作为开课实例的学期维度。
type TermEntity struct {
	Id        uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	Code      string         `gorm:"column:code;type:varchar(32);not null;default:'';uniqueIndex:uniq_course_term_code;" json:"code"`
	Name      string         `gorm:"column:name;type:varchar(64);not null;default:'';" json:"name"`
	StartsOn  *time.Time     `gorm:"column:starts_on;type:date;" json:"startsOn"`
	EndsOn    *time.Time     `gorm:"column:ends_on;type:date;" json:"endsOn"`
	Status    int8           `gorm:"column:status;not null;default:0;" json:"status"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-"`
}

func (itself *TermEntity) TableName() string {
	return termTableName
}

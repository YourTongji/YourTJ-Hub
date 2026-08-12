package pk

import (
	"time"

	"gorm.io/gorm"
)

const courseNatureTableName = "pk_course_nature"
const courseNatureByCalendarTableName = "pk_course_nature_by_calendar"

// CourseNatureEntity 一系统课程性质字典（courseLabelId → 名称）。
type CourseNatureEntity struct {
	CourseLabelId   uint64         `gorm:"primaryKey;column:course_label_id;not null;" json:"courseLabelId"`
	CourseLabelName string         `gorm:"column:course_label_name;type:varchar(128);not null;default:'';" json:"courseLabelName"`
	CalendarId      uint64         `gorm:"column:calendar_id;not null;default:0;" json:"calendarId"`
	CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `json:"-"`
}

func (itself *CourseNatureEntity) TableName() string {
	return courseNatureTableName
}

// CourseNatureByCalendarEntity 课程性质按学期快照：course-pk-sync 按学期删除重建，保留历史不覆盖全局字典。
type CourseNatureByCalendarEntity struct {
	CalendarId      uint64    `gorm:"primaryKey;column:calendar_id;not null;" json:"calendarId"`
	CourseLabelId   uint64    `gorm:"primaryKey;column:course_label_id;not null;" json:"courseLabelId"`
	CourseLabelName string    `gorm:"column:course_label_name;type:varchar(128);not null;default:'';" json:"courseLabelName"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

func (itself *CourseNatureByCalendarEntity) TableName() string {
	return courseNatureByCalendarTableName
}

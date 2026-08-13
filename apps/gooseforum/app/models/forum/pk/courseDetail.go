package pk

import (
	"time"

	"gorm.io/gorm"
)

const courseDetailTableName = "pk_course_detail"

// CourseDetailEntity 一系统排课（教学班）主表：id 即一系统 teachingClassId。
// 字段名与一系统 manualArrange/page 返回逐项对齐（见 pk-login-and-export-sql.py 的列映射）。
type CourseDetailEntity struct {
	Id               uint64         `gorm:"primaryKey;column:id;not null;" json:"id"`
	Code             string         `gorm:"column:code;type:varchar(64);not null;default:'';index:idx_pk_course_detail_code;" json:"code"`
	Name             string         `gorm:"column:name;type:varchar(255);not null;default:'';" json:"name"`
	CourseLabelId    *uint64        `gorm:"column:course_label_id;index:idx_pk_course_detail_label;" json:"courseLabelId"`
	AssessmentMode   string         `gorm:"column:assessment_mode;type:varchar(64);not null;default:'';" json:"assessmentMode"`
	Period           *float64       `gorm:"column:period;" json:"period"`
	WeekHour         *float64       `gorm:"column:week_hour;" json:"weekHour"`
	Campus           string         `gorm:"column:campus;type:varchar(128);not null;default:'';" json:"campus"`
	Number           *int           `gorm:"column:number;" json:"number"`
	ElcNumber        *int           `gorm:"column:elc_number;" json:"elcNumber"`
	StartWeek        *int           `gorm:"column:start_week;" json:"startWeek"`
	EndWeek          *int           `gorm:"column:end_week;" json:"endWeek"`
	CourseCode       string         `gorm:"column:course_code;type:varchar(64);not null;default:'';index:idx_pk_course_detail_course_code;" json:"courseCode"`
	CourseName       string         `gorm:"column:course_name;type:varchar(255);not null;default:'';" json:"courseName"`
	Credit           *float64       `gorm:"column:credit;" json:"credit"`
	TeachingLanguage string         `gorm:"column:teaching_language;type:varchar(64);not null;default:'';" json:"teachingLanguage"`
	Faculty          string         `gorm:"column:faculty;type:varchar(255);not null;default:'';" json:"faculty"`
	CalendarId       uint64         `gorm:"column:calendar_id;not null;default:0;index:idx_pk_course_detail_calendar;" json:"calendarId"`
	NewCourseCode    string         `gorm:"column:new_course_code;type:varchar(64);not null;default:'';index:idx_pk_course_detail_new_course_code;" json:"newCourseCode"`
	NewCode          string         `gorm:"column:new_code;type:varchar(64);not null;default:'';index:idx_pk_course_detail_new_code;" json:"newCode"`
	CreatedAt        time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `json:"-"`
}

func (itself *CourseDetailEntity) TableName() string {
	return courseDetailTableName
}

package pk

import (
	"time"

	"gorm.io/gorm"
)

const teacherTableName = "pk_teacher"

// TeacherEntity 教学班授课教师：id 即一系统 teacherList[].id；arrange_info_text 保留原始排课文本，
// teacher_timeslots 由它派生重建。
type TeacherEntity struct {
	Id              uint64         `gorm:"primaryKey;column:id;not null;" json:"id"`
	TeachingClassId uint64         `gorm:"column:teaching_class_id;not null;default:0;index:idx_pk_teacher_class;" json:"teachingClassId"`
	TeacherCode     string         `gorm:"column:teacher_code;type:varchar(64);not null;default:'';index:idx_pk_teacher_code;" json:"teacherCode"`
	TeacherName     string         `gorm:"column:teacher_name;type:varchar(128);not null;default:'';index:idx_pk_teacher_name;" json:"teacherName"`
	ArrangeInfoText string         `gorm:"column:arrange_info_text;type:text;not null;default:'';" json:"arrangeInfoText"`
	CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `json:"-"`
}

func (itself *TeacherEntity) TableName() string {
	return teacherTableName
}

package course

import (
	"time"

	"gorm.io/gorm"
)

const instructorTableName = "course_instructor"

// Entity 教师：按 (name, department) 作为 importer 的自然键，支持团队授课。
type InstructorEntity struct {
	Id             uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	TeacherCode    string         `gorm:"column:teacher_code;type:varchar(64);not null;default:'';index:idx_course_instructor_teacher_code;" json:"teacherCode"`
	Name           string         `gorm:"column:name;type:varchar(64);not null;default:'';" json:"name"`
	NormalizedName string         `gorm:"column:normalized_name;type:varchar(64);not null;default:'';index:idx_course_instructor_normalized;" json:"normalizedName"`
	NamePinyin     string         `gorm:"column:name_pinyin;type:varchar(255);not null;default:'';" json:"namePinyin"`
	NameInitials   string         `gorm:"column:name_initials;type:varchar(64);not null;default:'';" json:"nameInitials"`
	Department     string         `gorm:"column:department;type:varchar(255);not null;default:'';" json:"department"`
	Title          string         `gorm:"column:title;type:varchar(64);not null;default:'';" json:"title"`
	Status         int8           `gorm:"column:status;not null;default:0;" json:"status"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `json:"-"`
}

func (itself *InstructorEntity) TableName() string {
	return instructorTableName
}

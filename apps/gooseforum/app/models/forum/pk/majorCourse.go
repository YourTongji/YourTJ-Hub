package pk

import (
	"time"
)

const majorCourseTableName = "pk_major_course"

// MajorCourseEntity 专业×教学班关联（一系统 majorList × teachingClassId）。
type MajorCourseEntity struct {
	MajorId   uint64    `gorm:"primaryKey;column:major_id;not null;" json:"majorId"`
	CourseId  uint64    `gorm:"primaryKey;column:course_id;not null;index:idx_pk_major_course_course;" json:"courseId"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

func (itself *MajorCourseEntity) TableName() string {
	return majorCourseTableName
}

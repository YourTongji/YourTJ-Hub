package pk

import (
	"time"
)

const teacherTimeslotTableName = "pk_teacher_timeslot"

// TeacherTimeslotEntity 教师时间片投影：由 pk_teacher.arrange_info_text 解析重建（course-pk-sync 触发），
// 复合主键与上游 teacher_timeslots 一致，供排课查询按 (calendar, day, section) 反查。
type TeacherTimeslotEntity struct {
	CalendarId      uint64     `gorm:"primaryKey;column:calendar_id;not null;index:idx_pk_timeslot_slot,priority:1;" json:"calendarId"`
	TeachingClassId uint64     `gorm:"primaryKey;column:teaching_class_id;not null;index:idx_pk_timeslot_class;" json:"teachingClassId"`
	OccupyDay       int        `gorm:"primaryKey;column:occupy_day;not null;index:idx_pk_timeslot_slot,priority:2;" json:"occupyDay"`
	OccupySection   int        `gorm:"primaryKey;column:occupy_section;not null;index:idx_pk_timeslot_slot,priority:3;" json:"occupySection"`
	TeacherCode     string     `gorm:"primaryKey;column:teacher_code;type:varchar(64);not null;default:'';" json:"teacherCode"`
	TeacherName     string     `gorm:"primaryKey;column:teacher_name;type:varchar(128);not null;default:'';" json:"teacherName"`
	SchemaVersion   string     `gorm:"column:schema_version;type:varchar(64);not null;default:'';" json:"-"`
	SyncedAt        *time.Time `gorm:"column:synced_at;" json:"-"`
}

func (itself *TeacherTimeslotEntity) TableName() string {
	return teacherTimeslotTableName
}

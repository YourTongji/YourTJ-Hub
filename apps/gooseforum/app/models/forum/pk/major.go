package pk

const (
	majorTableName       = "pk_major"
	majorCourseTableName = "pk_major_course"
)

// MajorEntity 专业字典：code/grade/name 三元组 + 所属学期。
// name 形如 "2025(03074 土木工程(国际班))"，code 为专业代号、grade 为入学年级。
type MajorEntity struct {
	Id            uint64 `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	Code          string `gorm:"column:code;type:varchar(64);not null;default:'';index:idx_pk_major_code;" json:"code"`
	Grade         int    `gorm:"column:grade;not null;default:0;index:idx_pk_major_grade;" json:"grade"`
	Name          string `gorm:"column:name;type:varchar(255);not null;default:'';" json:"name"`
	CalendarId    int    `gorm:"column:calendar_id;not null;default:0;" json:"calendarId"`
	SchemaVersion string `gorm:"column:schema_version;type:varchar(32);not null;default:'';" json:"-"`
	SyncedAt      int64  `gorm:"column:synced_at;not null;default:0;" json:"-"`
}

func (MajorEntity) TableName() string {
	return majorTableName
}

// MajorCourseEntity 专业-教学班关联（majorId, courseId），courseId 即 PkCourseDetail.Id。
type MajorCourseEntity struct {
	MajorId  uint64 `gorm:"primaryKey;column:major_id;not null;" json:"majorId"`
	CourseId uint64 `gorm:"primaryKey;column:course_id;not null;index:idx_pk_major_course_course;" json:"courseId"`
}

func (MajorCourseEntity) TableName() string {
	return majorCourseTableName
}

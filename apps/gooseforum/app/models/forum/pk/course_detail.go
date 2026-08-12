package pk

const courseDetailTableName = "pk_course_detail"

// CourseDetailEntity 教学班（teaching class）：一个教学班是一个具体开课班级。
// Id 即上游 teachingClassId；courseCode 为一系统课程码，code 为教学班内部编号，
// newCourseCode/newCode 为迁移后的新课程码（用于与课评目录 primary_code 对齐）。
type CourseDetailEntity struct {
	Id               uint64  `gorm:"primaryKey;column:id;not null;" json:"id"`
	Code             string  `gorm:"column:code;type:varchar(64);not null;default:'';index:idx_pk_course_detail_code;" json:"code"`
	Name             string  `gorm:"column:name;type:varchar(255);not null;default:'';" json:"name"`
	CourseLabelId    int     `gorm:"column:course_label_id;not null;default:0;" json:"courseLabelId"`
	AssessmentMode   string  `gorm:"column:assessment_mode;type:varchar(64);not null;default:'';" json:"assessmentMode"`
	Period           float64 `gorm:"column:period;not null;default:0;" json:"period"`
	WeekHour         float64 `gorm:"column:week_hour;not null;default:0;" json:"weekHour"`
	Campus           string  `gorm:"column:campus;type:varchar(64);not null;default:'';" json:"campus"`
	Number           int     `gorm:"column:number;not null;default:0;" json:"number"`
	ElcNumber        int     `gorm:"column:elc_number;not null;default:0;" json:"elcNumber"`
	StartWeek        int     `gorm:"column:start_week;not null;default:0;" json:"startWeek"`
	EndWeek          int     `gorm:"column:end_week;not null;default:0;" json:"endWeek"`
	CourseCode       string  `gorm:"column:course_code;type:varchar(64);not null;default:'';index:idx_pk_course_detail_course_code;" json:"courseCode"`
	CourseName       string  `gorm:"column:course_name;type:varchar(255);not null;default:'';" json:"courseName"`
	Credit           float64 `gorm:"column:credit;not null;default:0;" json:"credit"`
	TeachingLanguage string  `gorm:"column:teaching_language;type:varchar(64);not null;default:'';" json:"teachingLanguage"`
	Faculty          string  `gorm:"column:faculty;type:varchar(64);not null;default:'';" json:"faculty"`
	CalendarId       int     `gorm:"column:calendar_id;not null;default:0;index:idx_pk_course_detail_calendar;" json:"calendarId"`
	NewCourseCode    string  `gorm:"column:new_course_code;type:varchar(64);not null;default:'';index:idx_pk_course_detail_new_course_code;" json:"newCourseCode"`
	NewCode          string  `gorm:"column:new_code;type:varchar(64);not null;default:'';index:idx_pk_course_detail_new_code;" json:"newCode"`
	SchemaVersion    string  `gorm:"column:schema_version;type:varchar(32);not null;default:'';" json:"-"`
	SyncedAt         int64   `gorm:"column:synced_at;not null;default:0;" json:"-"`
}

func (CourseDetailEntity) TableName() string {
	return courseDetailTableName
}

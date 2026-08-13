package pk

const (
	campusTableName       = "pk_campus"
	facultyTableName      = "pk_faculty"
	languageTableName     = "pk_language"
	assessmentTableName   = "pk_assessment"
	courseNatureTableName = "pk_course_nature"
)

// CampusEntity 校区字典：单列主键 code（如 SP/JD），全局字典，同 code 只保留一行（last-write-wins）。
// CalendarId 仅为「最后写入学期」的元数据，不代表按学期隔离——与明细表 JOIN 时不得附加
// calendar_id 等值条件（详情见 rep_course/rep_dict/rep_teacher 的 LEFT JOIN）。
type CampusEntity struct {
	Campus        string `gorm:"primaryKey;column:campus;type:varchar(64);not null;" json:"campus"`
	CampusI18n    string `gorm:"column:campus_i18n;type:varchar(255);not null;default:'';" json:"campusI18n"`
	CalendarId    int    `gorm:"column:calendar_id;not null;default:0;index:idx_pk_campus_calendar;" json:"calendarId"`
	SchemaVersion string `gorm:"column:schema_version;type:varchar(32);not null;default:'';" json:"-"`
	SyncedAt      int64  `gorm:"column:synced_at;not null;default:0;" json:"-"`
}

func (CampusEntity) TableName() string {
	return campusTableName
}

// FacultyEntity 院系列表：单列主键 code（如 CS），全局字典，同 code 只保留一行（last-write-wins）。
// CalendarId 仅为「最后写入学期」的元数据，非按学期隔离——与明细表 JOIN 时不得附加 calendar_id 等值条件。
type FacultyEntity struct {
	Faculty       string `gorm:"primaryKey;column:faculty;type:varchar(64);not null;" json:"faculty"`
	FacultyI18n   string `gorm:"column:faculty_i18n;type:varchar(255);not null;default:'';" json:"facultyI18n"`
	CalendarId    int    `gorm:"column:calendar_id;not null;default:0;index:idx_pk_faculty_calendar;" json:"calendarId"`
	SchemaVersion string `gorm:"column:schema_version;type:varchar(32);not null;default:'';" json:"-"`
	SyncedAt      int64  `gorm:"column:synced_at;not null;default:0;" json:"-"`
}

func (FacultyEntity) TableName() string {
	return facultyTableName
}

// LanguageEntity 教学语言字典（如 ZH/EN）：单列主键，全局字典，同 code 只保留一行（last-write-wins）。
// CalendarId 仅为「最后写入学期」的元数据，非按学期隔离——与明细表 JOIN 时不得附加 calendar_id 等值条件。
type LanguageEntity struct {
	TeachingLanguage     string `gorm:"primaryKey;column:teaching_language;type:varchar(64);not null;" json:"teachingLanguage"`
	TeachingLanguageI18n string `gorm:"column:teaching_language_i18n;type:varchar(255);not null;default:'';" json:"teachingLanguageI18n"`
	CalendarId           int    `gorm:"column:calendar_id;not null;default:0;index:idx_pk_language_calendar;" json:"calendarId"`
	SchemaVersion        string `gorm:"column:schema_version;type:varchar(32);not null;default:'';" json:"-"`
	SyncedAt             int64  `gorm:"column:synced_at;not null;default:0;" json:"-"`
}

func (LanguageEntity) TableName() string {
	return languageTableName
}

// AssessmentEntity 考核方式字典（如 EXAM/CHECK）：单列主键，全局字典，同 code 只保留一行（last-write-wins）。
// CalendarId 仅为「最后写入学期」的元数据，非按学期隔离——与明细表 JOIN 时不得附加 calendar_id 等值条件。
type AssessmentEntity struct {
	AssessmentMode     string `gorm:"primaryKey;column:assessment_mode;type:varchar(64);not null;" json:"assessmentMode"`
	AssessmentModeI18n string `gorm:"column:assessment_mode_i18n;type:varchar(255);not null;default:'';" json:"assessmentModeI18n"`
	CalendarId         int    `gorm:"column:calendar_id;not null;default:0;index:idx_pk_assessment_calendar;" json:"calendarId"`
	SchemaVersion      string `gorm:"column:schema_version;type:varchar(32);not null;default:'';" json:"-"`
	SyncedAt           int64  `gorm:"column:synced_at;not null;default:0;" json:"-"`
}

func (AssessmentEntity) TableName() string {
	return assessmentTableName
}

// CourseNatureEntity 课程性质（coursenature_by_calendar）：课程标签按学期隔离，
// 复合主键 (calendar_id, course_label_id) 避免跨学期覆盖（上游 002 迁移语义）。
type CourseNatureEntity struct {
	CalendarId      int    `gorm:"primaryKey;column:calendar_id;not null;index:idx_pk_course_nature_calendar;" json:"calendarId"`
	CourseLabelId   int    `gorm:"primaryKey;column:course_label_id;not null;index:idx_pk_course_nature_label;" json:"courseLabelId"`
	CourseLabelName string `gorm:"column:course_label_name;type:varchar(255);not null;default:'';" json:"courseLabelName"`
	SchemaVersion   string `gorm:"column:schema_version;type:varchar(32);not null;default:'';" json:"-"`
	SyncedAt        int64  `gorm:"column:synced_at;not null;default:0;" json:"-"`
}

func (CourseNatureEntity) TableName() string {
	return courseNatureTableName
}

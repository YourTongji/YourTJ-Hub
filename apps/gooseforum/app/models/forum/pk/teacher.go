package pk

const (
	teacherTableName         = "pk_teacher"
	teacherTimeslotTableName = "pk_teacher_timeslot"
)

// TeacherEntity 教学班教师：一条记录是一个教学班的一位教师的时间安排文本。
// arrangeInfoText 每行一条安排（\n 分隔），形如
// "张伟(T001) 星期一1-2节[1-16周] 四平路校区 A101"。
type TeacherEntity struct {
	Id              uint64 `gorm:"primaryKey;column:id;not null;" json:"id"`
	TeachingClassId uint64 `gorm:"column:teaching_class_id;not null;default:0;index:idx_pk_teacher_class;" json:"teachingClassId"`
	TeacherCode     string `gorm:"column:teacher_code;type:varchar(64);not null;default:'';index:idx_pk_teacher_code;" json:"teacherCode"`
	TeacherName     string `gorm:"column:teacher_name;type:varchar(255);not null;default:'';index:idx_pk_teacher_name;" json:"teacherName"`
	// arrange_info_text 用 text 无默认值：MySQL 不允许 TEXT 列的裸 DEFAULT 子句（8.0.13+ 需表达式形式），
	// 避免三方言迁移歧义。空值在应用层以 "" 处理。
	ArrangeInfoText string `gorm:"column:arrange_info_text;type:text;not null;" json:"arrangeInfoText"`
	SchemaVersion   string `gorm:"column:schema_version;type:varchar(32);not null;default:'';" json:"-"`
	SyncedAt        int64  `gorm:"column:synced_at;not null;default:0;" json:"-"`
}

func (TeacherEntity) TableName() string {
	return teacherTableName
}

// TeacherTimeslotEntity 教师时间片投影：由 arrangeInfoText 解析构建，
// courses-by-time 查询的性能关键（避免逐行 LIKE）。懒构建 + 版本号重建
// （PK_AUX_SCHEMA_VERSION），未就绪时回退 arrangeInfoText LIKE 降级查询。
type TeacherTimeslotEntity struct {
	CalendarId      int    `gorm:"primaryKey;column:calendar_id;not null;" json:"calendarId"`
	TeachingClassId uint64 `gorm:"primaryKey;column:teaching_class_id;not null;" json:"teachingClassId"`
	OccupyDay       int    `gorm:"primaryKey;column:occupy_day;not null;" json:"occupyDay"`
	OccupySection   int    `gorm:"primaryKey;column:occupy_section;not null;" json:"occupySection"`
	TeacherCode     string `gorm:"primaryKey;column:teacher_code;type:varchar(64);not null;default:'';" json:"teacherCode"`
	TeacherName     string `gorm:"primaryKey;column:teacher_name;type:varchar(255);not null;default:'';" json:"teacherName"`
	SchemaVersion   string `gorm:"column:schema_version;type:varchar(32);not null;default:'';" json:"-"`
	SyncedAt        int64  `gorm:"column:synced_at;not null;default:0;" json:"-"`
}

func (TeacherTimeslotEntity) TableName() string {
	return teacherTimeslotTableName
}

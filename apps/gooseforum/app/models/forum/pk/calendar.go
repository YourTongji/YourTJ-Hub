package pk

const calendarTableName = "pk_calendar"

// CalendarEntity 学期（calendar）字典：一系统同步结果按学期隔离。
// CalendarId 即学期编号（如 20252026），CalendarIdI18n 为学期名称（如 "2025-2026 第一学期"）。
// schema_version / synced_at 为重建与部分更新用的元数据列（PRD §5.4.2）。
type CalendarEntity struct {
	CalendarId     int    `gorm:"primaryKey;column:calendar_id;not null;" json:"calendarId"`
	CalendarIdI18n string `gorm:"column:calendar_id_i18n;type:varchar(255);not null;default:'';" json:"calendarIdI18n"`
	SchemaVersion  string `gorm:"column:schema_version;type:varchar(32);not null;default:'';" json:"-"`
	SyncedAt       int64  `gorm:"column:synced_at;not null;default:0;" json:"-"`
}

func (CalendarEntity) TableName() string {
	return calendarTableName
}

package pk

const (
	fetchLogTableName = "pk_fetch_log"
	settingTableName  = "pk_setting"
)

// FetchLogEntity 一系统同步日志：fetch_time 为 Unix 秒，latest-update 端点取其最大值。
type FetchLogEntity struct {
	Id        uint64 `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	FetchTime int64  `gorm:"column:fetch_time;not null;default:0;index:idx_pk_fetch_log_time;" json:"fetchTime"`
	Msg       string `gorm:"column:msg;type:varchar(255);not null;default:'';" json:"msg"`
}

func (FetchLogEntity) TableName() string {
	return fetchLogTableName
}

// SettingEntity PK 模块键值存储：用于 teacher_timeslots 辅助表版本号
// （pk_aux_schema_version），判断是否需要重建时间片投影。
type SettingEntity struct {
	Key   string `gorm:"primaryKey;column:key;type:varchar(64);not null;" json:"key"`
	Value string `gorm:"column:value;type:varchar(255);not null;default:'';" json:"value"`
}

func (SettingEntity) TableName() string {
	return settingTableName
}

package pk

import (
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
)

const settingTableName = "pk_setting"

// SettingEntity PK 模块键值存储：用于 teacher_timeslots 辅助表版本号
// （pk_aux_schema_version），判断是否需要重建时间片投影。
type SettingEntity struct {
	Key   string `gorm:"primaryKey;column:key;type:varchar(64);not null;" json:"key"`
	Value string `gorm:"column:value;type:varchar(255);not null;default:'';" json:"value"`
}

func (SettingEntity) TableName() string {
	return settingTableName
}

func settingBuilder() *gorm.DB {
	return db.Connect().Table(settingTableName)
}

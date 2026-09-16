package sticker

import (
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
)

const tableName = "stickers"

// 表情包名约束：token 形态为 [:sticker:name:]，name 不允许包含
// 冒号/方括号/换行等会破坏 token 解析的字符，由服务层校验。
const MaxNameLen = 64

type Entity struct {
	Id        uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	Name      string    `gorm:"column:name;type:varchar(64);not null;uniqueIndex;" json:"name"`
	FileName  string    `gorm:"column:file_name;type:varchar(512);not null;default:'';" json:"fileName"`
	SortOrder int       `gorm:"column:sort_order;type:int;not null;default:0;index;" json:"sortOrder"`
	IsEnabled bool      `gorm:"column:is_enabled;type:boolean;not null;default:true;index;" json:"isEnabled"`
	CreatedBy uint64    `gorm:"column:created_by;not null;default:0;" json:"createdBy"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

func builder() *gorm.DB {
	return db.Connect().Table(tableName)
}

func (itself *Entity) TableName() string {
	return tableName
}

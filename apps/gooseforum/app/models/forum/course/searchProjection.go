package course

import (
	"time"

	"gorm.io/gorm"
)

const searchProjectionTableName = "course_search_projection"

// Entity 课程搜索投影状态：worker 消费 outbox 后记录已索引版本与错误，
// 供 rebuild/reconcile 对账；PG 仍是事实源，Meili 是可重建投影。
type SearchProjectionEntity struct {
	CourseId       uint64         `gorm:"primaryKey;column:course_id;not null;" json:"courseId"`
	IndexedVersion uint64         `gorm:"column:indexed_version;not null;default:0;" json:"indexedVersion"`
	IndexedAt      *time.Time     `gorm:"column:indexed_at;" json:"indexedAt"`
	LastError      string         `gorm:"column:last_error;type:text;not null;default:'';" json:"lastError"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `json:"-"`
}

func (itself *SearchProjectionEntity) TableName() string {
	return searchProjectionTableName
}

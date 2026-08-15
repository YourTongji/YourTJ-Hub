package wikiSyncRuns

import (
	"time"

	"gorm.io/gorm"
)

const tableName = "wiki_sync_runs"

// 同步状态。
const (
	StatusRunning = "running" // 同步进行中
	StatusSuccess = "success" // 成功
	StatusFailed  = "failed"  // 失败（错误详情见 Error）
)

// 触发源。
const (
	TriggerWebhook = "webhook" // GitHub webhook 即时触发
	TriggerTicker  = "ticker"  // 每日定时
	TriggerManual  = "manual"  // 管理端手动「立即同步」
)

// Entity wiki 同步运行日志（迁移 v21 新建，同步引擎后端落库）。
type Entity struct {
	Id            uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	HeadSha       string         `gorm:"column:head_sha;type:varchar(40);not null;default:'';" json:"headSha"`
	InsertedCount int            `gorm:"column:inserted_count;type:int;not null;default:0;" json:"insertedCount"`
	UpdatedCount  int            `gorm:"column:updated_count;type:int;not null;default:0;" json:"updatedCount"`
	DeletedCount  int            `gorm:"column:deleted_count;type:int;not null;default:0;" json:"deletedCount"`
	Status        string         `gorm:"column:status;type:varchar(16);not null;default:'';" json:"status"`
	Error         string         `gorm:"column:error;type:text;" json:"error"`
	DurationMs    int64          `gorm:"column:duration_ms;type:bigint;not null;default:0;" json:"durationMs"`
	TriggerSource string         `gorm:"column:trigger_source;type:varchar(16);not null;default:'';" json:"triggerSource"`
	StartedAt     *time.Time     `gorm:"column:started_at;" json:"startedAt"`
	FinishedAt    *time.Time     `gorm:"column:finished_at;" json:"finishedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;" json:"-"`
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}

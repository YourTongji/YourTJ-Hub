package course

import (
	"time"

	"gorm.io/gorm"
)

const importRunTableName = "course_import_run"

// Entity 一次目录/评价导入的运行记录：manifest_hash 唯一保证幂等，可断点续跑。
type ImportRunEntity struct {
	Id               uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	Source           string         `gorm:"column:source;type:varchar(64);not null;default:'';" json:"source"`
	ManifestHash     string         `gorm:"column:manifest_hash;type:varchar(64);not null;default:'';uniqueIndex:uniq_course_import_run_manifest;" json:"manifestHash"`
	Status           string         `gorm:"column:status;type:varchar(32);not null;default:'';" json:"status"`
	InsertedCount    int            `gorm:"column:inserted_count;not null;default:0;" json:"insertedCount"`
	UpdatedCount     int            `gorm:"column:updated_count;not null;default:0;" json:"updatedCount"`
	QuarantinedCount int            `gorm:"column:quarantined_count;not null;default:0;" json:"quarantinedCount"`
	ErrorCount       int            `gorm:"column:error_count;not null;default:0;" json:"errorCount"`
	StartedAt        *time.Time     `gorm:"column:started_at;" json:"startedAt"`
	FinishedAt       *time.Time     `gorm:"column:finished_at;" json:"finishedAt"`
	CreatedAt        time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `json:"-"`
}

// 导入状态
const (
	ImportStatusRunning   string = "running"
	ImportStatusCompleted string = "completed"
	ImportStatusFailed    string = "failed"
)

func (itself *ImportRunEntity) TableName() string {
	return importRunTableName
}

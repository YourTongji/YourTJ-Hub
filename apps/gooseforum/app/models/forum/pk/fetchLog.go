package pk

import (
	"time"

	"gorm.io/gorm"
)

const fetchLogTableName = "pk_fetch_log"

// FetchLog 状态
const (
	FetchStatusRunning   string = "running"
	FetchStatusCompleted string = "completed"
	FetchStatusFailed    string = "failed"
)

// FetchLogEntity 一系统同步运行日志：按学期（calendar_id）记录游标，进程崩溃后据此断点续跑（AC3）。
// 语义参照 ImportRunEntity：running/completed/failed + 计数；额外记录 last_committed_page 作为批量游标。
type FetchLogEntity struct {
	Id                uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	CalendarId        uint64         `gorm:"column:calendar_id;not null;default:0;index:idx_pk_fetch_log_calendar;" json:"calendarId"`
	Status            string         `gorm:"column:status;type:varchar(32);not null;default:'';" json:"status"`
	TotalPages        int            `gorm:"column:total_pages;not null;default:0;" json:"totalPages"`
	LastCommittedPage int            `gorm:"column:last_committed_page;not null;default:0;" json:"lastCommittedPage"`
	RowsWritten       int            `gorm:"column:rows_written;not null;default:0;" json:"rowsWritten"`
	ErrorMsg          string         `gorm:"column:error_msg;type:text;not null;default:'';" json:"errorMsg"`
	StartedAt         *time.Time     `gorm:"column:started_at;" json:"startedAt"`
	FinishedAt        *time.Time     `gorm:"column:finished_at;" json:"finishedAt"`
	CreatedAt         time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt         gorm.DeletedAt `json:"-"`
}

func (itself *FetchLogEntity) TableName() string {
	return fetchLogTableName
}

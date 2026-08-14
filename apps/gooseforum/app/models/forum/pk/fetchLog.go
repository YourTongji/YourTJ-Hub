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
//
// 并发与跨方言（review HIGH「唯一/租约」P1，issue #186）：
//   - LeaseVersion：单调递增的精确 CAS/lease token。ClaimFetchLog 以
//     「WHERE lease_version=<旧值> SET lease_version=lease_version+1」的条件 UPDATE 原子认领，
//     用 RowsAffected==1 判定是否赢得竞态。不用 started_at 时间戳做 token——不同方言时间精度
//     不一致，两次写入可能取同一值，CAS 失效。
//   - RunningKey：仅当 status='running' 时置为 calendar_id，其余状态置 NULL，配普通 UNIQUE 索引
//     uniq_pk_fetch_log_running_key。SQLite/PostgreSQL 均允许唯一索引含多个 NULL，故
//     completed/failed（NULL）行之间永不冲突，但同一 calendar 的两条 running 行必然冲突——这是
//     「同一 calendar 至多一条 running」的跨方言唯一保证（不依赖方言专属的 partial unique index）。
type FetchLogEntity struct {
	Id                uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	CalendarId        uint64         `gorm:"column:calendar_id;not null;default:0;index:idx_pk_fetch_log_calendar;" json:"calendarId"`
	RunningKey        *uint64        `gorm:"column:running_key;index:uniq_pk_fetch_log_running_key,unique;" json:"-"`
	Status            string         `gorm:"column:status;type:varchar(32);not null;default:'';" json:"status"`
	LeaseVersion      int            `gorm:"column:lease_version;not null;default:0;" json:"-"`
	TotalPages        int            `gorm:"column:total_pages;not null;default:0;" json:"totalPages"`
	LastCommittedPage int            `gorm:"column:last_committed_page;not null;default:0;" json:"lastCommittedPage"`
	RowsWritten       int            `gorm:"column:rows_written;not null;default:0;" json:"rowsWritten"`
	ErrorMsg          string         `gorm:"column:error_msg;type:text;not null;default:'';" json:"errorMsg"`
	StartedAt         *time.Time     `gorm:"column:started_at;" json:"startedAt"`
	FinishedAt        *time.Time     `gorm:"column:finished_at;" json:"finishedAt"`
	SchemaVersion     string         `gorm:"column:schema_version;type:varchar(64);not null;default:'';" json:"-"`
	SyncedAt          *time.Time     `gorm:"column:synced_at;" json:"-"`
	CreatedAt         time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt         gorm.DeletedAt `json:"-"`
}

func (itself *FetchLogEntity) TableName() string {
	return fetchLogTableName
}

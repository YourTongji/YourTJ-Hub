package taskQueue

import (
	"time"
)

const tableName = "task_queue"

// 任务状态常量
const (
	StatusPending  = 0 // 待处理
	StatusRunning  = 1 // 处理中
	StatusSuccess  = 2 // 处理成功
	StatusFailed   = 3 // 处理失败
	StatusRetrying = 4 // 重试中
)

// LeaseDuration 是任务租约时长（issue #138）。任务被领取时以 processed_at 记录
// 租约起点，worker 执行期间通过 RenewLease 心跳持续续约；超过 LeaseDuration
// 未续约的 Running 任务视为崩溃残留，由 RecoverStaleRunning 回收为 Pending
// 重新领取，避免任务永久卡在 Running。
const LeaseDuration = 10 * time.Minute

type Entity struct {
	Id          uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	Type        string    `gorm:"column:type;type:varchar(50);not null;" json:"type"`       // 任务类型
	Status      uint8     `gorm:"column:status;not null;default:0;index" json:"status"`     // 任务状态
	TaskJson    string    `gorm:"column:task_json;type:text;" json:"taskJson"`              // 任务数据
	RetryCount  uint8     `gorm:"column:retry_count;not null;default:0;" json:"retryCount"` // 重试次数
	LastError   string    `gorm:"column:last_error;type:text;" json:"lastError"`            // 最后一次错误信息
	CreatedAt   time.Time `gorm:"column:created_at;index;autoCreateTime;<-:create;" json:"createdAt"`
	ProcessedAt time.Time `gorm:"column:processed_at;" json:"processedAt"` // 处理时间
}

func (itself *Entity) TableName() string {
	return tableName
}

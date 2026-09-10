package wikiSyncRuns

import "time"

const tableName = "wiki_sync_runs"

// 同步状态
const (
	StatusRunning int8 = 0 // 运行中
	StatusSuccess int8 = 1 // 成功
	StatusFailed  int8 = 2 // 失败
)

// Entity wiki 同步运行日志：每次从 GitHub wiki 仓库拉取并投影到论坛记录一行。
type Entity struct {
	Id           uint64     `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	HeadSha      string     `gorm:"column:head_sha;type:varchar(64);not null;default:'';" json:"headSha"`
	Trigger      string     `gorm:"column:trigger;type:varchar(32);not null;default:'';" json:"trigger"` // manual | schedule | webhook
	Status       int8       `gorm:"column:status;not null;default:0;index:idx_wiki_sync_status,priority:1;" json:"status"`
	PagesAdded   int        `gorm:"column:pages_added;type:int;not null;default:0;" json:"pagesAdded"`
	PagesUpdated int        `gorm:"column:pages_updated;type:int;not null;default:0;" json:"pagesUpdated"`
	PagesDeleted int        `gorm:"column:pages_deleted;type:int;not null;default:0;" json:"pagesDeleted"`
	Error        string     `gorm:"column:error;type:text;" json:"error"`
	StartedAt    time.Time  `gorm:"column:started_at;autoCreateTime;<-:create;" json:"startedAt"`
	FinishedAt   *time.Time `gorm:"column:finished_at;" json:"finishedAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}

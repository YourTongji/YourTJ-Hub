package inboxmail

import (
	"encoding/json"
	"time"
)

// Campaign 生命周期（#773）：draft → scheduled → running → completed，
// 允许 running/paused 之间切换，cancelled 与 failed 为终态。
const (
	CampaignStatusDraft     = "draft"
	CampaignStatusScheduled = "scheduled"
	CampaignStatusRunning   = "running"
	CampaignStatusPaused    = "paused"
	CampaignStatusCompleted = "completed"
	CampaignStatusCancelled = "cancelled"
	CampaignStatusFailed    = "failed"
)

// 触发/时间方式：immediate 立即派发，scheduled 到点派发，event 由 Trigger Registry
// （#775）在事件发生时产生投递。
const (
	ScheduleTypeImmediate = "immediate"
	ScheduleTypeScheduled = "scheduled"
	ScheduleTypeEvent     = "event"
)

// CampaignRun 状态：pending 待 Worker 领取，running/paused 可恢复，#774 通过
// lease_token 续约与回收崩溃残留。
const (
	RunStatusPending   = "pending"
	RunStatusRunning   = "running"
	RunStatusPaused    = "paused"
	RunStatusCompleted = "completed"
	RunStatusFailed    = "failed"
	RunStatusCancelled = "cancelled"
)

// AudienceSpec 是受众描述信封：Kind 是 Audience Field Registry 的键（#772），
// Payload 由该 Kind 自行解释。禁止把私信正文、私有备注等敏感字段暴露为分群条件。
type AudienceSpec struct {
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// PresentationSpec 是呈现策略信封：弹窗开关与延迟属于 P0 固定字段，其余呈现扩展
// 走 Payload（Presentation Registry）。用户对弹窗的收纳/关闭/Snooze 状态存放在
// Delivery 上，不在此处。
type PresentationSpec struct {
	PopupEnabled      bool            `json:"popupEnabled"`
	PopupDelaySeconds int             `json:"popupDelaySeconds,omitempty"`
	Payload           json.RawMessage `json:"payload,omitempty"`
}

// CampaignEntity 是管理端一次性/定时/事件投递任务，固定引用一个已发布
// Message Version；受众、呈现、附件解耦存放，发布内容不可变。
type CampaignEntity struct {
	Id                  uint64           `gorm:"primaryKey;column:id;autoIncrement;not null" json:"id"`
	MessageId           uint64           `gorm:"column:message_id;not null;default:0;index:idx_inbox_campaign_message" json:"messageId"`
	MessageVersionId    uint64           `gorm:"column:message_version_id;not null;default:0;index:idx_inbox_campaign_message_version" json:"messageVersionId"`
	Name                string           `gorm:"column:name;type:varchar(128);not null;default:''" json:"name"`
	Status              string           `gorm:"column:status;type:varchar(16);not null;default:'draft';index:idx_inbox_campaign_status_schedule,priority:1" json:"status"`
	ScheduleType        string           `gorm:"column:schedule_type;type:varchar(16);not null;default:'immediate'" json:"scheduleType"`
	ScheduledAt         *time.Time       `gorm:"column:scheduled_at;type:timestamp;index:idx_inbox_campaign_status_schedule,priority:2" json:"scheduledAt,omitempty"`
	Audience            AudienceSpec     `gorm:"column:audience;type:json;serializer:json" json:"audience"`
	Presentation        PresentationSpec `gorm:"column:presentation;type:json;serializer:json" json:"presentation"`
	EstimatedRecipients uint64           `gorm:"column:estimated_recipients;not null;default:0" json:"estimatedRecipients"`
	MaxRecipients       uint64           `gorm:"column:max_recipients;not null;default:0" json:"maxRecipients"` // 0 = 不限额
	CreatedBy           uint64           `gorm:"column:created_by;not null;default:0" json:"createdBy"`
	UpdatedBy           uint64           `gorm:"column:updated_by;not null;default:0" json:"updatedBy"`
	PublishedAt         *time.Time       `gorm:"column:published_at;type:timestamp" json:"publishedAt,omitempty"`
	CreatedAt           time.Time        `gorm:"column:created_at;autoCreateTime;<-:create" json:"createdAt"`
	UpdatedAt           time.Time        `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (itself *CampaignEntity) TableName() string {
	return campaignTableName
}

// CampaignAttachmentEntity 是 Campaign 的奖励附件。Handler 是 Reward Handler
// Registry 的键（P0 为 badge，#777 扩展 points/material 等），Payload 由 handler
// 解释；附件与内容解耦，Campaign 修订不改写附件事实。
type CampaignAttachmentEntity struct {
	Id            uint64          `gorm:"primaryKey;column:id;autoIncrement;not null" json:"id"`
	CampaignId    uint64          `gorm:"column:campaign_id;not null;default:0;uniqueIndex:uniq_inbox_campaign_attachment_key,priority:1;index:idx_inbox_campaign_attachment_order,priority:1" json:"campaignId"`
	AttachmentKey string          `gorm:"column:attachment_key;type:varchar(64);not null;uniqueIndex:uniq_inbox_campaign_attachment_key,priority:2;check:chk_inbox_campaign_attachment_key,attachment_key <> ''" json:"attachmentKey"`
	Handler       string          `gorm:"column:handler;type:varchar(32);not null;default:'';index:idx_inbox_campaign_attachment_handler" json:"handler"`
	Name          string          `gorm:"column:name;type:varchar(128);not null;default:''" json:"name"`
	Description   string          `gorm:"column:description;type:varchar(500);not null;default:''" json:"description"`
	IconUrl       string          `gorm:"column:icon_url;type:varchar(512);not null;default:''" json:"iconUrl"`
	Payload       json.RawMessage `gorm:"column:payload;type:json;serializer:json" json:"payload"`
	SortOrder     int             `gorm:"column:sort_order;not null;default:0;index:idx_inbox_campaign_attachment_order,priority:2" json:"sortOrder"`
	ExpiresAt     *time.Time      `gorm:"column:expires_at;type:timestamp" json:"expiresAt,omitempty"` // nil = 永不过期
	CreatedAt     time.Time       `gorm:"column:created_at;autoCreateTime;<-:create" json:"createdAt"`
	UpdatedAt     time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (itself *CampaignAttachmentEntity) TableName() string {
	return campaignAttachmentTableName
}

// CampaignRunEntity 是一次具体派发执行：cursor 记录可恢复进度，计数列是供管理端
// 快速展示的物化聚合（真源始终是 inbox_delivery 行）。lease_token/lease_expires_at
// 供 #774 Worker 的原子领取与崩溃回收使用（taskQueue 同款 fencing 语义）。
type CampaignRunEntity struct {
	Id             uint64     `gorm:"primaryKey;column:id;autoIncrement;not null;index:idx_inbox_campaign_run_status,priority:2" json:"id"`
	CampaignId     uint64     `gorm:"column:campaign_id;not null;default:0;uniqueIndex:uniq_inbox_campaign_run_no,priority:1" json:"campaignId"`
	RunNo          int        `gorm:"column:run_no;not null;default:1;uniqueIndex:uniq_inbox_campaign_run_no,priority:2" json:"runNo"`
	Status         string     `gorm:"column:status;type:varchar(16);not null;default:'pending';index:idx_inbox_campaign_run_status,priority:1" json:"status"`
	Cursor         string     `gorm:"column:cursor;type:varchar(255);not null;default:''" json:"cursor"`
	ProcessedCount int64      `gorm:"column:processed_count;not null;default:0" json:"processedCount"`
	DeliveredCount int64      `gorm:"column:delivered_count;not null;default:0" json:"deliveredCount"`
	SkippedCount   int64      `gorm:"column:skipped_count;not null;default:0" json:"skippedCount"`
	FailedCount    int64      `gorm:"column:failed_count;not null;default:0" json:"failedCount"`
	LeaseToken     string     `gorm:"column:lease_token;type:varchar(36);not null;default:''" json:"-"`
	LeaseExpiresAt *time.Time `gorm:"column:lease_expires_at;type:timestamp" json:"-"`
	LastError      string     `gorm:"column:last_error;type:varchar(512);not null;default:''" json:"lastError"`
	StartedAt      *time.Time `gorm:"column:started_at;type:timestamp" json:"startedAt,omitempty"`
	FinishedAt     *time.Time `gorm:"column:finished_at;type:timestamp" json:"finishedAt,omitempty"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime;<-:create" json:"createdAt"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (itself *CampaignRunEntity) TableName() string {
	return campaignRunTableName
}

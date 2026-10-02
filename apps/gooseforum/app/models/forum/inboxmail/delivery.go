package inboxmail

import (
	"time"
)

// Delivery 状态：delivered 进入信箱；suppressed 因频控/受众规则未投递但保留事实
// 供管理端解释「为什么没收到」；failed/cancelled 为投递失败与撤回。
const (
	DeliveryStatusDelivered  = "delivered"
	DeliveryStatusSuppressed = "suppressed"
	DeliveryStatusFailed     = "failed"
	DeliveryStatusCancelled  = "cancelled"
)

// Delivery 来源：campaign 由 Campaign Run 批量物化，trigger 由事件 Trigger 产生。
// dedupe_key 的命名空间与之一一对应。
const (
	DeliverySourceCampaign = "campaign"
	DeliverySourceTrigger  = "trigger"
)

// Popup 状态：同一 Delivery 同时服务信箱与弹窗；关闭/收纳/稍后提醒/此封不再弹
// 只改变弹窗状态，不删除信件（epic #769 核心决策 2/8/9）。状态跨端共享。
const (
	PopupStatePending   = "pending"   // 从未展示
	PopupStateShown     = "shown"     // 已展示
	PopupStateDismissed = "dismissed" // 本次关闭
	PopupStateCollapsed = "collapsed" // 收纳为胶囊
	PopupStateSnoozed   = "snoozed"   // 稍后提醒（配合 popup_snooze_until）
	PopupStateMuted     = "muted"     // 此封不再弹
)

// Claim 状态（Delivery 侧物化投影，词汇与 #778 对齐）：none 无附件；unclaimed 有
// 可领取附件且尚未领取；partial 部分附件已领取；claimed 全部附件领取或已拥有
// （成功终态）；expired 附件已全部过期且未成功领取（终态，需要 #778 的结算扫描）。
// 真源是逐行的 inbox_claim.status（pending/granted/already_owned/failed/expired），
// 本列只是客户端渲染领取入口与进度用的按 Delivery 聚合。
const (
	ClaimStateNone      = "none"
	ClaimStateUnclaimed = "unclaimed"
	ClaimStatePartial   = "partial"
	ClaimStateClaimed   = "claimed"
	ClaimStateExpired   = "expired"
)

// DeliveryEntity 是物化的「每人一封」信件，也是信箱列表、未读、归档、待领取与
// 弹窗候选的唯一查询入口。message_version_id 固定指向发布时的不可变版本，
// 保证历史收件可重现。
//
// 索引与查询对应关系：
//   - (user_id, id)：信箱游标分页（倒序）；
//   - (user_id, is_read, id)：未读列表 / 未读计数；
//   - (user_id, is_archived, id)：归档列表；
//   - (user_id, claim_state, id)：待领取列表；
//   - (user_id, popup_state, id)：弹窗候选；
//   - (campaign_id, status)：Campaign 维度统计与收件人排查。
type DeliveryEntity struct {
	Id               uint64 `gorm:"primaryKey;column:id;autoIncrement;not null;index:idx_inbox_delivery_user_id,priority:2;index:idx_inbox_delivery_user_read_id,priority:3;index:idx_inbox_delivery_user_archived_id,priority:3;index:idx_inbox_delivery_user_claim_id,priority:3;index:idx_inbox_delivery_user_popup_id,priority:3" json:"id"`
	UserId           uint64 `gorm:"column:user_id;not null;default:0;index:idx_inbox_delivery_user_id,priority:1;index:idx_inbox_delivery_user_read_id,priority:1;index:idx_inbox_delivery_user_archived_id,priority:1;index:idx_inbox_delivery_user_claim_id,priority:1;index:idx_inbox_delivery_user_popup_id,priority:1" json:"userId"`
	MessageId        uint64 `gorm:"column:message_id;not null;default:0;index:idx_inbox_delivery_message" json:"messageId"`
	MessageVersionId uint64 `gorm:"column:message_version_id;not null;default:0;index:idx_inbox_delivery_message_version" json:"messageVersionId"`
	CampaignId       uint64 `gorm:"column:campaign_id;not null;default:0;index:idx_inbox_delivery_campaign_status,priority:1" json:"campaignId"` // trigger 投递同样写入真实 campaign_id（收件人排查 / 领取率统计）
	CampaignRunId    uint64 `gorm:"column:campaign_run_id;not null;default:0;index:idx_inbox_delivery_campaign_run" json:"campaignRunId"`
	SourceType       string `gorm:"column:source_type;type:varchar(16);not null;default:'campaign'" json:"sourceType"`
	TriggerEvent     string `gorm:"column:trigger_event;type:varchar(32);not null;default:''" json:"triggerEvent,omitempty"`
	TriggerEventId   string `gorm:"column:trigger_event_id;type:varchar(128);not null;default:''" json:"triggerEventId,omitempty"`
	// dedupe_key 必须非空（chk_inbox_delivery_dedupe_key）：GORM 会把 Go 零值 '' 写进
	// INSERT，若只靠唯一索引，第一次漏传会静默写入、第二次才撞唯一键，而本仓库把
	// gorm.ErrDuplicatedKey 当作「已投递」的幂等成功，收件人 B..N 会被静默跳过。
	DedupeKey         string     `gorm:"column:dedupe_key;type:varchar(255);not null;uniqueIndex:uniq_inbox_delivery_dedupe_key;check:chk_inbox_delivery_dedupe_key,dedupe_key <> ''" json:"dedupeKey"`
	Status            string     `gorm:"column:status;type:varchar(16);not null;default:'delivered';index:idx_inbox_delivery_campaign_status,priority:2" json:"status"`
	SuppressionReason string     `gorm:"column:suppression_reason;type:varchar(64);not null;default:''" json:"suppressionReason,omitempty"`
	IsRead            bool       `gorm:"column:is_read;type:boolean;not null;default:false;index:idx_inbox_delivery_user_read_id,priority:2" json:"isRead"`
	ReadAt            *time.Time `gorm:"column:read_at;type:timestamp" json:"readAt,omitempty"`
	IsArchived        bool       `gorm:"column:is_archived;type:boolean;not null;default:false;index:idx_inbox_delivery_user_archived_id,priority:2" json:"isArchived"`
	ArchivedAt        *time.Time `gorm:"column:archived_at;type:timestamp" json:"archivedAt,omitempty"`
	PopupState        string     `gorm:"column:popup_state;type:varchar(16);not null;default:'pending';index:idx_inbox_delivery_user_popup_id,priority:2" json:"popupState"`
	PopupSnoozeUntil  *time.Time `gorm:"column:popup_snooze_until;type:timestamp" json:"popupSnoozeUntil,omitempty"`
	PopupDecidedAt    *time.Time `gorm:"column:popup_decided_at;type:timestamp" json:"popupDecidedAt,omitempty"`
	ClaimState        string     `gorm:"column:claim_state;type:varchar(16);not null;default:'none';index:idx_inbox_delivery_user_claim_id,priority:2" json:"claimState"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime;<-:create" json:"createdAt"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (itself *DeliveryEntity) TableName() string {
	return deliveryTableName
}

package inboxmail

import "time"

// Claim 行状态：pending 已受理待 handler 执行；granted 奖励已发放；
// already_owned 用户已拥有该奖励——成功终态，不得重复创建（徽章场景）；
// failed 执行失败（达到重试上限或不可重试前的失败态）；expired 附件已过期且未成功
// 领取。成功终态只有 granted / already_owned；failed / expired 不再自行变化，但仍需
// #778 的扫描处理，因此用 IsSuccessfulClaimStatus / IsRetryableClaimStatus 分类，
// 不再用含混的「终态」概念。
const (
	ClaimStatusPending      = "pending"
	ClaimStatusGranted      = "granted"
	ClaimStatusAlreadyOwned = "already_owned"
	ClaimStatusFailed       = "failed"
	ClaimStatusExpired      = "expired"
)

// IsSuccessfulClaimStatus 报告 claim 是否已达成目的（granted / already_owned）。
// 这是领取流程的「跳过，已完成」过滤器：成功项不得重复创建 claim 行。
// failed / expired 也是不再自行变化的状态，但不属于成功，不能用本函数跳过。
func IsSuccessfulClaimStatus(status string) bool {
	switch status {
	case ClaimStatusGranted, ClaimStatusAlreadyOwned:
		return true
	default:
		return false
	}
}

// IsRetryableClaimStatus 报告 #778 的重试扫描是否需要重新处理该 claim：
// pending（尚未执行，继续跑）、failed（上次执行失败）、expired（附件过期而领取未完成，
// 需由扫描决定补发还是最终失败）。expired 明确归入重试集合：过期只是附件的属性，
// 不代表 claim 不可再推进。
// granted / already_owned 是成功，永不重试；未知状态也不重试（宁可漏扫也不重复发奖）。
func IsRetryableClaimStatus(status string) bool {
	switch status {
	case ClaimStatusPending, ClaimStatusFailed, ClaimStatusExpired:
		return true
	default:
		return false
	}
}

// ClaimEntity 是一次附件领取事实，受双重幂等保护：
//   - UNIQUE (delivery_id, attachment_id)：同一次投递的同一附件只能有一行；
//   - UNIQUE source_key：与前者同粒度（key = inbox:<delivery>:<attachment>，见
//     ClaimSourceKey），跨重试/跨端并发可直接 upsert；source_key 非空由
//     chk_inbox_claim_source_key 兜底（空串不是合法的领取事实）。
//
// handler 与 campaign_id 冗余存放：前者供重试调度直接分发，后者供 Campaign
// 领取率统计，避免额外的 join。
//
// status 是逐 claim 的真实状态（pending/granted/already_owned/failed/expired）；
// DeliveryEntity.claim_state 是按 Delivery 聚合的展示投影，两者独立更新。
type ClaimEntity struct {
	Id           uint64     `gorm:"primaryKey;column:id;autoIncrement;not null;index:idx_inbox_claim_user_id,priority:2" json:"id"`
	DeliveryId   uint64     `gorm:"column:delivery_id;not null;default:0;uniqueIndex:uniq_inbox_claim_delivery_attachment,priority:1" json:"deliveryId"`
	AttachmentId uint64     `gorm:"column:attachment_id;not null;default:0;uniqueIndex:uniq_inbox_claim_delivery_attachment,priority:2" json:"attachmentId"`
	UserId       uint64     `gorm:"column:user_id;not null;default:0;index:idx_inbox_claim_user_id,priority:1" json:"userId"`
	CampaignId   uint64     `gorm:"column:campaign_id;not null;default:0;index:idx_inbox_claim_campaign" json:"campaignId"`
	Handler      string     `gorm:"column:handler;type:varchar(32);not null;default:''" json:"handler"`
	SourceKey    string     `gorm:"column:source_key;type:varchar(255);not null;uniqueIndex:uniq_inbox_claim_source_key;check:chk_inbox_claim_source_key,source_key <> ''" json:"sourceKey"`
	Status       string     `gorm:"column:status;type:varchar(16);not null;default:'pending';index:idx_inbox_claim_status" json:"status"`
	ClaimedAt    time.Time  `gorm:"column:claimed_at;autoCreateTime;<-:create" json:"claimedAt"`
	GrantedAt    *time.Time `gorm:"column:granted_at;type:timestamp" json:"grantedAt,omitempty"`
	RetryCount   int        `gorm:"column:retry_count;not null;default:0" json:"retryCount"`
	LastError    string     `gorm:"column:last_error;type:varchar(512);not null;default:''" json:"lastError"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime;<-:create" json:"createdAt"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (itself *ClaimEntity) TableName() string {
	return claimTableName
}

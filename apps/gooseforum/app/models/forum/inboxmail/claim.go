package inboxmail

import "time"

// Claim 状态：pending 已受理待 handler 执行；granted 奖励已发放；
// already_owned 用户已拥有该奖励——成功终态，不得重复创建（徽章场景）；
// failed 达到重试上限后的失败终态。
const (
	ClaimStatusPending      = "pending"
	ClaimStatusGranted      = "granted"
	ClaimStatusAlreadyOwned = "already_owned"
	ClaimStatusFailed       = "failed"
)

// IsTerminalClaimStatus 报告该状态是否已结束（成功或失败），失败态可被 #778 的重试
// 扫描重新领取。
func IsTerminalClaimStatus(status string) bool {
	switch status {
	case ClaimStatusGranted, ClaimStatusAlreadyOwned, ClaimStatusFailed:
		return true
	default:
		return false
	}
}

// ClaimEntity 是一次附件领取事实，受双重幂等保护：
//   - UNIQUE (delivery_id, attachment_id)：同一次投递的同一附件只能有一行；
//   - UNIQUE source_key：Reward Handler 层幂等（例如同一徽章 code 对同一 delivery
//     不会重复授予），跨重试/跨端并发安全。
//
// handler 与 campaign_id 冗余存放：前者供重试调度直接分发，后者供 Campaign
// 领取率统计，避免额外的 join。
type ClaimEntity struct {
	Id           uint64     `gorm:"primaryKey;column:id;autoIncrement;not null;index:idx_inbox_claim_user_id,priority:2" json:"id"`
	DeliveryId   uint64     `gorm:"column:delivery_id;not null;default:0;uniqueIndex:uniq_inbox_claim_delivery_attachment,priority:1" json:"deliveryId"`
	AttachmentId uint64     `gorm:"column:attachment_id;not null;default:0;uniqueIndex:uniq_inbox_claim_delivery_attachment,priority:2" json:"attachmentId"`
	UserId       uint64     `gorm:"column:user_id;not null;default:0;index:idx_inbox_claim_user_id,priority:1" json:"userId"`
	CampaignId   uint64     `gorm:"column:campaign_id;not null;default:0;index:idx_inbox_claim_campaign" json:"campaignId"`
	Handler      string     `gorm:"column:handler;type:varchar(32);not null;default:''" json:"handler"`
	SourceKey    string     `gorm:"column:source_key;type:varchar(255);not null;default:'';uniqueIndex:uniq_inbox_claim_source_key" json:"sourceKey"`
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

package moderationAppeals

import "time"

const (
	Pending  = "pending"
	Approved = "approved"
	Rejected = "rejected"
	Stale    = "stale"
)

// Entity is the durable submission and decision audit; identity is never exported.
// Candidate content remains in post_revisions and inherits its deletion guards.
type Entity struct {
	OriginalReviewReason  string     `gorm:"type:text" json:"-"`
	OriginalReviewActorId uint64     `gorm:"not null;default:0" json:"-"`
	OriginalReviewedAt    *time.Time `json:"-"`
	Id                    uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	PostId                uint64     `gorm:"not null;index" json:"-"`
	RevisionId            uint64     `gorm:"not null;uniqueIndex:uniq_moderation_appeal_revision" json:"revisionId"`
	Reason                string     `gorm:"type:text;not null" json:"reason"`
	Status                string     `gorm:"type:varchar(16);not null;default:pending;index:idx_moderation_appeal_status_id,priority:1" json:"status"`
	ReviewReason          string     `gorm:"type:text" json:"reviewReason"`
	ReviewActorId         uint64     `gorm:"not null;default:0" json:"-"`
	CreatedAt             time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	ReviewedAt            *time.Time `json:"reviewedAt"`
}

func (*Entity) TableName() string { return "moderation_appeals" }

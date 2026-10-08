// Package anonymousIdentity owns persistent public personas and their private bindings.
package anonymousIdentity

import (
	"gorm.io/gorm"
	"time"
)

type Persona struct {
	UID                   string    `gorm:"primaryKey;type:varchar(32)" json:"publicUid"`
	Name                  string    `gorm:"type:text;not null" json:"name"`
	AvatarSeed            string    `gorm:"type:varchar(32);not null" json:"-"`
	NameSelectedAt        time.Time `json:"-"`
	NameChangeAvailableAt time.Time `json:"-"`
	Disabled              bool      `gorm:"not null;default:false" json:"-"`
	GovernanceDisabled    bool      `gorm:"not null;default:false" json:"-"`
	ShowContent           bool      `gorm:"not null;default:true" json:"-"`
}

func (Persona) TableName() string { return "anonymous_personas" }

// Bindings and reveal audits survive account closure, per the approved retention policy.
// Never serialize them into ordinary admin, moderation or public responses.
type Binding struct {
	OwnerID        uint64 `gorm:"primaryKey;autoIncrement:false" json:"-"`
	PersonaUID     string `gorm:"type:varchar(32);not null;uniqueIndex" json:"-"`
	SelectionBatch string `gorm:"type:varchar(32);not null" json:"-"`
	SelectionIndex int    `gorm:"not null" json:"-"`
}

func (Binding) TableName() string { return "anonymous_bindings" }

type Quota struct {
	OwnerID uint64 `gorm:"primaryKey;autoIncrement:false" json:"-"`
	Day     string `gorm:"primaryKey;type:varchar(10)" json:"-"`
	Used    int    `gorm:"not null;default:0" json:"-"`
}

func (Quota) TableName() string { return "anonymous_name_quotas" }

type Batch struct {
	ID         string    `gorm:"primaryKey;type:varchar(32)" json:"id"`
	OwnerID    uint64    `gorm:"not null;uniqueIndex:uniq_anonymous_request,priority:1" json:"-"`
	Day        string    `gorm:"type:varchar(10);not null;uniqueIndex:uniq_anonymous_request,priority:2;index" json:"day"`
	RequestKey string    `gorm:"type:varchar(128);not null;uniqueIndex:uniq_anonymous_request,priority:3" json:"-"`
	Words      []string  `gorm:"type:text;serializer:json;not null" json:"words"`
	ExpiresAt  time.Time `gorm:"index" json:"expiresAt"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (Batch) TableName() string { return "anonymous_name_batches" }

type RevealAudit struct {
	Action     string    `gorm:"type:varchar(32);not null;default:reveal" json:"-"`
	ID         uint64    `gorm:"primaryKey" json:"-"`
	ActorID    uint64    `gorm:"not null" json:"-"`
	PersonaUID string    `gorm:"type:varchar(32);not null;index" json:"-"`
	OwnerID    uint64    `gorm:"not null" json:"-"`
	Reason     string    `gorm:"type:text;not null" json:"-"`
	TraceID    string    `gorm:"type:varchar(128);not null" json:"-"`
	CreatedAt  time.Time `json:"-"`
}

func (RevealAudit) TableName() string { return "anonymous_reveal_audits" }

func GetMap(conn *gorm.DB, uids []string) (map[string]Persona, error) {
	result := make(map[string]Persona)
	if len(uids) == 0 {
		return result, nil
	}
	var rows []Persona
	if err := conn.Where("uid IN ?", uids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.UID] = row
	}
	return result, nil
}

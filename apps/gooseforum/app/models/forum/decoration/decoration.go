package decoration

import "time"

// Definition stores administrator-managed styles; colors and asset paths are
// validated by decorationservice before they can reach a renderer.
type Definition struct {
	AssetScale  float64  `gorm:"not null;default:1.25" json:"assetScale"`
	AssetURL    string   `gorm:"size:160;not null;default:''" json:"assetUrl,omitempty"`
	Code        string   `gorm:"primaryKey;size:64" json:"code"`
	Name        string   `gorm:"size:80;not null" json:"name"`
	Kind        string   `gorm:"size:24;not null" json:"kind"`
	LightColors []string `gorm:"type:json;serializer:json" json:"lightColors"`
	DarkColors  []string `gorm:"type:json;serializer:json" json:"darkColors"`
	IsEnabled   bool     `gorm:"not null" json:"isEnabled"`
	SortOrder   int      `gorm:"not null;default:0" json:"sortOrder"`
}

func (Definition) TableName() string { return "decoration_definitions" }

type Entitlement struct {
	UserID    uint64 `gorm:"primaryKey;autoIncrement:false"`
	Code      string `gorm:"primaryKey;size:64"`
	Source    string `gorm:"primaryKey;size:24;not null"`
	SourceKey string `gorm:"size:128;not null"`
	GrantedAt time.Time
	RevokedAt *time.Time
}

func (Entitlement) TableName() string { return "user_decorations" }

type Equipment struct {
	UserID        uint64 `gorm:"primaryKey;autoIncrement:false"`
	AvatarFrame   string `gorm:"size:64;not null;default:''" json:"avatarFrame"`
	NicknameStyle string `gorm:"size:64;not null;default:''" json:"nicknameStyle"`
}

func (Equipment) TableName() string { return "decoration_equipment" }

// Audit is also the durable idempotency receipt. Retried grants cannot revive
// an entitlement after a later revoke.
type Audit struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	RequestKey string    `gorm:"size:128;not null;uniqueIndex:uniq_decoration_request" json:"requestKey"`
	UserID     uint64    `gorm:"not null;index" json:"userId"`
	Code       string    `gorm:"size:64;not null" json:"code"`
	Action     string    `gorm:"size:24;not null" json:"action"`
	Source     string    `gorm:"size:24;not null" json:"source"`
	ActorID    uint64    `json:"actorId"`
	Reason     string    `gorm:"size:512;not null" json:"reason"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (Audit) TableName() string { return "decoration_audit" }

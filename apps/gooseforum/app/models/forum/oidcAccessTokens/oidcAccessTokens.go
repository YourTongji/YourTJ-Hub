// Package oidcAccessTokens persists opaque access tokens issued by the
// built-in OIDC provider. Only the token ID, subject and metadata are stored;
// the raw bearer token is never persisted.
package oidcAccessTokens

import "time"

const tableName = "oidc_access_tokens"

// Entity represents one issued opaque access token row.
type Entity struct {
	Id           uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	TokenId      string    `gorm:"column:token_id;type:varchar(128);not null;default:'';index:idx_token_id,unique" json:"tokenId"` // token ID (inside the encrypted bearer token)
	Subject      uint64    `gorm:"column:subject;not null;default:0;index" json:"subject"`                                         // forum user id
	ClientId     string    `gorm:"column:client_id;type:varchar(128);not null;default:'';index" json:"clientId"`                   // OIDC client id
	Scopes       string    `gorm:"column:scopes;type:varchar(512);not null;default:'';" json:"scopes"`                             // space-delimited scopes
	TokenVersion uint64    `gorm:"column:token_version;not null;default:0;" json:"tokenVersion"`                                   // users.token_version at issue time
	ExpiresAt    time.Time `gorm:"column:expires_at;index;not null;" json:"expiresAt"`                                             // token lifetime
	Revoked      bool      `gorm:"column:revoked;not null;default:false;" json:"revoked"`                                          // explicit revocation
	CreatedAt    time.Time `gorm:"column:created_at;index;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}

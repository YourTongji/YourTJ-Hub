// Package oidcAuthRequests stores in-flight OIDC authorization requests for
// the built-in OIDC provider (authorization code flow, PKCE enforced).
package oidcAuthRequests

import "time"

const tableName = "oidc_auth_requests"

// Entity persists an authorization request between the /authorize entry and
// the login completion bridge. The authorization code is stored encrypted
// (see oidcservice); the raw bearer token is never persisted.
type Entity struct {
	Id             uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	RequestId      string    `gorm:"column:request_id;type:varchar(128);not null;default:'';index:idx_request_id,unique" json:"requestId"` // auth request ID (redirect login bridge key)
	ClientId       string    `gorm:"column:client_id;type:varchar(128);not null;default:'';index" json:"clientId"`                         // OIDC client id
	Scopes         string    `gorm:"column:scopes;type:varchar(512);not null;default:'';" json:"scopes"`                                   // space-delimited scopes
	RedirectUri    string    `gorm:"column:redirect_uri;type:varchar(1024);not null;default:'';" json:"redirectUri"`                       // registered redirect URI
	State          string    `gorm:"column:state;type:varchar(512);not null;default:'';" json:"state"`                                     // client state, echoed back
	Nonce          string    `gorm:"column:nonce;type:varchar(512);not null;default:'';" json:"nonce"`                                     // client nonce, bound to id_token
	ResponseType   string    `gorm:"column:response_type;type:varchar(32);not null;default:'';" json:"responseType"`                       // OIDC response_type (code only)
	CodeChallenge  string    `gorm:"column:code_challenge;type:varchar(512);not null;default:'';" json:"codeChallenge"`                    // PKCE S256 challenge
	AuthCode       string    `gorm:"column:auth_code;type:varchar(1024);not null;default:'';" json:"-"`                                    // encrypted authorization code (single-use)
	Subject        uint64    `gorm:"column:subject;not null;default:0;" json:"subject"`                                                    // authenticated forum user id
	BrowserBinding string    `gorm:"column:browser_binding;type:varchar(64);not null;default:'';" json:"-"`                                // SHA-256 hash of the browser-binding cookie value (never the raw value)
	AuthTime       time.Time `gorm:"column:auth_time;" json:"authTime"`                                                                    // time the user was authenticated
	Done           bool      `gorm:"column:done;not null;default:false;" json:"done"`                                                      // login completed, code issued
	Used           bool      `gorm:"column:used;not null;default:false;index" json:"used"`                                                 // code consumed at token endpoint
	ExpiresAt      time.Time `gorm:"column:expires_at;index;not null;" json:"expiresAt"`                                                   // auth request lifetime
	CreatedAt      time.Time `gorm:"column:created_at;index;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}

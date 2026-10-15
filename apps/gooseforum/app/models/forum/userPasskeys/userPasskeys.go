package userPasskeys

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	credentialTableName = "user_passkey_credentials"
	challengeTableName  = "user_passkey_challenges"
)

// CredentialEntity stores the globally unique credential identifier and
// encrypted go-webauthn credential record. The encrypted record contains the
// public key, authenticator counter and flags, AAGUID, and transports.
type CredentialEntity struct {
	CredentialID   []byte     `gorm:"column:credential_id;primaryKey;not null" json:"-"`
	UserID         uint64     `gorm:"column:user_id;not null;index:idx_passkey_user_active" json:"-"`
	CredentialData string     `gorm:"column:credential_data;type:text;not null" json:"-"`
	Name           string     `gorm:"column:name;type:varchar(80);not null;default:''" json:"name"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime;<-:create;index" json:"createdAt"`
	LastUsedAt     *time.Time `gorm:"column:last_used_at" json:"lastUsedAt,omitempty"`
	RevokedAt      *time.Time `gorm:"column:revoked_at;index:idx_passkey_user_active" json:"-"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"-"`
}

func (CredentialEntity) TableName() string { return credentialTableName }

// ChallengeEntity stores server-side WebAuthn session state and one-use
// reauthentication proofs. IDs are hashes of opaque client tokens; neither
// ceremony state nor proof tokens are held only in process memory.
type ChallengeEntity struct {
	ID          string    `gorm:"column:id;type:varchar(64);primaryKey;not null" json:"-"`
	Kind        string    `gorm:"column:kind;type:varchar(24);not null;index:idx_passkey_challenge_kind_expiry" json:"-"`
	UserID      uint64    `gorm:"column:user_id;not null;default:0;index" json:"-"`
	SessionJti  string    `gorm:"column:session_jti;type:varchar(64);not null;default:''" json:"-"`
	SessionData string    `gorm:"column:session_data;type:text;not null;default:''" json:"-"`
	Attempts    int       `gorm:"column:attempts;not null;default:0" json:"-"`
	ExpiresAt   time.Time `gorm:"column:expires_at;not null;index:idx_passkey_challenge_kind_expiry;index:idx_passkey_challenge_expires_at" json:"-"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime;<-:create" json:"-"`
}

func (ChallengeEntity) TableName() string { return challengeTableName }

func Models() []any { return []any{&CredentialEntity{}, &ChallengeEntity{}} }

func CountActiveCredentials(tx *gorm.DB, userID uint64) (int64, error) {
	var count int64
	err := tx.Model(&CredentialEntity{}).Where("user_id = ? AND revoked_at IS NULL", userID).Count(&count).Error
	return count, err
}

func ListActiveCredentials(tx *gorm.DB, userID uint64) ([]CredentialEntity, error) {
	var rows []CredentialEntity
	err := tx.Where("user_id = ? AND revoked_at IS NULL", userID).Order("created_at DESC, credential_id ASC").Find(&rows).Error
	return rows, err
}

func FindActiveCredential(tx *gorm.DB, credentialID []byte, lock bool) (*CredentialEntity, error) {
	query := tx.Where("credential_id = ? AND revoked_at IS NULL", credentialID)
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var row CredentialEntity
	if err := query.First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func CreateCredential(tx *gorm.DB, row *CredentialEntity) error {
	var existing CredentialEntity
	err := tx.Where("credential_id = ?", row.CredentialID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tx.Create(row).Error
	}
	if err != nil {
		return err
	}
	if existing.UserID != row.UserID || existing.RevokedAt == nil {
		return gorm.ErrDuplicatedKey
	}
	return tx.Model(&existing).Updates(map[string]any{
		"credential_data": row.CredentialData,
		"name":            row.Name,
		"revoked_at":      nil,
		"created_at":      time.Now(),
		"last_used_at":    nil,
	}).Error
}

func RenameCredential(tx *gorm.DB, userID uint64, credentialID []byte, name string) (bool, error) {
	result := tx.Model(&CredentialEntity{}).
		Where("user_id = ? AND credential_id = ? AND revoked_at IS NULL", userID, credentialID).
		Update("name", name)
	return result.RowsAffected == 1, result.Error
}

func RevokeCredential(tx *gorm.DB, userID uint64, credentialID []byte, now time.Time) (bool, error) {
	result := tx.Model(&CredentialEntity{}).
		Where("user_id = ? AND credential_id = ? AND revoked_at IS NULL", userID, credentialID).
		Update("revoked_at", now)
	return result.RowsAffected == 1, result.Error
}

func SaveChallenge(tx *gorm.DB, row *ChallengeEntity) error { return tx.Create(row).Error }

func FindChallenge(tx *gorm.DB, id, kind string, lock bool) (*ChallengeEntity, error) {
	query := tx.Where("id = ? AND kind = ? AND expires_at > ?", id, kind, time.Now())
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var row ChallengeEntity
	if err := query.First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func ConsumeChallenge(tx *gorm.DB, id, kind string, now time.Time) (bool, error) {
	result := tx.Where("id = ? AND kind = ? AND expires_at > ?", id, kind, now).Delete(&ChallengeEntity{})
	return result.RowsAffected == 1, result.Error
}

func IncrementChallengeAttempt(tx *gorm.DB, id, kind string, maxAttempts int) error {
	return tx.Model(&ChallengeEntity{}).
		Where("id = ? AND kind = ? AND expires_at > ? AND attempts < ?", id, kind, time.Now(), maxAttempts).
		UpdateColumn("attempts", gorm.Expr("attempts + 1")).
		Error
}

func ConsumeReauthenticationProof(tx *gorm.DB, id string, userID uint64, sessionJti string, now time.Time) (bool, error) {
	result := tx.Where("id = ? AND kind = ? AND user_id = ? AND session_jti = ? AND expires_at > ?", id, "reauth", userID, sessionJti, now).
		Delete(&ChallengeEntity{})
	return result.RowsAffected == 1, result.Error
}

func CleanupExpired(tx *gorm.DB, now time.Time) error {
	return tx.Where("expires_at <= ?", now).Delete(&ChallengeEntity{}).Error
}

func DeleteUserState(tx *gorm.DB, userID uint64) error {
	if err := tx.Where("user_id = ?", userID).Delete(&ChallengeEntity{}).Error; err != nil {
		return err
	}
	return tx.Where("user_id = ?", userID).Delete(&CredentialEntity{}).Error
}

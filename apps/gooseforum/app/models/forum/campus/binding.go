// Package campus owns the private campus identity connection. No school payloads
// are retained: only encrypted credentials and an HMAC identity key are stored.
package campus

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Binding struct {
	UserID             uint64 `gorm:"primaryKey;autoIncrement:false"`
	IdentityKey        string `gorm:"size:64;not null;uniqueIndex"`
	Revision           string `gorm:"size:64;not null"`
	Sealed             string `gorm:"type:text;not null"`
	NeedsAuthorization bool   `gorm:"not null;default:false"`
	LeaseUntil         int64  `gorm:"not null;default:0"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (Binding) TableName() string { return "campus_identity_bindings" }

var ErrChanged = errors.New("campus_connection_changed")

type Store struct{ DB *gorm.DB }

func (s Store) Get(id uint64) (Binding, error) {
	var b Binding
	err := s.DB.First(&b, "user_id = ?", id).Error
	return b, err
}

// Replace uses database uniqueness and optimistic concurrency. created_at tracks
// the current identity binding and survives same-identity reauthorization. The old
// identity is not released until this statement commits.
func (s Store) Replace(b Binding, previous string) error {
	if previous == "" {
		return s.DB.Create(&b).Error
	}
	r := s.DB.Model(&Binding{}).Where("user_id = ? AND revision = ? AND lease_until < ?", b.UserID, previous, time.Now().Unix()).Updates(map[string]any{
		"created_at":   gorm.Expr("CASE WHEN identity_key <> ? THEN ? ELSE created_at END", b.IdentityKey, time.Now()),
		"identity_key": b.IdentityKey, "revision": b.Revision, "sealed": b.Sealed,
		"needs_authorization": false, "lease_until": 0,
	})
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected != 1 {
		return ErrChanged
	}
	return nil
}
func (s Store) Delete(id uint64, revision string) error {
	r := s.DB.Where("user_id = ? AND revision = ?", id, revision).Delete(&Binding{})
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected != 1 {
		return ErrChanged
	}
	return nil
}
func (s Store) DeleteForUser(id uint64) error {
	return s.DB.Where("user_id = ?", id).Delete(&Binding{}).Error
}
func (s Store) Acquire(b Binding) error {
	r := s.DB.Model(&Binding{}).Where("user_id = ? AND revision = ? AND sealed = ? AND lease_until < ?", b.UserID, b.Revision, b.Sealed, time.Now().Unix()).Update("lease_until", time.Now().Add(time.Minute).Unix())
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected != 1 {
		return ErrChanged
	}
	return nil
}
func (s Store) Finish(b Binding, sealed string, reauth bool) error {
	r := s.DB.Model(&Binding{}).Where("user_id = ? AND revision = ? AND sealed = ?", b.UserID, b.Revision, b.Sealed).Updates(map[string]any{"sealed": sealed, "revision": b.Revision, "needs_authorization": reauth, "lease_until": 0})
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected != 1 {
		return ErrChanged
	}
	return nil
}

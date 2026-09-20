package campus

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// IdentityReservation prevents repeated automatic signup after unlink, replacement
// or account closure. It retains only an HMAC: no user association or credentials.
// Explicit binding to an existing account is still permitted.
type IdentityReservation struct {
	IdentityKey string `gorm:"size:64;primaryKey"`
}

func (IdentityReservation) TableName() string { return "campus_identity_reservations" }

var ErrIdentityUsed = errors.New("campus_identity_previously_bound")

// ReserveSignup must share the account-creation transaction. The unique insert
// serializes first-time claims even when there is no binding row to lock yet.
func (s Store) ReserveSignup(key string) error {
	r := s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&IdentityReservation{IdentityKey: key})
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected != 1 {
		return ErrIdentityUsed
	}
	return nil
}

// BackfillIdentityReservations is idempotent and runs after schema migration,
// before serving requests. Never infer historical identities from mutable email.
func BackfillIdentityReservations(db *gorm.DB) error {
	return db.Exec(`INSERT INTO campus_identity_reservations (identity_key)
		SELECT identity_key FROM campus_identity_bindings WHERE identity_key <> ''
		ON CONFLICT (identity_key) DO NOTHING`).Error
}

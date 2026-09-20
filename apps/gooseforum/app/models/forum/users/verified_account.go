package users

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrSignupQuota = errors.New("signup quota reached")

// GetForAuthenticationTx reads the current account under the caller's transaction.
// Deleted users are deliberately excluded; locks serialize account state changes.
func GetForAuthenticationTx(tx *gorm.DB, id uint64) (EntityComplete, error) {
	var user EntityComplete
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, id).Error
	return user, err
}

// CreateVerifiedAccountTx reserves the private email against both current and
// pending addresses before inserting. Callers couple identity creation to this tx.
func CreateVerifiedAccountTx(tx *gorm.DB, user *EntityComplete, maxDaily int) error {
	if err := CheckEmailClaimTx(tx, user.Email, 0); err != nil {
		return err
	}
	if maxDaily >= 0 {
		now := time.Now()
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		var count int64
		if err := tx.Model(&EntityComplete{}).Where("created_at >= ?", start).Count(&count).Error; err != nil {
			return err
		}
		if count >= int64(maxDaily) {
			return ErrSignupQuota
		}
	}
	return tx.Create(user).Error
}

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
	if err := CheckSignupQuotaTx(tx, maxDaily); err != nil {
		return err
	}
	return tx.Create(user).Error
}

// CheckSignupQuotaTx serializes all self-service registration paths until their
// account transaction commits. SQLite serializes writers; a stale read cannot
// commit an over-quota insert. Closed accounts still consume today's allowance.
func CheckSignupQuotaTx(tx *gorm.DB, maxDaily int) error {
	if maxDaily >= 0 {
		if tx.Name() == "postgres" {
			// Global, transaction-scoped lock; distinct email claims share it.
			if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(0x79746a7369676e)).Error; err != nil {
				return err
			}
		}
		now := tx.NowFunc()
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		var count int64
		if err := tx.Unscoped().Model(&EntityComplete{}).Where("created_at >= ?", start).Count(&count).Error; err != nil {
			return err
		}
		if count >= int64(maxDaily) {
			return ErrSignupQuota
		}
	}
	return nil
}

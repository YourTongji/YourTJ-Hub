package users

import "gorm.io/gorm"

// LockAnonymousOwnerTx serializes persona writes across processes and rechecks
// the live account. SQLite's write lock and PostgreSQL's row lock cover the same
// quota/creation critical section; cached credentials cannot revive a closed user.
func LockAnonymousOwnerTx(tx *gorm.DB, id uint64) (EntityComplete, error) {
	var user EntityComplete
	result := tx.Model(&EntityComplete{}).Where("id = ? AND is_frozen = ? AND actor_type = ? AND anonymous_governance_blocked = ?", id, StatusNormal, ActorTypeHuman, false).UpdateColumn("id", gorm.Expr("id"))
	if result.Error != nil {
		return user, result.Error
	}
	if result.RowsAffected != 1 {
		return user, gorm.ErrRecordNotFound
	}
	err := tx.First(&user, id).Error
	return user, err
}

func LockAnonymousOwnerForReadTx(tx *gorm.DB, id uint64) (EntityComplete, error) {
	var user EntityComplete
	result := tx.Model(&EntityComplete{}).Where("id = ? AND actor_type = ?", id, ActorTypeHuman).UpdateColumn("id", gorm.Expr("id"))
	if result.Error != nil {
		return user, result.Error
	}
	if result.RowsAffected != 1 {
		return user, gorm.ErrRecordNotFound
	}
	return user, tx.First(&user, id).Error
}

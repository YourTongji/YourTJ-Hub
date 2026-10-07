package anonymousIdentity

import (
	"errors"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

// ValidateWriterTx holds the owner row until the content transaction commits.
// Existing-content callers acquire post/topic locks first, matching Agent source
// authorization's content-before-participant order.
func ValidateWriterTx(tx *gorm.DB, owner uint64, uid string) error {
	if uid == "" {
		return nil
	}
	if _, err := users.LockAnonymousOwnerTx(tx, owner); err != nil {
		return err
	}
	var binding Binding
	if err := tx.First(&binding, "owner_id = ? AND persona_uid = ?", owner, uid).Error; err != nil {
		return err
	}
	var p Persona
	if err := tx.First(&p, "uid = ?", uid).Error; err != nil {
		return err
	}
	if p.Disabled || p.GovernanceDisabled {
		return errors.New("anonymous.unavailable")
	}
	return nil
}

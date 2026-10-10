package anonymousidentityservice

import (
	"strings"
	"time"

	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

// Govern applies a scoped moderation decision. The identity restriction applies
// to its private owner as well, so switching to member cannot bypass governance.
// Self reactivation never clears this lock.
func (s Service) Govern(actor uint64, uid, reason string, disabled bool) error {
	if !ValidateReason(reason) || !ValidatePublicUID(uid) {
		return ErrUnavailable
	}
	trace, err := randomID()
	if err != nil {
		return err
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
		return governTx(tx, actor, uid, reason, disabled, trace, s.Now)
	})
}

func governTx(tx *gorm.DB, actor uint64, uid, reason string, disabled bool, trace string, now func() time.Time) error {
	var binding identity.Binding
	if err := tx.First(&binding, "persona_uid = ?", uid).Error; err != nil {
		return err
	}
	owner := binding.OwnerID
	var user users.EntityComplete
	if err := tx.Model(&user).Where("id = ?", owner).UpdateColumn("id", gorm.Expr("id")).Error; err != nil {
		return err
	}
	audit := identity.RevealAudit{ActorID: actor, PersonaUID: uid, OwnerID: owner, Reason: strings.TrimSpace(reason), TraceID: trace, CreatedAt: now(), Action: "governance.restore"}
	if disabled {
		audit.Action = "governance.disable"
	}
	if err := tx.Create(&audit).Error; err != nil {
		return err
	}
	if err := tx.Model(&identity.Persona{}).Where("uid = ?", uid).Update("governance_disabled", disabled).Error; err != nil {
		return err
	}
	return tx.Model(&users.EntityComplete{}).Where("id = ?", owner).Update("anonymous_governance_blocked", disabled).Error
}

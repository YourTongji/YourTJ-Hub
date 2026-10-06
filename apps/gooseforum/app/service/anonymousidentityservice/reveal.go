package anonymousidentityservice

import (
	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"gorm.io/gorm"
	"strings"
)

type RevealedOwner struct {
	PublicUID string `json:"publicUid"`
	UserID    uint64 `json:"userId"`
	Username  string `json:"username"`
}

func (s Service) Reveal(actor uint64, uid, reason, trace string) (RevealedOwner, error) {
	var result RevealedOwner
	if !ValidateReason(reason) || len(trace) > 128 {
		return result, ErrUnavailable
	}
	if trace == "" {
		var err error
		trace, err = randomID()
		if err != nil {
			return result, err
		}
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		current, err := users.LockAnonymousOwnerTx(tx, actor)
		if err != nil {
			return err
		}
		allowed, err := rolePermissionRs.HasExplicitTx(tx, current.RoleId, permission.RevealAnonymousIdentity.Id())
		if err != nil {
			return err
		}
		if !allowed {
			return ErrUnavailable
		}
		var binding identity.Binding
		if err := tx.First(&binding, "persona_uid = ?", uid).Error; err != nil {
			return err
		}
		var owner users.EntityComplete
		if err := tx.Unscoped().First(&owner, binding.OwnerID).Error; err != nil {
			return err
		}
		audit := identity.RevealAudit{Action: "reveal", ActorID: actor, PersonaUID: uid, OwnerID: binding.OwnerID, Reason: strings.TrimSpace(reason), TraceID: trace, CreatedAt: s.Now()}
		if err := tx.Create(&audit).Error; err != nil {
			return err
		}
		result = RevealedOwner{uid, owner.Id, owner.Username}
		return nil
	})
	if err != nil {
		return RevealedOwner{}, err
	}
	return result, nil
}

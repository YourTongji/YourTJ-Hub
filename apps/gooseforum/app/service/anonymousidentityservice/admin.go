package anonymousidentityservice

import (
	"strings"
	"time"

	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"gorm.io/gorm"
)

type AdminListQuery struct {
	Page     int    `json:"page" validate:"min=1,max=100000"`
	PageSize int    `json:"pageSize" validate:"min=1,max=50"`
	Search   string `json:"search" validate:"max=128"`
	Status   string `json:"status" validate:"oneof=all active disabled banned"`
	Reason   string `json:"reason" validate:"required,max=512"`
}

type AdminOwner struct {
	UserID   uint64 `json:"userId"`
	Username string `json:"username"`
	Closed   bool   `json:"closed"`
	Frozen   bool   `json:"frozen"`
}

type AdminIdentity struct {
	PublicPersona
	Owner              AdminOwner `json:"owner"`
	Disabled           bool       `json:"disabled"`
	GovernanceDisabled bool       `json:"governanceDisabled"`
	SelectedAt         time.Time  `json:"selectedAt"`
}

type AdminList struct {
	Items    []AdminIdentity `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
}

// Both capabilities are read from the current role inside the transaction.
// Admin's wildcard never substitutes for the explicit private-binding grant.
func authorizeAdminTx(tx *gorm.DB, actor uint64) error {
	current, err := users.LockAnonymousOwnerTx(tx, actor)
	if err != nil {
		return err
	}
	for _, capability := range []permission.Enum{permission.RevealAnonymousIdentity, permission.UserManager} {
		allowed, err := rolePermissionRs.HasExplicitTx(tx, current.RoleId, capability.Id())
		if err != nil {
			return err
		}
		if !allowed && capability == permission.UserManager {
			allowed, err = rolePermissionRs.HasExplicitTx(tx, current.RoleId, permission.Admin.Id())
			if err != nil {
				return err
			}
		}
		if !allowed {
			return ErrUnavailable
		}
	}
	return nil
}

// ListAdmin releases each page of private mappings only after its restricted
// audit commits. Closed owners remain available for historical governance.
func (s Service) ListAdmin(actor uint64, q AdminListQuery) (AdminList, error) {
	if !ValidateReason(q.Reason) || q.Page < 1 || q.Page > 100000 || q.PageSize < 1 || q.PageSize > 50 || len([]rune(q.Search)) > 128 {
		return AdminList{}, ErrUnavailable
	}
	if q.Status != "all" && q.Status != "active" && q.Status != "disabled" && q.Status != "banned" {
		return AdminList{}, ErrUnavailable
	}
	trace, err := randomID()
	if err != nil {
		return AdminList{}, err
	}
	result := AdminList{Items: []AdminIdentity{}, Page: q.Page, PageSize: q.PageSize}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := authorizeAdminTx(tx, actor); err != nil {
			return err
		}
		query := tx.Table("anonymous_personas AS p").Joins("JOIN anonymous_bindings AS b ON b.persona_uid = p.uid").Joins("JOIN users AS u ON u.id = b.owner_id")
		if search := strings.TrimSpace(q.Search); search != "" {
			like := "%" + strings.ToLower(search) + "%"
			query = query.Where("LOWER(p.name) LIKE ? OR LOWER(p.uid) LIKE ? OR LOWER(u.username) LIKE ? OR CAST(u.id AS TEXT) = ?", like, like, like, search)
		}
		switch q.Status {
		case "active":
			query = query.Where("p.disabled = ? AND p.governance_disabled = ?", false, false)
		case "disabled":
			query = query.Where("p.disabled = ? AND p.governance_disabled = ?", true, false)
		case "banned":
			query = query.Where("p.governance_disabled = ?", true)
		}
		if err := query.Count(&result.Total).Error; err != nil {
			return err
		}
		var rows []struct {
			UID, Name, Username          string
			OwnerID                      uint64
			Disabled, GovernanceDisabled bool
			NameSelectedAt               time.Time
			DeletedAt                    gorm.DeletedAt
			IsFrozen                     int8
		}
		if err := query.Select("p.uid, p.name, p.disabled, p.governance_disabled, p.name_selected_at, b.owner_id, u.username, u.deleted_at, u.is_frozen").Order("p.name_selected_at DESC, p.uid ASC").Limit(q.PageSize).Offset((q.Page - 1) * q.PageSize).Scan(&rows).Error; err != nil {
			return err
		}
		audits := make([]identity.RevealAudit, 0, len(rows))
		for _, row := range rows {
			result.Items = append(result.Items, AdminIdentity{
				PublicPersona: Public(identity.Persona{UID: row.UID, Name: row.Name}),
				Owner:         AdminOwner{UserID: row.OwnerID, Username: row.Username, Closed: row.DeletedAt.Valid, Frozen: row.IsFrozen != users.StatusNormal},
				Disabled:      row.Disabled, GovernanceDisabled: row.GovernanceDisabled, SelectedAt: row.NameSelectedAt,
			})
			audits = append(audits, identity.RevealAudit{Action: "admin.list", ActorID: actor, PersonaUID: row.UID, OwnerID: row.OwnerID, Reason: strings.TrimSpace(q.Reason), TraceID: trace, CreatedAt: s.Now()})
		}
		if len(audits) == 0 {
			audits = append(audits, identity.RevealAudit{Action: "admin.list", ActorID: actor, Reason: strings.TrimSpace(q.Reason), TraceID: trace, CreatedAt: s.Now()})
		}
		return tx.Create(&audits).Error
	})
	if err != nil {
		return AdminList{}, err
	}
	return result, nil
}

// GovernAdmin allows governance without an existing post, with the same audit
// and owner-wide publishing restriction as scoped moderation.
func (s Service) GovernAdmin(actor uint64, uid, reason string, disabled bool) (uint64, error) {
	if !ValidateReason(reason) || len(uid) != 32 {
		return 0, ErrUnavailable
	}
	var binding identity.Binding
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := authorizeAdminTx(tx, actor); err != nil {
			return err
		}
		if err := tx.First(&binding, "persona_uid = ?", uid).Error; err != nil {
			return err
		}
		s.DB = tx
		return s.Govern(actor, uid, reason, disabled)
	})
	if err != nil {
		return 0, err
	}
	return binding.OwnerID, nil
}

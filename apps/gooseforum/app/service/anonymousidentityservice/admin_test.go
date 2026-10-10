package anonymousidentityservice

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
)

func exerciseAdminIdentities(t *testing.T, s Service) {
	t.Helper()
	uid := strings.Repeat("a", 32)
	second := strings.Repeat("b", 32)
	for _, row := range []any{
		&identity.Persona{UID: uid, Name: "躲进云里的猫", AvatarSeed: "secret-seed", NameSelectedAt: s.Now()},
		&identity.Persona{UID: second, Name: "抱着松果的熊", AvatarSeed: "other-secret", Disabled: true, NameSelectedAt: s.Now()},
		&identity.Binding{OwnerID: 1, PersonaUID: uid}, &identity.Binding{OwnerID: 2, PersonaUID: second},
		&rolePermissionRs.Entity{RoleId: 3, PermissionId: permission.Admin.Id(), Effective: 1},
	} {
		if err := s.DB.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	q := AdminListQuery{Page: 1, PageSize: 1, Status: "all", Reason: "investigate reports"}
	if result, err := s.ListAdmin(3, q); err == nil || len(result.Items) != 0 || result.Total != 0 {
		t.Fatal("admin wildcard revealed a mapping", result, err)
	}
	if owner, err := s.GovernAdmin(3, uid, "investigate reports", true); err == nil || owner != 0 {
		t.Fatal("wildcard permitted governance", owner, err)
	}
	grant := rolePermissionRs.Entity{RoleId: 3, PermissionId: permission.RevealAnonymousIdentity.Id(), Effective: 1}
	if err := s.DB.Create(&grant).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.GovernAdmin(3, strings.Repeat("g", 32), "moderation decision", true); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("non-hex public uid was not rejected: %v", err)
	}
	if err := s.DB.Delete(&users.EntityComplete{}, 2).Error; err != nil {
		t.Fatal(err)
	}
	result, err := s.ListAdmin(3, q)
	if err != nil || result.Total != 2 || len(result.Items) != 1 || result.Items[0].Owner.UserID != 1 {
		t.Fatal(result, err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "secret") || strings.Contains(string(encoded), "avatarSeed") {
		t.Fatal("seed leaked", string(encoded))
	}
	var audits []identity.RevealAudit
	if err := s.DB.Find(&audits).Error; err != nil || len(audits) != 1 || audits[0].Action != "admin.list" || audits[0].OwnerID != 1 {
		t.Fatal(audits, err)
	}
	q.Page = 2
	result, err = s.ListAdmin(3, q)
	if err != nil || len(result.Items) != 1 || !result.Items[0].Owner.Closed {
		t.Fatal("closed owner missing", result, err)
	}
	q.Page, q.Search = 1, "private-owner-1"
	result, err = s.ListAdmin(3, q)
	if err != nil || result.Total != 1 || len(result.Items) != 1 {
		t.Fatal("owner search", result, err)
	}
	q.Reason = "  "
	if result, err = s.ListAdmin(3, q); err == nil || len(result.Items) != 0 {
		t.Fatal("empty reason revealed mapping", result, err)
	}
	q.Reason, q.Search = "investigate reports", ""
	if err := s.DB.Model(&users.EntityComplete{}).Where("id = 1").Update("is_frozen", 1).Error; err != nil {
		t.Fatal(err)
	}
	for _, disabled := range []bool{true, false} {
		owner, err := s.GovernAdmin(3, uid, "moderation decision", disabled)
		if err != nil || owner != 1 {
			t.Fatal(owner, err)
		}
		var persona identity.Persona
		var user users.EntityComplete
		s.DB.First(&persona, "uid = ?", uid)
		s.DB.First(&user, 1)
		if persona.GovernanceDisabled != disabled || user.AnonymousGovernanceBlocked != disabled || user.IsFrozen != 1 {
			t.Fatal("governance changed independent freeze", persona, user)
		}
	}
	if err := s.DB.Model(&grant).Update("effective", 0).Error; err != nil {
		t.Fatal(err)
	}
	if result, err := s.ListAdmin(3, q); err == nil || len(result.Items) != 0 {
		t.Fatal("revoked grant revealed", result, err)
	}
	if err := s.DB.Model(&grant).Update("effective", 1).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Migrator().DropTable(&identity.RevealAudit{}); err != nil {
		t.Fatal(err)
	}
	if result, err := s.ListAdmin(3, q); err == nil || len(result.Items) != 0 || result.Total != 0 {
		t.Fatal("audit failure leaked mappings", result, err)
	}
	if _, err := s.GovernAdmin(3, uid, "moderation decision", true); err == nil {
		t.Fatal("audit failure permitted governance")
	}
	var persona identity.Persona
	if err := s.DB.First(&persona, "uid = ?", uid).Error; err != nil || persona.GovernanceDisabled {
		t.Fatal("audit failure changed status", persona, err)
	}
}

func TestAdminIdentitiesAreExplicitAuditedAndAtomic(t *testing.T) {
	exerciseAdminIdentities(t, setup(t, false))
}
func TestPostgreSQLAdminIdentitiesAreExplicitAuditedAndAtomic(t *testing.T) {
	exerciseAdminIdentities(t, setup(t, true))
}

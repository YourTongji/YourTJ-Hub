package routes

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/anonymousidentityservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/gin-gonic/gin"
)

func TestAdminAnonymousIdentityHTTPContract(t *testing.T) {
	conn, _ := setupAccountContractTest(t)
	if err := conn.AutoMigrate(&identity.Persona{}, &identity.Binding{}, &identity.RevealAudit{}, &rolePermissionRs.Entity{}); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	RegisterByGin(router)
	owner := createHTTPContractUser(t, conn, contractTestID())
	actor := createHTTPContractUser(t, conn, contractTestID())
	actor.RoleId = contractTestID()
	if err := conn.Model(actor).Update("role_id", actor.RoleId).Error; err != nil {
		t.Fatal(err)
	}
	uid := strings.Repeat("a", 32)
	for _, row := range []any{&identity.Persona{UID: uid, Name: "躲进云里的猫", AvatarSeed: "private-seed", NameSelectedAt: time.Date(2026, 10, 7, 8, 0, 0, 0, time.UTC)}, &identity.Binding{OwnerID: owner.Id, PersonaUID: uid}, &rolePermissionRs.Entity{RoleId: actor.RoleId, PermissionId: permission.Admin.Id(), Effective: 1}} {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	permission.InvalidateRole(actor.RoleId)
	token := contractSessionToken(t, actor)
	list := "/api/admin/anonymous-identities/list"
	govern := "/api/admin/anonymous-identities/govern"
	body := `{"page":1,"pageSize":10,"search":"","status":"all","reason":"review reports"}`
	for _, path := range []string{list, govern} {
		if rec := serveAuthSecurityJSON(router, http.MethodPost, path, body, ""); rec.Code != http.StatusUnauthorized {
			t.Fatal("unauthenticated", rec.Code, rec.Body.String())
		}
		if rec := serveJSON(router, path, body, token); rec.Code != http.StatusForbidden || rec.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("wildcard accessed private endpoint", rec.Code, rec.Body.String())
		}
	}
	grant := rolePermissionRs.Entity{RoleId: actor.RoleId, PermissionId: permission.RevealAnonymousIdentity.Id(), Effective: 1}
	if err := conn.Create(&grant).Error; err != nil {
		t.Fatal(err)
	}
	permission.InvalidateRole(actor.RoleId)
	rec := serveJSON(router, list, body, token)
	envelope := decodeContractEnvelope(t, rec)
	fixture := contractFixture(t, "admin-anonymous-list-success.json")
	var expected anonymousidentityservice.AdminList
	if err := json.Unmarshal(fixture.Result, &expected); err != nil {
		t.Fatal(err)
	}
	expected.Items[0].Owner.UserID = owner.Id
	expected.Items[0].Owner.Username = owner.Username
	encoded, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	fixture.Result = encoded
	assertFixtureEnvelope(t, envelope, fixture)
	var result anonymousidentityservice.AdminList
	if err := json.Unmarshal(envelope.Result, &result); err != nil || len(result.Items) != 1 || result.Items[0].Owner.UserID != owner.Id || rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal(result, err, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "private-seed") {
		t.Fatal("seed leaked")
	}
	if rec = serveJSON(router, govern, `{"publicUid":"`+uid+`","reason":"moderate reports"}`, token); decodeContractEnvelope(t, rec).Code == 0 {
		t.Fatal("missing status accepted", rec.Code, rec.Body.String())
	}
	for _, status := range []string{"true", "false"} {
		rec = serveJSON(router, govern, `{"publicUid":"`+uid+`","disabled":`+status+`,"reason":"moderate reports"}`, token)
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "anonymous-govern-success.json"))
	}
	if err := conn.Migrator().DropTable(&identity.RevealAudit{}); err != nil {
		t.Fatal(err)
	}
	envelope = decodeContractEnvelope(t, serveJSON(router, list, body, token))
	if envelope.Code == 0 || strings.Contains(string(envelope.Result), owner.Username) {
		t.Fatal("audit failure released mapping", envelope)
	}
}

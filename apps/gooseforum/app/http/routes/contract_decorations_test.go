package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/decoration"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/decorationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func setupDecorationHTTP(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(&decoration.Definition{}, &decoration.Entitlement{}, &decoration.Equipment{}, &decoration.Audit{}, &rolePermissionRs.Entity{}); err != nil {
		t.Fatal(err)
	}
	for _, model := range []any{&decoration.Definition{}, &decoration.Entitlement{}, &decoration.Equipment{}, &decoration.Audit{}} {
		if err := conn.Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatal(err)
		}
	}
	self := router.Group("/api", middleware.CSRFProtection, middleware.JWTAuthCheck)
	self.GET("user/decorations", UpButterReq(api.MyDecorations))
	self.POST("user/decorations/equip", middleware.CheckWritableAccount, UpLimitedJsonReq(4096, api.EquipDecorations))
	admin := self.Group("admin", middleware.CheckWritableAccount)
	admin.GET("decorations", middleware.CheckPermission(permission.SiteManager), UpButterReq(api.DecorationList))
	admin.POST("decoration-save", middleware.CheckPermission(permission.SiteManager), UpLimitedJsonReq(4096, api.SaveDecoration))
	admin.GET("user-decorations", middleware.CheckPermission(permission.UserManager), UpQueryReq(api.UserDecorations))
	admin.POST("decoration-grant", middleware.CheckPermission(permission.UserManager), UpLimitedJsonReq(4096, api.GrantDecoration))
	admin.POST("decoration-revoke", middleware.CheckPermission(permission.UserManager), UpLimitedJsonReq(4096, api.RevokeDecoration))
	return conn, router
}

func TestDecorationEquipmentHTTPContract(t *testing.T) {
	conn, router := setupDecorationHTTP(t)
	owner := createHTTPContractUser(t, conn, contractTestID())
	other := createHTTPContractUser(t, conn, contractTestID())
	manager := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, manager.Id, permission.UserManager)
	ownerToken, adminToken := contractSessionToken(t, owner), contractSessionToken(t, manager)
	call := func(method, path, body, token, fixture string) {
		t.Helper()
		rec := serveAuthSecurityJSON(router, method, path, body, token)
		if rec.Code != 200 {
			t.Fatalf("%s %s: %d %s", method, path, rec.Code, rec.Body.String())
		}
		if fixture != "" {
			assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, fixture))
		} else if decodeContractEnvelope(t, rec).Code != 0 {
			t.Fatal(rec.Body.String())
		}
	}
	call("GET", "/api/user/decorations", "", ownerToken, "decoration-state-empty.json")
	call("GET", fmt.Sprintf("/api/admin/user-decorations?userId=%d", owner.Id), "", adminToken, "decoration-admin-state.json")
	configured := []decoration.Definition{
		{Code: "frame-lantern", Name: "Lantern frame", Kind: decorationservice.AvatarFrame, LightColors: []string{"#166534", "#0D9488"}, DarkColors: []string{"#86EFAC", "#2DD4BF"}, IsEnabled: true, SortOrder: 10, AssetURL: "/static/decorations/frame-lantern.png"},
		{Code: "name-moss", Name: "Moss name", Kind: decorationservice.NicknameStyle, LightColors: []string{"#065F46", "#115E59"}, DarkColors: []string{"#6EE7B7", "#99F6E4"}, IsEnabled: true, SortOrder: 20},
	}
	if err := conn.Create(&configured).Error; err != nil {
		t.Fatal(err)
	}
	call("POST", "/api/user/decorations/equip", `{"avatarFrame":"frame-lantern","nicknameStyle":""}`, ownerToken, "permission-denied.json")
	grantBody := fmt.Sprintf(`{"userId":%d,"code":"frame-lantern","reason":"award","requestKey":"contract-grant-frame"}`, owner.Id)
	call("POST", "/api/admin/decoration-grant", grantBody, adminToken, "decoration-state-owned.json")
	call("POST", "/api/admin/decoration-grant", grantBody, adminToken, "decoration-state-owned.json")
	rec := serveAuthSecurityJSON(router, "POST", "/api/admin/decoration-grant", fmt.Sprintf(`{"userId":%d,"code":"name-moss","reason":"award","requestKey":"contract-grant-name"}`, owner.Id), adminToken)
	if envelope := decodeContractEnvelope(t, rec); envelope.Code != 0 {
		t.Fatal(rec.Body.String())
	}
	call("POST", "/api/user/decorations/equip", fmt.Sprintf(`{"userId":%d,"avatarFrame":"frame-lantern","nicknameStyle":"name-moss"}`, other.Id), ownerToken, "decoration-state-equipped.json")
	call("GET", "/api/user/decorations", "", contractSessionToken(t, other), "decoration-state-unowned.json")
	call("POST", "/api/user/decorations/equip", `{}`, ownerToken, "invalid-params.json")
	call("POST", "/api/user/decorations/equip", `{"avatarFrame":"name-moss","nicknameStyle":"frame-lantern"}`, ownerToken, "permission-denied.json")
	call("GET", "/api/user/decorations", "", ownerToken, "decoration-state-equipped.json")
	if rec := serveAuthSecurityJSON(router, "POST", "/api/user/decorations/equip", `{`, ownerToken); rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed body: %d %s", rec.Code, rec.Body.String())
	}
	if rec := serveAuthSecurityJSON(router, "POST", "/api/user/decorations/equip", strings.Repeat("x", 4097), ownerToken); rec.Code != http.StatusBadRequest {
		t.Fatalf("large body: %d", rec.Code)
	}
	// A cookie write from another origin must be rejected before mutation.
	if rec := serveAuthSecurityJSONWithCookie(router, "POST", "/api/user/decorations/equip", `{"avatarFrame":"","nicknameStyle":""}`, ownerToken); rec.Code != http.StatusForbidden {
		t.Fatalf("cookie without CSRF: %d %s", rec.Code, rec.Body.String())
	}
	call("GET", "/api/user/decorations", "", ownerToken, "decoration-state-equipped.json")
	call("POST", "/api/admin/decoration-revoke", fmt.Sprintf(`{"userId":%d,"code":"frame-lantern","reason":"undo","requestKey":"contract-revoke-frame"}`, owner.Id), adminToken, "")
	call("POST", "/api/admin/decoration-grant", grantBody, adminToken, "")
	rec = serveAuthSecurityJSON(router, "GET", fmt.Sprintf("/api/admin/user-decorations?userId=%d", owner.Id), "", adminToken)
	var state decorationservice.AdminState
	if err := json.Unmarshal(decodeContractEnvelope(t, rec).Result, &state); err != nil {
		t.Fatal(err)
	}
	if len(state.OwnedCodes) != 1 || state.OwnedCodes[0] != "name-moss" || state.Equipped.AvatarFrame != "" || len(state.Audits) != 4 {
		t.Fatalf("revoke/replay admin state=%+v", state)
	}

}

func TestDecorationAdminPermissionsAndDefinitionHTTP(t *testing.T) {
	conn, router := setupDecorationHTTP(t)
	ordinary := createHTTPContractUser(t, conn, contractTestID())
	userManager := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, userManager.Id, permission.UserManager)
	siteManager := createContractSiteManager(t, conn)
	for _, scenario := range []struct {
		method, path, body, token string
		status                    int
	}{
		{"GET", "/api/user/decorations", "", "", 401},
		{"GET", "/api/admin/decorations", "", contractSessionToken(t, ordinary), 403},
		{"GET", "/api/admin/decorations", "", contractSessionToken(t, userManager), 403},
		{"GET", fmt.Sprintf("/api/admin/user-decorations?userId=%d", ordinary.Id), "", contractSessionToken(t, siteManager), 403},
		{"POST", "/api/admin/decoration-grant", `{}`, contractSessionToken(t, ordinary), 403},
	} {
		rec := serveAuthSecurityJSON(router, scenario.method, scenario.path, scenario.body, scenario.token)
		if rec.Code != scenario.status {
			t.Fatalf("%s: %d %s", scenario.path, rec.Code, rec.Body.String())
		}
	}
	rec := serveAuthSecurityJSON(router, "GET", "/api/admin/decorations", "", contractSessionToken(t, siteManager))
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "decoration-list-success.json"))
	style := decorationservice.Style{Code: "frame-lantern", Name: "Lantern frame", Kind: decorationservice.AvatarFrame, LightColors: []string{"#166534", "#0D9488"}, DarkColors: []string{"#86EFAC", "#2DD4BF"}, IsEnabled: true, SortOrder: 10, AssetURL: "/static/decorations/frame-lantern.png"}
	body, err := json.Marshal(style)
	if err != nil {
		t.Fatal(err)
	}
	rec = serveAuthSecurityJSON(router, "POST", "/api/admin/decoration-save", string(body), contractSessionToken(t, siteManager))
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "decoration-style-success.json"))
	style.Kind = decorationservice.NicknameStyle
	body, err = json.Marshal(style)
	if err != nil {
		t.Fatal(err)
	}
	rec = serveAuthSecurityJSON(router, "POST", "/api/admin/decoration-save", string(body), contractSessionToken(t, siteManager))
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "invalid-params.json"))
	style.Kind = decorationservice.AvatarFrame
	style.LightColors = []string{"url(javascript:alert(1))"}
	body, err = json.Marshal(style)
	if err != nil {
		t.Fatal(err)
	}
	rec = serveAuthSecurityJSON(router, "POST", "/api/admin/decoration-save", string(body), contractSessionToken(t, siteManager))
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "invalid-params.json"))
	var count int64
	conn.Model(&decoration.Audit{}).Where("action = ?", "definition").Count(&count)
	if count != 1 {
		t.Fatalf("definition audit count=%d", count)
	}
	if err := conn.Model(&users.EntityComplete{}).Where("id = ?", ordinary.Id).Update("is_frozen", users.StatusFrozen).Error; err != nil {
		t.Fatal(err)
	}
	userservice.InvalidateUserInfoCache(ordinary.Id)
	rec = serveAuthSecurityJSON(router, "POST", "/api/user/decorations/equip", `{"avatarFrame":"","nicknameStyle":""}`, contractSessionToken(t, ordinary))
	if rec.Code != 403 {
		t.Fatalf("frozen equipment write: %d", rec.Code)
	}
}

func TestDecorationFrameScaleHTTP(t *testing.T) {
	conn, router := setupDecorationHTTP(t)
	manager := createContractSiteManager(t, conn)
	token := contractSessionToken(t, manager)
	style := decorationservice.Style{Code: "frame-scale", Name: "Scale frame", Kind: decorationservice.AvatarFrame, LightColors: []string{"#166534"}, DarkColors: []string{"#86EFAC"}, IsEnabled: true, AssetURL: "/static/decorations/frame-lantern.png"}
	for _, scale := range []float64{0, 0.5, 2, 1.1} {
		style.AssetScale = scale
		body, err := json.Marshal(style)
		if err != nil {
			t.Fatal(err)
		}
		rec := serveAuthSecurityJSON(router, "POST", "/api/admin/decoration-save", string(body), token)
		var response struct {
			Code   int
			Result decorationservice.Style
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		want := scale
		if scale == 0 {
			want = decorationservice.DefaultAssetScale
		}
		if rec.Code != 200 || response.Code != 0 || response.Result.AssetScale != want {
			t.Fatalf("save scale=%v response=%s", scale, rec.Body.String())
		}
		var stored decoration.Definition
		if err := conn.First(&stored, "code = ?", style.Code).Error; err != nil {
			t.Fatal(err)
		}
		if stored.AssetScale != want {
			t.Fatalf("persisted scale=%v want=%v", stored.AssetScale, want)
		}
		styles, err := decorationservice.Catalog(t.Context())
		if err != nil || len(styles) != 1 || styles[0].AssetScale != want {
			t.Fatalf("catalog scale=%+v err=%v", styles, err)
		}
	}
	for _, scale := range []float64{-1, 0.49, 2.01} {
		style.AssetScale = scale
		body, err := json.Marshal(style)
		if err != nil {
			t.Fatal(err)
		}
		rec := serveAuthSecurityJSON(router, "POST", "/api/admin/decoration-save", string(body), token)
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "invalid-params.json"))
	}
}

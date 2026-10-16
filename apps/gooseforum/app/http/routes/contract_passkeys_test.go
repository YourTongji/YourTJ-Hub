package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPasskeys"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userTotpRecoveryCodes"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/passkeyservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/totpservice"
	"github.com/gin-gonic/gin"
	otptotp "github.com/pquerna/otp/totp"
	"gorm.io/gorm"
)

func setupPasskeyContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(&userPasskeys.CredentialEntity{}, &userPasskeys.ChallengeEntity{}); err != nil {
		t.Fatalf("migrate passkey tables: %v", err)
	}
	if err := conn.AutoMigrate(&userTotpRecoveryCodes.Entity{}); err != nil {
		t.Fatalf("migrate TOTP recovery-code table: %v", err)
	}
	router.GET("/api/auth/passkey/status", api.PasskeyStatus)
	router.POST("/api/auth/passkey/login/begin", api.PasskeyAuthenticationBegin)
	router.POST("/api/auth/passkey/login/finish", api.PasskeyAuthenticationFinish)
	loginAPI := router.Group("/api").Use(middleware.CSRFProtection, middleware.JWTAuthCheck)
	loginAPI.POST("/user/passkeys/reauth", middleware.CheckWritableAccount, api.PasskeyReauthenticate)
	loginAPI.POST("/user/passkeys/registration/begin", middleware.CheckWritableAccount, api.PasskeyRegistrationBegin)
	loginAPI.POST("/user/passkeys/registration/finish", middleware.CheckWritableAccount, api.PasskeyRegistrationFinish)
	loginAPI.GET("/user/passkeys", api.PasskeyCredentialList)
	loginAPI.POST("/user/passkeys/rename", middleware.CheckWritableAccount, api.PasskeyCredentialRename)
	loginAPI.POST("/user/passkeys/revoke", middleware.CheckWritableAccount, api.PasskeyCredentialRevoke)
	return conn, router
}

func servePasskeyJSON(router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func mustPasskeyJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal passkey test request: %v", err)
	}
	return encoded
}

func TestPasskeyStatusDisabledByDefault(t *testing.T) {
	_, router := setupPasskeyContractTest(t)
	response := decodeContractEnvelope(t, servePasskeyJSON(router, http.MethodGet, "/api/auth/passkey/status", "", ""))
	assertResultObjectKeysMatchFixture(t, response, contractFixture(t, "passkey-status-disabled.json"))
}

func TestPasskeyManagementRequiresStrongReauthenticationAndOneUseProof(t *testing.T) {
	conn, router := setupPasskeyContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	credentialID := []byte{0x01, 0x02, 0x03, 0x04}
	if err := conn.Create(&userPasskeys.CredentialEntity{
		CredentialID:   credentialID,
		UserID:         user.Id,
		CredentialData: "encrypted-test-credential",
		Name:           "old name",
	}).Error; err != nil {
		t.Fatalf("seed credential: %v", err)
	}

	wrong := servePasskeyJSON(router, http.MethodPost, "/api/user/passkeys/reauth", `{"encryptedPassword":"invalid"}`, token)
	if response := decodeContractEnvelope(t, wrong); response.Code == 0 {
		t.Fatal("invalid password unexpectedly received a management proof")
	}

	body, err := json.Marshal(map[string]string{"encryptedPassword": encryptLoginPassword(t, "secret123")})
	if err != nil {
		t.Fatal(err)
	}
	reauth := decodeContractEnvelope(t, servePasskeyJSON(router, http.MethodPost, "/api/user/passkeys/reauth", string(body), token))
	if reauth.Code != 0 {
		t.Fatalf("strong reauthentication failed: %+v", reauth)
	}
	var proof struct {
		Proof string `json:"proof"`
	}
	if err := json.Unmarshal(reauth.Result, &proof); err != nil || proof.Proof == "" {
		t.Fatalf("missing proof in response %s: %v", reauth.Result, err)
	}

	revokeBody := mustPasskeyJSON(t, map[string]string{
		"proof":        proof.Proof,
		"credentialId": "AQIDBA",
	})
	revoked := decodeContractEnvelope(t, servePasskeyJSON(router, http.MethodPost, "/api/user/passkeys/revoke", string(revokeBody), token))
	if revoked.Code != 0 {
		t.Fatalf("credential revoke failed: %+v", revoked)
	}
	var stored userPasskeys.CredentialEntity
	if err := conn.Where("credential_id = ?", credentialID).First(&stored).Error; err != nil || stored.RevokedAt == nil {
		t.Fatalf("credential was not revoked: row=%+v err=%v", stored, err)
	}

	reuseBody := mustPasskeyJSON(t, map[string]string{
		"proof":        proof.Proof,
		"credentialId": "AQIDBA",
		"name":         "reused",
	})
	reused := decodeContractEnvelope(t, servePasskeyJSON(router, http.MethodPost, "/api/user/passkeys/rename", string(reuseBody), token))
	if reused.Code == 0 {
		t.Fatal("one-use proof unexpectedly succeeded a second time")
	}
}

func TestPasskeyConfigurationCanBeEnabledOnlyWithExplicitOriginsAndAppIdentities(t *testing.T) {
	keys := []string{"passkeys.enabled", "passkeys.rp_id", "passkeys.display_name", "passkeys.web_origins", "passkeys.apple_team_id", "passkeys.ios_bundle_id", "passkeys.android_package_name", "passkeys.android_sha256_cert_fingerprints", "app.env"}
	previous := make(map[string]any, len(keys))
	for _, key := range keys {
		previous[key] = preferences.GetRaw(key)
	}
	t.Cleanup(func() {
		for key, value := range previous {
			if value != nil {
				preferences.Set(key, value)
			} else {
				switch key {
				case "passkeys.enabled":
					preferences.Set(key, false)
				case "passkeys.display_name":
					preferences.Set(key, "YourTJ")
				case "passkeys.ios_bundle_id":
					preferences.Set(key, "tj.yourtj.forumApp")
				case "passkeys.android_package_name":
					preferences.Set(key, "tj.yourtj.forum_app")
				case "passkeys.web_origins", "passkeys.android_sha256_cert_fingerprints":
					preferences.Set(key, []string{})
				default:
					preferences.Set(key, "")
				}
			}
		}
	})
	preferences.Set("app.env", "production")
	preferences.Set("passkeys.enabled", true)
	preferences.Set("passkeys.rp_id", "f.yourtj.de")
	preferences.Set("passkeys.display_name", "YourTJ")
	preferences.Set("passkeys.web_origins", []string{"https://f.yourtj.de"})
	preferences.Set("passkeys.apple_team_id", "A1B2C3D4E5")
	preferences.Set("passkeys.ios_bundle_id", "tj.yourtj.forumApp")
	preferences.Set("passkeys.android_package_name", "tj.yourtj.forum_app")
	preferences.Set("passkeys.android_sha256_cert_fingerprints", []string{strings.Repeat("a", 64)})
	if _, err := passkeyservice.LoadConfig(); err != nil {
		t.Fatalf("complete passkey config rejected: %v", err)
	}

	preferences.Set("passkeys.web_origins", []string{"http://f.yourtj.de"})
	if _, err := passkeyservice.LoadConfig(); err == nil {
		t.Fatal("production HTTP origin unexpectedly enabled passkeys")
	}
}

func TestPasskeyReauthenticationRequiresTOTPWhenEnabled(t *testing.T) {
	conn, router := setupPasskeyContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	setup, err := totpservice.Setup(user.Id)
	if err != nil {
		t.Fatalf("setup TOTP: %v", err)
	}
	code, err := otptotp.GenerateCode(setup.Secret, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := totpservice.Enable(user.Id, code); err != nil {
		t.Fatalf("enable TOTP: %v", err)
	}
	base := map[string]string{"encryptedPassword": encryptLoginPassword(t, "secret123")}
	body := mustPasskeyJSON(t, base)
	missing := decodeContractEnvelope(t, servePasskeyJSON(router, http.MethodPost, "/api/user/passkeys/reauth", string(body), token))
	if missing.Code == 0 {
		t.Fatal("password alone unexpectedly bypassed enabled TOTP")
	}
	base["totpCode"] = code
	body = mustPasskeyJSON(t, base)
	valid := decodeContractEnvelope(t, servePasskeyJSON(router, http.MethodPost, "/api/user/passkeys/reauth", string(body), token))
	if valid.Code != 0 {
		t.Fatalf("password + TOTP reauthentication failed: %+v", valid)
	}
}

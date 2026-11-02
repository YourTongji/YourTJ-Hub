package controllers

import (
	"encoding/json"
	"fmt"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/tokenservice"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestActivationFailureHasPrivateOwnerRecovery(t *testing.T) {
	key := "activation-recovery-test-signing-key-2026"
	old := preferences.GetString("app.signingKey")
	preferences.Set("app.signingKey", key)
	t.Cleanup(func() { preferences.Set("app.signingKey", old) })
	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenservice.ActivationClaims{UserId: 42, Email: "owner-private@example.test", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))}}).SignedString([]byte(key))
	if err != nil {
		t.Fatal(err)
	}
	forged, err := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenservice.ActivationClaims{UserId: 42, Email: "owner-private@example.test", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))}}).SignedString([]byte("different-untrusted-test-key"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, token, message string }{
		{"missing", "", "activationMissingLink"}, {"invalid", "not-a-token", "activationInvalidLink"}, {"expired", expired, "activationExpired"}, {"forged expired", forged, "activationInvalidLink"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/activate?token="+tc.token+"&redirect=https://attacker.example", nil)
			c.Request.Header.Set("Accept-Language", "en")
			ActivateAccount(c)
			body := recorder.Body.String()
			if !strings.Contains(body, i18n.T("en", tc.message)) {
				t.Errorf("wrong reason for %s", tc.name)
			}
			if !strings.Contains(body, `href="/settings?tab=profile"`) {
				t.Error("missing owner recovery link")
			}
			if strings.Contains(body, "support@gooseforum.online") || strings.Contains(body, "attacker.example") || strings.Contains(body, "owner-private@example.test") {
				t.Error("response exposed an unrelated destination or account data")
			}
			if recorder.Header().Get("Cache-Control") != "private, no-store" || recorder.Header().Get("Referrer-Policy") != "no-referrer" {
				t.Error("activation result must not cache or disclose its token URL")
			}
			if recorder.Code != http.StatusOK {
				t.Fatalf("status=%d", recorder.Code)
			}
		})
	}
}

func activationContactConfig(t *testing.T, email string) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatal(err)
	}
	old := pageConfig.GetByPageType(pageConfig.SiteSettings)
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.SiteSettings).Delete(&pageConfig.Entity{})
		if old.Id != 0 {
			conn.Create(&old)
		}
		hotdataserve.ClearSiteSettingsConfigCache()
	})
	entity := old
	entity.PageType = pageConfig.SiteSettings
	data, err := json.Marshal(pageConfig.SiteSettingsConfig{SiteName: "Campus test", SiteEmail: email})
	if err != nil {
		t.Fatal(err)
	}
	entity.Config = string(data)
	if pageConfig.CreateOrSave(&entity) != 1 {
		t.Fatal("save site config")
	}
	hotdataserve.ClearSiteSettingsConfigCache()
}

func TestActivationRecoveryUsesOnlyConfiguredContact(t *testing.T) {
	for _, email := range []string{"help@campus.example", "bad\r\nBcc: theft@example.test", "javascript:alert(1)", "example@example.example", ""} {
		t.Run(email, func(t *testing.T) {
			activationContactConfig(t, email)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/activate?lang=en", nil)
			ActivateAccount(c)
			body := recorder.Body.String()
			expected := email == "help@campus.example"
			if strings.Contains(body, "mailto:") != expected {
				t.Fatalf("unexpected contact action for %q", email)
			}
			if expected && !strings.Contains(html.UnescapeString(body), "mailto:"+url.PathEscape(email)) {
				t.Fatal("configured destination missing")
			}
			if !strings.Contains(body, "Campus test") {
				t.Fatal("site identity missing")
			}
		})
	}
}

// Browser acceptance renders the real controller and embedded template. Output
// is opt-in, synthetic and ignored; the browser test owns its temporary folder.
func TestActivationRecoveryBrowserFixtures(t *testing.T) {
	dir := os.Getenv("YOURTJ_TEST_HTML_DIR")
	if dir == "" {
		t.Skip("browser fixture output not requested")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	activationContactConfig(t, "help@campus.example")
	key := "activation-browser-test-key-2026"
	old := preferences.GetString("app.signingKey")
	preferences.Set("app.signingKey", key)
	t.Cleanup(func() { preferences.Set("app.signingKey", old) })
	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenservice.ActivationClaims{UserId: 42, Email: "owner-private@example.test", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))}}).SignedString([]byte(key))
	if err != nil {
		t.Fatal(err)
	}
	for _, locale := range []string{"zh", "en", "ja", "de"} {
		for _, tc := range []struct{ name, token string }{{"missing", ""}, {"invalid", "not-a-token"}, {"expired", expired}} {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/activate?lang="+locale+"&token="+tc.token, nil)
			ActivateAccount(c)
			if err := os.WriteFile(filepath.Join(dir, tc.name+"-"+locale+".html"), recorder.Body.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestInvalidActivationDoesNotDistinguishAccountState(t *testing.T) {
	key := "activation-state-test-signing-key-2026"
	old := preferences.GetString("app.signingKey")
	preferences.Set("app.signingKey", key)
	t.Cleanup(func() { preferences.Set("app.signingKey", old) })
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	user := users.EntityComplete{Username: fmt.Sprintf("activation-recovery-%d", time.Now().UnixNano()), Email: "activation-current@example.test"}
	if err := conn.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&user) })
	missing, err := tokenservice.GenerateActivationToken(1<<63-1, "private-missing@example.test")
	if err != nil {
		t.Fatal(err)
	}
	stale, err := tokenservice.GenerateActivationToken(user.Id, "private-stale@example.test")
	if err != nil {
		t.Fatal(err)
	}
	var expected string
	for _, token := range []string{"not-a-token", missing, stale} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodGet, "/activate?lang=en&token="+token, nil)
		ActivateAccount(c)
		if expected == "" {
			expected = recorder.Body.String()
		} else if recorder.Body.String() != expected {
			t.Fatal("invalid links distinguish unavailable and stale account targets")
		}
	}
}

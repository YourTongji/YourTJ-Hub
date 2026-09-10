package pkservice

import (
	"os"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

func writeEncryptedCookieSetting(t *testing.T, cookie string) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate page_config: %v", err)
	}
	conn.Where("page_type = ?", pageConfig.OneSystemSettings).Delete(&pageConfig.Entity{})
	if cookie != "" {
		enc, err := securestore.EncryptPurpose(cookie, securestore.OneSystemCookiePurpose)
		if err != nil {
			t.Fatalf("encrypt cookie: %v", err)
		}
		if err := conn.Create(&pageConfig.Entity{PageType: pageConfig.OneSystemSettings, Config: jsonopt.Encode(pageConfig.OneSystemSettingsStorage{CookieEncrypted: enc})}).Error; err != nil {
			t.Fatalf("write settings: %v", err)
		}
	}
	hotdataserve.ClearOnesystemSettingsConfigCache()
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.OneSystemSettings).Delete(&pageConfig.Entity{})
		hotdataserve.ClearOnesystemSettingsConfigCache()
	})
}

func TestResolveCookiePrecedence(t *testing.T) {
	t.Setenv(envOnesystemCookie, "env-cookie")

	// 1) 显式参数优先
	if got, err := ResolveCookie("flag-cookie"); err != nil || got != "flag-cookie" {
		t.Errorf("flag precedence: got=%q err=%v", got, err)
	}
	// 2) 环境变量次之
	if got, err := ResolveCookie(""); err != nil || got != "env-cookie" {
		t.Errorf("env precedence: got=%q err=%v", got, err)
	}
	// 3) 管理端设置（加密落库、读取时解密）
	os.Unsetenv(envOnesystemCookie)
	writeEncryptedCookieSetting(t, "settings-cookie")
	if got, err := ResolveCookie(""); err != nil || got != "settings-cookie" {
		t.Errorf("settings precedence: got=%q err=%v", got, err)
	}
	// 4) 全缺失 → 明确报错
	writeEncryptedCookieSetting(t, "")
	if _, err := ResolveCookie(""); err == nil {
		t.Error("expected error when no credential configured")
	}
}

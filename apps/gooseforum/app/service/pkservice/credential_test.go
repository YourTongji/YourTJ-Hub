package pkservice

import (
	"os"
	"strings"
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
	if got, err := ResolveCookie("flag-cookie", "", ""); err != nil || got != "flag-cookie" {
		t.Errorf("flag precedence: got=%q err=%v", got, err)
	}
	// 2) 环境变量次之
	if got, err := ResolveCookie("", "", ""); err != nil || got != "env-cookie" {
		t.Errorf("env precedence: got=%q err=%v", got, err)
	}
	// 3) 管理端设置（加密落库、读取时解密）
	os.Unsetenv(envOnesystemCookie)
	writeEncryptedCookieSetting(t, "settings-cookie")
	if got, err := ResolveCookie("", "", ""); err != nil || got != "settings-cookie" {
		t.Errorf("settings precedence: got=%q err=%v", got, err)
	}
	// 4) 全缺失（无 cookie 也无账号密码）→ 明确报错
	writeEncryptedCookieSetting(t, "")
	if _, err := ResolveCookie("", "", ""); err == nil {
		t.Error("expected error when no credential configured")
	}
}

func TestResolveCookieFallsBackToSnoPasswordLogin(t *testing.T) {
	// 无 Cookie 来源时，提供账号密码（参数或环境变量）→ 自动 SSO 登录换取会话 Cookie。
	ep, _ := newLoginFixture(t, "s3cret-pass")

	orig := defaultOnesystemLoginEndpoints
	t.Cleanup(func() { defaultOnesystemLoginEndpoints = orig })
	defaultOnesystemLoginEndpoints = func() onesystemLoginEndpoints { return ep }

	// 清除环境变量与管理端设置，仅提供账号密码参数。
	os.Unsetenv(envOnesystemCookie)
	os.Unsetenv(envOnesystemSno)
	os.Unsetenv(envOnesystemPassword)
	writeEncryptedCookieSetting(t, "")

	cookie, err := ResolveCookie("", "1951234", "s3cret-pass")
	if err != nil {
		t.Fatalf("ResolveCookie with sno/password: %v", err)
	}
	if !strings.Contains(cookie, "JWTUser=jwt") {
		t.Fatalf("cookie from login = %q, want JWTUser=jwt", cookie)
	}

	// 环境变量形式同样生效。
	cookie, err = ResolveCookie("", "", "")
	if err == nil {
		t.Fatalf("expected error without env credentials, got cookie %q", cookie)
	}
	t.Setenv(envOnesystemSno, "1951234")
	t.Setenv(envOnesystemPassword, "s3cret-pass")
	cookie, err = ResolveCookie("", "", "")
	if err != nil {
		t.Fatalf("ResolveCookie with env credentials: %v", err)
	}
	if !strings.Contains(cookie, "JWTUser=jwt") {
		t.Fatalf("cookie from env login = %q", cookie)
	}
}

func TestResolveSnoPassword(t *testing.T) {
	os.Unsetenv(envOnesystemSno)
	os.Unsetenv(envOnesystemPassword)

	sno, pwd := resolveSnoPassword("flag-sno", "flag-pwd")
	if sno != "flag-sno" || pwd != "flag-pwd" {
		t.Fatalf("flag priority: %q/%q", sno, pwd)
	}
	sno, pwd = resolveSnoPassword("", "")
	if sno != "" || pwd != "" {
		t.Fatalf("empty: %q/%q", sno, pwd)
	}
	t.Setenv(envOnesystemSno, "env-sno")
	t.Setenv(envOnesystemPassword, "env-pwd")
	sno, pwd = resolveSnoPassword("", "")
	if sno != "env-sno" || pwd != "env-pwd" {
		t.Fatalf("env fallback: %q/%q", sno, pwd)
	}
	sno, pwd = resolveSnoPassword("flag-sno", "")
	if sno != "flag-sno" || pwd != "env-pwd" {
		t.Fatalf("mixed: %q/%q", sno, pwd)
	}
}

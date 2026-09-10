package api

import (
	"net/http"
	"strings"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

func setupOnesystemSettingsTest(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate page_config: %v", err)
	}
	conn.Where("page_type = ?", pageConfig.OneSystemSettings).Delete(&pageConfig.Entity{})
	hotdataserve.ClearOnesystemSettingsConfigCache()
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.OneSystemSettings).Delete(&pageConfig.Entity{})
		hotdataserve.ClearOnesystemSettingsConfigCache()
	})
}

func readOnesystemSettings() pageConfig.OneSystemSettingsConfig {
	return hotdataserve.GetOnesystemSettingsConfigCache()
}

func TestSaveOnesystemSettingsEncryptsAtRest(t *testing.T) {
	setupOnesystemSettingsTest(t)
	cookie := "JWTUser=abc; JSESSIONID=def"

	res := SaveOnesystemSettings(component.BetterRequest[SaveOnesystemSettingsReq]{Params: SaveOnesystemSettingsReq{Cookie: &cookie}})
	if res.Code != http.StatusOK {
		t.Fatalf("save failed: code=%d", res.Code)
	}

	cfg := readOnesystemSettings()
	if strings.Contains(cfg.CookieEncrypted, "JWTUser") || strings.Contains(cfg.CookieEncrypted, "JSESSIONID") {
		t.Fatalf("plaintext cookie leaked into page_config: %q", cfg.CookieEncrypted)
	}
	if cfg.CookieEncrypted == "" {
		t.Fatal("expected encrypted cookie to be stored")
	}
	plain, err := securestore.DecryptPurpose(cfg.CookieEncrypted, securestore.OneSystemCookiePurpose)
	if err != nil {
		t.Fatalf("decrypt stored cookie: %v", err)
	}
	if plain != cookie {
		t.Errorf("round-trip = %q, want %q", plain, cookie)
	}

	// GET 只回显是否已配置，不回显密文/明文。
	getRes := GetOnesystemSettings(component.BetterRequest[component.Null]{})
	result, ok := getRes.Data.Result.(map[string]any)
	if !ok {
		t.Fatalf("GET result type = %T", getRes.Data.Result)
	}
	if result["cookieConfigured"] != true {
		t.Errorf("cookieConfigured = %v, want true", result["cookieConfigured"])
	}
}

func TestSaveOnesystemSettingsClearsOnEmpty(t *testing.T) {
	setupOnesystemSettingsTest(t)
	initial := "JWTUser=abc"
	SaveOnesystemSettings(component.BetterRequest[SaveOnesystemSettingsReq]{Params: SaveOnesystemSettingsReq{Cookie: &initial}})

	empty := ""
	SaveOnesystemSettings(component.BetterRequest[SaveOnesystemSettingsReq]{Params: SaveOnesystemSettingsReq{Cookie: &empty}})
	if cfg := readOnesystemSettings(); cfg.CookieEncrypted != "" {
		t.Errorf("cookie not cleared, still stored: %q", cfg.CookieEncrypted)
	}
}

func TestSaveOnesystemSettingsOmittedFieldsKeepCredentials(t *testing.T) {
	setupOnesystemSettingsTest(t)
	graduate := "graduate-cookie"
	if res := SaveOnesystemSettings(component.BetterRequest[SaveOnesystemSettingsReq]{Params: SaveOnesystemSettingsReq{GraduateCookie: &graduate}}); res.Code != http.StatusOK {
		t.Fatalf("save graduate cookie failed: code=%d", res.Code)
	}
	undergraduate := "undergraduate-cookie"
	if res := SaveOnesystemSettings(component.BetterRequest[SaveOnesystemSettingsReq]{Params: SaveOnesystemSettingsReq{UndergraduateCookie: &undergraduate}}); res.Code != http.StatusOK {
		t.Fatalf("save undergraduate cookie failed: code=%d", res.Code)
	}

	if res := SaveOnesystemSettings(component.BetterRequest[SaveOnesystemSettingsReq]{Params: SaveOnesystemSettingsReq{}}); res.Code != http.StatusOK {
		t.Fatalf("save omitted settings failed: code=%d", res.Code)
	}
	if cfg := readOnesystemSettings(); cfg.CookieEncrypted == "" || cfg.GraduateCookieEncrypted == "" {
		t.Fatal("omitted fields cleared an existing audience credential")
	}
}

func TestSaveOnesystemSettingsKeepsOtherAudience(t *testing.T) {
	setupOnesystemSettingsTest(t)

	graduate := "graduate-cookie"
	if res := SaveOnesystemSettings(component.BetterRequest[SaveOnesystemSettingsReq]{Params: SaveOnesystemSettingsReq{GraduateCookie: &graduate}}); res.Code != http.StatusOK {
		t.Fatalf("save graduate cookie failed: code=%d", res.Code)
	}
	undergraduate := "undergraduate-cookie"
	if res := SaveOnesystemSettings(component.BetterRequest[SaveOnesystemSettingsReq]{Params: SaveOnesystemSettingsReq{UndergraduateCookie: &undergraduate}}); res.Code != http.StatusOK {
		t.Fatalf("save undergraduate cookie failed: code=%d", res.Code)
	}

	cfg := readOnesystemSettings()
	if cfg.CookieEncrypted == "" || cfg.GraduateCookieEncrypted == "" {
		t.Fatalf("audience credentials = %#v, want both credentials configured", cfg)
	}
	if plain, err := securestore.DecryptPurpose(cfg.CookieEncrypted, securestore.OneSystemCookiePurpose); err != nil || plain != undergraduate {
		t.Fatalf("undergraduate cookie decrypt = %q, err %v; want %q", plain, err, undergraduate)
	}
	if plain, err := securestore.DecryptPurpose(cfg.GraduateCookieEncrypted, securestore.OneSystemCookiePurpose); err != nil || plain != graduate {
		t.Fatalf("graduate cookie decrypt = %q, err %v; want %q", plain, err, graduate)
	}
}

func TestGetOnesystemSettingsUnconfigured(t *testing.T) {
	setupOnesystemSettingsTest(t)
	getRes := GetOnesystemSettings(component.BetterRequest[component.Null]{})
	result, ok := getRes.Data.Result.(map[string]any)
	if !ok {
		t.Fatalf("GET result type = %T", getRes.Data.Result)
	}
	if result["cookieConfigured"] != false {
		t.Errorf("cookieConfigured = %v, want false", result["cookieConfigured"])
	}
}

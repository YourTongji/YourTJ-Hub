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
	const cookie = "JWTUser=abc; JSESSIONID=def"

	res := SaveOnesystemSettings(component.BetterRequest[SaveOnesystemSettingsReq]{Params: SaveOnesystemSettingsReq{Cookie: cookie}})
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
	SaveOnesystemSettings(component.BetterRequest[SaveOnesystemSettingsReq]{Params: SaveOnesystemSettingsReq{Cookie: "JWTUser=abc"}})

	SaveOnesystemSettings(component.BetterRequest[SaveOnesystemSettingsReq]{Params: SaveOnesystemSettingsReq{Cookie: ""}})
	if cfg := readOnesystemSettings(); cfg.CookieEncrypted != "" {
		t.Errorf("cookie not cleared, still stored: %q", cfg.CookieEncrypted)
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

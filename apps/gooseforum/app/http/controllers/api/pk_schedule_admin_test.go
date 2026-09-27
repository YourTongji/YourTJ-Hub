package api

import (
	"net/http"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// setupPkScheduleAdminTest 迁移 page_config 并清理排课定时同步配置（含缓存），
// 保证每个用例从默认配置出发。
func setupPkScheduleAdminTest(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate page_config: %v", err)
	}
	conn.Unscoped().Where("page_type = ?", pageConfig.PkSyncSchedule).Delete(&pageConfig.Entity{})
	hotdataserve.ClearPkSyncScheduleConfigCache()
	t.Cleanup(hotdataserve.ClearPkSyncScheduleConfigCache)
}

func readPkSyncScheduleConfig() pageConfig.PkSyncScheduleConfig {
	return hotdataserve.GetPkSyncScheduleConfigCache()
}

func TestGetPkSyncScheduleSettingsReturnsDefault(t *testing.T) {
	setupPkScheduleAdminTest(t)
	res := GetPkSyncScheduleSettings(component.BetterRequest[component.Null]{})
	if res.Code != http.StatusOK || res.Data.Code != component.SUCCESS {
		t.Fatalf("get failed: code=%d data=%+v", res.Code, res.Data)
	}
	view, ok := res.Data.Result.(PkSyncScheduleSettingsView)
	if !ok {
		t.Fatalf("result type = %T, want PkSyncScheduleSettingsView", res.Data.Result)
	}
	if view.Enabled || view.Schedule != "30 2 * * *" || view.Depth != 1 || view.Audience != "undergraduate" {
		t.Fatalf("default view = %+v, want enabled=false schedule=30 2 * * * depth=1 audience=undergraduate", view)
	}
}

func TestSavePkSyncScheduleSettingsValid(t *testing.T) {
	setupPkScheduleAdminTest(t)
	res := SavePkSyncScheduleSettings(component.BetterRequest[SavePkSyncScheduleSettingsReq]{
		Params: SavePkSyncScheduleSettingsReq{Enabled: true, Schedule: "0 3 * * *", Term: "121", Depth: 3, Audience: "graduate"},
	})
	if res.Code != http.StatusOK || res.Data.Code != component.SUCCESS {
		t.Fatalf("save failed: code=%d data=%+v", res.Code, res.Data)
	}
	cfg := readPkSyncScheduleConfig()
	if !cfg.Enabled || cfg.Schedule != "0 3 * * *" || cfg.Term != "121" || cfg.Depth != 3 || cfg.Audience != "graduate" {
		t.Fatalf("persisted config = %+v", cfg)
	}
	// GET 回显与持久化一致。
	getRes := GetPkSyncScheduleSettings(component.BetterRequest[component.Null]{})
	view := getRes.Data.Result.(PkSyncScheduleSettingsView)
	if !view.Enabled || view.Schedule != "0 3 * * *" || view.Audience != "graduate" {
		t.Fatalf("get view = %+v", view)
	}
}

func TestSavePkSyncScheduleSettingsRejectsInvalidCron(t *testing.T) {
	setupPkScheduleAdminTest(t)
	invalid := []string{"", "not-a-cron", "61 * * * *", "30 2 * *", "0 0 0 * *", "0 0 * 0 *", "@every 1d", "*/1e1 * * * *", "*/2.5 * * * *"}
	for _, spec := range invalid {
		res := SavePkSyncScheduleSettings(component.BetterRequest[SavePkSyncScheduleSettingsReq]{
			Params: SavePkSyncScheduleSettingsReq{Enabled: true, Schedule: spec},
		})
		if res.Data.Code != component.FAIL {
			t.Fatalf("cron %q enabled: expected FAIL, got %+v", spec, res.Data)
		}
		if cfg := readPkSyncScheduleConfig(); cfg.Enabled {
			t.Fatalf("cron %q must not persist an enabled schedule", spec)
		}
	}
}

func TestSavePkSyncScheduleSettingsDisableClearsCron(t *testing.T) {
	setupPkScheduleAdminTest(t)
	enable := SavePkSyncScheduleSettings(component.BetterRequest[SavePkSyncScheduleSettingsReq]{
		Params: SavePkSyncScheduleSettingsReq{Enabled: true, Schedule: "30 2 * * *", Term: "121"},
	})
	if enable.Data.Code != component.SUCCESS {
		t.Fatalf("enable failed: %+v", enable.Data)
	}
	disable := SavePkSyncScheduleSettings(component.BetterRequest[SavePkSyncScheduleSettingsReq]{
		Params: SavePkSyncScheduleSettingsReq{Enabled: false},
	})
	if disable.Data.Code != component.SUCCESS {
		t.Fatalf("disable failed: %+v", disable.Data)
	}
	cfg := readPkSyncScheduleConfig()
	if cfg.Enabled {
		t.Fatalf("config = %+v, want disabled", cfg)
	}
}

func TestSavePkSyncScheduleSettingsClampsDepthAndDefaultAudience(t *testing.T) {
	setupPkScheduleAdminTest(t)
	res := SavePkSyncScheduleSettings(component.BetterRequest[SavePkSyncScheduleSettingsReq]{
		Params: SavePkSyncScheduleSettingsReq{Enabled: true, Schedule: "30 2 * * *", Depth: 99},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("save failed: %+v", res.Data)
	}
	cfg := readPkSyncScheduleConfig()
	if cfg.Depth != maxPkSyncDepth {
		t.Errorf("depth = %d, want clamped %d", cfg.Depth, maxPkSyncDepth)
	}
	if cfg.Audience != string(pk.AudienceUndergraduate) {
		t.Errorf("audience = %q, want undergraduate (default)", cfg.Audience)
	}
}

func TestSavePkSyncScheduleSettingsRejectsInvalidAudience(t *testing.T) {
	setupPkScheduleAdminTest(t)
	res := SavePkSyncScheduleSettings(component.BetterRequest[SavePkSyncScheduleSettingsReq]{
		Params: SavePkSyncScheduleSettingsReq{Enabled: true, Schedule: "30 2 * * *", Audience: "master"},
	})
	if res.Data.Code != component.FAIL {
		t.Fatalf("invalid audience: expected FAIL, got %+v", res.Data)
	}
}

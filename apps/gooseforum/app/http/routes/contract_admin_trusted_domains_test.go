package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// 外链免确认白名单（issue #1200）的管理端读写契约：保存期归一化（小写/去重/去空行）
// 与整单拒绝（code=1 + admin.externalDomain.invalid + params.domain，不落库），
// 读取期对升级前存量行（缺键）补内置默认清单、显式空数组保持为空。

func TestAdminSaveSiteSettingsNormalizesTrustedDomains(t *testing.T) {
	conn, router := setupAdminSiteContractTest(t)
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.SiteSettings).Delete(&pageConfig.Entity{})
	})
	body := `{"settings":{"siteName":"s","siteLogo":"","siteDescription":"","siteKeywords":"","siteUrl":"","siteEmail":"","externalLinks":"","trustedExternalDomains":[" GitHub.com ","github.com","","bilibili.com"]}}`
	recorder := serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/save-site-settings", body)
	env := decodeContractEnvelope(t, recorder)
	if env.Code != 0 {
		t.Fatalf("envelope = %#v, want success", env)
	}
	stored := pageConfig.GetConfigByPageType(pageConfig.SiteSettings, pageConfig.SiteSettingsConfig{})
	want := []string{"github.com", "bilibili.com"}
	if len(stored.TrustedExternalDomains) != len(want) {
		t.Fatalf("stored trustedExternalDomains = %#v, want %#v", stored.TrustedExternalDomains, want)
	}
	for i := range want {
		if stored.TrustedExternalDomains[i] != want[i] {
			t.Fatalf("stored trustedExternalDomains = %#v, want %#v", stored.TrustedExternalDomains, want)
		}
	}
}

func TestAdminSaveSiteSettingsRejectsInvalidTrustedDomains(t *testing.T) {
	cases := map[string]string{
		"scheme":            "https://github.com",
		"path":              "github.com/topics",
		"port":              "github.com:443",
		"wildcard":          "*.github.com",
		"public suffix":     "github.io",
		"ip literal":        "192.168.1.1",
		"single label":      "localhost",
		"unknowable string": "not a domain",
	}
	for name, entry := range cases {
		t.Run(name, func(t *testing.T) {
			conn, router := setupAdminSiteContractTest(t)
			t.Cleanup(func() {
				conn.Where("page_type = ?", pageConfig.SiteSettings).Delete(&pageConfig.Entity{})
			})
			body := fmt.Sprintf(`{"settings":{"siteName":"s","siteLogo":"","siteDescription":"","siteKeywords":"","siteUrl":"","siteEmail":"","externalLinks":"","trustedExternalDomains":[%q]}}`, entry)
			recorder := serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/save-site-settings", body)
			env := decodeContractEnvelope(t, recorder)
			if env.Code != 1 || env.MessageCode != "admin.externalDomain.invalid" {
				t.Fatalf("envelope = code %d messageCode %q, want code 1 + admin.externalDomain.invalid", env.Code, env.MessageCode)
			}
			if env.Params["domain"] != entry {
				t.Fatalf("params.domain = %v, want the offending entry %q", env.Params["domain"], entry)
			}
			stored := pageConfig.GetConfigByPageType(pageConfig.SiteSettings, pageConfig.SiteSettingsConfig{})
			if stored.SiteName != "" || len(stored.TrustedExternalDomains) != 0 {
				t.Fatalf("rejected config was persisted: %#v", stored)
			}
		})
	}
}

func TestAdminGetSiteSettingsTrustedDomainCompatibility(t *testing.T) {
	t.Run("legacy row without the field reports the built-in whitelist", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.SiteSettings, json.RawMessage(`{"siteName":"旧站点"}`))
		recorder := serveAdminSiteRaw(t, conn, router, http.MethodGet, "/api/admin/site-settings", "")
		var payload struct {
			Result struct {
				TrustedExternalDomains []string `json:"trustedExternalDomains"`
			} `json:"result"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode site settings %s: %v", recorder.Body.String(), err)
		}
		want := defaultconfig.GetDefaultSiteSettingsConfig().TrustedExternalDomains
		if len(payload.Result.TrustedExternalDomains) != len(want) {
			t.Fatalf("trustedExternalDomains = %#v, want the built-in default %#v", payload.Result.TrustedExternalDomains, want)
		}
	})

	t.Run("explicit empty whitelist survives the round trip", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.SiteSettings, json.RawMessage(`{"siteName":"新站点","trustedExternalDomains":[]}`))
		recorder := serveAdminSiteRaw(t, conn, router, http.MethodGet, "/api/admin/site-settings", "")
		if body := recorder.Body.String(); body == "" {
			t.Fatal("empty site settings response")
		}
		var payload struct {
			Result struct {
				TrustedExternalDomains []string `json:"trustedExternalDomains"`
			} `json:"result"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode site settings %s: %v", recorder.Body.String(), err)
		}
		if payload.Result.TrustedExternalDomains == nil || len(payload.Result.TrustedExternalDomains) != 0 {
			t.Fatalf("trustedExternalDomains = %#v, want an explicit empty array", payload.Result.TrustedExternalDomains)
		}
	})
}

// TestAdminSaveSiteSettingsInvalidatesSiteSettingsCache 白名单是 SSR payload 与
// /api/external-link-policy 的共同数据源：保存后缓存必须失效，否则两个客户端
// 面在 TTL 内仍按旧清单豁免确认层。
func TestAdminSaveSiteSettingsInvalidatesSiteSettingsCache(t *testing.T) {
	conn, router := setupAdminSiteContractTest(t)
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.SiteSettings).Delete(&pageConfig.Entity{})
		hotdataserve.ClearSiteSettingsConfigCache()
	})
	// 预热缓存（未配置 → 内置默认清单）。
	if len(hotdataserve.GetSiteSettingsConfigCache().TrustedExternalDomains) == 0 {
		t.Fatal("preheated cache has an empty whitelist, want the built-in default")
	}
	body := `{"settings":{"siteName":"s","siteLogo":"","siteDescription":"","siteKeywords":"","siteUrl":"","siteEmail":"","externalLinks":"","trustedExternalDomains":[]}}`
	recorder := serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/save-site-settings", body)
	if env := decodeContractEnvelope(t, recorder); env.Code != 0 {
		t.Fatalf("envelope = %#v, want success", env)
	}
	if domains := hotdataserve.GetSiteSettingsConfigCache().TrustedExternalDomains; len(domains) != 0 {
		t.Fatalf("cached whitelist = %#v, want the freshly saved empty list", domains)
	}
}

// TestAdminSaveSiteSettingsWithoutWhitelistFieldPreservesStoredList 部署期间仍打开的
// 旧管理端页面（bundle 不含 trustedExternalDomains）保存其他站点设置时，不能把
// 白名单静默清空：键缺失按「保留现有清单」处理，显式空数组才是清空（issue #1200 review）。
func TestAdminSaveSiteSettingsWithoutWhitelistFieldPreservesStoredList(t *testing.T) {
	conn, router := setupAdminSiteContractTest(t)
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.SiteSettings).Delete(&pageConfig.Entity{})
	})
	persistContractPageConfig(t, conn, pageConfig.SiteSettings, pageConfig.SiteSettingsConfig{
		SiteName:               "旧站点",
		TrustedExternalDomains: []string{"github.com", "zhihu.com"},
	})

	body := `{"settings":{"siteName":"新站点","siteLogo":"","siteDescription":"","siteKeywords":"","siteUrl":"","siteEmail":"","externalLinks":""}}`
	recorder := serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/save-site-settings", body)
	if env := decodeContractEnvelope(t, recorder); env.Code != 0 {
		t.Fatalf("envelope = %#v, want success", env)
	}
	stored := pageConfig.GetConfigByPageType(pageConfig.SiteSettings, pageConfig.SiteSettingsConfig{})
	if stored.SiteName != "新站点" {
		t.Fatalf("siteName = %q, want the submitted value", stored.SiteName)
	}
	if len(stored.TrustedExternalDomains) != 2 || stored.TrustedExternalDomains[0] != "github.com" {
		t.Fatalf("stored trustedExternalDomains = %#v, want the previous list preserved", stored.TrustedExternalDomains)
	}

	t.Run("explicit empty array still clears the list", func(t *testing.T) {
		body := `{"settings":{"siteName":"新站点","siteLogo":"","siteDescription":"","siteKeywords":"","siteUrl":"","siteEmail":"","externalLinks":"","trustedExternalDomains":[]}}`
		recorder := serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/save-site-settings", body)
		if env := decodeContractEnvelope(t, recorder); env.Code != 0 {
			t.Fatalf("envelope = %#v, want success", env)
		}
		stored := pageConfig.GetConfigByPageType(pageConfig.SiteSettings, pageConfig.SiteSettingsConfig{})
		if len(stored.TrustedExternalDomains) != 0 {
			t.Fatalf("stored trustedExternalDomains = %#v, want an explicit empty list", stored.TrustedExternalDomains)
		}
	})
}

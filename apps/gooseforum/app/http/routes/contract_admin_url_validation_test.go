package routes

import (
	"net/http"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// 管理员可配置 URL 的保存期校验（issue #409）：危险 scheme、编码/空白混淆、
// 协议相对 URL 与不符合字段策略的值被稳定拒绝（code=1 + admin.url.invalid），
// 且不落库；合法 http(s) 与站内相对路径正常保存。

func TestAdminSaveSiteSettingsRejectsUnsafeURLs(t *testing.T) {
	cases := map[string]string{
		"javascript scheme":      `{"settings":{"siteName":"s","siteUrl":"javascript:alert(1)","siteLogo":"","siteDescription":"","siteKeywords":"","siteEmail":"","externalLinks":""}}`,
		"data scheme":            `{"settings":{"siteName":"s","siteUrl":"data:text/html;base64,PHNjcmlwdD4=","siteLogo":"","siteDescription":"","siteKeywords":"","siteEmail":"","externalLinks":""}}`,
		"control char in scheme": "{\"settings\":{\"siteName\":\"s\",\"siteUrl\":\"java\\nscript:alert(1)\",\"siteLogo\":\"\",\"siteDescription\":\"\",\"siteKeywords\":\"\",\"siteEmail\":\"\",\"externalLinks\":\"\"}}",
		"protocol relative":      `{"settings":{"siteName":"s","siteUrl":"//evil.example.com","siteLogo":"","siteDescription":"","siteKeywords":"","siteEmail":"","externalLinks":""}}`,
		"entity encoded scheme":  `{"settings":{"siteName":"s","siteUrl":"jav&#x61;script:alert(1)","siteLogo":"","siteDescription":"","siteKeywords":"","siteEmail":"","externalLinks":""}}`,
		"relative siteUrl":       `{"settings":{"siteName":"s","siteUrl":"/only","siteLogo":"","siteDescription":"","siteKeywords":"","siteEmail":"","externalLinks":""}}`,
		"unsafe logo":            `{"settings":{"siteName":"s","siteUrl":"","siteLogo":"javascript:alert(1)","siteDescription":"","siteKeywords":"","siteEmail":"","externalLinks":""}}`,
		"no host http":           `{"settings":{"siteName":"s","siteUrl":"https://","siteLogo":"","siteDescription":"","siteKeywords":"","siteEmail":"","externalLinks":""}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			conn, router := setupAdminSiteContractTest(t)
			t.Cleanup(func() {
				conn.Where("page_type = ?", pageConfig.SiteSettings).Delete(&pageConfig.Entity{})
			})
			recorder := serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/save-site-settings", body)
			env := decodeContractEnvelope(t, recorder)
			if env.Code != 1 || env.MessageCode != "admin.url.invalid" {
				t.Fatalf("envelope = code %d messageCode %q, want code 1 + admin.url.invalid", env.Code, env.MessageCode)
			}
			stored := pageConfig.GetConfigByPageType(pageConfig.SiteSettings, pageConfig.SiteSettingsConfig{})
			if stored.SiteUrl != "" || stored.SiteLogo != "" {
				t.Fatalf("unsafe config was persisted: %#v", stored)
			}
		})
	}
}

func TestAdminSaveSiteSettingsAllowsSafeURLs(t *testing.T) {
	conn, router := setupAdminSiteContractTest(t)
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.SiteSettings).Delete(&pageConfig.Entity{})
	})
	body := `{"settings":{"siteName":"s","siteUrl":"https://example.com","siteLogo":"/static/logo.webp","siteDescription":"","siteKeywords":"","siteEmail":"","externalLinks":""}}`
	recorder := serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/save-site-settings", body)
	env := decodeContractEnvelope(t, recorder)
	if env.Code != 0 {
		t.Fatalf("envelope = %#v, want success", env)
	}
	stored := pageConfig.GetConfigByPageType(pageConfig.SiteSettings, pageConfig.SiteSettingsConfig{})
	if stored.SiteUrl != "https://example.com" || stored.SiteLogo != "/static/logo.webp" {
		t.Fatalf("stored = %#v, want safe values persisted", stored)
	}
}

func TestAdminSaveSiteChromeRejectsUnsafeURLs(t *testing.T) {
	cases := map[string]string{
		"nav javascript": `{"settings":{"header":[{"id":"x","enabled":true,"type":"link","label":"X","url":"javascript:alert(1)"}],"mainMenu":[],"resources":[],"sidebarGroups":[],"footerInfo":{"primary":[],"list":[]},"brandType":"default","brandText":"","brandImage":""}}`,
		"footer data":    `{"settings":{"header":[],"mainMenu":[],"resources":[],"sidebarGroups":[],"footerInfo":{"primary":[],"list":[{"name":"X","url":"data:text/html,x"}]},"brandType":"default","brandText":"","brandImage":""}}`,
		"brand image":    `{"settings":{"header":[],"mainMenu":[],"resources":[],"sidebarGroups":[],"footerInfo":{"primary":[],"list":[]},"brandType":"image","brandText":"","brandImage":"javascript:alert(1)"}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			conn, router := setupAdminSiteContractTest(t)
			t.Cleanup(func() {
				conn.Where("page_type = ?", pageConfig.SiteChrome).Delete(&pageConfig.Entity{})
			})
			recorder := serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/save-site-chrome", body)
			env := decodeContractEnvelope(t, recorder)
			if env.Code != 1 || env.MessageCode != "admin.url.invalid" {
				t.Fatalf("envelope = code %d messageCode %q, want code 1 + admin.url.invalid", env.Code, env.MessageCode)
			}
		})
	}
}

func TestAdminSaveSponsorsRejectsUnsafeURLs(t *testing.T) {
	cases := map[string]string{
		"sponsor link javascript": `{"sponsorsInfo":{"sponsors":{"level0":[{"link":"javascript:alert(1)","message":"m","avatarUrl":"","name":"A"}]},"content":{"title":"","description":""},"contact":{"title":"","description":"","buttonText":"","buttonLink":""},"rules":[]}}`,
		"sponsor relative link":   `{"sponsorsInfo":{"sponsors":{"level0":[{"link":"/path","message":"m","avatarUrl":"","name":"A"}]},"content":{"title":"","description":""},"contact":{"title":"","description":"","buttonText":"","buttonLink":""},"rules":[]}}`,
		"avatar data uri":         `{"sponsorsInfo":{"sponsors":{"level0":[{"link":"https://example.com","message":"m","avatarUrl":"data:image/svg+xml,<svg onload=alert(1)>","name":"A"}]},"content":{"title":"","description":""},"contact":{"title":"","description":"","buttonText":"","buttonLink":""},"rules":[]}}`,
		"contact javascript":      `{"sponsorsInfo":{"sponsors":{"level0":[]},"content":{"title":"","description":""},"contact":{"title":"","description":"","buttonText":"","buttonLink":"javascript:alert(1)"},"rules":[]}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			conn, router := setupAdminPagesContractTest(t)
			t.Cleanup(func() {
				conn.Where("page_type = ?", pageConfig.SponsorsPage).Delete(&pageConfig.Entity{})
			})
			recorder := serveAdminPagesRaw(t, conn, router, http.MethodPost, "/api/admin/save-sponsors", body)
			env := decodeContractEnvelope(t, recorder)
			if env.Code != 1 || env.MessageCode != "admin.url.invalid" {
				t.Fatalf("envelope = code %d messageCode %q, want code 1 + admin.url.invalid", env.Code, env.MessageCode)
			}
		})
	}
}

func TestAdminSaveSponsorsAllowsContactMailto(t *testing.T) {
	conn, router := setupAdminPagesContractTest(t)
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.SponsorsPage).Delete(&pageConfig.Entity{})
	})
	body := `{"sponsorsInfo":{"sponsors":{"level1":[{"link":"https://b.example.com","message":"m","avatarUrl":"/static/sponsor.webp","name":"B"}]},"content":{"title":"","description":""},"contact":{"title":"","description":"","buttonText":"","buttonLink":"mailto:contact@example.com"},"rules":[]}}`
	recorder := serveAdminPagesRaw(t, conn, router, http.MethodPost, "/api/admin/save-sponsors", body)
	env := decodeContractEnvelope(t, recorder)
	if env.Code != 0 {
		t.Fatalf("envelope = %#v, want success", env)
	}
	hotdataserve.ClearSponsorsConfigCache()
	stored := pageConfig.GetConfigByPageType(pageConfig.SponsorsPage, pageConfig.SponsorsConfig{})
	if len(stored.Sponsors.Level1) != 1 || stored.Sponsors.Level1[0].Link != "https://b.example.com" {
		t.Fatalf("stored = %#v, want safe sponsor persisted", stored.Sponsors)
	}
	if stored.Contact.ButtonLink != "mailto:contact@example.com" {
		t.Fatalf("contact buttonLink = %q, want mailto preserved", stored.Contact.ButtonLink)
	}
}

func TestAdminSaveFriendLinksRejectsUnsafeURLs(t *testing.T) {
	cases := map[string]string{
		"relative url":      `{"linksInfo":[{"name":"g","links":[{"name":"A","desc":"d","url":"/relative","logoUrl":"","status":1}]}]}`,
		"javascript scheme": `{"linksInfo":[{"name":"g","links":[{"name":"A","desc":"d","url":"javascript:alert(1)","logoUrl":"","status":1}]}]}`,
		"unsafe logo":       `{"linksInfo":[{"name":"g","links":[{"name":"A","desc":"d","url":"https://a.example.com","logoUrl":"file:///etc/passwd","status":1}]}]}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			conn, router := setupAdminPagesContractTest(t)
			t.Cleanup(func() {
				conn.Where("page_type = ?", pageConfig.FriendShipLinks).Delete(&pageConfig.Entity{})
			})
			recorder := serveAdminPagesRaw(t, conn, router, http.MethodPost, "/api/admin/save-friend-links", body)
			env := decodeContractEnvelope(t, recorder)
			if env.Code != 1 || env.MessageCode != "admin.url.invalid" {
				t.Fatalf("envelope = code %d messageCode %q, want code 1 + admin.url.invalid", env.Code, env.MessageCode)
			}
		})
	}
}

func TestAdminSaveFriendLinksAllowsAbsoluteHTTP(t *testing.T) {
	conn, router := setupAdminPagesContractTest(t)
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.FriendShipLinks).Delete(&pageConfig.Entity{})
	})
	body := `{"linksInfo":[{"name":"g","links":[{"name":"A","desc":"d","url":"https://a.example.com","logoUrl":"/static/logo.webp","status":1}]}]}`
	recorder := serveAdminPagesRaw(t, conn, router, http.MethodPost, "/api/admin/save-friend-links", body)
	env := decodeContractEnvelope(t, recorder)
	if env.Code != 0 {
		t.Fatalf("envelope = %#v, want success", env)
	}
	hotdataserve.ClearFriendLinksConfigCache()
	stored := pageConfig.GetConfigByPageType(pageConfig.FriendShipLinks, []pageConfig.FriendLinksGroup(nil))
	if len(stored) != 1 || len(stored[0].Links) != 1 || stored[0].Links[0].Url != "https://a.example.com" {
		t.Fatalf("stored = %#v, want safe friend link persisted", stored)
	}
}

package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/buildinfo"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/themeservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 本文件覆盖 SiteManager 权限组 24 条站点设置路由的契约测试（issue #277 P3 切片五）。
// 各 pageConfig 行在子测试间清理，避免同进程共享库串扰；GET 成功场景一律显式持久化
// 确定性配置，不依赖 defaultconfig 内嵌默认值；security/posting/rateLimit 三类行由
// 共享 harness（setupHTTPContractTest）管理，本文件覆盖后由双方 cleanup 依次恢复。

// 契约主题配置：与 fixtures/admin-site-theme-success.json 完全一致（全字段、合法值，
// normalize 恒等）。名称必须是内建默认主题名 gf-light/gf-dark，否则 normalize 会把
// 未知名称改写成 fallback 主题名。
const contractSiteThemeLightJSON = `{"name":"gf-light","label":"契约亮色","colorScheme":"light","tokens":{` +
	`"color-base-100":"#ffffff","color-base-200":"#f6f6f6","color-base-300":"#e6e6e6",` +
	`"color-base-content":"#1a1a1a","color-icon-muted":"#6b6b6b","color-line":"#d6d6d6",` +
	`"color-primary":"#2563eb","color-primary-content":"#ffffff","color-secondary":"#0d9488",` +
	`"color-secondary-content":"#ffffff","color-accent":"#7c3aed","color-accent-content":"#ffffff",` +
	`"color-neutral":"#374151","color-neutral-content":"#ffffff","color-info":"#0284c7",` +
	`"color-info-content":"#ffffff","color-success":"#16a34a","color-success-content":"#ffffff",` +
	`"color-warning":"#d97706","color-warning-content":"#1a1a1a","color-error":"#dc2626",` +
	`"color-error-content":"#ffffff","radius-selector":"0.5rem","radius-field":"0.25rem",` +
	`"radius-box":"0.75rem","size-selector":"0.25rem","size-field":"0.25rem","border":"1px","depth":"0"}}`

const contractSiteThemeDarkJSON = `{"name":"gf-dark","label":"契约暗色","colorScheme":"dark","tokens":{` +
	`"color-base-100":"#1f2937","color-base-200":"#111827","color-base-300":"#030712",` +
	`"color-base-content":"#f9fafb","color-icon-muted":"#9ca3af","color-line":"#374151",` +
	`"color-primary":"#3b82f6","color-primary-content":"#ffffff","color-secondary":"#14b8a6",` +
	`"color-secondary-content":"#ffffff","color-accent":"#8b5cf6","color-accent-content":"#ffffff",` +
	`"color-neutral":"#4b5563","color-neutral-content":"#ffffff","color-info":"#38bdf8",` +
	`"color-info-content":"#ffffff","color-success":"#22c55e","color-success-content":"#ffffff",` +
	`"color-warning":"#f59e0b","color-warning-content":"#1a1a1a","color-error":"#ef4444",` +
	`"color-error-content":"#ffffff","radius-selector":"0.5rem","radius-field":"0.25rem",` +
	`"radius-box":"0.75rem","size-selector":"0.25rem","size-field":"0.25rem","border":"1px","depth":"0"}}`

const contractSiteThemeConfigJSON = `{"version":3,"enabled":true,"themes":[` +
	contractSiteThemeLightJSON + `,` + contractSiteThemeDarkJSON +
	`],"publishedAt":"2026-01-02T03:04:05Z"}`

// setupAdminSiteContractTest 在共享 harness（setupHTTPContractTest）之上注册
// SiteManager 权限组 24 条路由，中间件链与 route4api.go 的生产注册保持一致
// （JWTAuthCheck + CheckWritableAccount 公共链 + CheckPermission(SiteManager) 子组）。
// 站点设置相关缓存（含 onesystem 凭证缓存与主题缓存）在首尾清空，避免共享进程内串扰。
func setupAdminSiteContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(&rolePermissionRs.Entity{}); err != nil {
		t.Fatalf("migrate admin site contract tables: %v", err)
	}
	clearAdminSiteCaches()
	t.Cleanup(clearAdminSiteCaches)

	siteAPI := router.Group("/api/admin",
		middleware.JWTAuthCheck,
		middleware.CheckWritableAccount,
		middleware.CheckPermission(permission.SiteManager),
	)
	siteAPI.GET("/server-version", UpButterReq(api.ServerVersion))
	siteAPI.GET("/site-settings", UpButterReq(api.GetSiteSettings))
	siteAPI.POST("/save-site-settings", UpButterReq(api.SaveSiteSettings))
	siteAPI.GET("/site-chrome", UpButterReq(api.GetSiteChrome))
	siteAPI.POST("/save-site-chrome", UpButterReq(api.SaveSiteChrome))
	siteAPI.GET("/site-theme", UpButterReq(api.GetSiteTheme))
	siteAPI.POST("/save-site-theme", UpButterReq(api.SaveSiteTheme))
	siteAPI.POST("/publish-site-theme", UpButterReq(api.PublishSiteTheme))
	siteAPI.GET("/security-settings", UpButterReq(api.GetSecuritySettings))
	siteAPI.POST("/save-security-settings", UpButterReq(api.SaveSecuritySettings))
	siteAPI.GET("/posting-settings", UpButterReq(api.GetPostingSettings))
	siteAPI.POST("/save-posting-settings", UpButterReq(api.SavePostingSettings))
	siteAPI.GET("/rate-limit-settings", UpButterReq(api.GetRateLimitSettings))
	siteAPI.POST("/save-rate-limit-settings", UpButterReq(api.SaveRateLimitSettings))
	siteAPI.GET("/http-notify-settings", UpButterReq(api.GetHttpNotifySettings))
	siteAPI.POST("/save-http-notify-settings", UpButterReq(api.SaveHttpNotifySettings))
	siteAPI.GET("/onesystem-settings", UpButterReq(api.GetOnesystemSettings))
	siteAPI.POST("/save-onesystem-settings", UpButterReq(api.SaveOnesystemSettings))
	siteAPI.GET("/ai-summary-settings", UpButterReq(api.GetAiSummarySettings))
	siteAPI.POST("/save-ai-summary-settings", UpButterReq(api.SaveAiSummarySettings))
	siteAPI.GET("/terms-of-service", UpButterReq(api.GetTermsOfService))
	siteAPI.POST("/save-terms-of-service", UpButterReq(api.SaveTermsOfService))
	siteAPI.GET("/privacy-policy", UpButterReq(api.GetPrivacyPolicy))
	siteAPI.POST("/save-privacy-policy", UpButterReq(api.SavePrivacyPolicy))
	return conn, router
}

func clearAdminSiteCaches() {
	hotdataserve.ClearSiteSettingsConfigCache()
	hotdataserve.ClearSiteChromeConfigCache()
	hotdataserve.ClearHttpNotifyConfigCache()
	hotdataserve.ClearOnesystemSettingsConfigCache()
	hotdataserve.ClearAiSummarySettingsConfigCache()
	hotdataserve.ClearTermsOfServiceConfigCache()
	hotdataserve.ClearPrivacyPolicyConfigCache()
	themeservice.ClearCaches()
}

// createContractSiteManager 创建登录用户并授予 SiteManager 权限
// （复用 grantContractPermission：独立角色 ID，规避 10min 权限缓存串扰）。
func createContractSiteManager(t *testing.T, conn *gorm.DB) *users.EntityComplete {
	t.Helper()
	user := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, user.Id, permission.SiteManager)
	return user
}

// serveAdminSiteOK 以 SiteManager 身份调用路由并断言 HTTP 200 + fixture 信封。
func serveAdminSiteOK(t *testing.T, conn *gorm.DB, router *gin.Engine, method, path, body, fixture string) {
	t.Helper()
	recorder := serveAdminSiteRaw(t, conn, router, method, path, body)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixture))
}

// serveAdminSiteRaw 以 SiteManager 身份调用路由，返回原始 recorder 供动态结果的结构化断言。
func serveAdminSiteRaw(t *testing.T, conn *gorm.DB, router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	manager := createContractSiteManager(t, conn)
	recorder := serveAuthSecurityJSON(router, method, path, body, contractSessionToken(t, manager))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	return recorder
}

// adminSiteGuardScenarios 跑 24 条路由公共的中间件守卫场景：
// 未登录 401 / 冻结账号 403 / 无 SiteManager 权限 403（params.permission="站点管理"）。
func adminSiteGuardScenarios(t *testing.T, method, path, fixturePrefix string) {
	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAdminSiteContractTest(t)
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixturePrefix+"-unauthenticated.json"))
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		if err := conn.Model(user).Update("is_frozen", users.StatusFrozen).Error; err != nil {
			t.Fatalf("freeze contract user: %v", err)
		}
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("frozen account status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixturePrefix+"-forbidden.json"))
	})

	t.Run("user without SiteManager returns 403", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("permission denied status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixturePrefix+"-permission-denied.json"))
	})
}

// decodeSiteResult 把成功信封的 result 解码为 map 供动态字段（时间戳等）结构化断言。
func decodeSiteResult(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	response := decodeContractEnvelope(t, recorder)
	if response.Code != 0 {
		t.Fatalf("envelope = %#v, want success", response)
	}
	var result map[string]any
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("decode result %s: %v", response.Result, err)
	}
	return result
}

func TestAdminServerVersionHTTPContract(t *testing.T) {
	path := "/api/admin/server-version"

	t.Run("success returns the compiled build metadata", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		recorder := serveAdminSiteRaw(t, conn, router, http.MethodGet, path, "")
		response := decodeContractEnvelope(t, recorder)
		var info buildinfo.Info
		if err := json.Unmarshal(response.Result, &info); err != nil {
			t.Fatalf("decode server version result %s: %v", response.Result, err)
		}
		if want := buildinfo.Get(); info != want {
			t.Fatalf("server version = %#v, want buildinfo.Get() %#v", info, want)
		}
	})

	adminSiteGuardScenarios(t, http.MethodGet, path, "admin-server-version")
}

func TestAdminGetSiteSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/site-settings"

	t.Run("success returns the stored site settings", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.SiteSettings, pageConfig.SiteSettingsConfig{
			SiteName:        "契约站点",
			SiteLogo:        "/static/logo.webp",
			SiteDescription: "契约描述",
			SiteKeywords:    "契约,测试",
			SiteUrl:         "https://contract.example.test",
			SiteEmail:       "hi@contract.example.test",
			ExternalLinks:   "https://example.test",
		})
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-site-settings-success.json")
	})

	adminSiteGuardScenarios(t, http.MethodGet, path, "admin-site-settings")
}

func TestAdminSaveSiteSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/save-site-settings"

	t.Run("success replaces the stored site settings", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.SiteSettings).Delete(&pageConfig.Entity{})
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"siteName":"新站点","siteLogo":"","siteDescription":"d","siteKeywords":"k","siteUrl":"https://new.example.test","siteEmail":"","externalLinks":""}}`,
			"admin-save-site-settings-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.SiteSettings, pageConfig.SiteSettingsConfig{})
		if stored.SiteName != "新站点" || stored.SiteUrl != "https://new.example.test" {
			t.Fatalf("stored site settings = %#v, want submitted values", stored)
		}
	})

	adminSiteGuardScenarios(t, http.MethodPost, path, "admin-save-site-settings")
}

func TestAdminGetSiteChromeHTTPContract(t *testing.T) {
	path := "/api/admin/site-chrome"

	t.Run("success returns the stored chrome configuration", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.SiteChrome, pageConfig.SiteChromeConfig{
			Header: []pageConfig.ChromeItem{
				{ID: "sponsors", Enabled: true, Type: "link", Label: "Sponsors", I18nLabel: "shell.nav.sponsors", URL: "/sponsors"},
			},
			MainMenu: []pageConfig.ChromeItem{
				{ID: "home", Enabled: true, Type: "link", Label: "首页", I18nLabel: "shell.nav.home", URL: "/"},
			},
			Resources: []pageConfig.ChromeItem{
				{ID: "docs", Enabled: false, Type: "link", Label: "文档", I18nLabel: "shell.nav.docs", URL: "/docs"},
			},
			SidebarGroups: []pageConfig.ChromeGroup{
				{ID: "g1", Title: "契约分组", I18nLabel: "shell.group.contract", Items: []pageConfig.ChromeItem{
					{ID: "i1", Enabled: true, Type: "link", Label: "条目", I18nLabel: "shell.item.contract", URL: "/item"},
				}},
			},
			FooterInfo: pageConfig.FooterInfo{
				Primary: []pageConfig.PItem{{Content: "契约页脚"}},
				List:    []pageConfig.FooterItem{{Name: "GitHub", Url: "https://example.test"}},
			},
			BrandType: "text",
			BrandText: "契约品牌",
		})
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-site-chrome-success.json")
	})

	adminSiteGuardScenarios(t, http.MethodGet, path, "admin-site-chrome")
}

func TestAdminSaveSiteChromeHTTPContract(t *testing.T) {
	path := "/api/admin/save-site-chrome"

	t.Run("success replaces the stored chrome configuration", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.SiteChrome).Delete(&pageConfig.Entity{})
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"header":[],"mainMenu":[],"resources":[],"sidebarGroups":[],"footerInfo":{"primary":[],"list":[]},"brandType":"text","brandText":"新品牌","brandImage":""}}`,
			"admin-save-site-chrome-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.SiteChrome, pageConfig.SiteChromeConfig{})
		if stored.BrandText != "新品牌" {
			t.Fatalf("stored chrome = %#v, want submitted brandText", stored)
		}
	})

	adminSiteGuardScenarios(t, http.MethodPost, path, "admin-save-site-chrome")
}

func TestAdminGetSiteThemeHTTPContract(t *testing.T) {
	path := "/api/admin/site-theme"

	t.Run("success returns the normalized stored theme configuration", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.SiteTheme, json.RawMessage(contractSiteThemeConfigJSON))
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-site-theme-success.json")
	})

	adminSiteGuardScenarios(t, http.MethodGet, path, "admin-site-theme")
}

func TestAdminSaveSiteThemeHTTPContract(t *testing.T) {
	path := "/api/admin/save-site-theme"

	t.Run("success stages an unpublished draft and keeps the published themes", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.SiteTheme, json.RawMessage(contractSiteThemeConfigJSON))
		recorder := serveAdminSiteRaw(t, conn, router, http.MethodPost, path,
			`{"settings":{"enabled":false,"themes":[`+contractSiteThemeLightJSON+`,`+contractSiteThemeDarkJSON+`]}}`)
		result := decodeSiteResult(t, recorder)
		if result["enabled"] != true {
			t.Fatalf("result.enabled = %#v, want the previously published value true", result["enabled"])
		}
		if result["publishedAt"] != "2026-01-02T03:04:05Z" {
			t.Fatalf("result.publishedAt = %#v, want the previously published timestamp", result["publishedAt"])
		}
		prepublish, ok := result["prepublish"].(map[string]any)
		if !ok {
			t.Fatalf("result.prepublish = %#v, want staged draft", result["prepublish"])
		}
		if prepublish["enabled"] != false {
			t.Fatalf("prepublish.enabled = %#v, want submitted false", prepublish["enabled"])
		}
		if updatedAt, _ := prepublish["updatedAt"].(string); updatedAt == "" {
			t.Fatal("prepublish.updatedAt is empty, want a server-stamped RFC 3339 time")
		}
		themes, _ := prepublish["themes"].([]any)
		if len(themes) != 2 {
			t.Fatalf("prepublish.themes = %#v, want two staged themes", prepublish["themes"])
		}
		stored := themeservice.LoadConfig()
		if stored.Prepublish == nil || stored.Prepublish.Enabled {
			t.Fatalf("stored prepublish = %#v, want the staged draft with enabled=false", stored.Prepublish)
		}
	})

	adminSiteGuardScenarios(t, http.MethodPost, path, "admin-save-site-theme")
}

func TestAdminPublishSiteThemeHTTPContract(t *testing.T) {
	path := "/api/admin/publish-site-theme"

	t.Run("success promotes the staged draft and stamps publishedAt", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.SiteTheme, json.RawMessage(contractSiteThemeConfigJSON))
		serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/save-site-theme",
			`{"settings":{"enabled":false,"themes":[`+contractSiteThemeLightJSON+`,`+contractSiteThemeDarkJSON+`]}}`)
		result := decodeSiteResult(t, serveAdminSiteRaw(t, conn, router, http.MethodPost, path, `{}`))
		if result["enabled"] != false {
			t.Fatalf("result.enabled = %#v, want the promoted draft value false", result["enabled"])
		}
		if _, exists := result["prepublish"]; exists {
			t.Fatalf("result.prepublish = %#v, want the draft cleared after publish", result["prepublish"])
		}
		publishedAt, _ := result["publishedAt"].(string)
		if publishedAt == "" || publishedAt == "2026-01-02T03:04:05Z" {
			t.Fatalf("result.publishedAt = %q, want a freshly stamped RFC 3339 time", publishedAt)
		}
	})

	t.Run("without a draft the publish is a no-op", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.SiteTheme, json.RawMessage(contractSiteThemeConfigJSON))
		result := decodeSiteResult(t, serveAdminSiteRaw(t, conn, router, http.MethodPost, path, `{}`))
		if result["publishedAt"] != "2026-01-02T03:04:05Z" {
			t.Fatalf("result.publishedAt = %#v, want the untouched stored timestamp", result["publishedAt"])
		}
		if _, exists := result["prepublish"]; exists {
			t.Fatalf("result.prepublish = %#v, want no draft", result["prepublish"])
		}
	})

	adminSiteGuardScenarios(t, http.MethodPost, path, "admin-publish-site-theme")
}

func TestAdminGetSecuritySettingsHTTPContract(t *testing.T) {
	path := "/api/admin/security-settings"

	t.Run("success returns the stored security settings", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.SecuritySettings, pageConfig.SecurityAndRegistration{
			EnableSignup:            true,
			EnableEmailVerification: false,
			AllowedDomains:          []string{"tongji.edu.cn"},
			ReservedUsernames:       []string{"admin"},
			BannedUsernames:         []string{"banned-contract"},
			SensitiveWords:          []string{"违禁词"},
			SensitiveAction:         "review",
			CaptchaRequired:         false,
		})
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-security-settings-success.json")
	})

	adminSiteGuardScenarios(t, http.MethodGet, path, "admin-security-settings")
}

func TestAdminSaveSecuritySettingsHTTPContract(t *testing.T) {
	path := "/api/admin/save-security-settings"

	t.Run("success replaces the stored security settings", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"enableSignup":false,"enableEmailVerification":true,"allowedDomains":[],"reservedUsernames":[],"bannedUsernames":["banned-contract-new"],"sensitiveWords":[],"sensitiveAction":"block","captchaRequired":true}}`,
			"admin-save-security-settings-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.SecuritySettings, pageConfig.SecurityAndRegistration{})
		if stored.EnableSignup || !stored.CaptchaRequired || stored.SensitiveAction != "block" {
			t.Fatalf("stored security settings = %#v, want submitted values", stored)
		}
		if len(stored.BannedUsernames) != 1 || stored.BannedUsernames[0] != "banned-contract-new" {
			t.Fatalf("stored bannedUsernames = %#v, want the submitted entry", stored.BannedUsernames)
		}
	})

	adminSiteGuardScenarios(t, http.MethodPost, path, "admin-save-security-settings")
}

func TestAdminGetPostingSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/posting-settings"

	t.Run("success returns the stored posting settings", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		posting := pageConfig.PostingContent{}
		posting.TextControl.MinPostLength = 5
		posting.TextControl.MaxPostLength = 20000
		posting.TextControl.MinTitleLength = 4
		posting.TextControl.MaxTitleLength = 120
		posting.TextControl.NewUserPostCooldownMinutes = 10
		posting.UploadControl.AllowAttachments = true
		posting.UploadControl.AuthorizedExtensions = []string{"png", "jpg"}
		posting.UploadControl.MaxAttachmentSizeKb = 2048
		posting.UploadControl.MaxDailyUploadsPerUser = 20
		posting.UploadControl.NewUserUploadCooldownMinutes = 30
		posting.LLMS = pageConfig.LLMSConfig{Enabled: true, FullText: false, Files: true}
		persistContractPageConfig(t, conn, pageConfig.PostingSettings, posting)
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-posting-settings-success.json")
	})

	adminSiteGuardScenarios(t, http.MethodGet, path, "admin-posting-settings")
}

func TestAdminSavePostingSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/save-posting-settings"

	t.Run("success replaces the stored posting settings", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"textControl":{"minPostLength":1,"maxPostLength":1000,"minTitleLength":2,"maxTitleLength":50,"newUserPostCooldownMinutes":0},"uploadControl":{"allowAttachments":false,"authorizedExtensions":["webp"],"maxAttachmentSizeKb":512,"maxDailyUploadsPerUser":5,"newUserUploadCooldownMinutes":0},"llms":{"enabled":false,"fullText":true,"files":false}}}`,
			"admin-save-posting-settings-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.PostingSettings, pageConfig.PostingContent{})
		if stored.TextControl.MaxPostLength != 1000 || stored.UploadControl.MaxAttachmentSizeKb != 512 || !stored.LLMS.FullText {
			t.Fatalf("stored posting settings = %#v, want submitted values", stored)
		}
	})

	adminSiteGuardScenarios(t, http.MethodPost, path, "admin-save-posting-settings")
}

func TestAdminGetRateLimitSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/rate-limit-settings"

	t.Run("success returns the stored rate-limit settings", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.RateLimitSettings, pageConfig.RateLimitConfig{
			Enabled:   true,
			SkipAdmin: true,
			Actions: []pageConfig.RateLimitRule{
				{Action: "topic-write", WindowSeconds: 60, LimitPerIp: 5, LimitPerUser: 3},
			},
			NewUserCaptchaAfterPosts: 3,
			NewUserCaptchaDays:       7,
			MinSubmitSeconds:         2,
		})
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-rate-limit-settings-success.json")
	})

	adminSiteGuardScenarios(t, http.MethodGet, path, "admin-rate-limit-settings")
}

func TestAdminSaveRateLimitSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/save-rate-limit-settings"

	t.Run("success replaces the stored rate-limit settings", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"enabled":false,"skipAdmin":false,"actions":[{"action":"login","windowSeconds":30,"limitPerIp":10,"limitPerUser":0}],"newUserCaptchaAfterPosts":0,"newUserCaptchaDays":0,"minSubmitSeconds":0}}`,
			"admin-save-rate-limit-settings-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.RateLimitSettings, pageConfig.RateLimitConfig{})
		if stored.Enabled || len(stored.Actions) != 1 || stored.Actions[0].Action != "login" {
			t.Fatalf("stored rate-limit settings = %#v, want submitted values", stored)
		}
	})

	adminSiteGuardScenarios(t, http.MethodPost, path, "admin-save-rate-limit-settings")
}

func TestAdminGetHttpNotifySettingsHTTPContract(t *testing.T) {
	path := "/api/admin/http-notify-settings"

	t.Run("success returns the stored notify settings including the cleartext secret", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.HttpNotify, pageConfig.HttpNotifyConfig{
			Enabled: true,
			Endpoints: []pageConfig.HttpNotifyEndpoint{
				{Id: "ep1", Name: "契约 webhook", Enabled: true, URL: "https://hook.example.test/notify",
					Secret: "contract-secret", Events: []string{"topic.created"}, TimeoutSeconds: 5},
			},
		})
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-http-notify-settings-success.json")
	})

	adminSiteGuardScenarios(t, http.MethodGet, path, "admin-http-notify-settings")
}

func TestAdminSaveHttpNotifySettingsHTTPContract(t *testing.T) {
	path := "/api/admin/save-http-notify-settings"

	t.Run("success replaces the stored notify settings", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.HttpNotify).Delete(&pageConfig.Entity{})
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"enabled":true,"endpoints":[{"id":"ep9","name":"新 webhook","enabled":true,"url":"https://new.example.test/hook","secret":"new-secret","events":["post.created"],"timeoutSeconds":3,"failureCount":0,"lastError":"","abnormalTerminated":false}]}}`,
			"admin-save-http-notify-settings-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.HttpNotify, pageConfig.HttpNotifyConfig{})
		if len(stored.Endpoints) != 1 || stored.Endpoints[0].Id != "ep9" || stored.Endpoints[0].Secret != "new-secret" {
			t.Fatalf("stored notify settings = %#v, want submitted endpoint", stored.Endpoints)
		}
	})

	adminSiteGuardScenarios(t, http.MethodPost, path, "admin-save-http-notify-settings")
}

func TestAdminGetOnesystemSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/onesystem-settings"

	t.Run("success reports only whether the credential is configured", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		conn.Where("page_type = ?", pageConfig.OneSystemSettings).Delete(&pageConfig.Entity{})
		hotdataserve.ClearOnesystemSettingsConfigCache()
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.OneSystemSettings).Delete(&pageConfig.Entity{})
			hotdataserve.ClearOnesystemSettingsConfigCache()
		})
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-onesystem-settings-success.json")
	})

	adminSiteGuardScenarios(t, http.MethodGet, path, "admin-onesystem-settings")
}

func TestAdminSaveOnesystemSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/save-onesystem-settings"

	t.Run("success encrypts the cookie and stores only the ciphertext", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.OneSystemSettings).Delete(&pageConfig.Entity{})
			hotdataserve.ClearOnesystemSettingsConfigCache()
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"cookie":"session=contract-cookie"}`, "admin-save-onesystem-settings-success.json")
		stored := pageConfig.GetByPageType(pageConfig.OneSystemSettings)
		if !strings.Contains(stored.Config, `"cookieEncrypted"`) {
			t.Fatalf("stored onesystem config = %q, want a cookieEncrypted field", stored.Config)
		}
		if strings.Contains(stored.Config, "contract-cookie") {
			t.Fatalf("stored onesystem config = %q, want no plaintext cookie", stored.Config)
		}
		result := decodeSiteResult(t, serveAdminSiteRaw(t, conn, router, http.MethodGet, "/api/admin/onesystem-settings", ""))
		if result["cookieConfigured"] != true {
			t.Fatalf("cookieConfigured = %#v, want true after saving a cookie", result["cookieConfigured"])
		}
	})

	t.Run("blank cookie clears the stored credential", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.OneSystemSettings).Delete(&pageConfig.Entity{})
			hotdataserve.ClearOnesystemSettingsConfigCache()
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"cookie":"session=contract-cookie"}`, "admin-save-onesystem-settings-success.json")
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"cookie":"  "}`, "admin-save-onesystem-settings-success.json")
		result := decodeSiteResult(t, serveAdminSiteRaw(t, conn, router, http.MethodGet, "/api/admin/onesystem-settings", ""))
		if result["cookieConfigured"] != false {
			t.Fatalf("cookieConfigured = %#v, want false after clearing", result["cookieConfigured"])
		}
	})

	t.Run("cookie longer than 4096 characters fails validation", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		manager := createContractSiteManager(t, conn)
		recorder := serveAuthSecurityJSON(router, http.MethodPost, path,
			`{"cookie":"`+strings.Repeat("a", 4097)+`"}`, contractSessionToken(t, manager))
		if recorder.Code != http.StatusOK {
			t.Fatalf("validation failure status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-save-onesystem-settings-invalid-params.json"))
	})

	adminSiteGuardScenarios(t, http.MethodPost, path, "admin-save-onesystem-settings")
}

func TestAdminGetAiSummarySettingsHTTPContract(t *testing.T) {
	path := "/api/admin/ai-summary-settings"

	t.Run("success returns the stored AI summary settings", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.AiSummarySettings, pageConfig.AiSummaryConfig{
			Enabled:         true,
			GlobalPerMinute: 10,
		})
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-ai-summary-settings-success.json")
	})

	adminSiteGuardScenarios(t, http.MethodGet, path, "admin-ai-summary-settings")
}

func TestAdminSaveAiSummarySettingsHTTPContract(t *testing.T) {
	path := "/api/admin/save-ai-summary-settings"

	t.Run("success replaces the stored AI summary settings", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.AiSummarySettings).Delete(&pageConfig.Entity{})
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"enabled":true,"globalPerMinute":20}}`, "admin-save-ai-summary-settings-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.AiSummarySettings, pageConfig.AiSummaryConfig{})
		if !stored.Enabled || stored.GlobalPerMinute != 20 {
			t.Fatalf("stored AI summary settings = %#v, want submitted values", stored)
		}
	})

	adminSiteGuardScenarios(t, http.MethodPost, path, "admin-save-ai-summary-settings")
}

func TestAdminGetTermsOfServiceHTTPContract(t *testing.T) {
	path := "/api/admin/terms-of-service"

	t.Run("success returns the stored terms-of-service configuration", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.TermsOfService, pageConfig.TermsOfServiceConfig{
			Enabled: true,
			Content: "# 服务条款\n契约内容",
		})
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-terms-of-service-success.json")
	})

	adminSiteGuardScenarios(t, http.MethodGet, path, "admin-terms-of-service")
}

func TestAdminSaveTermsOfServiceHTTPContract(t *testing.T) {
	path := "/api/admin/save-terms-of-service"

	t.Run("success replaces the stored terms without persisting rendered HTML", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.TermsOfService).Delete(&pageConfig.Entity{})
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"enabled":true,"content":"# 新条款"}}`, "admin-save-terms-of-service-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.TermsOfService, pageConfig.TermsOfServiceConfig{})
		if !stored.Enabled || stored.Content != "# 新条款" {
			t.Fatalf("stored terms = %#v, want submitted values", stored)
		}
		entity := pageConfig.GetByPageType(pageConfig.TermsOfService)
		if strings.Contains(entity.Config, "htmlContent") {
			t.Fatalf("stored terms config = %q, want no rendered HTML field", entity.Config)
		}
	})

	adminSiteGuardScenarios(t, http.MethodPost, path, "admin-save-terms-of-service")
}

func TestAdminGetPrivacyPolicyHTTPContract(t *testing.T) {
	path := "/api/admin/privacy-policy"

	t.Run("success returns the stored privacy-policy configuration", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.PrivacyPolicy, pageConfig.PrivacyPolicyConfig{
			Enabled: true,
			Content: "# 隐私政策\n契约内容",
		})
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-privacy-policy-success.json")
	})

	adminSiteGuardScenarios(t, http.MethodGet, path, "admin-privacy-policy")
}

func TestAdminSavePrivacyPolicyHTTPContract(t *testing.T) {
	path := "/api/admin/save-privacy-policy"

	t.Run("success replaces the stored policy without persisting rendered HTML", func(t *testing.T) {
		conn, router := setupAdminSiteContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.PrivacyPolicy).Delete(&pageConfig.Entity{})
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"enabled":false,"content":"# 新政策"}}`, "admin-save-privacy-policy-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.PrivacyPolicy, pageConfig.PrivacyPolicyConfig{})
		if stored.Enabled || stored.Content != "# 新政策" {
			t.Fatalf("stored policy = %#v, want submitted values", stored)
		}
		entity := pageConfig.GetByPageType(pageConfig.PrivacyPolicy)
		if strings.Contains(entity.Config, "htmlContent") {
			t.Fatalf("stored policy config = %q, want no rendered HTML field", entity.Config)
		}
	})

	adminSiteGuardScenarios(t, http.MethodPost, path, "admin-save-privacy-policy")
}

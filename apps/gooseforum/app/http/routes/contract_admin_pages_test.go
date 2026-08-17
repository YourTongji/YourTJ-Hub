package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 本文件覆盖 PageManager 权限组 6 条页面设置路由的契约测试（issue #277 P3 切片四）。
// friendShipLinks / sponsors / announcement 三类 pageConfig 行在各子测试间清理，
// 避免同进程共享库中的配置串扰；GET 成功场景一律显式持久化确定性配置，
// 不依赖 defaultconfig 内嵌默认值。

// setupAdminPagesContractTest 在共享 harness（setupHTTPContractTest）之上注册
// PageManager 权限组 6 条路由，中间件链与 route4api.go 的生产注册保持一致
// （JWTAuthCheck + CheckWritableAccount 公共链 + CheckPermission(PageManager) 子组）。
func setupAdminPagesContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(&rolePermissionRs.Entity{}); err != nil {
		t.Fatalf("migrate admin pages contract tables: %v", err)
	}

	pagesAPI := router.Group("/api/admin",
		middleware.JWTAuthCheck,
		middleware.CheckWritableAccount,
		middleware.CheckPermission(permission.PageManager),
	)
	pagesAPI.GET("/friend-links", UpButterReq(api.GetFriendLinks))
	pagesAPI.POST("/save-friend-links", UpButterReq(api.SaveFriendLinks))
	pagesAPI.GET("/sponsors", UpButterReq(api.GetSponsors))
	pagesAPI.POST("/save-sponsors", UpButterReq(api.SaveSponsors))
	pagesAPI.GET("/announcement", UpButterReq(api.GetAnnouncement))
	pagesAPI.POST("/save-announcement", UpButterReq(api.SaveAnnouncement))
	return conn, router
}

// createContractPageManager 创建登录用户并授予 PageManager 权限
// （复用 grantContractPermission：独立角色 ID，规避 10min 权限缓存串扰）。
func createContractPageManager(t *testing.T, conn *gorm.DB) *users.EntityComplete {
	t.Helper()
	user := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, user.Id, permission.PageManager)
	return user
}

// persistContractPageConfig 持久化确定性页面配置并在子测试结束后删除对应行。
func persistContractPageConfig(t *testing.T, conn *gorm.DB, pageType string, config any) {
	t.Helper()
	persistHTTPContractConfig(t, conn, pageType, config)
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageType).Delete(&pageConfig.Entity{})
	})
}

// serveAdminPagesOK 以 PageManager 身份调用路由并断言 HTTP 200 + fixture 信封。
func serveAdminPagesOK(t *testing.T, conn *gorm.DB, router *gin.Engine, method, path, body, fixture string) {
	t.Helper()
	recorder := serveAdminPagesRaw(t, conn, router, method, path, body)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixture))
}

// serveAdminPagesRaw 以 PageManager 身份调用路由，返回原始 recorder 供结构化断言。
func serveAdminPagesRaw(t *testing.T, conn *gorm.DB, router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	manager := createContractPageManager(t, conn)
	recorder := serveAuthSecurityJSON(router, method, path, body, contractSessionToken(t, manager))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	return recorder
}

// adminPagesGuardScenarios 跑 6 条路由公共的中间件守卫场景：
// 未登录 401 / 冻结账号 403 / 无 PageManager 权限 403（params.permission="页面管理"）。
func adminPagesGuardScenarios(t *testing.T, method, path, fixturePrefix string) {
	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAdminPagesContractTest(t)
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "auth-required.json"))
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAdminPagesContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		if err := conn.Model(user).Update("is_frozen", users.StatusFrozen).Error; err != nil {
			t.Fatalf("freeze contract user: %v", err)
		}
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("frozen account status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "account-frozen.json"))
	})

	t.Run("user without PageManager returns 403", func(t *testing.T) {
		conn, router := setupAdminPagesContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("permission denied status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-announcement-permission-denied.json"))
	})
}

func TestAdminGetFriendLinksHTTPContract(t *testing.T) {
	path := "/api/admin/friend-links"

	t.Run("success returns the stored groups with null links normalized", func(t *testing.T) {
		conn, router := setupAdminPagesContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.FriendShipLinks, []pageConfig.FriendLinksGroup{
			{
				Name:  "契约分组",
				Emoji: "🔗",
				Color: "#123456",
				Links: []pageConfig.LinkItem{
					{Name: "同济", Desc: "契约链接", Url: "https://example.test", LogoUrl: "/static/logo.webp", Status: 1},
				},
			},
			{},
		})
		serveAdminPagesOK(t, conn, router, http.MethodGet, path, "", "admin-friend-links-success.json")
	})

	adminPagesGuardScenarios(t, http.MethodGet, path, "admin-friend-links")
}

func TestAdminSaveFriendLinksHTTPContract(t *testing.T) {
	path := "/api/admin/save-friend-links"

	t.Run("success replaces the stored configuration", func(t *testing.T) {
		conn, router := setupAdminPagesContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.FriendShipLinks).Delete(&pageConfig.Entity{})
		})
		serveAdminPagesOK(t, conn, router, http.MethodPost, path,
			`{"linksInfo":[{"name":"新分组","links":[{"name":"A","desc":"d","url":"https://a.test","logoUrl":"","status":1}]}]}`,
			"admin-agent-disable-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.FriendShipLinks, []pageConfig.FriendLinksGroup(nil))
		if len(stored) != 1 || stored[0].Name != "新分组" || len(stored[0].Links) != 1 {
			t.Fatalf("stored friend links = %#v, want one group with one link", stored)
		}
	})

	adminPagesGuardScenarios(t, http.MethodPost, path, "admin-save-friend-links")
}

func TestAdminGetSponsorsHTTPContract(t *testing.T) {
	path := "/api/admin/sponsors"

	t.Run("success returns the stored sponsors configuration", func(t *testing.T) {
		conn, router := setupAdminPagesContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.SponsorsPage, pageConfig.SponsorsConfig{
			Sponsors: pageConfig.Sponsors{
				Level0: []pageConfig.SponsorItem{
					{Link: "https://example.test", Message: "契约赞助", AvatarUrl: "/static/sponsor.webp", Name: "契约赞助商"},
				},
				Level1: []pageConfig.SponsorItem{},
				Level2: []pageConfig.SponsorItem{},
				Level3: []pageConfig.SponsorItem{},
			},
			Content: pageConfig.SponsorsPageIntro{Title: "赞助", Description: "感谢支持"},
			Contact: pageConfig.SponsorsContact{
				Title: "成为赞助商", Description: "欢迎联系", ButtonText: "联系我们", ButtonLink: "mailto:hi@example.test",
			},
			Rules: []pageConfig.SponsorsRule{{Content: "链接需稳定"}},
		})
		serveAdminPagesOK(t, conn, router, http.MethodGet, path, "", "admin-sponsors-success.json")
	})

	adminPagesGuardScenarios(t, http.MethodGet, path, "admin-sponsors")
}

func TestAdminSaveSponsorsHTTPContract(t *testing.T) {
	path := "/api/admin/save-sponsors"

	t.Run("success replaces the stored configuration and fills blank defaults", func(t *testing.T) {
		conn, router := setupAdminPagesContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.SponsorsPage).Delete(&pageConfig.Entity{})
		})
		serveAdminPagesOK(t, conn, router, http.MethodPost, path,
			`{"sponsorsInfo":{"sponsors":{"level1":[{"link":"https://b.test","message":"m","avatarUrl":"","name":"B"}]},"content":{"title":"赞助","description":"感谢"},"contact":{"title":"","description":"","buttonText":"","buttonLink":""},"rules":[]}}`,
			"admin-agent-disable-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.SponsorsPage, pageConfig.SponsorsConfig{})
		if len(stored.Sponsors.Level1) != 1 || stored.Sponsors.Level1[0].Name != "B" {
			t.Fatalf("stored sponsors = %#v, want one level1 sponsor", stored.Sponsors)
		}
		if stored.Sponsors.Level0 == nil || len(stored.Sponsors.Level0) != 0 {
			t.Fatalf("stored level0 = %#v, want empty non-null array", stored.Sponsors.Level0)
		}
		if stored.Contact.Title == "" {
			t.Fatal("blank contact title was not filled from the built-in default")
		}
	})

	adminPagesGuardScenarios(t, http.MethodPost, path, "admin-save-sponsors")
}

func TestAdminGetAnnouncementHTTPContract(t *testing.T) {
	path := "/api/admin/announcement"

	t.Run("success returns the stored announcement configuration", func(t *testing.T) {
		conn, router := setupAdminPagesContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.Announcement, pageConfig.AnnouncementConfig{
			Enabled:     true,
			Content:     "契约公告",
			PublishedAt: "2026-01-02T03:04:05Z",
			Items: []pageConfig.AnnouncementItem{
				{ID: "a1", Title: "公告标题", Content: "公告内容", Enabled: true},
			},
		})
		serveAdminPagesOK(t, conn, router, http.MethodGet, path, "", "admin-announcement-success.json")
	})

	adminPagesGuardScenarios(t, http.MethodGet, path, "admin-announcement")
}

func TestAdminSaveAnnouncementHTTPContract(t *testing.T) {
	path := "/api/admin/save-announcement"

	t.Run("success replaces the stored configuration and stamps publishedAt", func(t *testing.T) {
		conn, router := setupAdminPagesContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.Announcement).Delete(&pageConfig.Entity{})
		})
		serveAdminPagesOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"enabled":true,"content":"新公告","publishedAt":"2000-01-01T00:00:00Z"}}`,
			"admin-agent-disable-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.Announcement, pageConfig.AnnouncementConfig{})
		if !stored.Enabled || stored.Content != "新公告" {
			t.Fatalf("stored announcement = %#v, want enabled with submitted content", stored)
		}
		if stored.PublishedAt == "" || stored.PublishedAt == "2000-01-01T00:00:00Z" {
			t.Fatalf("publishedAt = %q, want a server-stamped current time", stored.PublishedAt)
		}
	})

	t.Run("empty body saves zero-value settings with a stamped publishedAt", func(t *testing.T) {
		// Settings 的 validate:"required" 对结构体零值不生效（validator 对 struct 字段
		// 的 required 恒通过），宽松绑定下的空/畸形 body 会按零值配置保存。
		conn, router := setupAdminPagesContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.Announcement).Delete(&pageConfig.Entity{})
		})
		serveAdminPagesOK(t, conn, router, http.MethodPost, path, `{}`, "admin-agent-disable-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.Announcement, pageConfig.AnnouncementConfig{})
		if stored.Enabled || stored.Content != "" {
			t.Fatalf("stored announcement = %#v, want zero-value settings", stored)
		}
		if stored.PublishedAt == "" {
			t.Fatal("publishedAt is empty, want a server-stamped current time")
		}
	})

	adminPagesGuardScenarios(t, http.MethodPost, path, "admin-save-announcement")
}

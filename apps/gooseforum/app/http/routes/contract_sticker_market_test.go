package routes

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/safefetch"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/stickerservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 本文件覆盖贴纸市场（issue #1429）的公开浏览、账号订阅与管理端 6 条路由。
// 种子数据与 fixtures 中的固定 ID 一一对应：9201/9202 两个包、9301/9302 两张
// 已入库贴纸；上游请求一律由 contractMarketFetcher 应答，测试从不访问真实市场。

var (
	contractMarketFileA = stickerservice.MarketAssetPrefix + strings.Repeat("a", 64) + ".png"
	contractMarketFileB = stickerservice.MarketAssetPrefix + strings.Repeat("b", 64) + ".png"
	contractMarketSync  = time.Date(2026, 10, 29, 8, 0, 0, 0, time.UTC)
)

type contractMarketFetcher map[string]string

func (f contractMarketFetcher) FetchJSON(_ context.Context, raw string) (safefetch.Result, error) {
	final, _ := url.Parse(raw)
	body, ok := f[raw]
	if !ok {
		return safefetch.Result{}, &safefetch.FetchError{Class: safefetch.ErrorTimeout}
	}
	return safefetch.Result{StatusCode: http.StatusOK, FinalURL: final, ContentType: "application/json", Body: []byte(body)}, nil
}

func (f contractMarketFetcher) FetchImage(_ context.Context, _ string) (safefetch.Result, error) {
	return safefetch.Result{}, &safefetch.FetchError{Class: safefetch.ErrorTimeout}
}

func setupStickerMarketContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupAdminStickersContractTest(t)
	if err := conn.AutoMigrate(&sticker.MarketTopic{}, &sticker.MarketPack{}, &sticker.MarketItem{}, &taskQueue.Entity{}); err != nil {
		t.Fatalf("migrate sticker market tables: %v", err)
	}
	clean := func() {
		for _, model := range []any{&sticker.MarketTopic{}, &sticker.MarketPack{}, &sticker.MarketItem{}, &sticker.PackSubscription{}, &sticker.LibraryOwner{}} {
			conn.Where("1 = 1").Delete(model)
		}
		conn.Where("pack = ?", sticker.PackMarket).Delete(&sticker.Entity{})
		conn.Where("type = ?", stickerservice.TaskTypeMarketImport).Delete(&taskQueue.Entity{})
		db4fileconnect.Connect().Where("name LIKE ?", stickerservice.MarketAssetPrefix+"%").Delete(&filedata.Entity{})
		stickerservice.InvalidateCache()
	}
	clean()
	t.Cleanup(clean)
	restore := stickerservice.SwapMarketFetcher(contractMarketFetcher{})
	t.Cleanup(restore)

	router.GET("/api/forum/sticker-market", middleware.RateLimit(middleware.RateLimitStickerList), UpQueryReq(api.StickerMarket))
	router.GET("/api/forum/sticker-market/pack", middleware.RateLimit(middleware.RateLimitStickerList), UpQueryReq(api.StickerMarketPack))
	forum := router.Group("/api/forum", middleware.CSRFProtection, middleware.JWTAuthCheck)
	forum.GET("my-sticker-packs", middleware.NoUpdateUserActivity, UpButterReq(api.MyStickerPacks))
	forum.POST("my-sticker-pack-save", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpLimitedJsonReq(64<<10, api.SaveMyStickerPack))
	forum.POST("my-sticker-pack-delete", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpLimitedJsonReq(64<<10, api.DeleteMyStickerPack))
	admin := router.Group("/api/admin",
		middleware.JWTAuthCheck,
		middleware.CheckWritableAccount,
		middleware.CheckPermission(permission.SiteManager),
	)
	admin.GET("/sticker-market", UpQueryReq(api.AdminStickerMarket))
	admin.POST("/sticker-market-sync", UpButterReq(api.AdminStickerMarketSync))
	admin.GET("/sticker-market-pack", UpQueryReq(api.AdminStickerMarketPack))
	admin.POST("/sticker-market-import", UpButterReq(api.AdminStickerMarketImport))
	admin.POST("/sticker-market-pack-save", UpButterReq(api.AdminStickerMarketPackSave))
	admin.POST("/sticker-market-item-save", UpButterReq(api.AdminStickerMarketItemSave))
	router.GET("/file/img/*filename", api.GetFileByFileName)
	return conn, router
}

// seedContractMarket writes the catalog the market fixtures describe: one
// published, partially imported pack and one synchronized but unimported pack.
const contractMarketIcon = "https://cdn3.ldstatic.com/original/pack-icon.webp"

func seedContractMarket(t *testing.T, conn *gorm.DB) {
	t.Helper()
	imported := contractMarketSync.Add(5 * time.Minute)
	rows := []any{
		&sticker.MarketTopic{Source: stickerservice.MarketSourceID, UpstreamID: "anime", Label: "动画"},
		&sticker.Entity{Id: 9301, Name: "m_" + strings.Repeat("a", 40), DisplayName: "Pack One 1", Pack: sticker.PackMarket, FileName: contractMarketFileA},
		&sticker.Entity{Id: 9302, Name: "m_" + strings.Repeat("b", 40), DisplayName: "Pack One 2", Pack: sticker.PackMarket, FileName: contractMarketFileB, SortOrder: 1},
		&sticker.MarketPack{Id: 9201, Source: stickerservice.MarketSourceID, UpstreamID: "group-1", Name: "Pack One", Topic: "anime", UpstreamCount: 3,
			Status: sticker.MarketStatusPartial, Published: true, ItemCount: 2, FailedCount: 1, ImportTotal: 3, CoverStickerID: 9301,
			IconURL: contractMarketIcon, SyncedAt: contractMarketSync, ImportedAt: &imported},
		&sticker.MarketPack{Id: 9202, Source: stickerservice.MarketSourceID, UpstreamID: "group-2", Name: "Pack Two", Topic: "anime", UpstreamCount: 2,
			Status: sticker.MarketStatusNew, UpstreamOrder: 1, IconURL: contractMarketIcon, SyncedAt: contractMarketSync},
		&sticker.MarketItem{PackID: 9201, UpstreamID: "e1", StickerID: 9301},
		&sticker.MarketItem{PackID: 9201, UpstreamID: "e2", StickerID: 9302, SortOrder: 1},
		&sticker.MarketItem{PackID: 9201, UpstreamID: "e3", SortOrder: 2, Error: "blocked"},
	}
	for _, row := range rows {
		if err := conn.Create(row).Error; err != nil {
			t.Fatalf("seed market row %T: %v", row, err)
		}
	}
	// GORM Create skips a false bool, so the column default would mark these official.
	if err := conn.Model(&sticker.Entity{}).Where("pack = ?", sticker.PackMarket).Update("is_official", false).Error; err != nil {
		t.Fatal(err)
	}
	stickerservice.InvalidateCache()
}

func TestStickerMarketBrowseHTTPContract(t *testing.T) {
	conn, router := setupStickerMarketContractTest(t)
	seedContractMarket(t, conn)

	response := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/sticker-market", "", "")
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "sticker-market-success.json"))

	response = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/sticker-market/pack?id=9201", "", "")
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "sticker-market-pack-success.json"))

	// An unpublished pack is invisible, but its stickers still resolve in old posts.
	response = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/sticker-market/pack?id=9202", "", "")
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "sticker-market-pack-unavailable.json"))

	// Market stickers never join the official directory.
	response = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/stickers", "", "")
	if strings.Contains(response.Body.String(), "m_aaaa") {
		t.Fatalf("official directory listed a market sticker: %s", response.Body.String())
	}
}

func TestStickerMarketSubscriptionHTTPContract(t *testing.T) {
	conn, router := setupStickerMarketContractTest(t)
	seedContractMarket(t, conn)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)

	t.Run("anonymous visitors cannot subscribe", func(t *testing.T) {
		response := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-pack-save", `{"packId":9201}`, "")
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", response.Code)
		}
	})

	response := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-pack-save", `{"packId":9201}`, token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "my-sticker-pack-save-success.json"))
	// Saving twice is idempotent.
	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-pack-save", `{"packId":9201}`, token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "my-sticker-pack-save-success.json"))
	response = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/my-sticker-packs", "", token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "my-sticker-packs-success.json"))

	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-pack-save", `{"packId":9202}`, token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "sticker-market-pack-unavailable.json"))

	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-pack-delete", `{"packId":9201}`, token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "my-sticker-action-success.json"))
	response = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/my-sticker-packs", "", token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "admin-sticker-list-empty.json"))

	t.Run("the subscription limit is enforced", func(t *testing.T) {
		for id := uint64(1); id <= uint64(stickerservice.MaxPackSubscriptions); id++ {
			if err := conn.Create(&sticker.PackSubscription{UserID: user.Id, PackID: 100000 + id, SortOrder: int(id)}).Error; err != nil {
				t.Fatal(err)
			}
		}
		response := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-pack-save", `{"packId":9201}`, token)
		assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "my-sticker-pack-limit.json"))
	})
}

func TestAdminStickerMarketHTTPContract(t *testing.T) {
	t.Run("catalog lists every synchronized pack", func(t *testing.T) {
		conn, router := setupStickerMarketContractTest(t)
		seedContractMarket(t, conn)
		serveAdminStickersOK(t, conn, router, http.MethodGet, "/api/admin/sticker-market", "", "admin-sticker-market-success.json")
		serveAdminStickersOK(t, conn, router, http.MethodGet, "/api/admin/sticker-market-pack?id=9201", "", "admin-sticker-market-pack-success.json")
		serveAdminStickersOK(t, conn, router, http.MethodGet, "/api/admin/sticker-market-pack?id=987654", "", "admin-sticker-not-found.json")
	})

	t.Run("sync reads the whole upstream catalog", func(t *testing.T) {
		conn, router := setupStickerMarketContractTest(t)
		base := "https://s.pwsh.us.kg/assets/market/index/"
		group := func(id, name string, count int) string {
			return fmt.Sprintf(`{"id":%q,"name":%q,"topic":"anime","emojiCount":%d,"isArchived":false}`, id, name, count)
		}
		restore := stickerservice.SwapMarketFetcher(contractMarketFetcher{
			base + "topics.json": `{"topics":[{"id":"all","label":"全部","totalPages":1},{"id":"anime","label":"动画"}]}`,
			base + "page-1.json": `{"groups":[` + group("group-1", "Pack One", 3) + `,` + group("group-2", "Pack Two", 2) + `],"totalPages":1}`,
		})
		t.Cleanup(restore)
		serveAdminStickersOK(t, conn, router, http.MethodPost, "/api/admin/sticker-market-sync", `{}`, "admin-sticker-market-sync-success.json")
	})

	t.Run("an unreachable upstream keeps the last catalog", func(t *testing.T) {
		conn, router := setupStickerMarketContractTest(t)
		seedContractMarket(t, conn)
		serveAdminStickersOK(t, conn, router, http.MethodPost, "/api/admin/sticker-market-sync", `{}`, "admin-sticker-market-unavailable.json")
		if pack, err := sticker.MarketPackByID(9202); err != nil || pack.Missing {
			t.Fatalf("failed sync changed the catalog: %#v, %v", pack, err)
		}
	})

	t.Run("import queues one task per pack", func(t *testing.T) {
		conn, router := setupStickerMarketContractTest(t)
		seedContractMarket(t, conn)
		serveAdminStickersOK(t, conn, router, http.MethodPost, "/api/admin/sticker-market-import", `{"id":9202}`, "admin-sticker-market-import-success.json")
		serveAdminStickersOK(t, conn, router, http.MethodPost, "/api/admin/sticker-market-import", `{"id":9202}`, "admin-sticker-market-busy.json")
		var tasks int64
		conn.Model(&taskQueue.Entity{}).Where("type = ?", stickerservice.TaskTypeMarketImport).Count(&tasks)
		if tasks != 1 {
			t.Fatalf("queued tasks = %d, want 1", tasks)
		}
	})

	t.Run("publishing needs imported stickers", func(t *testing.T) {
		conn, router := setupStickerMarketContractTest(t)
		seedContractMarket(t, conn)
		serveAdminStickersOK(t, conn, router, http.MethodPost, "/api/admin/sticker-market-pack-save", `{"id":9201,"published":true,"sortOrder":5}`, "admin-sticker-market-pack-save-success.json")
		serveAdminStickersOK(t, conn, router, http.MethodPost, "/api/admin/sticker-market-pack-save", `{"id":9202,"published":true}`, "admin-sticker-market-not-ready.json")
	})

	t.Run("a withdrawn sticker stops resolving", func(t *testing.T) {
		conn, router := setupStickerMarketContractTest(t)
		seedContractMarket(t, conn)
		serveAdminStickersOK(t, conn, router, http.MethodPost, "/api/admin/sticker-market-item-save", `{"stickerId":9302,"isEnabled":false}`, "my-sticker-action-success.json")
		urls, err := stickerservice.ResolveURLs([]string{"m_" + strings.Repeat("b", 40)})
		if err != nil || len(urls) != 0 {
			t.Fatalf("withdrawn sticker resolved: %#v, %v", urls, err)
		}
		// Official stickers are managed elsewhere and cannot be withdrawn here.
		seedContractSticker(t, conn, contractStickerID, "smile", "", 1, true)
		serveAdminStickersOK(t, conn, router, http.MethodPost, "/api/admin/sticker-market-item-save", `{"stickerId":9101,"isEnabled":false}`, "admin-sticker-not-found.json")
	})

	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/admin/sticker-market"},
		{http.MethodPost, "/api/admin/sticker-market-sync"},
		{http.MethodPost, "/api/admin/sticker-market-import"},
	} {
		t.Run(route.path, func(t *testing.T) {
			adminStickerRouteGuardScenarios(t, setupStickerMarketContractTest, route.method, route.path)
		})
	}
}

// Market objects are content-addressed, so public ones may be cached for a day;
// every other stored file keeps the default no-store policy.
func TestStickerMarketFileCacheHeader(t *testing.T) {
	conn, router := setupStickerMarketContractTest(t)
	seedContractMarket(t, conn)
	for _, name := range []string{contractMarketFileA, "stickers/9f1c2d3e-0000-4000-8000-0000000000aa.png"} {
		if _, err := filedata.SaveFile(0, name, "image/png", contractTinyPNG); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = filedata.DeleteByName(name) })
	}
	seedContractSticker(t, conn, contractStickerID, "smile", "stickers/9f1c2d3e-0000-4000-8000-0000000000aa.png", 1, true)
	for id, name := range map[uint64]string{9301: contractMarketFileA, contractStickerID: "stickers/9f1c2d3e-0000-4000-8000-0000000000aa.png"} {
		if err := fileusageservice.SetStickerUsageTx(conn, id, 0, name, fileUsage.UsageStatusActive); err != nil {
			t.Fatal(err)
		}
	}

	response := serveAuthSecurityJSON(router, http.MethodGet, "/file/img/"+contractMarketFileA, "", "")
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "public, max-age=86400" {
		t.Fatalf("market file status = %d, Cache-Control = %q", response.Code, response.Header().Get("Cache-Control"))
	}
	response = serveAuthSecurityJSON(router, http.MethodGet, "/file/img/stickers/9f1c2d3e-0000-4000-8000-0000000000aa.png", "", "")
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("official file status = %d, Cache-Control = %q", response.Code, response.Header().Get("Cache-Control"))
	}
}

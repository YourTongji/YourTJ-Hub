package routes

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/meiliconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaceEditors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPageRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiSyncRuns"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupWikiContractTest 迁移 wiki 相关表并注册 wiki 路由（与 route4api.go 注册一致）。
func setupWikiContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&wikiNamespaces.Entity{},
		&wikiNamespaceEditors.Entity{},
		&wikiPages.Entity{},
		&wikiPageRevisions.Entity{},
		&wikiSyncRuns.Entity{},
		&rolePermissionRs.Entity{},
	); err != nil {
		t.Fatalf("migrate wiki contract tables: %v", err)
	}
	// 清空 wiki 测试表与固定 ID 种子，保证共享测试库上可重复运行。
	for _, model := range []any{
		&wikiSyncRuns.Entity{},
		&wikiPageRevisions.Entity{},
		&wikiPages.Entity{},
		&wikiNamespaceEditors.Entity{},
		&wikiNamespaces.Entity{},
	} {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean wiki contract table: %v", err)
		}
	}
	conn.Unscoped().Where("id IN ?", []uint64{6001, 6002, 6003}).Delete(&topics.Entity{})
	conn.Unscoped().Where("id IN ?", []uint64{11001, 11002, 11003}).Delete(&posts.Entity{})
	conn.Where("permission_id = ?", permission.PageManager.Id()).Delete(&rolePermissionRs.Entity{})
	// 清空 wiki 同步设置（webhook secret 密文/清除标记）：TestWikiWebhookSecretHTTPContract
	// 会写入 page_config，不清空会污染共享测试库、令后续 webhook 验签测试 403
	// （cleared=true 使 LoadWebhookSecret 恒返回空，fail-closed 生效）。
	conn.Where("page_type = ?", pageConfig.WikiSyncSettings).Delete(&pageConfig.Entity{})
	hotdataserve.ClearWikiSyncSettingsConfigCache()

	wikiApi := router.Group("/api/wiki")
	wikiApi.GET("tree", UpButterReq(api.WikiTree))
	wikiApi.GET("namespaces", UpButterReq(api.WikiNamespaces))
	wikiApi.GET("home", UpButterReq(api.WikiHome))
	// wiki 站内局内搜索（段落级 Meilisearch 索引，公开只读；与 route4api.go 注册一致）。
	wikiApi.GET("search", UpQueryReq(forum.WikiSearchJSON))
	// wiki GitHub webhook：PR merge 后即时同步（独立验签，无 JWT）。
	wikiApi.POST("webhook", api.WikiWebhook)
	adminWiki := router.Group("/api/admin/wiki", middleware.JWTAuthCheck, middleware.CheckWritableAccount, middleware.CheckPermission(permission.PageManager))
	adminWiki.GET("tree", UpButterReq(api.WikiAdminTree))
	adminWiki.GET("sync/status", UpButterReq(api.WikiSyncStatus))
	adminWiki.POST("sync", UpButterReq(api.WikiSyncRun))
	adminWiki.GET("sync/runs", UpButterReq(api.WikiSyncRuns))
	adminWiki.GET("sync/webhook-secret", UpButterReq(api.GetWikiWebhookSecret))
	adminWiki.POST("sync/webhook-secret", UpJsonReq(api.SaveWikiWebhookSecret))
	adminWiki.GET("sync/cdn", UpButterReq(api.GetWikiAssetCDN))
	adminWiki.POST("sync/cdn", UpJsonReq(api.SaveWikiAssetCDN))
	// 视图路由（与 route4api.go viewRoute 注册一致；JWTAuth 可选登录）。
	wikiView := router.Group("/wiki")
	wikiView.Use(middleware.JWTAuth)
	wikiView.GET("", forum.WikiHome)
	wikiView.GET("/*path", forum.WikiDetail)
	return conn, router
}

// ---------- SSR 视图（蓝图 smoke：X-Goose-Page JSON 载荷） ----------

func TestWikiHomeSSRPayload(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	alice := createHTTPContractUser(t, conn, contractTestID())
	seedWikiContract(t, conn, alice.Id)

	req := httptest.NewRequest(http.MethodGet, "/wiki", nil)
	req.Header.Set("X-Goose-Page", "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("wiki home SSR status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("wiki home SSR cache-control = %q, want no-store", rec.Header().Get("Cache-Control"))
	}
	var payload struct {
		Component string `json:"component"`
		Props     struct {
			Namespaces []map[string]any `json:"namespaces"`
			Recent     []map[string]any `json:"recent"`
			CanManage  bool             `json:"canManage"`
		} `json:"props"`
		Layout struct {
			Sidebar struct {
				Mode     string           `json:"mode"`
				WikiTree []map[string]any `json:"wikiTree"`
			} `json:"sidebar"`
		} `json:"layout"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode wiki home SSR payload: %v", err)
	}
	if payload.Component != "wiki.home" {
		t.Fatalf("wiki home component = %q, want wiki.home", payload.Component)
	}
	if len(payload.Props.Namespaces) != 1 || len(payload.Props.Recent) != 3 {
		t.Fatalf("wiki home props = %#v, want 1 namespace + 3 recent", payload.Props)
	}
	if payload.Props.CanManage {
		t.Fatal("anonymous wiki home canManage = true, want false")
	}
	if payload.Layout.Sidebar.Mode != "wiki" {
		t.Fatalf("wiki home sidebar mode = %q, want wiki", payload.Layout.Sidebar.Mode)
	}
	if len(payload.Layout.Sidebar.WikiTree) != 1 {
		t.Fatalf("wiki home sidebar wikiTree = %#v, want 1 namespace", payload.Layout.Sidebar.WikiTree)
	}
}

func TestWikiDetailSSRPayload(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	alice := createHTTPContractUser(t, conn, contractTestID())
	seedWikiContract(t, conn, alice.Id)
	// GitHub SSOT：canEdit 由 [wiki.git].repo 配置决定，未配置时关闭。
	prevRepo := preferences.GetString("wiki.git.repo", "")
	preferences.Set("wiki.git.repo", "")
	t.Cleanup(func() { preferences.Set("wiki.git.repo", prevRepo) })

	fetch := func(token string) (int, map[string]any) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/wiki/guide/getting-started", nil)
		req.Header.Set("X-Goose-Page", "true")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode wiki detail SSR payload: %v", err)
		}
		return rec.Code, payload
	}

	t.Run("anonymous sees read-only snapshot", func(t *testing.T) {
		code, payload := fetch("")
		if code != http.StatusOK {
			t.Fatalf("anonymous wiki detail status = %d, want 200: %v", code, payload)
		}
		if payload["component"] != "wiki.detail" {
			t.Fatalf("wiki detail component = %v, want wiki.detail", payload["component"])
		}
		props, _ := payload["props"].(map[string]any)
		page, _ := props["page"].(map[string]any)
		if page["title"] != "快速开始" || page["path"] != "guide/getting-started" {
			t.Fatalf("wiki detail page = %#v, want 快速开始/guide/getting-started", page)
		}
		// GitHub SSOT：未配置仓库时无站内编辑，外链为空。
		if page["canEdit"] != false {
			t.Fatalf("anonymous canEdit = %v, want false", page["canEdit"])
		}
		layout, _ := payload["layout"].(map[string]any)
		sidebar, _ := layout["sidebar"].(map[string]any)
		if sidebar["mode"] != "wiki" {
			t.Fatalf("wiki detail sidebar mode = %v, want wiki", sidebar["mode"])
		}
	})

	t.Run("repo configured exposes GitHub edit links", func(t *testing.T) {
		preferences.Set("wiki.git.repo", "https://github.com/YourTongji/YourTJ-Wiki.git")
		t.Cleanup(func() { preferences.Set("wiki.git.repo", "") })
		aliceToken := contractSessionToken(t, alice)
		code, payload := fetch(aliceToken)
		if code != http.StatusOK {
			t.Fatalf("wiki detail status = %d, want 200", code)
		}
		props, _ := payload["props"].(map[string]any)
		page, _ := props["page"].(map[string]any)
		if page["canEdit"] != true {
			t.Fatalf("repo-configured canEdit = %v, want true", page["canEdit"])
		}
		if editURL, _ := page["editUrl"].(string); !strings.Contains(editURL, "github.com/YourTongji/YourTJ-Wiki/edit/main/guide/getting-started.md") {
			t.Fatalf("wiki detail editUrl = %q, want GitHub edit link", editURL)
		}
		if historyURL, _ := page["historyUrl"].(string); !strings.Contains(historyURL, "github.com/YourTongji/YourTJ-Wiki/commits/main/guide/getting-started.md") {
			t.Fatalf("wiki detail historyUrl = %q, want GitHub history link", historyURL)
		}
	})
}

func TestWikiDetailSSRNotFound(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	alice := createHTTPContractUser(t, conn, contractTestID())
	seedWikiContract(t, conn, alice.Id)

	req := httptest.NewRequest(http.MethodGet, "/wiki/guide/nonexistent", nil)
	req.Header.Set("X-Goose-Page", "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("wiki detail not-found status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
}

// seedWikiContract 构造 wiki 契约测试数据：guide namespace + 3 个页面投影
// （GitHub SSOT：内容/标题直接落在 wiki_pages 投影列）。
func seedWikiContract(t *testing.T, conn *gorm.DB, aliceID uint64) {
	t.Helper()
	// 固定时间（CST +08:00），与 fixtures 中 updatedAt 一致。
	base := time.Date(2026, 8, 10, 14, 0, 0, 0, time.FixedZone("CST", 8*3600))
	if err := conn.Create(&wikiNamespaces.Entity{Id: 1, Name: "guide", Slug: &[]string{"guide"}[0], Description: "社区使用指南", SortOrder: 10, CreatedAt: base, UpdatedAt: base}).Error; err != nil {
		t.Fatalf("create wiki namespace: %v", err)
	}

	seedWikiPage := func(id uint64, path, title, content string, sortOrder int, createdAt time.Time) {
		t.Helper()
		topic := topics.Entity{
			Id: id + 5000, UserId: aliceID, Title: title, Status: 1, ProcessStatus: 0,
			TopicType: topics.TopicTypeWiki, FirstPostId: id + 10000, LastPostId: id + 10000,
			PostSeq: 1, VisibilityStatus: topics.VisibilityActive, RetentionStatus: topics.RetentionNormal,
			CreatedAt: createdAt, UpdatedAt: createdAt,
		}
		if err := conn.Create(&topic).Error; err != nil {
			t.Fatalf("create wiki topic %d: %v", id, err)
		}
		firstPost := posts.Entity{
			Id: id + 10000, TopicId: topic.Id, PostNo: 1, UserId: aliceID,
			Content: content, RenderedHTML: "", RenderedVersion: 1, ProcessStatus: posts.ProcessStatusNormal,
			VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
			CreatedAt: createdAt, UpdatedAt: createdAt,
		}
		if err := conn.Create(&firstPost).Error; err != nil {
			t.Fatalf("create wiki first post %d: %v", id, err)
		}
		if err := conn.Create(&wikiPages.Entity{
			Id: id, TopicId: topic.Id, Namespace: "guide", Path: path, SourcePath: path, SortOrder: sortOrder,
			Title: title, Content: content, RenderedHTML: content,
			PublishedRevisionNo: 1, CreatedAt: createdAt, UpdatedAt: createdAt,
		}).Error; err != nil {
			t.Fatalf("create wiki page %d: %v", id, err)
		}
	}

	seedWikiPage(1001, "guide/getting-started", "快速开始", "欢迎来到 YourTJ Wiki。", 1, base)
	seedWikiPage(1002, "guide/content", "内容规范", "内容规范正文。", 2, base.Add(30*time.Minute))
	seedWikiPage(1003, "guide/draft/ideas", "草稿想法", "草稿内容。", 3, base.Add(time.Hour))
}

func TestWikiTreeHTTPContract(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	alice := createHTTPContractUser(t, conn, contractTestID())
	seedWikiContract(t, conn, alice.Id)

	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/wiki/tree", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("wiki tree status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "wiki-tree-success.json"))
}

func TestWikiNamespacesHTTPContract(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	alice := createHTTPContractUser(t, conn, contractTestID())
	seedWikiContract(t, conn, alice.Id)

	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/wiki/namespaces", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("wiki namespaces status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	env := decodeContractEnvelope(t, rec)
	if env.Code != 0 {
		t.Fatalf("wiki namespaces code = %d, want 0: %s", env.Code, rec.Body.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(env.Result, &items); err != nil {
		t.Fatalf("decode wiki namespaces result: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("wiki namespaces count = %d, want 1", len(items))
	}
	ns := items[0]
	if ns["name"] != "guide" || ns["description"] != "社区使用指南" || ns["sortOrder"] != float64(10) || ns["pageCount"] != float64(3) {
		t.Fatalf("wiki namespaces[0] = %#v, want guide/社区使用指南/sortOrder 10/pageCount 3", ns)
	}
	if _, ok := ns["updatedAt"].(string); !ok {
		t.Fatalf("wiki namespaces[0].updatedAt = %#v, want RFC3339 string", ns["updatedAt"])
	}
	// review P2：namespace 卡需提供首个页面的完整路径供跳转。
	if ns["firstPagePath"] != "guide/getting-started" {
		t.Fatalf("wiki namespaces[0].firstPagePath = %#v, want guide/getting-started", ns["firstPagePath"])
	}
}

// TestWikiSearchHTTPContract 覆盖 /api/wiki/search 的公开只读契约：
// 空 query 走确定性空结果（不依赖 Meilisearch，契约断言字段形状与语义）；
// 非空 query 在 Meilisearch 不可用时应降级 searchUnavailable（HTTP 200）。
func TestWikiSearchHTTPContract(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	alice := createHTTPContractUser(t, conn, contractTestID())
	seedWikiContract(t, conn, alice.Id)

	t.Run("empty query returns deterministic empty result", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/wiki/search?q=", "", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("wiki search empty-query status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "wiki-search-empty.json"))
	})

	t.Run("query without Meilisearch degrades to searchUnavailable", func(t *testing.T) {
		if meiliconnect.IsAvailable() {
			t.Skip("Meilisearch available in this environment; searchUnavailable fallback not testable")
		}
		rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/wiki/search?q=%E4%BF%9D%E7%A0%94", "", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("wiki search unavailable status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		env := decodeContractEnvelope(t, rec)
		if env.Code != 0 {
			t.Fatalf("wiki search unavailable code = %d, want 0", env.Code)
		}
		var result struct {
			Query             string `json:"query"`
			Total             int64  `json:"total"`
			Items             []any  `json:"items"`
			SearchUnavailable bool   `json:"searchUnavailable"`
		}
		if err := json.Unmarshal(env.Result, &result); err != nil {
			t.Fatalf("decode wiki search result %q: %v", env.Result, err)
		}
		if result.Query != "保研" {
			t.Fatalf("wiki search unavailable query = %q, want 保研", result.Query)
		}
		if result.Total != 0 || len(result.Items) != 0 || !result.SearchUnavailable {
			t.Fatalf("wiki search unavailable result = %+v, want empty + searchUnavailable=true", result)
		}
	})
}

// TestWikiSearchHTTPContractBadQuery 非数字 limit 参数应被严格绑定拒绝为 400
// （UpQueryReq strict 模式：稳定 messageCode，不泄漏原始解析错误）。
func TestWikiSearchHTTPContractBadQuery(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	alice := createHTTPContractUser(t, conn, contractTestID())
	seedWikiContract(t, conn, alice.Id)

	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/wiki/search?q=x&limit=abc", "", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("wiki search bad limit status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "wiki-search-parse-failed.json"))
}

func TestWikiHomeHTTPContract(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	alice := createHTTPContractUser(t, conn, contractTestID())
	seedWikiContract(t, conn, alice.Id)

	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/wiki/home", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("wiki home status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	env := decodeContractEnvelope(t, rec)
	if env.Code != 0 {
		t.Fatalf("wiki home code = %d, want 0: %s", env.Code, rec.Body.String())
	}
	var data map[string]any
	if err := json.Unmarshal(env.Result, &data); err != nil {
		t.Fatalf("decode wiki home result: %v", err)
	}
	namespaces, ok := data["namespaces"].([]any)
	if !ok || len(namespaces) != 1 {
		t.Fatalf("wiki home namespaces = %#v, want 1 namespace", data["namespaces"])
	}
	recent, ok := data["recent"].([]any)
	if !ok || len(recent) != 3 {
		t.Fatalf("wiki home recent = %#v, want 3 recent pages", data["recent"])
	}
	// 最近更新按页面投影更新时间降序：1003（15:00）→ 1002（14:30）→ 1001（14:00）。
	first := recent[0].(map[string]any)
	if first["pageId"] != float64(1003) || first["path"] != "guide/draft/ideas" {
		t.Fatalf("wiki home recent[0] = %#v, want 1003/guide/draft/ideas", first)
	}
	second := recent[1].(map[string]any)
	if second["pageId"] != float64(1002) || second["path"] != "guide/content" {
		t.Fatalf("wiki home recent[1] = %#v, want 1002/guide/content", second)
	}
	third := recent[2].(map[string]any)
	if third["pageId"] != float64(1001) || third["path"] != "guide/getting-started" {
		t.Fatalf("wiki home recent[2] = %#v, want 1001/guide/getting-started", third)
	}
	// GitHub SSOT：无论坛编辑者概念，recent 条目不得再携带 editorId/editorName
	// （历史遗留字段恒为零值且违反 OpenAPI minimum:1，issue #291）。
	for i, raw := range recent {
		item := raw.(map[string]any)
		if _, ok := item["editorId"]; ok {
			t.Fatalf("wiki home recent[%d].editorId present, want removed under GitHub SSOT", i)
		}
		if _, ok := item["editorName"]; ok {
			t.Fatalf("wiki home recent[%d].editorName present, want removed under GitHub SSOT", i)
		}
	}
	// 首页 namespace 卡携带 firstPagePath。
	if ns0 := namespaces[0].(map[string]any); ns0["firstPagePath"] != "guide/getting-started" {
		t.Fatalf("wiki home namespaces[0].firstPagePath = %#v, want guide/getting-started", ns0["firstPagePath"])
	}
}

func TestWikiWebhookSecretHTTPContract(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	bob := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, bob.Id, permission.PageManager)
	bobToken := contractSessionToken(t, bob)

	// 该测试会持久化 wikiSyncSettings page_config 行，而 setupWikiContractTest
	// 只清空 wiki 表；中断的子测试会留下该行，翻转下次运行的 configured=false
	// 断言（共享测试库）。无论子测试结果如何都删除该行。
	t.Cleanup(func() {
		conn.Unscoped().Where("page_type = ?", pageConfig.WikiSyncSettings).Delete(&pageConfig.Entity{})
	})

	t.Run("unconfigured reports configured=false", func(t *testing.T) {
		// 清空管理端设置与旧配置，保证未配置态。
		prev := preferences.GetString("wiki.git.webhook_secret", "")
		preferences.Set("wiki.git.webhook_secret", "")
		t.Cleanup(func() { preferences.Set("wiki.git.webhook_secret", prev) })

		rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/admin/wiki/sync/webhook-secret", "", bobToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("webhook secret status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		env := decodeContractEnvelope(t, rec)
		if env.Code != 0 {
			t.Fatalf("webhook secret status code = %d, want 0", env.Code)
		}
		var result map[string]any
		if err := json.Unmarshal(env.Result, &result); err != nil {
			t.Fatalf("decode webhook secret status result: %v", err)
		}
		if result["configured"] != false {
			t.Fatalf("webhook secret configured = %v, want false", result["configured"])
		}
	})

	t.Run("manager saves secret then status reports configured=true", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/wiki/sync/webhook-secret",
			`{"secret":"s3cret-value"}`, bobToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("webhook secret save status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		env := decodeContractEnvelope(t, rec)
		if env.Code != 0 {
			t.Fatalf("webhook secret save code = %d, want 0: %s", env.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, env, contractFixture(t, "wiki-webhook-secret-save-success.json"))

		// 保存后 status 应报告 configured=true（密文经 securestore 落库）。
		rec = serveAuthSecurityJSON(router, http.MethodGet, "/api/admin/wiki/sync/webhook-secret", "", bobToken)
		env = decodeContractEnvelope(t, rec)
		var result map[string]any
		if err := json.Unmarshal(env.Result, &result); err != nil {
			t.Fatalf("decode webhook secret status after save: %v", err)
		}
		if result["configured"] != true {
			t.Fatalf("webhook secret configured after save = %v, want true", result["configured"])
		}

		// 用保存的 secret 验证 webhook 签名可达（端到端：securestore 解密链路）。
		body := `{"ref":"refs/heads/main"}`
		req := httptest.NewRequest(http.MethodPost, "/api/wiki/webhook", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Hub-Signature-256", webhookSignature("s3cret-value", body))
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("webhook with saved secret status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("manager clears secret", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/wiki/sync/webhook-secret",
			`{"secret":""}`, bobToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("webhook secret clear status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		env := decodeContractEnvelope(t, rec)
		if env.Code != 0 {
			t.Fatalf("webhook secret clear code = %d, want 0", env.Code)
		}
		rec = serveAuthSecurityJSON(router, http.MethodGet, "/api/admin/wiki/sync/webhook-secret", "", bobToken)
		env = decodeContractEnvelope(t, rec)
		var result map[string]any
		if err := json.Unmarshal(env.Result, &result); err != nil {
			t.Fatalf("decode webhook secret status after clear: %v", err)
		}
		if result["configured"] != false {
			t.Fatalf("webhook secret configured after clear = %v, want false", result["configured"])
		}
	})
}

func TestWikiAssetCDNHTTPContract(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	bob := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, bob.Id, permission.PageManager)
	bobToken := contractSessionToken(t, bob)

	// 该测试持久化 wikiSyncSettings page_config 行（AssetCDN 字段），
	// 中断的子测试会留下该行；无论结果如何都删除，避免污染共享测试库。
	t.Cleanup(func() {
		conn.Unscoped().Where("page_type = ?", pageConfig.WikiSyncSettings).Delete(&pageConfig.Entity{})
		hotdataserve.ClearWikiSyncSettingsConfigCache()
	})

	t.Run("unconfigured reports default self", func(t *testing.T) {
		// 清空管理端设置，保证未配置态。
		conn.Where("page_type = ?", pageConfig.WikiSyncSettings).Delete(&pageConfig.Entity{})
		hotdataserve.ClearWikiSyncSettingsConfigCache()

		rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/admin/wiki/sync/cdn", "", bobToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("asset cdn status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		env := decodeContractEnvelope(t, rec)
		if env.Code != 0 {
			t.Fatalf("asset cdn status code = %d, want 0", env.Code)
		}
		var result map[string]any
		if err := json.Unmarshal(env.Result, &result); err != nil {
			t.Fatalf("decode asset cdn status result: %v", err)
		}
		if result["cdn"] != "self" {
			t.Fatalf("asset cdn = %v, want self", result["cdn"])
		}
	})

	t.Run("manager saves jsDelivr then status reports it", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/wiki/sync/cdn",
			`{"cdn":"jsDelivr"}`, bobToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("asset cdn save status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		env := decodeContractEnvelope(t, rec)
		if env.Code != 0 {
			t.Fatalf("asset cdn save code = %d, want 0: %s", env.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, env, contractFixture(t, "wiki-asset-cdn-save-success.json"))

		rec = serveAuthSecurityJSON(router, http.MethodGet, "/api/admin/wiki/sync/cdn", "", bobToken)
		env = decodeContractEnvelope(t, rec)
		var result map[string]any
		if err := json.Unmarshal(env.Result, &result); err != nil {
			t.Fatalf("decode asset cdn status after save: %v", err)
		}
		if result["cdn"] != "jsDelivr" {
			t.Fatalf("asset cdn after save = %v, want jsDelivr", result["cdn"])
		}
	})

	t.Run("manager restores self", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/wiki/sync/cdn",
			`{"cdn":"self"}`, bobToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("asset cdn restore status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		env := decodeContractEnvelope(t, rec)
		if env.Code != 0 {
			t.Fatalf("asset cdn restore code = %d, want 0", env.Code)
		}
		rec = serveAuthSecurityJSON(router, http.MethodGet, "/api/admin/wiki/sync/cdn", "", bobToken)
		env = decodeContractEnvelope(t, rec)
		var result map[string]any
		if err := json.Unmarshal(env.Result, &result); err != nil {
			t.Fatalf("decode asset cdn status after restore: %v", err)
		}
		if result["cdn"] != "self" {
			t.Fatalf("asset cdn after restore = %v, want self", result["cdn"])
		}
	})
}

func TestWikiSyncStatusHTTPContract(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	bob := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, bob.Id, permission.PageManager)
	bobToken := contractSessionToken(t, bob)

	// 未配置 [wiki.git].repo → enabled=false（空库：pages 0/0）。
	prevRepo := preferences.GetString("wiki.git.repo", "")
	prevBranch := preferences.GetString("wiki.git.branch", "main")
	preferences.Set("wiki.git.repo", "")
	preferences.Set("wiki.git.branch", "main")
	t.Cleanup(func() {
		preferences.Set("wiki.git.repo", prevRepo)
		preferences.Set("wiki.git.branch", prevBranch)
	})

	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/admin/wiki/sync/status", "", bobToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("wiki sync status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "wiki-sync-status-success.json"))
}

// webhookSignature 按 GitHub 文档计算 X-Hub-Signature-256（HMAC-SHA256，sha256=<hex>）。
func webhookSignature(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestWikiWebhookSignature(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	_ = conn

	t.Run("no secret configured rejects with 403", func(t *testing.T) {
		prev := preferences.GetString("wiki.git.webhook_secret", "")
		preferences.Set("wiki.git.webhook_secret", "")
		t.Cleanup(func() { preferences.Set("wiki.git.webhook_secret", prev) })

		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/wiki/webhook", `{}`, "")
		if rec.Code != http.StatusForbidden {
			t.Fatalf("webhook no-secret status = %d, want 403: %s", rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode webhook 403 body: %v", err)
		}
		if body["error"] != "webhook not configured" {
			t.Fatalf("webhook 403 error = %v, want webhook not configured", body["error"])
		}
	})

	t.Run("valid signature accepted", func(t *testing.T) {
		prev := preferences.GetString("wiki.git.webhook_secret", "")
		preferences.Set("wiki.git.webhook_secret", "test-secret")
		t.Cleanup(func() { preferences.Set("wiki.git.webhook_secret", prev) })

		body := `{"ref":"refs/heads/main"}`
		req := httptest.NewRequest(http.MethodPost, "/api/wiki/webhook", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Hub-Signature-256", webhookSignature("test-secret", body))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("webhook valid signature status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode webhook 200 body: %v", err)
		}
		if resp["ok"] != true {
			t.Fatalf("webhook ok = %v, want true", resp["ok"])
		}
	})

	t.Run("tampered body rejected", func(t *testing.T) {
		prev := preferences.GetString("wiki.git.webhook_secret", "")
		preferences.Set("wiki.git.webhook_secret", "test-secret")
		t.Cleanup(func() { preferences.Set("wiki.git.webhook_secret", prev) })

		body := `{"ref":"refs/heads/main"}`
		req := httptest.NewRequest(http.MethodPost, "/api/wiki/webhook", strings.NewReader(`{"ref":"refs/heads/evil"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Hub-Signature-256", webhookSignature("test-secret", body))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("webhook tampered status = %d, want 401: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("missing signature rejected", func(t *testing.T) {
		prev := preferences.GetString("wiki.git.webhook_secret", "")
		preferences.Set("wiki.git.webhook_secret", "test-secret")
		t.Cleanup(func() { preferences.Set("wiki.git.webhook_secret", prev) })

		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/wiki/webhook", `{}`, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("webhook missing signature status = %d, want 401: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestWikiUnauthenticatedHTTPContract(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	seedWikiContract(t, conn, contractTestID())

	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/admin/wiki/sync/status", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wiki unauthenticated status = %d, want 401: %s", rec.Code, rec.Body.String())
	}
	env := decodeContractEnvelope(t, rec)
	if env.MessageCode != "auth.required" {
		t.Fatalf("wiki unauthenticated messageCode = %q, want auth.required", env.MessageCode)
	}
}

// TestWikiReadFailuresHTTPContract 注入数据库故障：公开 wiki 读接口与
// 管理端同步状态/运行日志必须返回 HTTP 500 + wiki.readFailed，
// 不能把 DB 故障伪装成空 wiki 或零计数（issue #287）。
func TestWikiReadFailuresHTTPContract(t *testing.T) {
	setup := func(t *testing.T) (*gorm.DB, *gin.Engine) {
		conn, router := setupWikiContractTest(t)
		alice := createHTTPContractUser(t, conn, contractTestID())
		seedWikiContract(t, conn, alice.Id)
		return conn, router
	}

	t.Run("tree returns 500 when wiki_namespaces is missing", func(t *testing.T) {
		conn, router := setup(t)
		if err := conn.Migrator().DropTable(&wikiNamespaces.Entity{}); err != nil {
			t.Fatalf("drop wiki_namespaces: %v", err)
		}
		rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/wiki/tree", "", "")
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("wiki tree failure status = %d, want 500: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "wiki-read-failed.json"))
	})

	t.Run("namespaces returns 500 when wiki_pages is missing", func(t *testing.T) {
		conn, router := setup(t)
		if err := conn.Migrator().DropTable(&wikiPages.Entity{}); err != nil {
			t.Fatalf("drop wiki_pages: %v", err)
		}
		rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/wiki/namespaces", "", "")
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("wiki namespaces failure status = %d, want 500: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "wiki-read-failed.json"))
	})

	t.Run("home returns 500 when topics is missing", func(t *testing.T) {
		conn, router := setup(t)
		if err := conn.Migrator().DropTable(&topics.Entity{}); err != nil {
			t.Fatalf("drop topics: %v", err)
		}
		rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/wiki/home", "", "")
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("wiki home failure status = %d, want 500: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "wiki-read-failed.json"))
	})

	t.Run("admin sync status returns 500 when wiki_pages is missing", func(t *testing.T) {
		conn, router := setup(t)
		bob := createHTTPContractUser(t, conn, contractTestID())
		grantContractPermission(t, conn, bob.Id, permission.PageManager)
		bobToken := contractSessionToken(t, bob)
		if err := conn.Migrator().DropTable(&wikiPages.Entity{}); err != nil {
			t.Fatalf("drop wiki_pages: %v", err)
		}
		rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/admin/wiki/sync/status", "", bobToken)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("wiki sync status failure status = %d, want 500: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "wiki-read-failed.json"))
	})

	t.Run("admin sync runs returns 500 when wiki_sync_runs is missing", func(t *testing.T) {
		conn, router := setup(t)
		bob := createHTTPContractUser(t, conn, contractTestID())
		grantContractPermission(t, conn, bob.Id, permission.PageManager)
		bobToken := contractSessionToken(t, bob)
		if err := conn.Migrator().DropTable(&wikiSyncRuns.Entity{}); err != nil {
			t.Fatalf("drop wiki_sync_runs: %v", err)
		}
		rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/admin/wiki/sync/runs", "", bobToken)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("wiki sync runs failure status = %d, want 500: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "wiki-read-failed.json"))
	})
}

// TestWikiEmptyDBHTTPContract 空库（无 namespace/页面）必须返回成功 + 空结果，
// 与 DB 故障（HTTP 500）可区分（issue #287 验收标准）。
func TestWikiEmptyDBHTTPContract(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	_ = conn // 表已迁移，无种子数据。

	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/wiki/tree", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("empty wiki tree status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	env := decodeContractEnvelope(t, rec)
	if env.Code != 0 {
		t.Fatalf("empty wiki tree code = %d, want 0: %s", env.Code, rec.Body.String())
	}
	var tree struct {
		Namespaces []any `json:"namespaces"`
	}
	if err := json.Unmarshal(env.Result, &tree); err != nil {
		t.Fatalf("decode empty wiki tree result: %v", err)
	}
	if len(tree.Namespaces) != 0 {
		t.Fatalf("empty wiki tree namespaces = %#v, want []", tree.Namespaces)
	}

	rec = serveAuthSecurityJSON(router, http.MethodGet, "/api/wiki/home", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("empty wiki home status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	env = decodeContractEnvelope(t, rec)
	var home struct {
		Namespaces []any `json:"namespaces"`
		Recent     []any `json:"recent"`
	}
	if err := json.Unmarshal(env.Result, &home); err != nil {
		t.Fatalf("decode empty wiki home result: %v", err)
	}
	if len(home.Namespaces) != 0 || len(home.Recent) != 0 {
		t.Fatalf("empty wiki home = %#v, want empty namespaces/recent", home)
	}
}

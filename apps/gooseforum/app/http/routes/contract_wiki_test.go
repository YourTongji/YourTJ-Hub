package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaceEditors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPageRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
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
		&rolePermissionRs.Entity{},
	); err != nil {
		t.Fatalf("migrate wiki contract tables: %v", err)
	}
	// 清空 wiki 测试表与固定 ID 种子，保证共享测试库上可重复运行。
	for _, model := range []any{
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

	wikiApi := router.Group("/api/wiki")
	wikiApi.GET("tree", UpButterReq(api.WikiTree))
	wikiApi.GET("namespaces", UpButterReq(api.WikiNamespaces))
	wikiApi.GET("home", UpButterReq(api.WikiHome))
	wikiApi.GET("revisions", UpQueryReq(api.WikiRevisions))
	wikiLoginApi := wikiApi.Use(middleware.JWTAuthCheck)
	wikiLoginApi.POST("pages", middleware.CheckWritableAccount, UpJsonReq(api.WikiCreatePage))
	wikiLoginApi.PUT("pages/:pageId", middleware.CheckWritableAccount, UpUriJsonReq(api.WikiEditPage))
	wikiLoginApi.POST("revisions/:revisionId/review", middleware.CheckWritableAccount, UpUriJsonReq(api.WikiReview))
	adminWiki := router.Group("/api/admin/wiki", middleware.JWTAuthCheck, middleware.CheckWritableAccount, middleware.CheckPermission(permission.PageManager))
	adminWiki.POST("namespaces", UpButterReq(api.WikiCreateNamespace))
	adminWiki.PUT("namespaces/:name", UpUriJsonReq(api.WikiUpdateNamespace))
	adminWiki.DELETE("namespaces/:name", UpUriReq(api.WikiDeleteNamespace))
	adminWiki.GET("namespaces/:name/editors", UpUriReq(api.WikiNamespaceEditors))
	adminWiki.PUT("namespaces/:name/editors", UpUriJsonReq(api.WikiSetEditors))
	adminWiki.GET("tree", UpButterReq(api.WikiAdminTree))
	adminWiki.PUT("tree", UpButterReq(api.WikiAdminTreeOps))
	adminWiki.GET("revisions", UpQueryReq(api.WikiAdminRevisions))
	// 视图路由（与 route4api.go viewRoute 注册一致；JWTAuth 可选登录）。
	wikiView := router.Group("/wiki")
	wikiView.Use(middleware.JWTAuth)
	wikiView.GET("", forum.WikiHome)
	wikiView.GET("/*path", forum.WikiDetail)
	return conn, router
}

// ---------- SSR 视图（蓝图 smoke：X-Goose-Page JSON 载荷 + pending 泄漏门禁） ----------

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
	if len(payload.Props.Namespaces) != 1 || len(payload.Props.Recent) != 2 {
		t.Fatalf("wiki home props = %#v, want 1 namespace + 2 recent", payload.Props)
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

func TestWikiDetailSSRPayloadAndPendingGate(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	alice := createHTTPContractUser(t, conn, contractTestID())
	seedWikiContract(t, conn, alice.Id)

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

	t.Run("anonymous sees approved snapshot without pending", func(t *testing.T) {
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
		if page["canEdit"] != false || page["canReview"] != false {
			t.Fatalf("anonymous canEdit/canReview = %v/%v, want false/false", page["canEdit"], page["canReview"])
		}
		// 蓝图风险项：pending 内容对公开用户不可见。
		if pending, exists := page["pending"]; !exists || pending != nil {
			t.Fatalf("anonymous wiki detail pending = %#v, want null (leak guard)", pending)
		}
		layout, _ := payload["layout"].(map[string]any)
		sidebar, _ := layout["sidebar"].(map[string]any)
		if sidebar["mode"] != "wiki" {
			t.Fatalf("wiki detail sidebar mode = %v, want wiki", sidebar["mode"])
		}
	})

	t.Run("editor sees pending banner data", func(t *testing.T) {
		aliceToken := contractSessionToken(t, alice)
		code, payload := fetch(aliceToken)
		if code != http.StatusOK {
			t.Fatalf("editor wiki detail status = %d, want 200", code)
		}
		props, _ := payload["props"].(map[string]any)
		page, _ := props["page"].(map[string]any)
		if page["canEdit"] != true {
			t.Fatalf("editor canEdit = %v, want true", page["canEdit"])
		}
		pending, ok := page["pending"].(map[string]any)
		if !ok || pending["title"] != "快速开始" || pending["content"] != "更新后的内容。" {
			t.Fatalf("editor pending = %#v, want pending revision view", page["pending"])
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

// seedWikiContract 构造 wiki 契约测试数据：guide namespace + 3 页面（2 已发布 + 1 草稿）。
func seedWikiContract(t *testing.T, conn *gorm.DB, aliceID uint64) {
	t.Helper()
	// 固定时间（CST +08:00），与 fixtures 中 updatedAt 一致。
	base := time.Date(2026, 8, 10, 14, 0, 0, 0, time.FixedZone("CST", 8*3600))
	if err := conn.Create(&wikiNamespaces.Entity{Id: 1, Name: "guide", Description: "社区使用指南", SortOrder: 10, CreatedAt: base, UpdatedAt: base}).Error; err != nil {
		t.Fatalf("create wiki namespace: %v", err)
	}
	if err := conn.Create(&wikiNamespaceEditors.Entity{Namespace: "guide", UserId: aliceID, AddedBy: aliceID, CreatedAt: base}).Error; err != nil {
		t.Fatalf("create wiki editor: %v", err)
	}

	seedWikiPage := func(id uint64, path, title, content string, sortOrder int, status int8, createdAt time.Time) {
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
		if err := conn.Create(&wikiPages.Entity{Id: id, TopicId: topic.Id, Namespace: "guide", Path: path, SortOrder: sortOrder, CreatedAt: createdAt, UpdatedAt: createdAt}).Error; err != nil {
			t.Fatalf("create wiki page %d: %v", id, err)
		}
		if err := conn.Create(&wikiPageRevisions.Entity{
			PageId: id, RevisionNo: 1, Title: title, Content: content,
			RenderedHTML: content, Status: status, EditorId: aliceID, CreatedAt: createdAt,
		}).Error; err != nil {
			t.Fatalf("create wiki revision %d: %v", id, err)
		}
	}

	seedWikiPage(1001, "guide/getting-started", "快速开始", "欢迎来到 YourTJ Wiki。", 1, wikiPageRevisions.StatusApproved, base)
	seedWikiPage(1002, "guide/content", "内容规范", "内容规范正文。", 2, wikiPageRevisions.StatusApproved, base.Add(30*time.Minute))
	seedWikiPage(1003, "guide/draft/ideas", "草稿想法", "草稿内容。", 3, wikiPageRevisions.StatusPending, base.Add(time.Hour))

	// pending 修订（待审队列；2026-08-12T18:05:00+08:00）。
	if err := conn.Create(&wikiPageRevisions.Entity{
		Id: 6001, PageId: 1001, RevisionNo: 2, Title: "快速开始", Content: "更新后的内容。",
		Status: wikiPageRevisions.StatusPending, EditorId: aliceID,
		CreatedAt: time.Date(2026, 8, 12, 18, 5, 0, 0, time.FixedZone("CST", 8*3600)),
	}).Error; err != nil {
		t.Fatalf("create wiki pending revision: %v", err)
	}
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
	if ns["name"] != "guide" || ns["description"] != "社区使用指南" || ns["sortOrder"] != float64(10) || ns["pageCount"] != float64(2) {
		t.Fatalf("wiki namespaces[0] = %#v, want guide/社区使用指南/sortOrder 10/pageCount 2", ns)
	}
	if _, ok := ns["updatedAt"].(string); !ok {
		t.Fatalf("wiki namespaces[0].updatedAt = %#v, want RFC3339 string", ns["updatedAt"])
	}
	// review P2：namespace 卡需提供首个 approved 页面的完整路径供跳转。
	if ns["firstPagePath"] != "guide/getting-started" {
		t.Fatalf("wiki namespaces[0].firstPagePath = %#v, want guide/getting-started", ns["firstPagePath"])
	}
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
	if !ok || len(recent) != 2 {
		t.Fatalf("wiki home recent = %#v, want 2 recent pages", data["recent"])
	}
	// 最近更新按修订时间降序：1002（14:30）→ 1001（14:00）。
	first := recent[0].(map[string]any)
	if first["pageId"] != float64(1002) || first["path"] != "guide/content" {
		t.Fatalf("wiki home recent[0] = %#v, want 1002/guide/content", first)
	}
	second := recent[1].(map[string]any)
	if second["pageId"] != float64(1001) || second["path"] != "guide/getting-started" {
		t.Fatalf("wiki home recent[1] = %#v, want 1001/guide/getting-started", second)
	}
	// 首页 namespace 卡携带 firstPagePath。
	if ns0 := namespaces[0].(map[string]any); ns0["firstPagePath"] != "guide/getting-started" {
		t.Fatalf("wiki home namespaces[0].firstPagePath = %#v, want guide/getting-started", ns0["firstPagePath"])
	}
}

func TestWikiCreatePageHTTPContract(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	alice := createHTTPContractUser(t, conn, contractTestID())
	seedWikiContract(t, conn, alice.Id)
	token := contractSessionToken(t, alice)

	rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/wiki/pages",
		`{"namespace":"guide","path":"guide/new-page","title":"新页面","content":"新内容"}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("wiki create page status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	env := decodeContractEnvelope(t, rec)
	if env.Code != 0 {
		t.Fatalf("wiki create page code = %d, want 0: %s", env.Code, rec.Body.String())
	}
	var result struct {
		PageId uint64 `json:"pageId"`
		Path   string `json:"path"`
	}
	if err := json.Unmarshal(env.Result, &result); err != nil {
		t.Fatalf("decode wiki create result: %v", err)
	}
	if result.PageId == 0 || result.Path != "guide/new-page" {
		t.Fatalf("wiki create result = %+v, want non-zero pageId + path", result)
	}
	// 清理本次创建产生的 topic/post，避免污染共享测试库。
	page := wikiPages.Get(result.PageId)
	if page.Id != 0 {
		conn.Where("id = ?", page.TopicId).Delete(&topics.Entity{})
		conn.Where("topic_id = ?", page.TopicId).Delete(&posts.Entity{})
	}
}

func TestWikiEditAndReviewHTTPContract(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	alice := createHTTPContractUser(t, conn, contractTestID())
	manager := createHTTPContractUser(t, conn, contractTestID())
	seedWikiContract(t, conn, alice.Id)
	grantContractPermission(t, conn, manager.Id, permission.PageManager)
	aliceToken := contractSessionToken(t, alice)
	managerToken := contractSessionToken(t, manager)

	t.Run("editor creates a pending revision", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPut, "/api/wiki/pages/1002",
			`{"title":"内容规范 v2","content":"更新后的内容。"}`, aliceToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("wiki edit status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		env := decodeContractEnvelope(t, rec)
		if env.Code != 0 {
			t.Fatalf("wiki edit code = %d, want 0: %s", env.Code, rec.Body.String())
		}
		var result struct {
			RevisionId uint64 `json:"revisionId"`
			Status     string `json:"status"`
		}
		if err := json.Unmarshal(env.Result, &result); err != nil {
			t.Fatalf("decode wiki edit result: %v", err)
		}
		if result.RevisionId == 0 || result.Status != "pending" {
			t.Fatalf("wiki edit result = %+v, want non-zero revisionId + pending", result)
		}
	})

	// 契约：超长标题（>512 字节）是请求级校验失败，create/edit 两端都必须
	// 映射为 common.request.invalidParams，而不是 wiki.path.invalid。
	t.Run("overlong title maps to request invalidParams on edit", func(t *testing.T) {
		longTitle := strings.Repeat("a", 513)
		rec := serveAuthSecurityJSON(router, http.MethodPut, "/api/wiki/pages/1002",
			`{"title":"`+longTitle+`","content":"x"}`, aliceToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("wiki edit overlong title status = %d, want 200 envelope: %s", rec.Code, rec.Body.String())
		}
		env := decodeContractEnvelope(t, rec)
		if env.MessageCode != "common.request.invalidParams" {
			t.Fatalf("wiki edit overlong title messageCode = %q, want common.request.invalidParams: %s", env.MessageCode, rec.Body.String())
		}
	})

	t.Run("overlong title maps to request invalidParams on create", func(t *testing.T) {
		longTitle := strings.Repeat("a", 513)
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/wiki/pages",
			`{"namespace":"guide","path":"guide/long-title","title":"`+longTitle+`","content":"x"}`, aliceToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("wiki create overlong title status = %d, want 200 envelope: %s", rec.Code, rec.Body.String())
		}
		env := decodeContractEnvelope(t, rec)
		if env.MessageCode != "common.request.invalidParams" {
			t.Fatalf("wiki create overlong title messageCode = %q, want common.request.invalidParams: %s", env.MessageCode, rec.Body.String())
		}
	})

	t.Run("non-manager cannot review", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/wiki/revisions/6001/review",
			`{"action":"approve"}`, aliceToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("wiki review denied status = %d, want 200 envelope: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "wiki-revision-review-forbidden.json"))
	})

	t.Run("manager approves the pending revision", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/wiki/revisions/6001/review",
			`{"action":"approve"}`, managerToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("wiki review status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		env := decodeContractEnvelope(t, rec)
		if env.Code != 0 {
			t.Fatalf("wiki review code = %d, want 0: %s", env.Code, rec.Body.String())
		}
		var result struct {
			RevisionId uint64 `json:"revisionId"`
			Status     string `json:"status"`
		}
		if err := json.Unmarshal(env.Result, &result); err != nil {
			t.Fatalf("decode wiki review result: %v", err)
		}
		if result.RevisionId != 6001 || result.Status != "approved" {
			t.Fatalf("wiki review result = %+v, want revisionId 6001 + approved", result)
		}
	})

	t.Run("already-reviewed revision conflicts", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/wiki/revisions/6001/review",
			`{"action":"reject"}`, managerToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("wiki review conflict status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "wiki-revision-review-conflict.json"))
	})
}

func TestWikiRevisionsHTTPContract(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	alice := createHTTPContractUser(t, conn, contractTestID())
	seedWikiContract(t, conn, alice.Id)

	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/wiki/revisions?pageId=1001", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("wiki revisions status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	env := decodeContractEnvelope(t, rec)
	// 契约：公开修订历史仅展示已发布（approved）修订，pending 内容不泄漏（蓝图风险项）。
	var items []map[string]any
	if err := json.Unmarshal(env.Result, &items); err != nil {
		t.Fatalf("decode wiki revisions result: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("wiki revisions count = %d, want 1 (pending excluded): %s", len(items), rec.Body.String())
	}
	if items[0]["revisionNo"] != float64(1) || items[0]["status"] != "approved" {
		t.Fatalf("wiki revisions[0] = %#v, want revisionNo 1 + approved", items[0])
	}
	if _, has := items[0]["content"]; !has {
		t.Fatalf("wiki revisions[0] missing content: %#v", items[0])
	}
}

func TestWikiAdminEditorsHTTPContract(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	alice := createHTTPContractUser(t, conn, contractTestID())
	bob := createHTTPContractUser(t, conn, contractTestID())
	seedWikiContract(t, conn, alice.Id)
	grantContractPermission(t, conn, bob.Id, permission.PageManager)
	bobToken := contractSessionToken(t, bob)

	t.Run("regular user denied", func(t *testing.T) {
		regular := createHTTPContractUser(t, conn, contractTestID())
		regularToken := contractSessionToken(t, regular)
		rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/admin/wiki/namespaces/guide/editors", "", regularToken)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("admin editors denied status = %d, want 403: %s", rec.Code, rec.Body.String())
		}
		env := decodeContractEnvelope(t, rec)
		if env.MessageCode != "permission.denied" {
			t.Fatalf("admin editors denied messageCode = %q, want permission.denied", env.MessageCode)
		}
	})

	t.Run("manager lists editors", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/admin/wiki/namespaces/guide/editors", "", bobToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("admin editors status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		var items []map[string]any
		if err := json.Unmarshal(decodeContractEnvelope(t, rec).Result, &items); err != nil {
			t.Fatalf("decode admin editors: %v", err)
		}
		if len(items) != 1 || items[0]["userId"] != float64(alice.Id) {
			t.Fatalf("admin editors = %#v, want exactly alice", items)
		}
	})

	t.Run("manager replaces editors", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPut, "/api/admin/wiki/namespaces/guide/editors",
			`{"userIds":[`+strconv.FormatUint(alice.Id, 10)+`]}`, bobToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("admin editors update status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		env := decodeContractEnvelope(t, rec)
		if env.Code != 0 {
			t.Fatalf("admin editors update code = %d, want 0: %s", env.Code, rec.Body.String())
		}
	})
}

func TestWikiAdminNamespaceHTTPContract(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	bob := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, bob.Id, permission.PageManager)
	bobToken := contractSessionToken(t, bob)

	t.Run("manager creates namespace", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/wiki/namespaces",
			`{"name":"deploy","description":"部署手册"}`, bobToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("admin namespace create status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		env := decodeContractEnvelope(t, rec)
		if env.Code != 0 {
			t.Fatalf("admin namespace create code = %d, want 0: %s", env.Code, rec.Body.String())
		}
	})

	t.Run("duplicate namespace conflicts", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/wiki/namespaces",
			`{"name":"deploy","description":"again"}`, bobToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("admin namespace conflict status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "wiki-namespace-create-conflict.json"))
	})

	t.Run("manager deletes empty namespace", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodDelete, "/api/admin/wiki/namespaces/deploy", "", bobToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("admin namespace delete status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		env := decodeContractEnvelope(t, rec)
		if env.Code != 0 {
			t.Fatalf("admin namespace delete code = %d, want 0: %s", env.Code, rec.Body.String())
		}
	})
}

func TestWikiUnauthenticatedHTTPContract(t *testing.T) {
	conn, router := setupWikiContractTest(t)
	seedWikiContract(t, conn, contractTestID())

	rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/wiki/pages",
		`{"namespace":"guide","path":"guide/x","title":"X","content":"x"}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wiki unauthenticated status = %d, want 401: %s", rec.Code, rec.Body.String())
	}
	env := decodeContractEnvelope(t, rec)
	if env.MessageCode != "auth.required" {
		t.Fatalf("wiki unauthenticated messageCode = %q, want auth.required", env.MessageCode)
	}
}

package searchservice

import (
	"errors"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/meiliconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
)

// wikiPruneTestPageID 回归测试专用高位 pageId：
// 共享的 wiki_pages Meilisearch 索引在开发环境同时被服务使用，
// 测试不能拿内存测试库的 1/2 号页去覆盖/误删真实 wiki 文档。
const wikiPruneTestPageID = uint64(9_000_001)

func setupWikiSearchTestDB(t *testing.T) {
	t.Helper()
	conn := dbconnect.Connect()
	models := []any{
		&topics.Entity{},
		&posts.Entity{},
		&wikiNamespaces.Entity{},
		&wikiPages.Entity{},
	}
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate wiki search schema: %v", err)
	}
	for _, model := range models {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean wiki search table: %v", err)
		}
	}
}

func seedWikiSearchPage(t *testing.T, ns, path, title, paraAnchorsJSON string) wikiPages.Entity {
	return seedWikiSearchPageWithID(t, ns, path, title, paraAnchorsJSON, 0)
}

// seedWikiSearchPageWithID 与 seedWikiSearchPage 相同，但允许指定页面 ID。
// 回归测试直写共享的 wiki_pages Meilisearch 索引：必须用远离真实数据的高位
// pageId（测试库内存库 ID 从 1 起），否则会覆盖/误删开发环境真实 wiki 的索引。
func seedWikiSearchPageWithID(t *testing.T, ns, path, title, paraAnchorsJSON string, pageID uint64) wikiPages.Entity {
	t.Helper()
	if !wikiNamespaces.Exists(ns) {
		if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: ns}); err != nil {
			t.Fatalf("create namespace %q: %v", ns, err)
		}
	}
	topic := topics.Entity{
		Title:            title,
		Status:           1,
		ProcessStatus:    topics.ProcessStatusNormal,
		TopicType:        topics.TopicTypeWiki,
		VisibilityStatus: topics.VisibilityActive,
		RetentionStatus:  topics.RetentionNormal,
	}
	if err := topics.Create(&topic); err != nil {
		t.Fatalf("create topic: %v", err)
	}
	post := posts.Entity{
		TopicId:          topic.Id,
		PostNo:           1,
		Content:          title,
		RenderedHTML:     title,
		ProcessStatus:    posts.ProcessStatusNormal,
		VisibilityStatus: posts.VisibilityActive,
		RetentionStatus:  posts.RetentionNormal,
	}
	if err := posts.Create(&post); err != nil {
		t.Fatalf("create post: %v", err)
	}
	topic.FirstPostId = post.Id
	topic.LastPostId = post.Id
	topic.PostSeq = 1
	if err := topics.Save(&topic); err != nil {
		t.Fatalf("save topic: %v", err)
	}
	page := wikiPages.Entity{
		Id:           pageID,
		TopicId:      topic.Id,
		Namespace:    ns,
		Path:         path,
		Title:        title,
		RenderedHTML: title,
		ParaAnchors:  paraAnchorsJSON,
	}
	if err := wikiPages.Create(&page); err != nil {
		t.Fatalf("create wiki page: %v", err)
	}
	return page
}

// TestWikiPageDocumentsShape 每段一文档：从 para_anchors JSON 构建段落文档，
// id 为 "<pageId>-<paraIndex>"，段落文本来自投影列（不二次解析 HTML）。
func TestWikiPageDocumentsShape(t *testing.T) {
	setupWikiSearchTestDB(t)
	page := seedWikiSearchPage(t, "guide", "guide/faq", "FAQ", `[
		{"index":1,"anchor":"s-1","headingId":"faq","headingText":"FAQ","text":"第一段正文"},
		{"index":2,"anchor":"s-2","headingId":"apply","headingText":"申请条件","text":"成绩均分不低于 3.0"}
	]`)

	docs := wikiPageDocuments(&page)
	if len(docs) != 2 {
		t.Fatalf("documents=%d, want 2: %+v", len(docs), docs)
	}
	if docs[0].ID != "1-1" || docs[1].ID != "1-2" {
		t.Fatalf("document ids=%s,%s, want 1-1,1-2", docs[0].ID, docs[1].ID)
	}
	if docs[1].Heading != "申请条件" || docs[1].Anchor != "s-2" || docs[1].Paragraph != "成绩均分不低于 3.0" {
		t.Fatalf("document[1]=%+v", docs[1])
	}
	if docs[0].PageId != page.Id || docs[0].Path != "guide/faq" || docs[0].Namespace != "guide" {
		t.Fatalf("document[0] page context=%+v", docs[0])
	}
}

// TestWikiPageDocumentsEmptyAnchors 无段落锚点（存量页/纯标题页）不产出文档。
func TestWikiPageDocumentsEmptyAnchors(t *testing.T) {
	setupWikiSearchTestDB(t)
	page := seedWikiSearchPage(t, "guide", "guide/empty", "空页", "")
	if docs := wikiPageDocuments(&page); len(docs) != 0 {
		t.Fatalf("documents=%d, want 0 (empty para_anchors)", len(docs))
	}
}

// TestSearchWikiPageHitsUnavailable 搜索服务不可用时返回 ErrSearchUnavailable
// （调用方降级为 searchUnavailable 空结果）。仅当 Meilisearch 不可用时生效；
// 本地已启动 Meilisearch 的环境跳过（此时验证真实搜索路径，见 build/query 测试）。
func TestSearchWikiPageHitsUnavailable(t *testing.T) {
	setupWikiSearchTestDB(t)
	if meiliconnect.IsAvailable() {
		t.Skip("Meilisearch available in this environment; unavailable fallback not testable")
	}
	_, err := SearchWikiPageHits("选课", 10)
	if err == nil || !errors.Is(err, ErrSearchUnavailable) {
		t.Fatalf("SearchWikiPageHits err=%v, want ErrSearchUnavailable", err)
	}
}

// TestIndexWikiPageDocumentsPrunesShrunkParagraphs 回归（review P1）：
// 增量索引必须先删后写——页面段落数减少时（3 段缩为 2 段）旧 <pageId>-n 文档
// 必须被清理，否则搜索仍命中已删除文本并把用户带到不存在的锚点；无段落页
// 同样清空该页索引，整页软删也清理。
// 依赖真实 Meilisearch（CI 无 Meilisearch 时跳过）。
func TestIndexWikiPageDocumentsPrunesShrunkParagraphs(t *testing.T) {
	setupWikiSearchTestDB(t)
	if !meiliconnect.IsAvailable() {
		t.Skip("Meilisearch not available; incremental pruning regression not testable")
	}
	ensureWikiPageIndexConfigured(t)
	// 用远离真实数据的高位 pageId：共享的 wiki_pages 索引在开发环境同时被
	// 服务使用，不能拿测试库的 1/2 号页去覆盖/误删真实 wiki 文档。
	page := seedWikiSearchPageWithID(t, "guide", "guide/faq", "FAQ", `[
		{"index":1,"anchor":"s-1","headingId":"faq","headingText":"FAQ","text":"成绩均分不低于 3.0"},
		{"index":2,"anchor":"s-2","headingId":"apply","headingText":"申请条件","text":"申请材料需齐全"}
	]`, wikiPruneTestPageID)
	t.Cleanup(func() { _ = DeleteWikiPageDocuments(page.Id) })
	if err := IndexWikiPageDocuments(page.Id); err != nil {
		t.Fatalf("index page: %v", err)
	}

	// 段落减少为 1 段：重新索引后旧段落关键词与旧锚点不得再命中。
	page.ParaAnchors = `[{"index":1,"anchor":"s-1","headingId":"faq","headingText":"FAQ","text":"成绩均分不低于 3.0"}]`
	if err := wikiPages.Save(&page); err != nil {
		t.Fatalf("save shrunk page: %v", err)
	}
	if err := IndexWikiPageDocuments(page.Id); err != nil {
		t.Fatalf("reindex shrunk page: %v", err)
	}
	assertNoPageHits(t, page.Id, "申请材料")

	// 无段落（纯标题页）：整页索引应清空，旧段落不再命中。
	page.ParaAnchors = ""
	if err := wikiPages.Save(&page); err != nil {
		t.Fatalf("save empty-paragraph page: %v", err)
	}
	if err := IndexWikiPageDocuments(page.Id); err != nil {
		t.Fatalf("reindex empty-paragraph page: %v", err)
	}
	assertNoPageHits(t, page.Id, "成绩")

	// 整页软删：索引侧清理，旧关键词不再命中。
	if err := DeleteWikiPageDocuments(page.Id); err != nil {
		t.Fatalf("delete page index: %v", err)
	}
	assertNoPageHits(t, page.Id, "成绩")
}

// TestWikiPageSearchTitleHitKeepsRawTitle 回归（review P2）：
// Meilisearch 的 _formatted.title 只用于判断标题命中，API 命中的 title
// 必须保持原始文本，避免前端再次高亮时把 <mark> 当作字面文本显示。
func TestWikiPageSearchTitleHitKeepsRawTitle(t *testing.T) {
	setupWikiSearchTestDB(t)
	if !meiliconnect.IsAvailable() {
		t.Skip("Meilisearch not available; title formatting regression not testable")
	}
	ensureWikiPageIndexConfigured(t)
	page := seedWikiSearchPageWithID(t, "guide", "guide/title-hit", "申请指南", `[
		{"index":1,"anchor":"s-1","headingId":"intro","headingText":"简介","text":"正文"}
	]`, wikiPruneTestPageID)
	t.Cleanup(func() { _ = DeleteWikiPageDocuments(page.Id) })
	if err := IndexWikiPageDocuments(page.Id); err != nil {
		t.Fatalf("index title-hit page: %v", err)
	}

	resp, err := SearchWikiPageHits("申请指南", 20)
	if err != nil {
		t.Fatalf("search title-hit page: %v", err)
	}
	for _, hit := range resp.Hits {
		if hit.PageId != page.Id {
			continue
		}
		if hit.Title != page.Title {
			t.Fatalf("title = %q, want raw title %q", hit.Title, page.Title)
		}
		if !hit.TitleHit {
			t.Fatal("titleHit = false, want true")
		}
		return
	}
	t.Fatalf("title-hit page %d not found in search results", page.Id)
}

// ensureWikiPageIndexConfigured 为 wiki_pages 索引补齐 searchable/filterable
// 配置（回归测试直接用单页增量路径，不走全量重建，需自行配置索引字段）。
func ensureWikiPageIndexConfigured(t *testing.T) {
	t.Helper()
	client := meiliconnect.GetClient()
	index := client.Index(WikiPageIndex)
	searchable := []string{"title", "heading", "paragraph"}
	if _, err := index.UpdateSearchableAttributes(&searchable); err != nil {
		t.Fatalf("configure wiki searchable: %v", err)
	}
	filterable := []any{"pageId", "namespace", "isPublic"}
	if _, err := index.UpdateFilterableAttributes(&filterable); err != nil {
		t.Fatalf("configure wiki filterable: %v", err)
	}
	waitIndexTask(t, client)
}

// assertNoPageHits 断言某页不再出现在搜索结果中（旧关键词被清理后不得命中）。
func assertNoPageHits(t *testing.T, pageID uint64, keyword string) {
	t.Helper()
	resp, err := SearchWikiPageHits(keyword, 20)
	if err != nil {
		t.Fatalf("search %q: %v", keyword, err)
	}
	for _, hit := range resp.Hits {
		if hit.PageId == pageID {
			t.Fatalf("page %d still hit for %q after cleanup: %+v", pageID, keyword, hit)
		}
	}
}

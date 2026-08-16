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

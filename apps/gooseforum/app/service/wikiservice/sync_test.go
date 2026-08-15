package wikiservice

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaceEditors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
)

// writeRepoFile 在临时仓库目录下写一个 md 文件（自动建父目录）。
func writeRepoFile(t *testing.T, root, rel string, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir repo dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write repo file %s: %v", rel, err)
	}
}

// TestApplyRepoToDBFirstSync 首次同步：仓库 md → wiki_pages/topics/posts 行，
// title 来自 frontmatter，content_hash 为正文 sha256。
func TestApplyRepoToDBFirstSync(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "docs/getting-started.md", "---\ntitle: 快速开始\norder: 2\n---\n\n# 标题\n\n正文内容")

	res := &SyncResult{}
	if err := applyRepoToDB(GitConfig{CloneDir: repo}, res); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if res.PagesAdded != 1 {
		t.Fatalf("PagesAdded=%d, want 1", res.PagesAdded)
	}
	page := wikiPages.GetByPath("docs/getting-started")
	if page.Id == 0 {
		t.Fatal("wiki_pages row missing after first sync")
	}
	if page.Title != "快速开始" {
		t.Fatalf("page title=%q, want frontmatter title 快速开始", page.Title)
	}
	if page.SortOrder != 2 {
		t.Fatalf("page sort_order=%d, want 2", page.SortOrder)
	}
	wantBody := "# 标题\n\n正文内容"
	if page.Content != wantBody {
		t.Fatalf("page content=%q, want %q", page.Content, wantBody)
	}
	if page.ContentHash != sha256Hex(wantBody) {
		t.Fatalf("content_hash=%q, want sha256(body)", page.ContentHash)
	}
	if page.RenderedHTML == "" {
		t.Fatal("rendered_html empty after first sync")
	}
	topic := topics.Get(page.TopicId)
	if topic.Id == 0 || topic.TopicType != topics.TopicTypeWiki || topic.Title != "快速开始" {
		t.Fatalf("topic after sync=%+v, want wiki topic with title", topic)
	}
	firstPost := posts.Get(topic.FirstPostId)
	if firstPost.Id == 0 || firstPost.Content != wantBody {
		t.Fatalf("first post after sync: id=%d content=%q", firstPost.Id, firstPost.Content)
	}
}

// TestApplyRepoToDBIdempotent 幂等：内容未变时重复同步零变更，不重复建行。
func TestApplyRepoToDBIdempotent(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "docs/page.md", "---\ntitle: 页面\n---\n\n# 标题\n\n正文")

	cfg := GitConfig{CloneDir: repo}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	res := &SyncResult{}
	if err := applyRepoToDB(cfg, res); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if res.PagesAdded != 0 || res.PagesUpdated != 0 || res.PagesDeleted != 0 {
		t.Fatalf("second sync added/updated/deleted=%d/%d/%d, want 0/0/0", res.PagesAdded, res.PagesUpdated, res.PagesDeleted)
	}
	pages := wikiPages.ListAll()
	if len(pages) != 1 {
		t.Fatalf("page count after second sync=%d, want 1", len(pages))
	}
}

// TestApplyRepoToDBUpdate 更新：md 内容变化 → PagesUpdated=1，
// content/title/rendered_html 同步刷新。
func TestApplyRepoToDBUpdate(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	rel := "docs/page.md"
	writeRepoFile(t, repo, rel, "---\ntitle: v1\n---\n\n# 标题\n\n旧正文")

	cfg := GitConfig{CloneDir: repo}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	page := wikiPages.GetByPath("docs/page")
	firstPost := posts.Get(topics.Get(page.TopicId).FirstPostId)
	oldRendered := firstPost.RenderedHTML

	writeRepoFile(t, repo, rel, "---\ntitle: v2\n---\n\n# 标题\n\n新正文")
	res := &SyncResult{}
	if err := applyRepoToDB(cfg, res); err != nil {
		t.Fatalf("update sync: %v", err)
	}
	if res.PagesUpdated != 1 {
		t.Fatalf("PagesUpdated=%d, want 1", res.PagesUpdated)
	}
	page = wikiPages.Get(page.Id)
	if page.Title != "v2" {
		t.Fatalf("page title after update=%q, want v2", page.Title)
	}
	if page.Content != "# 标题\n\n新正文" {
		t.Fatalf("page content after update=%q", page.Content)
	}
	if page.ContentHash != sha256Hex("# 标题\n\n新正文") {
		t.Fatal("content_hash not refreshed after update")
	}
	if page.RenderedHTML == "" || page.RenderedHTML == oldRendered {
		t.Fatal("rendered_html not refreshed after update")
	}
	topic := topics.Get(page.TopicId)
	got := posts.Get(topic.FirstPostId)
	if got.Content != "# 标题\n\n新正文" {
		t.Fatalf("first post content after update=%q", got.Content)
	}
}

// TestApplyRepoToDBDelete 删除：仓库移除 md → PagesDeleted=1，
// 页面软删（deleted_at 非空）+ topic visibility=USER_DELETED。
func TestApplyRepoToDBDelete(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "docs/gone.md", "---\ntitle: 删除\n---\n\n# 标题\n\n正文")

	cfg := GitConfig{CloneDir: repo}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	page := wikiPages.GetByPath("docs/gone")
	topicID := page.TopicId

	if err := os.Remove(filepath.Join(repo, "docs/gone.md")); err != nil {
		t.Fatal(err)
	}
	res := &SyncResult{}
	if err := applyRepoToDB(cfg, res); err != nil {
		t.Fatalf("delete sync: %v", err)
	}
	if res.PagesDeleted != 1 {
		t.Fatalf("PagesDeleted=%d, want 1", res.PagesDeleted)
	}
	unscoped := wikiPages.GetByPathUnscoped("docs/gone")
	if unscoped.Id == 0 || !unscoped.DeletedAt.Valid {
		t.Fatalf("page should be soft-deleted: id=%d deletedAt.Valid=%v", unscoped.Id, unscoped.DeletedAt.Valid)
	}
	topic := topics.UnscopedGet(topicID)
	if topic.VisibilityStatus != topics.VisibilityUserDeleted || !topic.DeletedAt.Valid {
		t.Fatalf("topic after delete: visibility=%q deletedAt.Valid=%v, want USER_DELETED/true", topic.VisibilityStatus, topic.DeletedAt.Valid)
	}
}

// TestApplyRepoToDBRestore 恢复：仓库重新出现已删页面 →
// (a) 内容未变：解除 wiki_pages/topic 软删、复用原 topic（评论/互动保留）；
// (b) 内容变化：同时刷新内容/哈希/首楼物化。
// 恢复必须走更新路径（含 topic 生命周期恢复），即使内容未变。
func TestApplyRepoToDBRestore(t *testing.T) {
	t.Run("unchanged content", func(t *testing.T) {
		setupWikiTestDB(t)
		repo := t.TempDir()
		rel := "docs/restore.md"
		content := "---\ntitle: 恢复\n---\n\n# 标题\n\n正文"
		writeRepoFile(t, repo, rel, content)

		cfg := GitConfig{CloneDir: repo}
		if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
			t.Fatalf("first sync: %v", err)
		}
		page := wikiPages.GetByPath("docs/restore")
		topicID := page.TopicId

		// 仓库移除 → 页面/topic 软删。
		if err := os.Remove(filepath.Join(repo, rel)); err != nil {
			t.Fatal(err)
		}
		if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
			t.Fatalf("delete sync: %v", err)
		}

		// 原样恢复 → 解除软删并复用原 topic；内容未变仍走更新路径。
		writeRepoFile(t, repo, rel, content)
		res := &SyncResult{}
		if err := applyRepoToDB(cfg, res); err != nil {
			t.Fatalf("restore sync: %v", err)
		}
		if res.PagesAdded != 0 || res.PagesDeleted != 0 {
			t.Fatalf("restore added/deleted=%d/%d, want 0/0", res.PagesAdded, res.PagesDeleted)
		}
		if res.PagesUpdated != 1 {
			t.Fatalf("PagesUpdated=%d, want 1 (restore must run update path)", res.PagesUpdated)
		}
		page = wikiPages.GetByPath("docs/restore")
		if page.Id == 0 {
			t.Fatal("page missing after restore")
		}
		if page.DeletedAt.Valid {
			t.Fatal("page still soft-deleted after restore")
		}
		if page.TopicId != topicID {
			t.Fatalf("topic id changed after restore: %d → %d, want reuse", topicID, page.TopicId)
		}
		if page.Content != "# 标题\n\n正文" {
			t.Fatalf("content changed on unchanged restore: %q", page.Content)
		}
		topic := topics.UnscopedGet(page.TopicId)
		if topic.Id == 0 || topic.DeletedAt.Valid {
			t.Fatalf("topic not restored: id=%d deletedAt.Valid=%v", topic.Id, topic.DeletedAt.Valid)
		}
		if topic.VisibilityStatus != topics.VisibilityActive {
			t.Fatalf("topic visibility after restore=%q, want ACTIVE", topic.VisibilityStatus)
		}
		if topic.RetentionStatus != topics.RetentionNormal {
			t.Fatalf("topic retention after restore=%q, want NORMAL", topic.RetentionStatus)
		}
		if got := posts.Get(topic.FirstPostId); got.Id == 0 || got.Content != "# 标题\n\n正文" {
			t.Fatalf("first post after restore: id=%d content=%q", got.Id, got.Content)
		}
	})

	t.Run("changed content", func(t *testing.T) {
		setupWikiTestDB(t)
		repo := t.TempDir()
		rel := "docs/restore.md"
		writeRepoFile(t, repo, rel, "---\ntitle: v1\n---\n\n# 标题\n\n旧正文")

		cfg := GitConfig{CloneDir: repo}
		if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
			t.Fatalf("first sync: %v", err)
		}
		page := wikiPages.GetByPath("docs/restore")
		topicID := page.TopicId

		if err := os.Remove(filepath.Join(repo, rel)); err != nil {
			t.Fatal(err)
		}
		if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
			t.Fatalf("delete sync: %v", err)
		}

		// 变化后恢复 → 解除软删并刷新内容/哈希/首楼。
		writeRepoFile(t, repo, rel, "---\ntitle: v2\n---\n\n# 标题\n\n新正文")
		res := &SyncResult{}
		if err := applyRepoToDB(cfg, res); err != nil {
			t.Fatalf("restore sync: %v", err)
		}
		if res.PagesUpdated != 1 {
			t.Fatalf("PagesUpdated=%d, want 1", res.PagesUpdated)
		}
		page = wikiPages.GetByPath("docs/restore")
		if page.Id == 0 || page.DeletedAt.Valid {
			t.Fatalf("page not restored: id=%d deletedAt.Valid=%v", page.Id, page.DeletedAt.Valid)
		}
		if page.TopicId != topicID {
			t.Fatalf("topic id changed after restore: %d → %d", topicID, page.TopicId)
		}
		if page.Title != "v2" || page.Content != "# 标题\n\n新正文" {
			t.Fatalf("page not refreshed: title=%q content=%q", page.Title, page.Content)
		}
		if page.ContentHash != sha256Hex("# 标题\n\n新正文") {
			t.Fatal("content_hash not refreshed after restore with changes")
		}
		topic := topics.UnscopedGet(page.TopicId)
		if topic.DeletedAt.Valid || topic.VisibilityStatus != topics.VisibilityActive {
			t.Fatalf("topic not restored: deletedAt.Valid=%v visibility=%q", topic.DeletedAt.Valid, topic.VisibilityStatus)
		}
		if got := posts.Get(topic.FirstPostId); got.Content != "# 标题\n\n新正文" {
			t.Fatalf("first post after restore=%q, want new body", got.Content)
		}
	})
}

// TestParseMarkdownFileFrontmatter frontmatter 解析边界：
// 无 frontmatter 用文件名、frontmatter 缺 title 兜底文件名、空正文。
func TestParseMarkdownFileFrontmatter(t *testing.T) {
	cases := []struct {
		name      string
		file      repoFile
		wantTitle string
		wantOrder int
		wantBody  string
	}{
		{
			name:      "frontmatter full",
			file:      repoFile{Path: "docs/a.md", Content: []byte("---\ntitle: 完整\norder: 3\ndescription: 描述\n---\n\n# 正文")},
			wantTitle: "完整",
			wantOrder: 3,
			wantBody:  "# 正文",
		},
		{
			name:      "no frontmatter uses filename",
			file:      repoFile{Path: "docs/b.md", Content: []byte("# 纯正文")},
			wantTitle: "b",
			wantOrder: 0,
			wantBody:  "# 纯正文",
		},
		{
			name:      "frontmatter missing title falls back to filename",
			file:      repoFile{Path: "docs/c.md", Content: []byte("---\norder: 1\n---\n\n正文")},
			wantTitle: "c",
			wantOrder: 1,
			wantBody:  "正文",
		},
		{
			name:      "empty body",
			file:      repoFile{Path: "docs/d.md", Content: []byte("---\ntitle: 空\n---\n\n   ")},
			wantTitle: "空",
			wantOrder: 0,
			wantBody:  "",
		},
		{
			name:      "frontmatter but no closing marker keeps raw body",
			file:      repoFile{Path: "docs/e.md", Content: []byte("---\ntitle: 未闭合\n\n# 正文")},
			wantTitle: "e",
			wantOrder: 0,
			wantBody:  "---\ntitle: 未闭合\n\n# 正文",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			title, order, _, body := parseMarkdownFile(tc.file)
			if title != tc.wantTitle {
				t.Fatalf("title=%q, want %q", title, tc.wantTitle)
			}
			if order != tc.wantOrder {
				t.Fatalf("order=%d, want %d", order, tc.wantOrder)
			}
			if body != tc.wantBody {
				t.Fatalf("body=%q, want %q", body, tc.wantBody)
			}
		})
	}
}

// TestScanRepoFilesSkipsNonMarkdown 扫描跳过 .git、隐藏目录、非 .md 文件；
// 结果按路径排序。
func TestScanRepoFilesSkipsNonMarkdown(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "docs/a.md", "# A")
	writeRepoFile(t, repo, "docs/b.md", "# B")
	writeRepoFile(t, repo, "docs/skip.txt", "not md")
	writeRepoFile(t, repo, ".hidden/c.md", "# hidden")
	writeRepoFile(t, repo, "docs/.secret.md", "# secret")
	writeRepoFile(t, repo, "docs/README.md", "# README")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, repo, ".git/config.md", "# git internal")

	files, err := scanRepoFiles(repo)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	// 只跳过隐藏目录与非 .md 文件；隐藏 .md 文件（docs/.secret.md）仍按
	// 普通 .md 收录（scanRepoFiles 只对目录做隐藏过滤）。
	if len(files) != 4 {
		t.Fatalf("scanned files=%d, want 4 (docs/.secret.md, docs/README.md, docs/a.md, docs/b.md): %+v", len(files), files)
	}
	wantPaths := []string{"docs/.secret.md", "docs/README.md", "docs/a.md", "docs/b.md"}
	for i, want := range wantPaths {
		if files[i].Path != want {
			t.Fatalf("scan[%d]=%q, want %q (all=%v)", i, files[i].Path, want, wantPaths)
		}
	}
}

// TestApplyRepoToDBNamespaceMetaFromIndex 顶层目录 index.md 的 frontmatter
// description/order → wiki_namespaces.description/sort_order（D4）。
func TestApplyRepoToDBNamespaceMetaFromIndex(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	// index.md 携带 description/order；页面文件正常同步。
	writeRepoFile(t, repo, "guide/index.md", "---\ntitle: 指南\ndescription: 社区使用指南\norder: 10\n---\n\n# 指南首页")
	writeRepoFile(t, repo, "guide/start.md", "---\ntitle: 开始\n---\n\n# 开始")

	cfg := GitConfig{CloneDir: repo}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	ns := wikiNamespaces.GetByName("guide")
	if ns.Id == 0 {
		t.Fatal("namespace guide missing")
	}
	if ns.Description != "社区使用指南" {
		t.Fatalf("namespace description=%q, want 社区使用指南", ns.Description)
	}
	if ns.SortOrder != 10 {
		t.Fatalf("namespace sort_order=%d, want 10", ns.SortOrder)
	}

	// 修改 index.md → 描述/排序更新；幂等检查。
	writeRepoFile(t, repo, "guide/index.md", "---\ntitle: 指南\ndescription: 新版描述\norder: 20\n---\n\n# 指南首页")
	res := &SyncResult{}
	if err := applyRepoToDB(cfg, res); err != nil {
		t.Fatalf("update sync: %v", err)
	}
	ns = wikiNamespaces.GetByName("guide")
	if ns.Description != "新版描述" || ns.SortOrder != 20 {
		t.Fatalf("namespace after update: desc=%q order=%d, want 新版描述/20", ns.Description, ns.SortOrder)
	}
}

// TestApplyRepoToDBDeleteNamespace 仓库顶层目录消失 → 命名空间自动删除（D5）。
// 页面先软删，命名空间行与贡献者记录一并清理。
func TestApplyRepoToDBDeleteNamespace(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "guide/start.md", "---\ntitle: 开始\n---\n\n# 开始")

	cfg := GitConfig{CloneDir: repo}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if !wikiNamespaces.Exists("guide") {
		t.Fatal("namespace guide should exist after first sync")
	}

	// 贡献者记录预置一条，验证删除时一并清理。
	if err := dbconnect.Connect().Create(&wikiNamespaceEditors.Entity{Namespace: "guide", UserId: 424242}).Error; err != nil {
		t.Fatalf("seed editor: %v", err)
	}

	// 删除整个顶层目录 → 页面软删 + 命名空间删除。
	if err := os.RemoveAll(filepath.Join(repo, "guide")); err != nil {
		t.Fatal(err)
	}
	res := &SyncResult{}
	if err := applyRepoToDB(cfg, res); err != nil {
		t.Fatalf("delete namespace sync: %v", err)
	}
	if res.PagesDeleted != 1 {
		t.Fatalf("PagesDeleted=%d, want 1", res.PagesDeleted)
	}
	if res.NamespacesDeleted != 1 {
		t.Fatalf("NamespacesDeleted=%d, want 1", res.NamespacesDeleted)
	}
	if wikiNamespaces.Exists("guide") {
		t.Fatal("namespace guide should be deleted")
	}
	page := wikiPages.GetByPathUnscoped("guide/start")
	if page.Id == 0 || !page.DeletedAt.Valid {
		t.Fatalf("page should be soft-deleted: id=%d deletedAt.Valid=%v", page.Id, page.DeletedAt.Valid)
	}
	// 贡献者记录清理。
	var editors int64
	if err := dbconnect.Connect().Table("wiki_namespace_editors").
		Where("namespace = ?", "guide").Count(&editors).Error; err != nil {
		t.Fatalf("count editors: %v", err)
	}
	if editors != 0 {
		t.Fatalf("editors after namespace delete=%d, want 0", editors)
	}

	// 目录重新出现 → 命名空间重建 + 页面恢复（复用原 topic）。
	// topic 已随页面软删，必须 Unscoped 读取（Get 带软删 scope 返回零值）。
	topicID := topics.UnscopedGet(page.TopicId).Id
	writeRepoFile(t, repo, "guide/start.md", "---\ntitle: 开始\n---\n\n# 开始")
	res = &SyncResult{}
	if err := applyRepoToDB(cfg, res); err != nil {
		t.Fatalf("restore namespace sync: %v", err)
	}
	if !wikiNamespaces.Exists("guide") {
		t.Fatal("namespace guide should be recreated")
	}
	page = wikiPages.GetByPath("guide/start")
	if page.Id == 0 || page.DeletedAt.Valid {
		t.Fatalf("page should be restored: id=%d deletedAt.Valid=%v", page.Id, page.DeletedAt.Valid)
	}
	if page.TopicId != topicID {
		t.Fatalf("topic changed after namespace restore: %d → %d, want reuse", topicID, page.TopicId)
	}
}

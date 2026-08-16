package wikiservice

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaceEditors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiSyncRuns"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
)

// setupWikiTestDB 迁移 wiki 相关表并清空。
// GitHub SSOT 后测试直接写 wiki_pages/topics 投影行，不再走 Create 写路径。
func setupWikiTestDB(t *testing.T) {
	t.Helper()
	conn := dbconnect.Connect()
	models := []any{
		&topics.Entity{},
		&posts.Entity{},
		&users.EntityComplete{},
		&rolePermissionRs.Entity{},
		&topicUserAction.Entity{},
		&wikiNamespaces.Entity{},
		&wikiNamespaceEditors.Entity{},
		&wikiPages.Entity{},
		&wikiSyncRuns.Entity{},
	}
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate wiki schema: %v", err)
	}
	for _, model := range models {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean wiki table: %v", err)
		}
	}
}

// wikiTestUserSeq 每次测试递增，保证用户 ID 唯一，避免 userservice 缓存串扰。
var wikiTestUserSeq uint64 = 1000

// seedWikiUser 创建测试用户；manager=true 时授予 PageManager（角色 ID 每次唯一）。
func seedWikiUser(t *testing.T, manager bool) uint64 {
	t.Helper()
	wikiTestUserSeq++
	id := wikiTestUserSeq
	user := &users.EntityComplete{
		Id:          id,
		Username:    fmt.Sprintf("wikiuser%d", id),
		Email:       fmt.Sprintf("wikiuser%d@example.test", id),
		IsActivated: users.ActivationSuccess,
	}
	if err := dbconnect.Connect().Create(user).Error; err != nil {
		t.Fatalf("create wiki test user: %v", err)
	}
	if !manager {
		return id
	}
	roleID := uint64(time.Now().UnixNano())
	if err := dbconnect.Connect().Model(&users.EntityComplete{}).Where("id = ?", id).Update("role_id", roleID).Error; err != nil {
		t.Fatalf("set role for wiki test user: %v", err)
	}
	if err := dbconnect.Connect().Create(&rolePermissionRs.Entity{RoleId: roleID, PermissionId: permission.PageManager.Id()}).Error; err != nil {
		t.Fatalf("grant PageManager to wiki test user: %v", err)
	}
	return id
}

// seedProjectedWikiPage 直接写 wiki_pages 投影行 + topics/posts 物化行
// （GitHub SSOT：内容来自仓库同步，测试绕过写路径直插投影表）。
func seedProjectedWikiPage(t *testing.T, ns, path, title string, updatedAt time.Time) wikiPages.Entity {
	t.Helper()
	if !wikiNamespaces.Exists(ns) {
		if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: ns}); err != nil {
			t.Fatalf("create namespace %q: %v", ns, err)
		}
	}
	topic := topics.Entity{
		Title:            title,
		UserId:           wikiSystemUserID,
		Status:           1,
		ProcessStatus:    topics.ProcessStatusNormal,
		TopicType:        topics.TopicTypeWiki,
		VisibilityStatus: topics.VisibilityActive,
		RetentionStatus:  topics.RetentionNormal,
	}
	if err := topics.Create(&topic); err != nil {
		t.Fatalf("create topic for %q: %v", path, err)
	}
	post := posts.Entity{
		TopicId:          topic.Id,
		PostNo:           1,
		UserId:           wikiSystemUserID,
		Content:          "# " + title,
		RenderedHTML:     "<p>" + title + "</p>",
		ProcessStatus:    posts.ProcessStatusNormal,
		VisibilityStatus: posts.VisibilityActive,
		RetentionStatus:  posts.RetentionNormal,
	}
	if err := posts.Create(&post); err != nil {
		t.Fatalf("create post for %q: %v", path, err)
	}
	topic.FirstPostId = post.Id
	topic.LastPostId = post.Id
	topic.PostSeq = 1
	if err := topics.Save(&topic); err != nil {
		t.Fatalf("save topic pointers for %q: %v", path, err)
	}
	page := wikiPages.Entity{
		TopicId:      topic.Id,
		Namespace:    ns,
		Path:         path,
		Title:        title,
		Content:      "# " + title,
		RenderedHTML: "<p>" + title + "</p>",
		ContentHash:  sha256Hex("# " + title),
		UpdatedAt:    updatedAt,
	}
	if err := wikiPages.Create(&page); err != nil {
		t.Fatalf("create wiki page %q: %v", path, err)
	}
	return page
}

func TestValidatePath(t *testing.T) {
	cases := []struct {
		input string
		ok    bool
	}{
		{"guide/getting-started", true},
		{"deployment/waline", true},
		{"guide/sub/page-name", true},
		{"同济新手教程/学校/简介", true},          // 中文命名空间与页面段（GitHub SSOT）
		{"中文/目录/页面", true},              // 纯中文路径
		{"Guide/Getting-Started", true}, // 保留大小写（不再小写归一）
		{"guide/UPPER", true},           // 大写段合法
		{"guide", false},                // 至少 namespace + 一个 slug 段
		{"guide/..", false},             // 禁止 ..
		{"guide/.hidden", false},        // 禁止点开头段
		{"guide/a b", false},            // 空格非法
		{"guide/a\tb", false},           // 控制字符非法
		{"guide/a:b", false},            // 保留字符非法
		{"guide/a*b", false},            // 保留字符非法
		{"guide/中文 空格", false},          // 中文路径含空格非法
		{"", false},
	}
	for _, tc := range cases {
		got, ok := ValidatePath(tc.input)
		if ok != tc.ok {
			t.Fatalf("ValidatePath(%q) ok=%v, want %v", tc.input, ok, tc.ok)
		}
		if ok && got != tc.input {
			t.Fatalf("ValidatePath(%q) normalized=%q, want unchanged (no lowercasing)", tc.input, got)
		}
	}
}

func TestValidateNamespace(t *testing.T) {
	cases := []struct {
		input string
		ok    bool
	}{
		{"guide", true},
		{"deployment", true},
		{"my-namespace", true},
		{"同济新手教程", true}, // 中文命名空间（GitHub 顶层目录名）
		{"使用指南", true},   // 中文命名空间
		{"Guide", true},  // 保留大小写
		{"UPPER", true},  // 保留大小写
		{"has space", false},
		{"中文 空格", false},   // 中间空格非法
		{" 前导空格", true},    // 首尾空格被 trim 后为合法名称（trim 后再校验）
		{".hidden", false}, // 点开头（隐藏目录）非法
		{"a:b", false},     // 保留字符非法
		{"a*b", false},     // 保留字符非法
		{"", false},
	}
	for _, tc := range cases {
		if got := ValidateNamespace(tc.input); got != tc.ok {
			t.Fatalf("ValidateNamespace(%q) = %v, want %v", tc.input, got, tc.ok)
		}
	}
}

func TestValidateNamespaceLengthByRunes(t *testing.T) {
	// 长度按字符（rune）计数：64 个中文字符合法，65 个非法。
	short := strings.Repeat("济", 64)
	if !ValidateNamespace(short) {
		t.Fatalf("ValidateNamespace(64 中文) = false, want true (按字符计数)")
	}
	long := strings.Repeat("济", 65)
	if ValidateNamespace(long) {
		t.Fatalf("ValidateNamespace(65 中文) = true, want false")
	}
}

// TestBuildTreeGroupsNamespacesAndActive 公开导航树：按 namespace 分组，
// path 为完整路径，active 标记当前页（投影表驱动，不再查修订表）。
func TestBuildTreeGroupsNamespacesAndActive(t *testing.T) {
	setupWikiTestDB(t)
	base := time.Now().Add(-24 * time.Hour)
	seedProjectedWikiPage(t, "docs", "docs/old", "Old", base)
	seedProjectedWikiPage(t, "docs", "docs/new", "New", base.Add(time.Hour))
	seedProjectedWikiPage(t, "guide", "guide/start", "Start", base.Add(2*time.Hour))

	tree, err := BuildTree("docs/new")
	if err != nil {
		t.Fatalf("BuildTree: %v", err)
	}
	if len(tree) != 2 {
		t.Fatalf("tree namespaces=%d, want 2: %+v", len(tree), tree)
	}
	var docs *TreeNamespace
	for i := range tree {
		if tree[i].Name == "docs" {
			docs = &tree[i]
			break
		}
	}
	if docs == nil {
		t.Fatal("tree missing docs namespace")
	}
	if len(docs.Pages) != 2 {
		t.Fatalf("docs pages=%d, want 2: %+v", len(docs.Pages), docs.Pages)
	}
	paths := map[string]bool{}
	activeCount := 0
	for _, p := range docs.Pages {
		paths[p.Path] = true
		if p.Active {
			activeCount++
			if p.Path != "docs/new" {
				t.Fatalf("unexpected active page %q", p.Path)
			}
		}
	}
	if activeCount != 1 {
		t.Fatalf("active pages=%d, want 1", activeCount)
	}
	if !paths["docs/old"] || !paths["docs/new"] {
		t.Fatalf("docs tree paths=%v", paths)
	}
}

// TestBuildTreeAPIRelativePaths 契约形状导航树：path 为 namespace 内相对路径。
func TestBuildTreeAPIRelativePaths(t *testing.T) {
	setupWikiTestDB(t)
	base := time.Now().Add(-24 * time.Hour)
	seedProjectedWikiPage(t, "docs", "docs/guide/tips", "Tips", base)
	seedProjectedWikiPage(t, "guide", "guide/start", "Start", base.Add(time.Hour))

	res, err := BuildTreeAPI()
	if err != nil {
		t.Fatalf("BuildTreeAPI: %v", err)
	}
	if len(res.Namespaces) != 2 {
		t.Fatalf("namespaces=%d, want 2: %+v", len(res.Namespaces), res.Namespaces)
	}
	var docs *TreeNamespace
	for i := range res.Namespaces {
		if res.Namespaces[i].Name == "docs" {
			docs = &res.Namespaces[i]
			break
		}
	}
	if docs == nil || len(docs.Pages) != 1 {
		t.Fatalf("docs tree: %+v", res.Namespaces)
	}
	if docs.Pages[0].Path != "guide/tips" {
		t.Fatalf("API tree path=%q, want relative guide/tips", docs.Pages[0].Path)
	}
	if docs.Pages[0].Active {
		t.Fatal("API tree page should not be active (no active path)")
	}
}

// TestBuildHomeRecentByUpdatedAt 首页最近更新按投影 updated_at 降序。
func TestBuildHomeRecentByUpdatedAt(t *testing.T) {
	setupWikiTestDB(t)
	base := time.Now().Add(-72 * time.Hour)
	seedProjectedWikiPage(t, "docs", "docs/old", "Old", base)
	seedProjectedWikiPage(t, "docs", "docs/mid", "Mid", base.Add(time.Hour))
	seedProjectedWikiPage(t, "docs", "docs/new", "New", base.Add(2*time.Hour))

	home, err := BuildHome()
	if err != nil {
		t.Fatalf("BuildHome: %v", err)
	}
	if len(home.Recent) != 3 {
		t.Fatalf("recent count=%d, want 3: %+v", len(home.Recent), home.Recent)
	}
	wantOrder := []string{"docs/new", "docs/mid", "docs/old"}
	for i, want := range wantOrder {
		if home.Recent[i].Path != want {
			t.Fatalf("recent[%d]=%q, want %q", i, home.Recent[i].Path, want)
		}
	}
}

// TestBuildNamespaceSummaries namespace 摘要：pageCount 与 firstPagePath
// 均来自投影表公开页面。
func TestBuildNamespaceSummaries(t *testing.T) {
	setupWikiTestDB(t)
	base := time.Now().Add(-24 * time.Hour)
	seedProjectedWikiPage(t, "docs", "docs/a", "A", base)
	seedProjectedWikiPage(t, "docs", "docs/b", "B", base.Add(time.Hour))
	seedProjectedWikiPage(t, "guide", "guide/start", "Start", base.Add(2*time.Hour))

	summaries, err := BuildNamespaceSummaries()
	if err != nil {
		t.Fatalf("BuildNamespaceSummaries: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("summaries=%d, want 2: %+v", len(summaries), summaries)
	}
	var docs *NamespaceSummary
	for i := range summaries {
		if summaries[i].Name == "docs" {
			docs = &summaries[i]
			break
		}
	}
	if docs == nil {
		t.Fatal("summaries missing docs")
	}
	if docs.PageCount != 2 {
		t.Fatalf("docs pageCount=%d, want 2", docs.PageCount)
	}
	if docs.FirstPagePath != "docs/a" {
		t.Fatalf("docs firstPagePath=%q, want docs/a", docs.FirstPagePath)
	}
}

// TestLoadPageDetailFromProjection 详情页 title/content/toc 直接来自投影列
// （rendered_html 快照 + toc JSON，不再查修订表）。
func TestLoadPageDetailFromProjection(t *testing.T) {
	setupWikiTestDB(t)
	page := seedProjectedWikiPage(t, "docs", "docs/detail", "详情页", time.Now().Add(-time.Hour))
	if err := dbconnect.Connect().Table("wiki_pages").Where("id = ?", page.Id).
		Update("toc", `[{"level":1,"id":"detail","text":"详情"}]`).Error; err != nil {
		t.Fatalf("set toc: %v", err)
	}
	page = wikiPages.Get(page.Id)
	topic := topics.Get(page.TopicId)
	detail, err := LoadPageDetail(&page, &topic)
	if err != nil {
		t.Fatalf("load detail: %v", err)
	}
	if detail.Title != "详情页" {
		t.Fatalf("detail.Title=%q, want projection title 详情页", detail.Title)
	}
	if detail.Content != page.RenderedHTML {
		t.Fatalf("detail.Content=%q, want projection rendered_html %q", detail.Content, page.RenderedHTML)
	}
	if len(detail.Toc) != 1 || detail.Toc[0].Level != 1 || detail.Toc[0].Text != "详情" {
		t.Fatalf("detail.Toc=%+v, want one item level=1 text=详情", detail.Toc)
	}
	if detail.UpdatedAt != page.UpdatedAt.Format(time.RFC3339) {
		t.Fatalf("detail.UpdatedAt=%q, want %q", detail.UpdatedAt, page.UpdatedAt.Format(time.RFC3339))
	}
}

// TestBuildContributorsFromProjection 贡献者读 wiki_pages.contributors_json 缓存；
// GitHub 贡献者无论坛账号，userId/avatarUrl 恒为空，仅 username/count。
func TestBuildContributorsFromProjection(t *testing.T) {
	setupWikiTestDB(t)
	page := seedProjectedWikiPage(t, "docs", "docs/contrib", "Contrib", time.Now())
	if err := dbconnect.Connect().Table("wiki_pages").Where("id = ?", page.Id).
		Update("contributors_json", `[{"name":"Alice","count":5},{"name":"Bob","count":2}]`).Error; err != nil {
		t.Fatalf("set contributors: %v", err)
	}
	contribs := BuildContributors(page.Id)
	if len(contribs) != 2 {
		t.Fatalf("contributors=%d, want 2: %+v", len(contribs), contribs)
	}
	if contribs[0].Username != "Alice" || contribs[0].Count != 5 {
		t.Fatalf("contributors[0]=%+v, want Alice/5", contribs[0])
	}
	if contribs[1].Username != "Bob" || contribs[1].Count != 2 {
		t.Fatalf("contributors[1]=%+v, want Bob/2", contribs[1])
	}
	if contribs[0].UserId != 0 || contribs[0].AvatarUrl != "" {
		t.Fatalf("contributor user linkage should be empty: %+v", contribs[0])
	}
	if got := BuildContributors(0); len(got) != 0 {
		t.Fatalf("unknown page contributors=%d, want 0", len(got))
	}
}

// TestHasPageManagerPermission PageManager（含 Admin）权限判定。
func TestHasPageManagerPermission(t *testing.T) {
	setupWikiTestDB(t)
	manager := seedWikiUser(t, true)
	plain := seedWikiUser(t, false)
	if !HasPageManagerPermission(manager) {
		t.Fatal("PageManager user should have permission")
	}
	if HasPageManagerPermission(plain) {
		t.Fatal("plain user should not have PageManager")
	}
	if HasPageManagerPermission(0) {
		t.Fatal("system user 0 should not have PageManager")
	}
}

// TestValidatePathLengthByRunes 整路径长度按码点（rune）计数 ≤255：
// 255 个 runes 合法、256 个非法；100 个中文字符（300 字节）按码点计数合法。
// 与 DB varchar(255) 字符语义及前端码点计数对齐（review 建议 3）。
// 注意：每段仍有 ≤64 的独立上限，构造长路径需多段分摊。
func TestValidatePathLengthByRunes(t *testing.T) {
	// ns=1 + 斜杠 + 四段 63/63/62/62 + 3 个斜杠 = 255 runes。
	path255 := "a/" +
		strings.Repeat("b", 63) + "/" +
		strings.Repeat("b", 63) + "/" +
		strings.Repeat("b", 62) + "/" +
		strings.Repeat("b", 62)
	if got, ok := ValidatePath(path255); !ok {
		t.Fatalf("ValidatePath(255 ascii runes) = false, want true")
	} else if got != path255 {
		t.Fatalf("ValidatePath(255 runes) normalized=%q, want unchanged", got)
	}
	// 100 个中文字符（300 字节）：按码点计数合法（此前按字节会被误拒）。
	chinese100 := "济/" + strings.Repeat("济", 63) + "/" + strings.Repeat("济", 34)
	if _, ok := ValidatePath(chinese100); !ok {
		t.Fatalf("ValidatePath(100 chinese runes, 300 bytes) = false, want true (counted by runes)")
	}
	// ns=1 + 四段 63/63/63/62 + 4 个斜杠 = 256 runes 非法。
	path256 := "a/" +
		strings.Repeat("b", 63) + "/" +
		strings.Repeat("b", 63) + "/" +
		strings.Repeat("b", 63) + "/" +
		strings.Repeat("b", 62)
	if _, ok := ValidatePath(path256); ok {
		t.Fatalf("ValidatePath(256 runes) = true, want false")
	}
}

// TestGitConfigEditURLPathEscapesSegments GitHub 外链逐段转义：
// `#`/`%` 等仓库合法目录字符必须被 PathEscape，避免 `#` 开启 fragment
// 或 `%` 被当转义前缀 → GitHub 404（review 建议 2）。
// 调用方传不带 .md 的仓库相对路径（source_path），扩展名由 EditURL 追加。
func TestGitConfigEditURLPathEscapesSegments(t *testing.T) {
	cfg := GitConfig{Repo: "https://github.com/YourTongji/YourTJ-Wiki.git", Branch: "main"}
	got := cfg.EditURL("C#/x")
	want := "https://github.com/YourTongji/YourTJ-Wiki/edit/main/C%23/x.md"
	if got != want {
		t.Fatalf("EditURL(C#/x) = %q, want %q", got, want)
	}
	got = cfg.EditURL("100%/README")
	want = "https://github.com/YourTongji/YourTJ-Wiki/edit/main/100%25/README.md"
	if got != want {
		t.Fatalf("EditURL(100%%) = %q, want %q", got, want)
	}
	// 普通 slug 路径保持原样（无转义副作用）。
	got = cfg.EditURL("guide/getting-started")
	want = "https://github.com/YourTongji/YourTJ-Wiki/edit/main/guide/getting-started.md"
	if got != want {
		t.Fatalf("EditURL(guide/getting-started) = %q, want %q", got, want)
	}
	// 中文段被转义（URL 安全）。
	got = cfg.EditURL("同济新手教程/start")
	if !strings.Contains(got, "/edit/main/%E5%90%8C%E6%B5%8E%E6%96%B0%E6%89%8B%E6%95%99%E7%A8%8B/start.md") {
		t.Fatalf("EditURL(chinese dir) = %q, want URL-escaped first segment", got)
	}
	// HistoryURL 同样逐段转义。
	got = cfg.HistoryURL("C#/x")
	want = "https://github.com/YourTongji/YourTJ-Wiki/commits/main/C%23/x.md"
	if got != want {
		t.Fatalf("HistoryURL(C#/x) = %q, want %q", got, want)
	}
}

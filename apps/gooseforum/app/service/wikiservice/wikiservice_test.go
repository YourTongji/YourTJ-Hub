package wikiservice

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
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
		&eventNotification.Entity{},
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
		t.Fatalf("build tree: %v", err)
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
	if len(docs.Nodes) != 2 {
		t.Fatalf("docs nodes=%d, want 2: %+v", len(docs.Nodes), docs.Nodes)
	}
	paths := map[string]bool{}
	activeCount := 0
	for _, p := range docs.Nodes {
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
		t.Fatalf("build tree api: %v", err)
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
	if docs == nil || len(docs.Nodes) != 1 {
		t.Fatalf("docs tree: %+v", res.Namespaces)
	}
	if docs.Nodes[0].Kind != WikiTreeNodeDirectory || len(docs.Nodes[0].Children) != 1 || docs.Nodes[0].Children[0].Path != "guide/tips" {
		t.Fatalf("API tree nodes=%+v, want guide directory with relative guide/tips", docs.Nodes)
	}
	if docs.Nodes[0].Children[0].Active {
		t.Fatal("API tree page should not be active (no active path)")
	}
}

// TestBuildTreePreservesRepositoryDirectories verifies that directory segments are
// emitted as tree nodes even when the directory has no index.md page. The content
// repository treats directories as hierarchy, not as a requirement for a parent
// markdown file.
func TestBuildTreePreservesRepositoryDirectories(t *testing.T) {
	setupWikiTestDB(t)
	base := time.Now().Add(-24 * time.Hour)
	index := seedProjectedWikiPage(t, "guide", "guide/index", "指南首页", base)
	admissionIndex := seedProjectedWikiPage(t, "guide", "guide/admission/index", "入学", base.Add(time.Minute))
	process := seedProjectedWikiPage(t, "guide", "guide/admission/process", "流程", base.Add(2*time.Minute))
	faq := seedProjectedWikiPage(t, "guide", "guide/faq/common", "常见问题", base.Add(3*time.Minute))
	if err := dbconnect.Connect().Table("wiki_pages").Where("id = ?", admissionIndex.Id).Update("sort_order", 2).Error; err != nil {
		t.Fatalf("set admission order: %v", err)
	}
	if err := dbconnect.Connect().Table("wiki_pages").Where("id = ?", process.Id).Update("sort_order", 1).Error; err != nil {
		t.Fatalf("set process order: %v", err)
	}

	tree, err := BuildTree("guide/admission/process")
	if err != nil {
		t.Fatalf("build tree: %v", err)
	}
	if len(tree) != 1 {
		t.Fatalf("namespaces=%d, want 1: %+v", len(tree), tree)
	}
	// Root index remains a page. admission has an index.md page and faq has no
	// index.md, but both directories must retain their descendants.
	nodes := tree[0].Nodes
	if len(nodes) != 3 {
		t.Fatalf("root nodes=%d, want index + admission + faq: %+v", len(nodes), nodes)
	}
	byPath := make(map[string]TreeNode, len(nodes))
	for _, node := range nodes {
		byPath[node.Path] = node
	}
	if node := byPath["guide/index"]; node.Kind != WikiTreeNodePage || node.PageId != index.Id {
		t.Fatalf("root index node=%+v", node)
	}
	admissionNode := byPath["guide/admission"]
	if admissionNode.Kind != WikiTreeNodeDirectory || len(admissionNode.Children) != 2 {
		t.Fatalf("admission directory=%+v", admissionNode)
	}
	if admissionNode.Children[0].PageId != process.Id || !admissionNode.Children[0].Active || admissionNode.Children[1].PageId != admissionIndex.Id {
		t.Fatalf("admission children=%+v, want process then index by sibling order", admissionNode.Children)
	}
	faqNode := byPath["guide/faq"]
	if faqNode.Kind != WikiTreeNodeDirectory || len(faqNode.Children) != 1 || faqNode.Children[0].PageId != faq.Id {
		t.Fatalf("faq directory=%+v", faqNode)
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
		t.Fatalf("build home: %v", err)
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
		t.Fatalf("build namespace summaries: %v", err)
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
// GitHub 贡献者无论坛账号，userId 恒 0；noreply 邮箱解析出 username 时
// avatarUrl/githubUrl 可用，否则三者皆空（前端降级首字母占位）。
func TestBuildContributorsFromProjection(t *testing.T) {
	setupWikiTestDB(t)
	page := seedProjectedWikiPage(t, "docs", "docs/contrib", "Contrib", time.Now())
	if err := dbconnect.Connect().Table("wiki_pages").Where("id = ?", page.Id).
		Update("contributors_json", `[
			{"name":"Alice","email":"12345+alice@users.noreply.github.com","username":"alice","count":5},
			{"name":"Bob","email":"bob@example.com","count":2}
		]`).Error; err != nil {
		t.Fatalf("set contributors: %v", err)
	}
	contribs := BuildContributors(page.Id)
	if len(contribs) != 2 {
		t.Fatalf("contributors=%d, want 2: %+v", len(contribs), contribs)
	}
	if contribs[0].Username != "Alice" || contribs[0].Count != 5 {
		t.Fatalf("contributors[0]=%+v, want Alice/5", contribs[0])
	}
	if contribs[0].UserId != 0 {
		t.Fatalf("contributor userId should be 0: %+v", contribs[0])
	}
	if contribs[0].AvatarUrl != "https://github.com/alice.png?size=56" {
		t.Fatalf("contributor avatarUrl = %q, want github avatar URL", contribs[0].AvatarUrl)
	}
	if contribs[0].GithubUrl != "https://github.com/alice" {
		t.Fatalf("contributor githubUrl = %q, want github profile URL", contribs[0].GithubUrl)
	}
	// 自定义邮箱：无头像/链接（降级）。
	if contribs[1].Username != "Bob" || contribs[1].Count != 2 {
		t.Fatalf("contributors[1]=%+v, want Bob/2", contribs[1])
	}
	if contribs[1].AvatarUrl != "" || contribs[1].GithubUrl != "" {
		t.Fatalf("custom-email contributor should have no avatar/link: %+v", contribs[1])
	}
	if got := BuildContributors(0); len(got) != 0 {
		t.Fatalf("unknown page contributors=%d, want 0", len(got))
	}
}

// TestGithubUsernameFromEmail 从 GitHub noreply 隐私邮箱解析用户名：
// 新版 {id}+{username}、旧版 {username}、自定义邮箱/非法用户名返回空。
func TestGithubUsernameFromEmail(t *testing.T) {
	cases := []struct {
		email string
		want  string
	}{
		{"12345+alice@users.noreply.github.com", "alice"},
		{"bob@users.noreply.github.com", "bob"},
		{"c-harlie@users.noreply.github.com", "c-harlie"},
		{"bob@example.com", ""},                    // 自定义邮箱
		{"", ""},                                   // 空
		{"-bad@users.noreply.github.com", ""},      // 连字符开头非法
		{"bad-@users.noreply.github.com", ""},      // 连字符结尾非法
		{"has space@users.noreply.github.com", ""}, // 空格非法
		{"12345+@users.noreply.github.com", ""},    // + 后为空
		{"@users.noreply.github.com", ""},          // local 为空
	}
	for _, tc := range cases {
		if got := githubUsernameFromEmail(tc.email); got != tc.want {
			t.Errorf("githubUsernameFromEmail(%q) = %q, want %q", tc.email, got, tc.want)
		}
	}
}

// TestBuildContributorsSnapshotAggregatesByUsername 贡献者快照按可解析 username
// 聚合（合并 GitHub 新旧 noreply 格式——username 相同；自定义邮箱降级按 email），
// 展示名取最近提交的 name，count 排序。
func TestBuildContributorsSnapshotAggregatesByUsername(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
	repo := t.TempDir()
	git := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	// 旧版 noreply 格式：{username}@users.noreply.github.com。
	git("config", "user.email", "old@users.noreply.github.com")
	git("config", "user.name", "Old Name")
	writeRepoFile(t, repo, "guide/start.md", "# 一")
	git("add", "-A")
	git("commit", "-q", "-m", "c1")
	// 同一人换新版 noreply 格式（{id}+{username}）+ 改显示名 → 按 username 合并为 1 人。
	git("config", "user.email", "12345+old@users.noreply.github.com")
	git("config", "user.name", "New Name")
	writeRepoFile(t, repo, "guide/start.md", "# 二")
	git("add", "-A")
	git("commit", "-q", "-m", "c2")
	// 自定义邮箱另一人（无法解析 username，按 email 聚合）。
	git("config", "user.email", "bob@example.com")
	git("config", "user.name", "Bob")
	writeRepoFile(t, repo, "guide/start.md", "# 三")
	git("add", "-A")
	git("commit", "-q", "-m", "c3")

	raw := buildContributorsSnapshot(repo, "guide/start.md")
	var got []gitContributor
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("unmarshal snapshot %q: %v", raw, err)
	}
	if len(got) != 2 {
		t.Fatalf("contributors=%d, want 2 (same user merged): %+v", len(got), got)
	}
	// 同人合并后 count=2，展示名 = 最近提交的 New Name。
	if got[0].Name != "New Name" || got[0].Count != 2 || got[0].Username != "old" {
		t.Fatalf("merged contributor = %+v, want New Name/2/old", got[0])
	}
	if got[1].Name != "Bob" || got[1].Count != 1 || got[1].Username != "" {
		t.Fatalf("custom email contributor = %+v, want Bob/1/no username", got[1])
	}
}

// TestRebuildGitTracesRefreshesUnchangedPages 浅克隆→全量升级后全量重建贡献者
// 缓存（review P1 回归）：applyRepoToDB 幂等跳过内容未变的页面，rebuildGitTraces
// 必须仍然刷新存量页面的 contributors_json——旧 depth-1 缓存只有一位作者，
// 且生产 source_path 不带 .md 后缀（gitLogPath 补全后 git pathspec 才能匹配）。
func TestRebuildGitTracesRefreshesUnchangedPages(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "guide/start.md", "# 一")
	initGitRepo(t, repo)
	// 第二个 noreply 作者提交 → git log 有两位作者。
	git := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("config", "user.email", "12345+alice@users.noreply.github.com")
	git("config", "user.name", "Alice")
	writeRepoFile(t, repo, "guide/start.md", "# 二")
	git("add", "-A")
	git("commit", "-q", "-m", "c2")

	page := seedProjectedWikiPage(t, "guide", "guide/start", "Start", time.Now())
	// 生产数据：source_path 不带 .md（gitLogPath 必须补全才能匹配 pathspec）。
	if err := dbconnect.Connect().Table("wiki_pages").Where("id = ?", page.Id).
		Update("source_path", "guide/start").Error; err != nil {
		t.Fatalf("set source_path: %v", err)
	}
	// 旧缓存：只有 1 位作者（模拟 depth-1 时代的缓存）。
	if err := dbconnect.Connect().Table("wiki_pages").Where("id = ?", page.Id).
		Update("contributors_json", `[{"name":"test","count":1}]`).Error; err != nil {
		t.Fatalf("set old contributors: %v", err)
	}

	rebuildGitTraces(GitConfig{CloneDir: repo})

	page = wikiPages.Get(page.Id)
	if page.ContributorsJSON == `[{"name":"test","count":1}]` {
		t.Fatal("contributors_json not refreshed after rebuildGitTraces")
	}
	var got []gitContributor
	if err := json.Unmarshal([]byte(page.ContributorsJSON), &got); err != nil {
		t.Fatalf("unmarshal refreshed contributors %q: %v", page.ContributorsJSON, err)
	}
	if len(got) != 2 {
		t.Fatalf("contributors=%d, want 2 (both git authors): %+v", len(got), got)
	}
	foundAlice := false
	for _, c := range got {
		if c.Username == "alice" {
			foundAlice = true
		}
	}
	if !foundAlice {
		t.Fatalf("alice missing after rebuild: %+v", got)
	}
	if page.LastCommitSha == "" {
		t.Fatal("last_commit_sha not set after rebuild")
	}
}

// TestBuildContributorsSnapshotFollowsRename 贡献者统计跨 Git 重命名历史
// （review P2 回归）：git mv 后在新路径 git log --follow 必须归因重命名前的
// 旧作者；source_path 不带 .md 的形式同样工作（gitLogPath 补全）。
func TestBuildContributorsSnapshotFollowsRename(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
	repo := t.TempDir()
	git := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	git("config", "user.email", "alice@users.noreply.github.com")
	git("config", "user.name", "Alice")
	writeRepoFile(t, repo, "guide/old.md", "# 一")
	git("add", "-A")
	git("commit", "-q", "-m", "c1")
	git("config", "user.email", "bob@users.noreply.github.com")
	git("config", "user.name", "Bob")
	git("mv", "guide/old.md", "guide/new.md")
	git("commit", "-q", "-m", "c2 rename")

	// 新路径：--follow 必须归因重命名前的 Alice。
	raw := buildContributorsSnapshot(repo, "guide/new.md")
	var got []gitContributor
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("unmarshal snapshot %q: %v", raw, err)
	}
	if len(got) != 2 {
		t.Fatalf("contributors=%d, want 2 (alice+bob via --follow): %+v", len(got), got)
	}
	byName := map[string]int{}
	for _, c := range got {
		byName[c.Name] = c.Count
	}
	if byName["Alice"] != 1 || byName["Bob"] != 1 {
		t.Fatalf("rename attribution wrong: %+v", got)
	}

	// source_path 形式（不带 .md）也必须工作：gitLogPath 补 .md。
	raw2 := buildContributorsSnapshot(repo, "guide/new")
	var got2 []gitContributor
	if err := json.Unmarshal([]byte(raw2), &got2); err != nil {
		t.Fatalf("unmarshal snapshot (no .md) %q: %v", raw2, err)
	}
	if len(got2) != 2 {
		t.Fatalf("contributors (no .md path)=%d, want 2: %+v", len(got2), got2)
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

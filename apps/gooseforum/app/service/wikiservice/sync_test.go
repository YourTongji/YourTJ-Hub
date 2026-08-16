package wikiservice

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaceEditors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiSyncRuns"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

func TestApplyRepoToDBRewritesRelativeReferences(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "同济指南/index.md", "---\ntitle: 指南\nslug: guide\n---\n\n# 指南")
	writeRepoFile(t, repo, "同济指南/other.md", "# 下一页")
	writeRepoFile(t, repo, "同济指南/nested/start.md", `# 开始

[下一页](../other.md?tab=2#section)
![图片](../../assets/a%20b.png?raw=1#preview)
[附件](../../assets/handout.pdf)
[外部](https://example.com/guide)
[根路径](/static/logo.svg)`)
	writeRepoFile(t, repo, "assets/a b.png", "png")
	writeRepoFile(t, repo, "assets/handout.pdf", "pdf")

	if err := applyRepoToDB(GitConfig{CloneDir: repo}, &SyncResult{}); err != nil {
		t.Fatalf("sync: %v", err)
	}
	page := wikiPages.GetByPath("guide/nested/start")
	if page.Id == 0 {
		t.Fatal("nested page missing after sync")
	}
	for _, want := range []string{
		`href="/wiki/guide/other?tab=2#section"`,
		`src="/wiki/_assets/assets/a%20b.png?raw=1#preview"`,
		`href="/wiki/_assets/assets/handout.pdf"`,
		`href="https://example.com/guide"`,
		`href="/static/logo.svg"`,
	} {
		if !strings.Contains(page.RenderedHTML, want) {
			t.Fatalf("rendered HTML missing %s: %s", want, page.RenderedHTML)
		}
	}
}

func TestApplyRepoToDBRejectsBrokenRelativeReferences(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "escapes repository",
			content: "# 开始\n\n![图片](../../outside.png)",
			want:    "escapes repository root",
		},
		{
			name:    "missing page",
			content: "# 开始\n\n[下一页](missing.md)",
			want:    `linked page "guide/missing.md" does not exist`,
		},
		{
			name:    "missing asset",
			content: "# 开始\n\n![图片](missing.png)",
			want:    `asset "guide/missing.png": asset not found`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupWikiTestDB(t)
			repo := t.TempDir()
			writeRepoFile(t, repo, "guide/start.md", tc.content)

			err := applyRepoToDB(GitConfig{CloneDir: repo}, &SyncResult{})
			if err == nil || !strings.Contains(err.Error(), "wiki source guide/start.md") || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("sync error = %v, want actionable error containing %q", err, tc.want)
			}
		})
	}
}

// TestApplyRepoToDBDegradesPerPageOnBrokenReferences review M1：
// 单个页面坏引用（missing 类）→ 该页跳过（保留 DB 旧版本）+ 聚合告警，
// 其余页面正常 upsert；只有安全类错误（仓库根逃逸）才整体失败。
func TestApplyRepoToDBDegradesPerPageOnBrokenReferences(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "guide/broken.md", "# 坏页\n\n[下一页](missing.md)")
	writeRepoFile(t, repo, "guide/good.md", "# 好页\n\n[下一页](broken.md)")
	writeRepoFile(t, repo, "guide/index.md", "---\ntitle: 指南\nslug: guide\n---\n\n# 指南")

	cfg := GitConfig{CloneDir: repo}
	res := &SyncResult{}
	err := applyRepoToDB(cfg, res)
	if err == nil {
		t.Fatal("sync error = nil, want aggregated per-page failure")
	}
	if !strings.Contains(err.Error(), "skip guide/broken") {
		t.Fatalf("sync error = %v, want skip guide/broken", err)
	}
	// 好页仍被创建；被跳过页不存在（首次同步，无旧版本可保留）。
	if page := wikiPages.GetByPath("guide/good"); page.Id == 0 {
		t.Fatal("good page missing after degraded sync")
	}
	if page := wikiPages.GetByPath("guide/broken"); page.Id != 0 {
		t.Fatalf("broken page should not be created, got id=%d", page.Id)
	}
}

// TestApplyRepoToDBKeepsEscapeFatal review M1：仓库根逃逸是安全类错误，
// 不允许 per-page 降级，必须整体失败（恶意链接不得通过「坏页跳过」绕过）。
func TestApplyRepoToDBKeepsEscapeFatal(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "guide/start.md", "# 开始\n\n![图片](../../outside.png)")
	writeRepoFile(t, repo, "guide/good.md", "# 好页")

	err := applyRepoToDB(GitConfig{CloneDir: repo}, &SyncResult{})
	if err == nil || !strings.Contains(err.Error(), "escapes repository root") {
		t.Fatalf("sync error = %v, want fatal escape error", err)
	}
	// 逃逸致命：整体失败，好页也不创建。
	if page := wikiPages.GetByPath("guide/good"); page.Id != 0 {
		t.Fatal("good page should not be created when sync fails fatally")
	}
}

func TestApplyRepoToDBRefreshesRelativeLinksAfterSlugChange(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "指南/index.md", "---\nslug: guide\n---\n\n# 指南")
	writeRepoFile(t, repo, "指南/start.md", "# 开始\n\n[下一页](../文档/other.md)")
	writeRepoFile(t, repo, "文档/index.md", "---\nslug: docs\n---\n\n# 文档")
	writeRepoFile(t, repo, "文档/other.md", "# 下一页")
	cfg := GitConfig{CloneDir: repo}

	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	start := wikiPages.GetByPath("guide/start")
	if !strings.Contains(start.RenderedHTML, `href="/wiki/docs/other"`) {
		t.Fatalf("first rendered link = %s, want /wiki/docs/other", start.RenderedHTML)
	}

	writeRepoFile(t, repo, "文档/index.md", "---\nslug: handbook\n---\n\n# 文档")
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("sync after slug change: %v", err)
	}
	start = wikiPages.GetByPath("guide/start")
	if !strings.Contains(start.RenderedHTML, `href="/wiki/handbook/other"`) {
		t.Fatalf("refreshed rendered link = %s, want /wiki/handbook/other", start.RenderedHTML)
	}
}

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

// initGitRepo 在临时目录初始化 git 仓库并提交全部文件（同步器 ensureClone
// 需要真实 git 仓库；测试环境缺 git 二进制时跳过）。
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
	git := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	git("config", "user.email", "test@example.test")
	git("config", "user.name", "test")
	git("add", "-A")
	git("commit", "-q", "-m", "init")
}

// TestMarkAllRunningAbandoned 崩溃恢复原语（issue #290）：全部 running 行统一
// 标记 failed + error + finished_at，终态行不受影响。
func TestMarkAllRunningAbandoned(t *testing.T) {
	setupWikiTestDB(t)
	running := wikiSyncRuns.Entity{Trigger: "manual", Status: wikiSyncRuns.StatusRunning}
	if err := wikiSyncRuns.Create(&running); err != nil {
		t.Fatalf("create running run: %v", err)
	}
	done := wikiSyncRuns.Entity{Trigger: "webhook", Status: wikiSyncRuns.StatusSuccess}
	if err := wikiSyncRuns.Create(&done); err != nil {
		t.Fatalf("create success run: %v", err)
	}

	n, err := wikiSyncRuns.MarkAllRunningAbandoned("abandoned: process restarted before run finished")
	if err != nil {
		t.Fatalf("mark abandoned: %v", err)
	}
	if n != 1 {
		t.Fatalf("abandoned count = %d, want 1", n)
	}
	got := wikiSyncRuns.GetById(running.Id)
	if got.Status != wikiSyncRuns.StatusFailed {
		t.Fatalf("abandoned run status = %d, want failed", got.Status)
	}
	if !strings.Contains(got.Error, "abandoned") {
		t.Fatalf("abandoned run error = %q, want abandoned marker", got.Error)
	}
	if got.FinishedAt == nil {
		t.Fatal("abandoned run finished_at not set")
	}
	if still := wikiSyncRuns.GetById(done.Id); still.Status != wikiSyncRuns.StatusSuccess {
		t.Fatalf("terminal run status = %d, want success untouched", still.Status)
	}
}

// TestSyncOnceReconcilesStaleRunningRuns 崩溃遗留的 running 行在下次同步开始前
// 被回收（issue #290）：残留行标记 failed，新 run 正常执行并 success。
func TestSyncOnceReconcilesStaleRunningRuns(t *testing.T) {
	setupWikiTestDB(t)
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
	repo := t.TempDir()
	writeRepoFile(t, repo, "docs/page.md", "---\ntitle: 页面\n---\n\n# 标题\n\n正文")
	initGitRepo(t, repo)

	stale := wikiSyncRuns.Entity{Trigger: "manual", Status: wikiSyncRuns.StatusRunning}
	if err := wikiSyncRuns.Create(&stale); err != nil {
		t.Fatalf("create stale running run: %v", err)
	}

	cfg := GitConfig{Repo: "file://" + repo, Branch: "main", CloneDir: filepath.Join(t.TempDir(), "clone")}
	res, err := SyncWithConfig(cfg, "manual")
	if err != nil {
		t.Fatalf("SyncWithConfig: %v", err)
	}
	if res.PagesAdded != 1 {
		t.Fatalf("PagesAdded = %d, want 1", res.PagesAdded)
	}

	got := wikiSyncRuns.GetById(stale.Id)
	if got.Status != wikiSyncRuns.StatusFailed {
		t.Fatalf("stale run status = %d, want failed (reconciled)", got.Status)
	}
	latest, err := wikiSyncRuns.Latest()
	if err != nil {
		t.Fatalf("load latest run: %v", err)
	}
	if latest.Id == stale.Id || latest.Status != wikiSyncRuns.StatusSuccess {
		t.Fatalf("latest run = id %d status %d, want new success run", latest.Id, latest.Status)
	}
}

// TestBuildSyncStatusReconcilesStaleRunning 状态读取时自愈（issue #290）：
// 进程内无同步锁占用时，残留 running 行在 BuildSyncStatus 内被标记 failed，
// 管理端刷新页面即可解除按钮禁用。
func TestBuildSyncStatusReconcilesStaleRunning(t *testing.T) {
	setupWikiTestDB(t)
	stale := wikiSyncRuns.Entity{Trigger: "manual", Status: wikiSyncRuns.StatusRunning}
	if err := wikiSyncRuns.Create(&stale); err != nil {
		t.Fatalf("create stale running run: %v", err)
	}

	status, err := BuildSyncStatus()
	if err != nil {
		t.Fatalf("build sync status: %v", err)
	}
	if status.LastRun == nil || status.LastRun.Status != "failed" {
		t.Fatalf("lastRun after status read = %+v, want failed", status.LastRun)
	}
}

// TestBuildSyncStatusKeepsLiveRunningWhileLockHeld 同步锁被占用（本进程同步
// 进行中）时，BuildSyncStatus 不得回收 running 行——误回收会杀死在途同步。
func TestBuildSyncStatusKeepsLiveRunningWhileLockHeld(t *testing.T) {
	setupWikiTestDB(t)
	live := wikiSyncRuns.Entity{Trigger: "manual", Status: wikiSyncRuns.StatusRunning}
	if err := wikiSyncRuns.Create(&live); err != nil {
		t.Fatalf("create live run: %v", err)
	}
	if !TryAcquireSyncLock() {
		t.Fatal("acquire sync lock")
	}
	defer ReleaseSyncLock()

	status, err := BuildSyncStatus()
	if err != nil {
		t.Fatalf("build sync status: %v", err)
	}
	if status.LastRun == nil || status.LastRun.Status != "running" {
		t.Fatalf("live run must stay running while lock held, got %+v", status.LastRun)
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

// TestApplyRepoToDBReconcilesParentsAfterTreeChanges verifies that parent_id is a
// derived cache of the nearest ancestor index page. Reconciliation happens after
// all files are upserted, so repository ordering, moves, deletes, and restores
// cannot leave a stale relationship behind.
func TestApplyRepoToDBReconcilesParentsAfterTreeChanges(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	cfg := GitConfig{CloneDir: repo}

	// Deliberately create the leaf before either ancestor index. scanRepoFiles
	// sorts paths, but the reconciliation pass must be correct independently of
	// create order.
	writeRepoFile(t, repo, "guide/admission/process.md", "---\ntitle: 流程\n---\n\n# 流程")
	writeRepoFile(t, repo, "guide/index.md", "---\ntitle: 指南\n---\n\n# 指南")
	writeRepoFile(t, repo, "guide/admission/index.md", "---\ntitle: 入学\n---\n\n# 入学")
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	root := wikiPages.GetByPath("guide/index")
	admission := wikiPages.GetByPath("guide/admission/index")
	process := wikiPages.GetByPath("guide/admission/process")
	if root.ParentId != 0 || admission.ParentId != root.Id || process.ParentId != admission.Id {
		t.Fatalf("initial parent chain root/admission/process=%d/%d/%d, want 0/%d/%d", root.ParentId, admission.ParentId, process.ParentId, root.Id, admission.Id)
	}

	// Removing the directory index leaves the page visible under a synthetic
	// directory node and falls back to the next ancestor index rather than
	// retaining the deleted page id.
	if err := os.Remove(filepath.Join(repo, "guide/admission/index.md")); err != nil {
		t.Fatal(err)
	}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("delete index sync: %v", err)
	}
	if process = wikiPages.GetByPath("guide/admission/process"); process.ParentId != root.Id {
		t.Fatalf("process parent after deleting index=%d, want root %d", process.ParentId, root.Id)
	}

	// Restoring the index must reconnect the unchanged child to its restored row.
	writeRepoFile(t, repo, "guide/admission/index.md", "---\ntitle: 入学\n---\n\n# 入学")
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("restore index sync: %v", err)
	}
	admission = wikiPages.GetByPath("guide/admission/index")
	process = wikiPages.GetByPath("guide/admission/process")
	if process.ParentId != admission.Id {
		t.Fatalf("process parent after restoring index=%d, want %d", process.ParentId, admission.Id)
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
	pages, err := wikiPages.ListAll()
	if err != nil {
		t.Fatalf("list pages after second sync: %v", err)
	}
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
// 无 frontmatter 用文件名、frontmatter 缺 title 兜底文件名、空正文、slug 解析。
func TestParseMarkdownFileFrontmatter(t *testing.T) {
	cases := []struct {
		name      string
		file      repoFile
		wantTitle string
		wantOrder int
		wantSlug  string
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
			name:      "frontmatter with slug",
			file:      repoFile{Path: "同济新手教程/index.md", Content: []byte("---\ntitle: 教程\nslug: tongji-freshman-guide\n---\n\n# 首页")},
			wantTitle: "教程",
			wantSlug:  "tongji-freshman-guide",
			wantBody:  "# 首页",
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
			title, order, _, slug, body := parseMarkdownFile(tc.file)
			if title != tc.wantTitle {
				t.Fatalf("title=%q, want %q", title, tc.wantTitle)
			}
			if order != tc.wantOrder {
				t.Fatalf("order=%d, want %d", order, tc.wantOrder)
			}
			if slug != tc.wantSlug {
				t.Fatalf("slug=%q, want %q", slug, tc.wantSlug)
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

// TestScanRepoFilesRejectsSymlinks review F2：Git 克隆会把仓库符号链接物化为
// symlink，若跟随读取可把服务器任意文件投影为公开 wiki 页（任意文件泄露）
// 或读 /dev/zero 类目标永久阻塞同步。扫描必须 lstat 拒绝符号链接。
func TestScanRepoFilesRejectsSymlinks(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "docs/ok.md", "# OK")
	writeRepoFile(t, repo, "docs/target.md", "# 目标")
	if err := os.Symlink(filepath.Join(repo, "docs/target.md"), filepath.Join(repo, "docs/link.md")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	_, err := scanRepoFiles(repo)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("scan error = %v, want symlink rejection", err)
	}
}

// TestScanRepoFilesRejectsOversize review F2：超大 .md（超 4MiB）拒绝，
// 防止恶意仓库撑爆内存。
func TestScanRepoFilesRejectsOversize(t *testing.T) {
	repo := t.TempDir()
	big := make([]byte, maxWikiPageBytes+1)
	for i := range big {
		big[i] = 'a'
	}
	writeRepoFile(t, repo, "docs/big.md", string(big))

	_, err := scanRepoFiles(repo)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("scan error = %v, want size rejection", err)
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
func namespaceSlugOrEmpty(t *testing.T, name string) string {
	t.Helper()
	ns := wikiNamespaces.GetByName(name)
	if ns.Id == 0 {
		t.Fatalf("namespace %s missing", name)
	}
	return ns.SlugOrEmpty()
}

// TestApplyRepoToDBSlugDefaultsToASCIIDirName 纯 ASCII 目录名且未声明 slug →
// 默认 slug=目录名，页面 path 首段=slug（URL 用 slug，D7）；index.md 变更
// frontmatter slug 后 slug 与页面 path 首段同步迁移。
func TestApplyRepoToDBSlugDefaultsToASCIIDirName(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "guide/index.md", "---\ntitle: 指南\n---\n\n# 指南首页")
	writeRepoFile(t, repo, "guide/start.md", "---\ntitle: 开始\n---\n\n# 开始")

	cfg := GitConfig{CloneDir: repo}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if got := namespaceSlugOrEmpty(t, "guide"); got != "guide" {
		t.Fatalf("slug=%q, want default dir name guide", got)
	}
	if page := wikiPages.GetByPath("guide/start"); page.Id == 0 {
		t.Fatal("page guide/start missing (path first segment = slug)")
	}

	// index.md 显式声明 slug → 覆盖目录名默认值，页面 path 首段跟随迁移。
	writeRepoFile(t, repo, "guide/index.md", "---\ntitle: 指南\nslug: tongji-guide\n---\n\n# 指南首页")
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("update sync: %v", err)
	}
	if got := namespaceSlugOrEmpty(t, "guide"); got != "tongji-guide" {
		t.Fatalf("slug=%q, want tongji-guide from frontmatter", got)
	}
	if page := wikiPages.GetByPath("tongji-guide/start"); page.Id == 0 {
		t.Fatal("page path first segment not migrated to new slug tongji-guide/start")
	}
	if page := wikiPages.GetByPath("guide/start"); page.Id != 0 {
		t.Fatal("old path guide/start should be gone after slug migration")
	}
}

// TestApplyRepoToDBSlugKeepsChineseNameNull 中文目录名且 index.md 未声明 slug
// → slug 保持 NULL，URL key 降级=显示名（页面 path 首段=中文目录名）；
// 仓库声明 slug 后 slug 填充且页面 path 首段迁移为 slug（D7 降级策略）。
func TestApplyRepoToDBSlugKeepsChineseNameNull(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "同济新手教程/index.md", "---\ntitle: 新手教程\n---\n\n# 首页")
	writeRepoFile(t, repo, "同济新手教程/start.md", "---\ntitle: 开始\n---\n\n# 开始")

	cfg := GitConfig{CloneDir: repo}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if got := namespaceSlugOrEmpty(t, "同济新手教程"); got != "" {
		t.Fatalf("slug=%q, want empty for chinese name without frontmatter slug", got)
	}
	// 降级：无 slug 时 URL key=显示名，页面 path 首段=中文目录名。
	if page := wikiPages.GetByPath("同济新手教程/start"); page.Id == 0 {
		t.Fatal("page 同济新手教程/start missing (URL key falls back to display name)")
	}

	// 仓库 index.md 声明 slug → 填充，页面 path 首段迁移为 slug。
	writeRepoFile(t, repo, "同济新手教程/index.md", "---\ntitle: 新手教程\nslug: tongji-freshman-guide\n---\n\n# 首页")
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("update sync: %v", err)
	}
	if got := namespaceSlugOrEmpty(t, "同济新手教程"); got != "tongji-freshman-guide" {
		t.Fatalf("slug=%q, want tongji-freshman-guide", got)
	}
	if page := wikiPages.GetByPath("tongji-freshman-guide/start"); page.Id == 0 {
		t.Fatal("page path first segment not migrated to slug after frontmatter slug added")
	}
	if page := wikiPages.GetByPath("同济新手教程/start"); page.Id != 0 {
		t.Fatal("old path 同济新手教程/start should be gone after slug migration")
	}
}

// TestApplyRepoToDBSlugConflictKeepsOldValue slug 已被其他命名空间占用 →
// 报错并保留旧值（fail-fast，run 标记 failed）。
// 两个命名空间同时索要 slug=guide：一个目录名默认（guide），一个 index.md
// frontmatter 显式声明（指南）。map 遍历顺序不确定，但无论谁先处理，
// 恰好一个持有 slug=guide、另一个冲突保留旧值（空），且同步报错。
func TestApplyRepoToDBSlugConflictKeepsOldValue(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "guide/start.md", "---\ntitle: 开始\n---\n\n# 开始")
	writeRepoFile(t, repo, "指南/index.md", "---\ntitle: 指南\nslug: guide\n---\n\n# 首页")
	writeRepoFile(t, repo, "指南/start.md", "---\ntitle: 开始\n---\n\n# 开始")

	cfg := GitConfig{CloneDir: repo}
	err := applyRepoToDB(cfg, &SyncResult{})
	if err == nil {
		t.Fatal("want error on slug conflict")
	}
	slugs := map[string]string{}
	namespaces, err := wikiNamespaces.List()
	if err != nil {
		t.Fatalf("list namespaces: %v", err)
	}
	for _, ns := range namespaces {
		slugs[ns.Name] = ns.SlugOrEmpty()
	}
	guideHas := slugs["guide"] == "guide"
	zhinanHas := slugs["指南"] == "guide"
	if guideHas == zhinanHas {
		t.Fatalf("slug=guide must be held by exactly one namespace, got guide=%q 指南=%q", slugs["guide"], slugs["指南"])
	}
}

// TestApplyRepoToDBSlugInvalidFromFrontmatter 声明非法 slug（大写/特殊字符）→
// fail-fast 报错且不落库（保留旧值 NULL）；页面本身仍正常同步。
func TestApplyRepoToDBSlugInvalidFromFrontmatter(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "guide/index.md", "---\ntitle: 指南\nslug: Bad_Slug!\n---\n\n# 指南首页")

	cfg := GitConfig{CloneDir: repo}
	err := applyRepoToDB(cfg, &SyncResult{})
	if err == nil {
		t.Fatal("want error on invalid slug")
	}
	if got := namespaceSlugOrEmpty(t, "guide"); got != "" {
		t.Fatalf("slug=%q, want empty (invalid slug rejected, keep old value)", got)
	}
	if page := wikiPages.GetByPath("guide/index"); page.Id == 0 {
		t.Fatal("page should still be synced despite invalid namespace slug")
	}
}

// TestResolvePageByURLPath D7 路由语义（URL 用 slug）：
//  1. slug 首段路径直查命中（新 URL，如 /wiki/tongji-freshman-guide/start）；
//  2. 中文显示名 URL 回退：无 slug 时 path 首段=显示名，直接命中；
//  3. 声明 slug 后旧链接兼容：首段=显示名的 URL 按 name→urlKey 重建命中。
func TestResolvePageByURLPath(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "同济新手教程/index.md", "---\ntitle: 新手教程\nslug: tongji-freshman-guide\n---\n\n# 首页")
	writeRepoFile(t, repo, "同济新手教程/start.md", "---\ntitle: 开始\n---\n\n# 开始")

	cfg := GitConfig{CloneDir: repo}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	// 新 URL：slug 首段直查命中。
	if page := ResolvePageByURLPath("tongji-freshman-guide/start"); page.Id == 0 {
		t.Fatal("slug URL tongji-freshman-guide/start should resolve directly")
	}
	// 旧链接兼容：中文显示名 URL（声明 slug 前的链接）按 name→slug 重建命中。
	if page := ResolvePageByURLPath("同济新手教程/start"); page.Id == 0 {
		t.Fatal("display-name URL 同济新手教程/start should resolve via name→slug rebuild")
	}
	// 未命中：不存在的页面。
	if page := ResolvePageByURLPath("tongji-freshman-guide/nope"); page.Id != 0 {
		t.Fatal("nonexistent page should not resolve")
	}
	if page := ResolvePageByURLPath("不存在的命名空间/start"); page.Id != 0 {
		t.Fatal("nonexistent namespace should not resolve")
	}
}

// TestResolvePageByURLPathFallbackUnassigned 中文目录未声明 slug 时
// path 首段=显示名（降级），直查即命中（无需回退）。
func TestResolvePageByURLPathFallbackUnassigned(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "同济新手教程/index.md", "---\ntitle: 新手教程\n---\n\n# 首页")
	writeRepoFile(t, repo, "同济新手教程/start.md", "---\ntitle: 开始\n---\n\n# 开始")

	cfg := GitConfig{CloneDir: repo}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if page := ResolvePageByURLPath("同济新手教程/start"); page.Id == 0 {
		t.Fatal("fallback URL (display name as URL key) should resolve directly")
	}
}

// TestApplyRepoToDBCDNSwitchDoesNotNotifyWatchers review P2：切换 CDN 只改变
// RenderedHTML（正文/标题/排序未变）时更新页面但不发送 wiki_updated 通知；
// 正文真正变化时仍通知（避免 CDN 切换给所有订阅者发误导通知）。
func TestApplyRepoToDBCDNSwitchDoesNotNotifyWatchers(t *testing.T) {
	setupWikiTestDB(t)
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&eventNotification.Entity{}, &pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate notification/page_config: %v", err)
	}
	conn.Unscoped().Where("page_type = ?", pageConfig.WikiSyncSettings).Delete(&pageConfig.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&eventNotification.Entity{})
	hotdataserve.ClearWikiSyncSettingsConfigCache()
	t.Cleanup(func() {
		conn.Unscoped().Where("page_type = ?", pageConfig.WikiSyncSettings).Delete(&pageConfig.Entity{})
		conn.Unscoped().Where("1 = 1").Delete(&eventNotification.Entity{})
		hotdataserve.ClearWikiSyncSettingsConfigCache()
	})

	repo := t.TempDir()
	writeRepoFile(t, repo, "docs/page.md", "---\ntitle: 页面\n---\n\n# 标题\n\n![图](../assets/a.png)")
	writeRepoFile(t, repo, "assets/a.png", "png")
	cfg := GitConfig{
		CloneDir: repo,
		Repo:     "https://github.com/YourTongji/YourTJ-Wiki.git",
		Branch:   "main",
	}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	page := wikiPages.GetByPath("docs/page")
	if !strings.Contains(page.RenderedHTML, "/wiki/_assets/") {
		t.Fatalf("first render should use self route: %s", page.RenderedHTML)
	}
	// 订阅者：任意用户 watch 该 topic。
	watcher := seedWikiUser(t, false)
	if !topicUserAction.SetWatched(watcher, page.TopicId, true) {
		t.Fatal("set watcher failed")
	}

	countNotifications := func() int64 {
		var n int64
		conn.Table("event_notification").
			Where("topic_id = ? AND event_type = ?", page.TopicId, eventNotification.EventTypeWikiUpdated).
			Count(&n)
		return n
	}

	// CDN 切换 → 仅渲染变化：页面更新但 watcher 不通知。
	if err := conn.Create(&pageConfig.Entity{
		PageType: pageConfig.WikiSyncSettings,
		Config:   `{"assetCDN":"jsDelivr"}`,
	}).Error; err != nil {
		t.Fatalf("persist cdn setting: %v", err)
	}
	hotdataserve.ClearWikiSyncSettingsConfigCache()
	res := &SyncResult{}
	if err := applyRepoToDB(cfg, res); err != nil {
		t.Fatalf("cdn-switch sync: %v", err)
	}
	if res.PagesUpdated != 1 {
		t.Fatalf("PagesUpdated=%d, want 1", res.PagesUpdated)
	}
	page = wikiPages.GetByPath("docs/page")
	if !strings.Contains(page.RenderedHTML, "cdn.jsdelivr.net") {
		t.Fatalf("cdn render should use jsDelivr mirror: %s", page.RenderedHTML)
	}
	if got := countNotifications(); got != 0 {
		t.Fatalf("wiki_updated notifications after CDN-only re-render = %d, want 0", got)
	}

	// 正文变化 → 通知照常发送。
	writeRepoFile(t, repo, "docs/page.md", "---\ntitle: 页面\n---\n\n# 标题\n\n![图](../assets/a.png)\n\n新增段落")
	res = &SyncResult{}
	if err := applyRepoToDB(cfg, res); err != nil {
		t.Fatalf("content-change sync: %v", err)
	}
	if got := countNotifications(); got != 1 {
		t.Fatalf("wiki_updated notifications after content change = %d, want 1", got)
	}
}

// TestApplyRepoToDBInvalidNestedPathFailsFast 非法嵌套页面路径 → 同步整体
// 失败（fail-fast），绝不静默跳过并报告成功（issue #283）。
// 根级 README/CONTRIBUTING 等元文件仍显式排除、不阻断同步。
func TestApplyRepoToDBInvalidNestedPathFailsFast(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	// 合法中文路径页面（正常投影）。
	writeRepoFile(t, repo, "同济新手教程/start.md", "---\ntitle: 开始\n---\n\n# 开始")
	// 非法嵌套路径：段含空格（保留字符，跨平台可创建；不用冒号——Windows
	// 会把 "foo:bar" 解释为 NTFS Alternate Data Stream 语法，filepath.Walk
	// 不会枚举到该文件，测试无法覆盖目标）。
	writeRepoFile(t, repo, "guide/foo bar.md", "---\ntitle: 非法\n---\n\n# 非法")
	// 根级元文件：显式排除，不阻断。
	writeRepoFile(t, repo, "README.md", "# Wiki")
	writeRepoFile(t, repo, "CONTRIBUTING.md", "# 贡献指南")

	cfg := GitConfig{CloneDir: repo}
	err := applyRepoToDB(cfg, &SyncResult{})
	if err == nil {
		t.Fatal("want error for invalid nested page path, got nil")
	}
	if !strings.Contains(err.Error(), "guide/foo bar") {
		t.Fatalf("error should name the invalid path, got: %v", err)
	}
	if !strings.Contains(err.Error(), "invalid segment") {
		t.Fatalf("error should include the reason, got: %v", err)
	}
	// 非法路径存在时不得投影任何页面（fail-fast 在任何写入之前）。
	pages, err := wikiPages.ListAll()
	if err != nil {
		t.Fatalf("list all pages: %v", err)
	}
	if len(pages) != 0 {
		t.Fatalf("pages projected despite invalid path: %d, want 0 (fail-fast before any write)", len(pages))
	}
}

// TestApplyRepoToDBValidChinesePathProjects 中文目录/文件名路径正常投影
// （issue #283 根因回归：ASCII-only 校验已放宽为 Unicode 目录名兼容）。
func TestApplyRepoToDBValidChinesePathProjects(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "同济新手教程/学校/简介.md", "---\ntitle: 简介\n---\n\n# 简介")
	writeRepoFile(t, repo, "同济新手教程/学校/社团活动.md", "---\ntitle: 社团活动\n---\n\n# 社团活动")

	cfg := GitConfig{CloneDir: repo}
	res := &SyncResult{}
	if err := applyRepoToDB(cfg, res); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if res.PagesAdded != 2 {
		t.Fatalf("PagesAdded=%d, want 2 (中文嵌套路径全部投影)", res.PagesAdded)
	}
	if page := wikiPages.GetByPath("同济新手教程/学校/简介"); page.Id == 0 {
		t.Fatal("中文嵌套页面 同济新手教程/学校/简介 missing after sync")
	}
	if page := wikiPages.GetByPath("同济新手教程/学校/社团活动"); page.Id == 0 {
		t.Fatal("中文嵌套页面 同济新手教程/学校/社团活动 missing after sync")
	}
}

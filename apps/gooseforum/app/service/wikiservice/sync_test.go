package wikiservice

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	writeRepoFile(t, repo, "guide/index.md", "---\ntitle: 指南\n---\n\n# 指南")
	writeRepoFile(t, repo, "guide/other.md", "# 下一页")
	writeRepoFile(t, repo, "guide/nested/start.md", `# 开始

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
	writeRepoFile(t, repo, "guide/index.md", "---\ntitle: 指南\n---\n\n# 指南")

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

func TestApplyRepoToDBRefreshesRelativeLinksAfterDirectoryRename(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "指南/index.md", "---\ntitle: 指南\n---\n\n# 指南")
	writeRepoFile(t, repo, "指南/start.md", "# 开始\n\n[下一页](../文档/other.md)")
	writeRepoFile(t, repo, "文档/index.md", "---\ntitle: 文档\n---\n\n# 文档")
	writeRepoFile(t, repo, "文档/other.md", "# 下一页")
	cfg := GitConfig{CloneDir: repo}

	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	start := wikiPages.GetByPath("指南/start")
	// 相对链接解析为目标页面 URL；中文目录段按 URL 编码输出。
	if !strings.Contains(start.RenderedHTML, `href="/wiki/%E6%96%87%E6%A1%A3/other"`) {
		t.Fatalf("first rendered link = %s, want /wiki/文档/other (URL-encoded)", start.RenderedHTML)
	}

	// 目录重命名（git mv 语义：文件移动 + 内容不变）：URL 首段随目录名变化。
	// slug 移除后仓库内相对链接以仓库真实路径为准——作者必须同步更新引用
	// （交叉引用维护成本；旧链接不回退解析）。未更新引用时同步按坏链接跳过
	// 该页（review M1），更新引用后页面重新渲染指向新路径。
	if err := os.Rename(filepath.Join(repo, "文档"), filepath.Join(repo, "handbook")); err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, repo, "指南/start.md", "# 开始\n\n[下一页](../handbook/other.md)")
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("sync after directory rename: %v", err)
	}
	start = wikiPages.GetByPath("指南/start")
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

// TestUnshallowMarkerSurvivesProjectionFailure 评论 P1 回归：unshallow 后
// 首次投影失败、修复后重试仍刷新未变化页面的贡献者缓存。
// 机制：ensureClone 在 unshallow 成功后写持久化标记（.git/wiki-trace-rebuild），
// rebuildGitTraces 完成后删除；投影失败/崩溃后标记保留，下次同步的 ensureClone
// 仍返回 unshallowed=true——.git/shallow 已被 git 删除，仅靠局部变量会永久
// 丢失升级机会，depth-1 快照残留。
func TestUnshallowMarkerSurvivesProjectionFailure(t *testing.T) {
	setupWikiTestDB(t)
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
	// 源仓库：两位作者两次提交（贡献者统计依赖完整历史）。
	src := t.TempDir()
	git := func(dir string, args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git(src, "init", "-q", "-b", "main")
	git(src, "config", "user.email", "alice@users.noreply.github.com")
	git(src, "config", "user.name", "Alice")
	writeRepoFile(t, src, "docs/page.md", "---\ntitle: 页面\n---\n\n# 一")
	git(src, "add", "-A")
	git(src, "commit", "-q", "-m", "c1")
	git(src, "config", "user.email", "12345+bob@users.noreply.github.com")
	git(src, "config", "user.name", "Bob")
	writeRepoFile(t, src, "docs/page.md", "---\ntitle: 页面\n---\n\n# 二")
	git(src, "add", "-A")
	git(src, "commit", "-q", "-m", "c2")

	// 存量浅克隆（模拟 v1 --depth=1 部署）。
	cloneDir := filepath.Join(t.TempDir(), "clone")
	if out, err := exec.Command("git", "clone", "-q", "--depth=1", "file://"+src, cloneDir).CombinedOutput(); err != nil {
		t.Fatalf("shallow clone: %v: %s", err, out)
	}
	if _, err := os.Stat(filepath.Join(cloneDir, ".git", "shallow")); err != nil {
		t.Fatal("test setup: expected shallow marker file")
	}

	cfg := GitConfig{Repo: "file://" + src, Branch: "main", CloneDir: cloneDir}
	// 预置页面 + depth-1 时代缓存（只有 1 位作者）。
	page := seedProjectedWikiPage(t, "docs", "docs/page", "页面", time.Now())
	if err := dbconnect.Connect().Table("wiki_pages").Where("id = ?", page.Id).
		Update("source_path", "docs/page").Error; err != nil {
		t.Fatalf("set source_path: %v", err)
	}
	if err := dbconnect.Connect().Table("wiki_pages").Where("id = ?", page.Id).
		Update("contributors_json", `[{"name":"test","count":1}]`).Error; err != nil {
		t.Fatalf("set old contributors: %v", err)
	}

	// 第一次同步：unshallow 成功 + 写标记；随后模拟投影失败（不消费标记）。
	head, unshallowed, err := ensureClone(cfg)
	if err != nil {
		t.Fatalf("ensureClone: %v", err)
	}
	if head == "" || !unshallowed {
		t.Fatalf("ensureClone after unshallow = (%q, %v), want unshallowed=true", head, unshallowed)
	}
	markerPath := filepath.Join(cloneDir, ".git", unshallowMarkerFile)
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatal("unshallow marker not written after unshallow")
	}

	// 第二次同步（修复后重试）：.git/shallow 已删除，但标记保留 → 仍须重建。
	if _, unshallowed, err = ensureClone(cfg); err != nil {
		t.Fatalf("ensureClone retry: %v", err)
	}
	if !unshallowed {
		t.Fatal("unshallowed lost after projection failure: marker present but ensureClone returned false")
	}

	// 重建完成：贡献者缓存刷新为 2 位作者，标记删除，后续同步不再重建。
	rebuildGitTraces(cfg)
	page = wikiPages.Get(page.Id)
	var got []gitContributor
	if err := json.Unmarshal([]byte(page.ContributorsJSON), &got); err != nil {
		t.Fatalf("unmarshal refreshed contributors %q: %v", page.ContributorsJSON, err)
	}
	if len(got) != 2 {
		t.Fatalf("contributors=%d, want 2 (both authors after rebuild): %+v", len(got), got)
	}
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Fatal("unshallow marker not removed after rebuildGitTraces")
	}
	if _, unshallowed, err = ensureClone(cfg); err != nil {
		t.Fatalf("ensureClone after rebuild: %v", err)
	}
	if unshallowed {
		t.Fatal("unshallowed should be false after marker consumed")
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

// seedTopicInteractions 给 topic 预置互动：回复 + 点赞/收藏/订阅（watcher）。
// 返回回复 ID，供收养后断言互动仍在原 topic。
func seedTopicInteractions(t *testing.T, topicID uint64, userIDs []uint64) uint64 {
	t.Helper()
	reply := posts.Entity{
		TopicId:          topicID,
		PostNo:           2,
		UserId:           userIDs[0],
		Content:          "回复内容",
		RenderedHTML:     "<p>回复内容</p>",
		ProcessStatus:    posts.ProcessStatusNormal,
		VisibilityStatus: posts.VisibilityActive,
		RetentionStatus:  posts.RetentionNormal,
	}
	if err := posts.Create(&reply); err != nil {
		t.Fatalf("create reply: %v", err)
	}
	now := time.Now()
	for _, uid := range userIDs {
		if !topicUserAction.SetLikedAt(uid, topicID, &now) {
			t.Fatalf("set liked for user %d", uid)
		}
		if !topicUserAction.SetBookmarkedAt(uid, topicID, &now) {
			t.Fatalf("set bookmarked for user %d", uid)
		}
		if !topicUserAction.SetWatchedAt(uid, topicID, &now) {
			t.Fatalf("set watched for user %d", uid)
		}
	}
	return reply.Id
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
			name:      "frontmatter extra fields ignored",
			file:      repoFile{Path: "同济新手教程/index.md", Content: []byte("---\ntitle: 教程\nslug: tongji-freshman-guide\n---\n\n# 首页")},
			wantTitle: "教程",
			wantOrder: 0,
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

// TestResolvePageByURLPath 路由语义（URL = 仓库目录名）：
// path 即仓库相对路径（去 .md），首段即目录名，直查命中；无 slug 重建。
func TestResolvePageByURLPath(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "同济新手教程/index.md", "---\ntitle: 新手教程\n---\n\n# 首页")
	writeRepoFile(t, repo, "同济新手教程/start.md", "---\ntitle: 开始\n---\n\n# 开始")

	cfg := GitConfig{CloneDir: repo}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	// 中文目录名 URL 直查命中（path 首段 = 仓库目录名）。
	if page := ResolvePageByURLPath("同济新手教程/start"); page.Id == 0 {
		t.Fatal("URL 同济新手教程/start should resolve directly (dir name = URL key)")
	}
	if page := ResolvePageByURLPath("同济新手教程/index"); page.Id == 0 {
		t.Fatal("URL 同济新手教程/index should resolve directly")
	}
	// 未命中：不存在的页面/命名空间。
	if page := ResolvePageByURLPath("同济新手教程/nope"); page.Id != 0 {
		t.Fatal("nonexistent page should not resolve")
	}
	if page := ResolvePageByURLPath("不存在的命名空间/start"); page.Id != 0 {
		t.Fatal("nonexistent namespace should not resolve")
	}
	// slug 已移除：旧 slug URL 不回退解析（404 语义）。
	if page := ResolvePageByURLPath("tongji-freshman-guide/start"); page.Id != 0 {
		t.Fatal("removed slug URL should not resolve (no fallback)")
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

// ---------- issue #288：重命名/移动收养 ----------

// TestApplyRepoToDBRenamePreservesInteractions Git 重命名（内容不变）：
// 同一逻辑页面的 path 迁移 + 原 topic 复用，回复/点赞/收藏/订阅全部保留，
// watcher 通知与 URL 解析跟随新路径。
func TestApplyRepoToDBRenamePreservesInteractions(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	content := "---\ntitle: 文档\n---\n\n# 标题\n\n正文"
	writeRepoFile(t, repo, "docs/a.md", content)

	cfg := GitConfig{CloneDir: repo}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	page := wikiPages.GetByPath("docs/a")
	topicID := page.TopicId
	replyID := seedTopicInteractions(t, topicID, []uint64{4242, 4243})

	// 重命名 docs/a.md → docs/b.md（内容不变）。
	if err := os.Rename(filepath.Join(repo, "docs/a.md"), filepath.Join(repo, "docs/b.md")); err != nil {
		t.Fatal(err)
	}
	res := &SyncResult{}
	if err := applyRepoToDB(cfg, res); err != nil {
		t.Fatalf("rename sync: %v", err)
	}
	if res.PagesAdded != 0 || res.PagesDeleted != 0 {
		t.Fatalf("rename added/deleted=%d/%d, want 0/0 (must not create+delete)", res.PagesAdded, res.PagesDeleted)
	}
	if res.PagesUpdated != 1 {
		t.Fatalf("PagesUpdated=%d, want 1 (adoption runs the update path)", res.PagesUpdated)
	}
	moved := wikiPages.GetByPath("docs/b")
	if moved.Id == 0 {
		t.Fatal("page docs/b missing after rename")
	}
	if moved.TopicId != topicID {
		t.Fatalf("topic id changed after rename: %d → %d, want reuse", topicID, moved.TopicId)
	}
	if moved.SourcePath != "docs/b" {
		t.Fatalf("source_path=%q, want docs/b", moved.SourcePath)
	}
	if old := wikiPages.GetByPathUnscoped("docs/a"); old.Id != 0 {
		t.Fatalf("old path docs/a still exists after rename (id=%d, deleted=%v)", old.Id, old.DeletedAt.Valid)
	}
	// 互动保留：回复 + 点赞/收藏/订阅。
	if reply := posts.UnscopedGet(replyID); reply.Id == 0 || reply.TopicId != topicID {
		t.Fatalf("reply lost after rename: id=%d topic=%d", reply.Id, reply.TopicId)
	}
	for _, uid := range []uint64{4242, 4243} {
		ua := topicUserAction.GetByTopicId(uid, topicID)
		if ua.Id == 0 || ua.LikedAt == nil || ua.BookmarkedAt == nil || ua.WatchedAt == nil {
			t.Fatalf("interaction lost after rename for user %d: %+v", uid, ua)
		}
	}
	// topic 生命周期保持活跃。
	topic := topics.Get(topicID)
	if topic.Id == 0 || topic.DeletedAt.Valid || topic.VisibilityStatus != topics.VisibilityActive {
		t.Fatalf("topic after rename: id=%d deleted=%v visibility=%q", topic.Id, topic.DeletedAt.Valid, topic.VisibilityStatus)
	}
	// watcher 通知跟随新路径。
	notif := eventNotification.GetLatestByTopicAndType(topicID, eventNotification.EventTypeWikiUpdated)
	if notif.Id == 0 {
		t.Fatal("watcher notification missing after rename")
	}
	if notif.Payload.Extra.ProfileURL != "/wiki/docs/b" {
		t.Fatalf("notification profileUrl=%q, want /wiki/docs/b", notif.Payload.Extra.ProfileURL)
	}
	// URL 解析：新路径命中，旧路径 404（文档化的旧链接行为）。
	if page := ResolvePageByURLPath("docs/b"); page.Id == 0 {
		t.Fatal("new URL docs/b should resolve")
	}
	if page := ResolvePageByURLPath("docs/a"); page.Id != 0 {
		t.Fatal("old URL docs/a should not resolve (page migrated)")
	}
}

// TestApplyRepoToDBRenameWithContentChange 重命名 + 内容修改（同一次同步）：
// 无稳定信号可判定为同页 → 定义为新建 + 软删旧页（互动保留在旧 topic 上），
// 不猜测合并（fail-safe）。
func TestApplyRepoToDBRenameWithContentChange(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "docs/a.md", "---\ntitle: v1\n---\n\n# 标题\n\n旧正文")

	cfg := GitConfig{CloneDir: repo}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	page := wikiPages.GetByPath("docs/a")
	topicID := page.TopicId
	replyID := seedTopicInteractions(t, topicID, []uint64{4242})

	if err := os.Rename(filepath.Join(repo, "docs/a.md"), filepath.Join(repo, "docs/b.md")); err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, repo, "docs/b.md", "---\ntitle: v2\n---\n\n# 标题\n\n新正文")
	res := &SyncResult{}
	if err := applyRepoToDB(cfg, res); err != nil {
		t.Fatalf("rename+edit sync: %v", err)
	}
	if res.PagesAdded != 1 || res.PagesDeleted != 1 {
		t.Fatalf("rename+edit added/deleted=%d/%d, want 1/1", res.PagesAdded, res.PagesDeleted)
	}
	// 旧页软删、互动保留在旧 topic；新页全新 topic。
	old := wikiPages.GetByPathUnscoped("docs/a")
	if old.Id == 0 || !old.DeletedAt.Valid {
		t.Fatalf("old page should be soft-deleted: id=%d deleted=%v", old.Id, old.DeletedAt.Valid)
	}
	if reply := posts.UnscopedGet(replyID); reply.Id == 0 || reply.TopicId != topicID {
		t.Fatal("old topic interactions must be preserved on the soft-deleted page's topic")
	}
	created := wikiPages.GetByPath("docs/b")
	if created.Id == 0 || created.TopicId == topicID {
		t.Fatalf("new page should have a fresh topic: id=%d topic=%d", created.Id, created.TopicId)
	}
}

// TestApplyRepoToDBMoveNestedDirectory 目录内移动（docs/guide/tips.md →
// docs/other/tips.md）：同 topic 复用 + parent_id 重算到新父 index 页。
// parent_id 语义（#303）：最近祖先 index 页；目录无 index.md 时归到 0。
func TestApplyRepoToDBMoveNestedDirectory(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	writeRepoFile(t, repo, "docs/guide/index.md", "---\ntitle: guide\n---\n\n# guide")
	writeRepoFile(t, repo, "docs/guide/tips.md", "---\ntitle: tips\n---\n\n# tips")
	writeRepoFile(t, repo, "docs/other/index.md", "---\ntitle: other\n---\n\n# other")

	cfg := GitConfig{CloneDir: repo}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	tips := wikiPages.GetByPath("docs/guide/tips")
	topicID := tips.TopicId
	parentGuide := wikiPages.GetByPath("docs/guide/index")
	if tips.ParentId != parentGuide.Id {
		t.Fatalf("parent_id=%d, want guide index page %d", tips.ParentId, parentGuide.Id)
	}

	// 移动到 docs/other/tips.md（先建目标目录——git 中 other/index.md 文件与
	// other/ 目录可共存，文件系统 rename 需要目标目录存在）。
	if err := os.MkdirAll(filepath.Join(repo, "docs/other"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(repo, "docs/guide/tips.md"), filepath.Join(repo, "docs/other/tips.md")); err != nil {
		t.Fatal(err)
	}
	res := &SyncResult{}
	if err := applyRepoToDB(cfg, res); err != nil {
		t.Fatalf("move sync: %v", err)
	}
	if res.PagesAdded != 0 || res.PagesDeleted != 0 {
		t.Fatalf("move added/deleted=%d/%d, want 0/0", res.PagesAdded, res.PagesDeleted)
	}
	moved := wikiPages.GetByPath("docs/other/tips")
	if moved.Id == 0 || moved.TopicId != topicID {
		t.Fatalf("moved page: id=%d topic=%d, want same topic %d", moved.Id, moved.TopicId, topicID)
	}
	parentOther := wikiPages.GetByPath("docs/other/index")
	if moved.ParentId != parentOther.Id {
		t.Fatalf("parent_id after move=%d, want other index page %d", moved.ParentId, parentOther.Id)
	}
}

// TestApplyRepoToDBMoveAcrossNamespaces 跨命名空间移动 docs/a.md → guide/a.md：
// 同 topic 复用；旧命名空间无页面后自动删除，新命名空间自动创建。
func TestApplyRepoToDBMoveAcrossNamespaces(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	content := "---\ntitle: 跨域\n---\n\n# 标题\n\n正文"
	writeRepoFile(t, repo, "docs/a.md", content)
	writeRepoFile(t, repo, "guide/stay.md", "---\ntitle: stay\n---\n\n# stay")

	cfg := GitConfig{CloneDir: repo}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	page := wikiPages.GetByPath("docs/a")
	topicID := page.TopicId

	if err := os.Rename(filepath.Join(repo, "docs/a.md"), filepath.Join(repo, "guide/a.md")); err != nil {
		t.Fatal(err)
	}
	res := &SyncResult{}
	if err := applyRepoToDB(cfg, res); err != nil {
		t.Fatalf("cross-ns move sync: %v", err)
	}
	if res.PagesAdded != 0 || res.PagesDeleted != 0 {
		t.Fatalf("cross-ns move added/deleted=%d/%d, want 0/0", res.PagesAdded, res.PagesDeleted)
	}
	moved := wikiPages.GetByPath("guide/a")
	if moved.Id == 0 || moved.TopicId != topicID {
		t.Fatalf("moved page: id=%d topic=%d, want same topic %d", moved.Id, moved.TopicId, topicID)
	}
	if !wikiNamespaces.Exists("guide") {
		t.Fatal("namespace guide should exist")
	}
	if wikiNamespaces.Exists("docs") {
		t.Fatal("namespace docs should be deleted (no pages left after move)")
	}
}

// TestApplyRepoToDBDeleteRecreateAdoptsSoftDeleted 删除后重建（新路径、同内容）：
// 软删行按 content_hash 收养 → 恢复 + 迁移，复用原 topic（两步 rename）。
func TestApplyRepoToDBDeleteRecreateAdoptsSoftDeleted(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	content := "---\ntitle: 重建\n---\n\n# 标题\n\n正文"
	writeRepoFile(t, repo, "docs/a.md", content)

	cfg := GitConfig{CloneDir: repo}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	page := wikiPages.GetByPath("docs/a")
	topicID := page.TopicId

	// 删除 → 软删。
	if err := os.Remove(filepath.Join(repo, "docs/a.md")); err != nil {
		t.Fatal(err)
	}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("delete sync: %v", err)
	}
	// 新路径重建（同内容）。
	writeRepoFile(t, repo, "docs/b.md", content)
	res := &SyncResult{}
	if err := applyRepoToDB(cfg, res); err != nil {
		t.Fatalf("recreate sync: %v", err)
	}
	if res.PagesAdded != 0 || res.PagesDeleted != 0 {
		t.Fatalf("recreate added/deleted=%d/%d, want 0/0", res.PagesAdded, res.PagesDeleted)
	}
	reborn := wikiPages.GetByPath("docs/b")
	if reborn.Id == 0 || reborn.DeletedAt.Valid {
		t.Fatalf("page not restored: id=%d deleted=%v", reborn.Id, reborn.DeletedAt.Valid)
	}
	if reborn.TopicId != topicID {
		t.Fatalf("topic changed after delete/recreate: %d → %d", topicID, reborn.TopicId)
	}
	topic := topics.UnscopedGet(topicID)
	if topic.Id == 0 || topic.DeletedAt.Valid || topic.VisibilityStatus != topics.VisibilityActive {
		t.Fatalf("topic not restored after delete/recreate: id=%d deleted=%v visibility=%q",
			topic.Id, topic.DeletedAt.Valid, topic.VisibilityStatus)
	}
	if old := wikiPages.GetByPathUnscoped("docs/a"); old.Id != 0 {
		t.Fatalf("old path docs/a still present after adoption (id=%d)", old.Id)
	}
}

// TestApplyRepoToDBCopySameContentFailsSafe 复制（同内容两页并存）：同 hash 多个
// wanted → 不收养（不猜测哪页是原页），新文件走新建、原页原地保留。
func TestApplyRepoToDBCopySameContentFailsSafe(t *testing.T) {
	setupWikiTestDB(t)
	repo := t.TempDir()
	content := "---\ntitle: 复制\n---\n\n# 标题\n\n正文"
	writeRepoFile(t, repo, "docs/a.md", content)

	cfg := GitConfig{CloneDir: repo}
	if err := applyRepoToDB(cfg, &SyncResult{}); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	page := wikiPages.GetByPath("docs/a")
	topicID := page.TopicId

	// 复制 a.md → b.md（两页并存）。
	writeRepoFile(t, repo, "docs/b.md", content)
	res := &SyncResult{}
	if err := applyRepoToDB(cfg, res); err != nil {
		t.Fatalf("copy sync: %v", err)
	}
	if res.PagesAdded != 1 {
		t.Fatalf("copy PagesAdded=%d, want 1 (new page created)", res.PagesAdded)
	}
	copied := wikiPages.GetByPath("docs/b")
	if copied.Id == 0 || copied.TopicId == topicID {
		t.Fatalf("copied page should have a fresh topic: id=%d topic=%d", copied.Id, copied.TopicId)
	}
	original := wikiPages.GetByPath("docs/a")
	if original.Id == 0 || original.TopicId != topicID {
		t.Fatalf("original page must keep its topic: id=%d topic=%d", original.Id, original.TopicId)
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
		t.Fatalf("list pages: %v", err)
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

package wikiservice

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/recovery"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiSyncRuns"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"go.yaml.in/yaml/v3"
	"gorm.io/gorm"
)

// wikiSystemUserID GitHub 同步创建的页面 topic/post 占位 user_id。
// 不创建真实用户行：0 仅作系统占位（作者展示为空，不影响互动/搜索）。
const wikiSystemUserID uint64 = 0

// ---------- 配置 ----------

// GitConfig GitHub wiki 仓库同步配置（config.toml [wiki.git] section）。
type GitConfig struct {
	Repo     string // 仓库地址（https://github.com/YourTongji/YourTJ-Wiki.git）
	Branch   string // 默认分支（main）
	CloneDir string // 本地 clone 目录（默认 ./storage/wiki-repo）
	Schedule string // 定时同步 cron spec（默认 "0 3 * * *"）
}

// LoadGitConfig 读取 [wiki.git] 配置。
func LoadGitConfig() GitConfig {
	return GitConfig{
		Repo:     preferences.GetString("wiki.git.repo", ""),
		Branch:   preferences.GetString("wiki.git.branch", "main"),
		CloneDir: preferences.GetString("wiki.git.clone_dir", "./storage/wiki-repo"),
		Schedule: preferences.GetString("wiki.git.schedule", "0 3 * * *"),
	}
}

// LoadWebhookSecret 读取 GitHub webhook 验签密钥（D1/安全：secret 存 securestore）。
// 读取优先级：
//  1. 管理端设置（securestore 加密落库，读取时解密）；
//  2. 兼容旧配置 config.toml [wiki.git].webhook_secret（明文；已配置时仍生效，
//     但管理端保存后以管理端为准）。
//
// 返回空串表示未配置（webhook 端点 403 fail-closed）。
func LoadWebhookSecret() string {
	cfg := hotdataserve.GetWikiSyncSettingsConfigCache()
	// 管理端显式清除过密钥 → 即使 config.toml 存在旧明文也保持禁用（fail-closed），
	// 避免管理员误以为已禁用而旧密钥仍生效（review L4）。
	if cfg.WebhookSecretCleared {
		return ""
	}
	if v := strings.TrimSpace(cfg.WebhookSecretEncrypted); v != "" {
		plain, err := securestore.DecryptPurpose(v, securestore.WikiWebhookSecretPurpose)
		if err != nil {
			slog.Warn("wiki webhook secret decrypt failed (signingKey rotated?), falling back to config",
				"error", err)
		} else if strings.TrimSpace(plain) != "" {
			return strings.TrimSpace(plain)
		}
	}
	return strings.TrimSpace(preferences.GetString("wiki.git.webhook_secret", ""))
}

// Enabled 返回 wiki git 同步是否启用（repo 配置非空）。
func (c GitConfig) Enabled() bool {
	return c.Repo != ""
}

// RepoPath 返回规范化 "owner/repo"（供 GitHub 外链拼接；失败返回空）。
func (c GitConfig) RepoPath() string {
	repo := c.Repo
	repo = strings.TrimSuffix(repo, ".git")
	repo = strings.TrimSuffix(repo, "/")
	for _, prefix := range []string{"https://github.com/", "http://github.com/", "git@github.com:"} {
		if strings.HasPrefix(repo, prefix) {
			repo = strings.TrimPrefix(repo, prefix)
			break
		}
	}
	if strings.Contains(repo, "/") && !strings.HasPrefix(repo, "/") {
		return repo
	}
	return ""
}

// githubPathEscape 对仓库相对路径逐段做 URL 路径转义：仓库目录名/文件名允许
// `#`/`%` 等字符（validSegment 未拒绝），但 GitHub 外链拼接时 `#` 会开启
// URL fragment、`%` 可能被当作转义前缀 → 404。逐段 PathEscape 保留 `/` 分隔。
func githubPathEscape(pagePath string) string {
	segs := strings.Split(pagePath, "/")
	for i, seg := range segs {
		segs[i] = url.PathEscape(seg)
	}
	return strings.Join(segs, "/")
}

// EditURL 返回某页面的 GitHub 编辑外链（{repo}/edit/{branch}/{path}.md）。
func (c GitConfig) EditURL(pagePath string) string {
	if repo := c.RepoPath(); repo != "" {
		return "https://github.com/" + repo + "/edit/" + c.Branch + "/" + githubPathEscape(pagePath) + ".md"
	}
	return ""
}

// HistoryURL 返回某页面的 GitHub 历史外链（{repo}/commits/{branch}/{path}.md）。
func (c GitConfig) HistoryURL(pagePath string) string {
	if repo := c.RepoPath(); repo != "" {
		return "https://github.com/" + repo + "/commits/" + c.Branch + "/" + githubPathEscape(pagePath) + ".md"
	}
	return ""
}

// ---------- 同步状态 ----------

// SyncStatus 同步面板状态。
type SyncStatus struct {
	Enabled    bool           `json:"enabled"`
	Repo       string         `json:"repo"`
	Branch     string         `json:"branch"`
	HeadSha    string         `json:"headSha"`
	LastRun    *SyncRunView   `json:"lastRun,omitempty"`
	RecentRuns []SyncRunView  `json:"recentRuns,omitempty"`
	Pages      SyncPageCounts `json:"pages"`
}

// SyncRunView 一次同步运行视图。
type SyncRunView struct {
	Id           uint64     `json:"id"`
	HeadSha      string     `json:"headSha"`
	Trigger      string     `json:"trigger"`
	Status       string     `json:"status"` // running | success | failed
	PagesAdded   int        `json:"pagesAdded"`
	PagesUpdated int        `json:"pagesUpdated"`
	PagesDeleted int        `json:"pagesDeleted"`
	Error        string     `json:"error,omitempty"`
	StartedAt    time.Time  `json:"startedAt"`
	FinishedAt   *time.Time `json:"finishedAt,omitempty"`
}

// SyncPageCounts 页面计数。
type SyncPageCounts struct {
	Total      int64 `json:"total"`
	Namespaces int64 `json:"namespaces"`
}

// statusString 映射同步状态 int8 → 字符串。
func statusString(s int8) string {
	switch s {
	case wikiSyncRuns.StatusSuccess:
		return "success"
	case wikiSyncRuns.StatusFailed:
		return "failed"
	default:
		return "running"
	}
}

// BuildSyncStatus 构建同步面板状态。
func BuildSyncStatus() SyncStatus {
	// 自愈（issue #290）：锁空闲 = 本进程无在途同步，running 行只可能来自
	// 崩溃/写失败遗留 → 回收，避免管理端 lastRun.status=running 永久禁用
	// 手动同步（页面刷新即恢复可用）。锁被占用时跳过（同步真在运行）。
	ReconcileStaleRuns()
	cfg := LoadGitConfig()
	status := SyncStatus{
		Enabled: cfg.Enabled(),
		Repo:    cfg.Repo,
		Branch:  cfg.Branch,
	}
	if latest := wikiSyncRuns.Latest(); latest.Id != 0 {
		status.HeadSha = latest.HeadSha
		last := ToRunView(latest)
		status.LastRun = &last
	}
	for _, r := range wikiSyncRuns.ListRecent(10) {
		status.RecentRuns = append(status.RecentRuns, ToRunView(r))
	}
	dbconnect.Connect().Table("wiki_pages").Count(&status.Pages.Total)
	dbconnect.Connect().Table("wiki_namespaces").Count(&status.Pages.Namespaces)
	return status
}

// ToRunView 把同步运行实体映射为视图（控制器层导出用）。
func ToRunView(r wikiSyncRuns.Entity) SyncRunView {
	return SyncRunView{
		Id:           r.Id,
		HeadSha:      r.HeadSha,
		Trigger:      r.Trigger,
		Status:       statusString(r.Status),
		PagesAdded:   r.PagesAdded,
		PagesUpdated: r.PagesUpdated,
		PagesDeleted: r.PagesDeleted,
		Error:        r.Error,
		StartedAt:    r.StartedAt,
		FinishedAt:   r.FinishedAt,
	}
}

// ---------- 并发防重入 ----------

// syncMu 是进程内同步互斥锁。整个 wiki 同步（SyncWithConfig/syncOnce）全程
// 持锁，ReconcileStaleRuns 在锁空闲时把库中遗留 running 行回收为 failed。
// 该回收逻辑依赖「同一数据库上至多一个进程运行同步」的部署假设：当前部署
// 每个环境只有一个应用容器（deploy/docker-compose.yaml，main/dev 各一实例），
// 进程内锁即唯一仲裁者。若未来同一实例水平扩容（多副本共享同一数据库），
// 进程内锁无法互斥跨进程同步，本回收逻辑必须换成 DB 级租约/锁，否则会误杀
// 其他进程正在执行的同步。
var syncMu sync.Mutex

// syncPending 运行期间到达的 webhook push 合并标记：锁释放后补跑一次，
// 避免投影停留在旧 head（并发丢弃 push 会 stale 到下次定时同步）。
var syncPending atomic.Bool

// ErrSyncAlreadyRunning 同步已在运行（webhook/定时/手动并发时）。
var ErrSyncAlreadyRunning = errors.New("wiki sync already running")

// TryAcquireSyncLock 尝试获取同步锁（防重入；webhook/定时/手动并发时只跑一个）。
func TryAcquireSyncLock() bool {
	return syncMu.TryLock()
}

// ReleaseSyncLock 释放同步锁。
func ReleaseSyncLock() {
	syncMu.Unlock()
}

// abandonedRunErrMsg 崩溃/重启遗留 running 行的回收标记文案（issue #290）。
const abandonedRunErrMsg = "abandoned: process restarted or run interrupted before completion"

// markAbandonedRuns 把库中遗留的 running 运行行标记为 failed（issue #290
// 崩溃恢复）。调用方必须持有同步锁（或已确认锁空闲）：同步全程持锁，锁
// 空闲时 running 行只可能来自崩溃/重启/MarkFinished 失败 → 可安全回收。
// 返回回收的行数。
func markAbandonedRuns() int64 {
	n, err := wikiSyncRuns.MarkAllRunningAbandoned(abandonedRunErrMsg)
	if err != nil {
		slog.Warn("wiki sync: reconcile stale running runs failed", "error", err)
		return 0
	}
	if n > 0 {
		slog.Warn("wiki sync: reconciled abandoned running runs", "count", n)
	}
	return n
}

// reconcileStaleRuns 回收遗留 running 运行行（issue #290 崩溃恢复）：锁空闲时
// running 行只可能来自进程崩溃/重启或 MarkFinished 失败 → 统一标记 failed，
// 让管理端 lastRun 呈现可恢复状态（手动同步按钮不再被永久禁用）。锁被占用
// （同步真在运行）时跳过，避免误杀在途同步。供状态读取与进程启动时调用。
func ReconcileStaleRuns() {
	if !TryAcquireSyncLock() {
		return // 锁被占用 = 同步真在运行，running 行是活的，不得回收
	}
	defer ReleaseSyncLock()
	markAbandonedRuns()
}

// ---------- git 操作 ----------

// ensureClone 确保本地 clone 存在且 remote 与配置一致（无则 clone，有则 fetch + reset --hard）。
// 返回当前 head SHA。本地工作区永不手动修改，reset --hard 比 pull 更确定。
// 配置变更（repo 换源）时丢弃缓存 clone 重建，避免投影到错误仓库。
func ensureClone(cfg GitConfig) (string, error) {
	if cfg.Repo == "" {
		return "", fmt.Errorf("wiki git repo not configured")
	}
	if cfg.CloneDir == "" {
		cfg.CloneDir = "./storage/wiki-repo"
	}
	if _, err := os.Stat(filepath.Join(cfg.CloneDir, ".git")); err == nil {
		if out, err := runGit(cfg.CloneDir, "remote", "get-url", "origin"); err == nil {
			if !sameRemote(strings.TrimSpace(out), cfg.Repo) {
				slog.Warn("wiki sync: cached clone remote mismatch, recloning",
					"cached", strings.TrimSpace(out), "configured", cfg.Repo)
				if err := os.RemoveAll(cfg.CloneDir); err != nil {
					return "", fmt.Errorf("remove stale clone: %w", err)
				}
			}
		}
	}
	if _, err := os.Stat(filepath.Join(cfg.CloneDir, ".git")); err == nil {
		if out, err := runGit(cfg.CloneDir, "fetch", "origin", cfg.Branch); err != nil {
			return "", fmt.Errorf("git fetch: %v: %s", err, out)
		}
		if out, err := runGit(cfg.CloneDir, "reset", "--hard", "origin/"+cfg.Branch); err != nil {
			return "", fmt.Errorf("git reset: %v: %s", err, out)
		}
	} else {
		if err := os.MkdirAll(cfg.CloneDir, 0o755); err != nil {
			return "", fmt.Errorf("mkdir clone dir: %w", err)
		}
		// 新 clone：--depth=1 只拉默认分支最新（同步只关心 head）。
		if out, err := runGit("", "clone", "--depth=1", "--branch", cfg.Branch, cfg.Repo, cfg.CloneDir); err != nil {
			return "", fmt.Errorf("git clone: %v: %s", err, out)
		}
	}
	out, err := runGit(cfg.CloneDir, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %v: %s", err, out)
	}
	return strings.TrimSpace(out), nil
}

// sameRemote 归一化比较仓库地址（忽略尾部 / 与 .git）。
func sameRemote(a, b string) bool {
	norm := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.TrimSuffix(s, "/")
		s = strings.TrimSuffix(s, ".git")
		return s
	}
	return norm(a) == norm(b)
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return buf.String(), err
	}
	return buf.String(), nil
}

// ---------- 仓库文件扫描 ----------

// repoFile 仓库中的一个 .md 文件。
type repoFile struct {
	Path    string // 相对仓库根，如 "docs/guide.md"
	Content []byte
	Hash    string // sha256(content)
}

// scanRepoFiles 递归扫描 clone 目录下的 .md 文件（排除 .git、隐藏目录）。
func scanRepoFiles(cloneDir string) ([]repoFile, error) {
	var files []repoFile
	err := filepath.Walk(cloneDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}
		rel, err := filepath.Rel(cloneDir, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(content)
		files = append(files, repoFile{
			Path:    filepath.ToSlash(rel),
			Content: content,
			Hash:    hex.EncodeToString(sum[:]),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// gitContributor 仓库 git log 贡献者快照条目（BuildContributors 数据源）。
type gitContributor struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// buildContributorsSnapshot 从 git log 统计某文件贡献者（同步时写入页面缓存）。
// 公开仓库无鉴权；失败返回空快照（不阻断同步）。
func buildContributorsSnapshot(cloneDir, relPath string) string {
	out, err := runGit(cloneDir, "log", "--pretty=format:%an", "--", relPath)
	if err != nil || strings.TrimSpace(out) == "" {
		return ""
	}
	counts := make(map[string]int)
	for _, line := range strings.Split(out, "\n") {
		name := strings.TrimSpace(line)
		if name != "" {
			counts[name]++
		}
	}
	type item struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	items := make([]item, 0, len(counts))
	for name, c := range counts {
		items = append(items, item{Name: name, Count: c})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Count > items[j].Count })
	data, err := json.Marshal(items)
	if err != nil {
		return ""
	}
	return string(data)
}

// ---------- frontmatter 解析 ----------

// pageFrontmatter md 文件 frontmatter（--- 块）。
type pageFrontmatter struct {
	Title       string `yaml:"title"`
	Order       int    `yaml:"order"`
	Description string `yaml:"description"`
	Slug        string `yaml:"slug"`
}

// parseMarkdownFile 解析 md：frontmatter + 正文（去掉 frontmatter 块）。
// title 缺失时用文件名（去 .md）兜底。
// 返回 (title, order, description, slug, body)；slug 仅在 frontmatter 显式
// 声明时非空（命名空间 URL 标识，与显示名分离）。
func parseMarkdownFile(f repoFile) (title string, order int, description string, slug string, body string) {
	content := string(f.Content)
	title = strings.TrimSuffix(filepath.Base(f.Path), ".md")
	if strings.HasPrefix(content, "---") {
		rest := content[3:]
		if idx := strings.Index(rest, "\n---"); idx >= 0 {
			fm := rest[:idx]
			var parsed pageFrontmatter
			if err := yaml.Unmarshal([]byte(fm), &parsed); err == nil {
				if parsed.Title != "" {
					title = parsed.Title
				}
				order = parsed.Order
				description = parsed.Description
				slug = strings.TrimSpace(parsed.Slug)
			}
			// 正文 = 第二个 --- 之后，去掉前导空行。
			body = strings.TrimLeft(rest[idx+4:], "\n")
		} else {
			body = content
		}
	} else {
		body = content
	}
	body = strings.TrimSpace(body)
	return
}

// ---------- 同步执行 ----------

// SyncResult 同步结果。
type SyncResult struct {
	HeadSha           string `json:"headSha"`
	PagesAdded        int    `json:"pagesAdded"`
	PagesUpdated      int    `json:"pagesUpdated"`
	PagesDeleted      int    `json:"pagesDeleted"`
	NamespacesDeleted int    `json:"namespacesDeleted,omitempty"`
}

// SyncAccepted 手动触发同步的立即响应（同步异步执行，进度由 status/runs 轮询）。
type SyncAccepted struct {
	Accepted bool `json:"accepted"`
}

// Sync 执行一次 GitHub → 论坛投影同步（webhook / 定时 / 手动共用）。
// 幂等：内容 hash 不变则跳过（重复同步零变更）。
func Sync(trigger string) (*SyncResult, error) {
	return SyncWithConfig(LoadGitConfig(), trigger)
}

// SyncWithConfig 使用显式配置执行一次同步（测试注入本地仓库用；
// 生产路径 Sync 内部读取 [wiki.git] 配置）。
func SyncWithConfig(cfg GitConfig, trigger string) (*SyncResult, error) {
	if !cfg.Enabled() {
		return nil, fmt.Errorf("wiki git sync not configured ([wiki.git].repo empty)")
	}
	if !TryAcquireSyncLock() {
		syncPending.Store(true)
		return nil, ErrSyncAlreadyRunning
	}
	// 崩溃恢复（issue #290）：此刻锁空闲刚被本调用持有 = 本进程无其他在途
	// 同步，库中 running 行必为崩溃/重启/MarkFinished 失败残留 → 回收，
	// 避免管理端 lastRun.status=running 永久禁用手动同步。锁已持有，
	// markAbandonedRuns 不再重入锁（不调用 ReconcileStaleRuns）。
	markAbandonedRuns()
	defer func() {
		ReleaseSyncLock()
		// 运行期间到达的 webhook push 合并补跑一次（只补一次，避免链式同步）。
		if syncPending.CompareAndSwap(true, false) {
			// review MEDIUM：补跑在调用方 goroutine 内执行，panic 同样会终止
			// 进程（webhook/manual 外层已有 Recover，此处再兜底，覆盖
			// startup/cron 之外的直接调用方）。
			defer recovery.Recover("wiki_sync_pending_rerun")
			// 补跑同样持锁（issue #290）：保持「running 行存在 ⇒ 同步锁被持有」
			// 不变量，状态读取时的遗留行回收才不会误杀在途同步。抢锁失败 =
			// 另一同步已在进行，其完成后投影即最新 head，补跑可安全丢弃。
			if TryAcquireSyncLock() {
				defer ReleaseSyncLock()
				if _, err := syncOnce(cfg, "webhook"); err != nil {
					slog.Warn("wiki sync: pending rerun failed", "error", err)
				}
			}
		}
	}()
	return syncOnce(cfg, trigger)
}

// syncOnce 执行一次同步主体（调用方持有同步锁；不重入锁）。
func syncOnce(cfg GitConfig, trigger string) (*SyncResult, error) {
	run := wikiSyncRuns.Entity{Trigger: trigger, Status: wikiSyncRuns.StatusRunning}
	if err := wikiSyncRuns.Create(&run); err != nil {
		return nil, fmt.Errorf("create sync run: %w", err)
	}

	head, err := ensureClone(cfg)
	if err != nil {
		if merr := wikiSyncRuns.MarkFinished(run.Id, wikiSyncRuns.StatusFailed, "", 0, 0, 0, err.Error()); merr != nil {
			slog.Error("wiki sync: mark run failed (clone) failed", "runId", run.Id, "error", merr)
		}
		return nil, err
	}

	result := &SyncResult{HeadSha: head}
	if err := applyRepoToDB(cfg, result); err != nil {
		if merr := wikiSyncRuns.MarkFinished(run.Id, wikiSyncRuns.StatusFailed, head, result.PagesAdded, result.PagesUpdated, result.PagesDeleted, err.Error()); merr != nil {
			slog.Error("wiki sync: mark run failed (projection) failed", "runId", run.Id, "error", merr)
		}
		return result, err
	}
	if merr := wikiSyncRuns.MarkFinished(run.Id, wikiSyncRuns.StatusSuccess, head, result.PagesAdded, result.PagesUpdated, result.PagesDeleted, ""); merr != nil {
		slog.Error("wiki sync: mark run success failed", "runId", run.Id, "error", merr)
	}
	if result.NamespacesDeleted > 0 {
		slog.Info("wiki sync: namespaces deleted", "count", result.NamespacesDeleted, "headSha", head)
	}
	return result, nil
}

// wantedPage 仓库 md 解析后的目标页面。
// path 首段 = URL key（slug，降级=显示名）；displayName = 仓库目录名（显示名，
// 可中文）；namespace = URL key（wiki_pages.namespace 列）；sourcePath = 仓库
// 真实相对路径（GitHub 外链拼接用，与 URL 解耦）。
type wantedPage struct {
	path        string
	sourcePath  string
	namespace   string
	displayName string
	title       string
	order       int
	body        string
}

// nsSlugPlan slug 定稿结果（D7：URL 用 slug，显示名与 URL key 分离）。
// urlKey：该命名空间页面 path 首段/namespace 列的目标值；
// setSlug：是否写入 slug 列（false = 非法/冲突时保持旧值）；
// slug：要写入 slug 列的值（nil = 置 NULL，即仓库未声明且无法推导）。
type nsSlugPlan struct {
	urlKey  string
	setSlug bool
	slug    *string
}

// migratePagePath 迁移单页 path/namespace（slug 变更时 path 首段同步）。
// 幂等：目标 path 与当前一致时零变更。
func migratePagePath(page *wikiPages.Entity, newPath, newNamespace string) error {
	if page.Path == newPath && page.Namespace == newNamespace {
		return nil
	}
	return dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		return tx.Table("wiki_pages").Where("id = ?", page.Id).Updates(map[string]any{
			"path":       newPath,
			"namespace":  newNamespace,
			"updated_at": time.Now(),
		}).Error
	})
}

// namespaceURLKey 返回命名空间的当前 URL key（slug 已分配时用 slug，
// 未分配时降级用显示名——与同步器 2.6 的降级策略一致，供 D5 删除计数等
// 按 URL key 查询 wiki_pages.namespace 列的场景使用）。
func namespaceURLKey(ns *wikiNamespaces.Entity) string {
	if ns == nil {
		return ""
	}
	if s := ns.SlugOrEmpty(); s != "" {
		return s
	}
	return ns.Name
}

// applyRepoToDB 把仓库当前文件树投影到 DB（核心幂等 diff）。
func applyRepoToDB(cfg GitConfig, result *SyncResult) error {
	files, err := scanRepoFiles(cfg.CloneDir)
	if err != nil {
		return fmt.Errorf("scan repo files: %w", err)
	}

	// 1. 收集现有页面（含软删，用于恢复）。
	existing := wikiPages.ListAll()
	byPath := make(map[string]*wikiPages.Entity, len(existing))
	for _, p := range existing {
		byPath[p.Path] = p
	}
	for _, p := range listAllUnscoped() {
		if _, ok := byPath[p.Path]; !ok {
			byPath[p.Path] = p
		}
	}

	// 2. 顶层目录 index.md → 命名空间元数据（D4/D7：description/sort_order/slug
	// 跟随 frontmatter；slug 为 URL 友好标识，与显示名 name 分离）。
	// 仅当 index.md 实际携带 description/order/slug 时才应用，
	// 避免无 frontmatter 的 index.md 清空命名空间元数据。
	type nsMeta struct {
		description string
		order       int
		slug        string
		carried     bool // index.md 实际携带元数据（review L1：缺失时不得用零值覆盖）
	}
	namespaceMeta := map[string]nsMeta{}
	for _, f := range files {
		parts := strings.Split(f.Path, "/")
		if len(parts) < 2 || parts[len(parts)-1] != "index.md" {
			continue
		}
		_, order, description, slug, _ := parseMarkdownFile(f)
		if description == "" && order == 0 && slug == "" {
			continue
		}
		namespaceMeta[parts[0]] = nsMeta{description: description, order: order, slug: slug, carried: true}
	}

	// 2.5 slug 定稿：推导每个仓库命名空间的目标 slug/URL key。
	// 冲突策略：slug 已被其他命名空间占用 → 报错并保留旧值（fail-fast，
	// run 标记 failed，避免 URL 与仓库不一致）；非法 slug 拒绝落库。
	// 中文目录无 slug 声明 → 降级 URL key=显示名 + 告警（不 fail）。
	// 仓库内部冲突：同一次同步中两个命名空间推导出相同 urlKey（如目录 guide
	// 默认 slug=guide，另一目录 frontmatter 声明 slug: guide）→ 后处理方
	// 报错并降级为显示名（唯一索引保护；两行都尚不存在时 DB 冲突检测不可用）。
	var errs []string
	repoDisplayNames := make(map[string]struct{}, len(files))
	for _, f := range files {
		rel := strings.TrimSuffix(f.Path, ".md")
		norm, ok := ValidatePath(rel)
		if !ok {
			continue
		}
		repoDisplayNames[NamespaceOf(norm)] = struct{}{}
	}
	plans := make(map[string]nsSlugPlan, len(repoDisplayNames))
	urlKeyOwners := make(map[string]string, len(repoDisplayNames)) // urlKey → 显示名
	for nsName := range repoDisplayNames {
		ns := wikiNamespaces.GetByName(nsName)
		meta := namespaceMeta[nsName]
		wantSlug := meta.slug
		if wantSlug == "" && isPureASCIISlug(nsName) {
			wantSlug = nsName
		}
		keepCurrent := func() nsSlugPlan {
			cur := ns.SlugOrEmpty()
			if cur == "" {
				cur = nsName
			}
			return nsSlugPlan{urlKey: cur}
		}
		if wantSlug != "" {
			if !ValidateSlug(wantSlug) {
				errs = append(errs, fmt.Sprintf("invalid slug for namespace %s: %q", nsName, wantSlug))
				plans[nsName] = keepCurrent()
				continue
			}
			if ns.Id != 0 && wantSlug != ns.SlugOrEmpty() && wikiNamespaces.SlugExists(wantSlug, ns.Id) {
				errs = append(errs, fmt.Sprintf("slug conflict for namespace %s: %q already used by another namespace (keep old value)", nsName, wantSlug))
				plans[nsName] = keepCurrent()
				continue
			}
			if owner, taken := urlKeyOwners[wantSlug]; taken {
				errs = append(errs, fmt.Sprintf("slug conflict for namespace %s: %q already used by namespace %s in repo (keep old value)", nsName, wantSlug, owner))
				plans[nsName] = keepCurrent()
				continue
			}
			urlKeyOwners[wantSlug] = nsName
			slug := wantSlug
			plans[nsName] = nsSlugPlan{urlKey: wantSlug, setSlug: true, slug: &slug}
		} else {
			// 降级：无 slug 声明 → URL key=显示名；仓库权威，slug 列置 NULL。
			if !isPureASCIISlug(nsName) {
				slog.Warn("wiki sync: namespace without slug falls back to display name as URL key",
					"namespace", nsName)
			}
			// 降级 urlKey=显示名也可能与已分配 slug 冲突（如中文目录名恰为另一
			// 目录的 slug）；同样报错并退回自身显示名（保持可读，宁可告警）。
			if owner, taken := urlKeyOwners[nsName]; taken {
				errs = append(errs, fmt.Sprintf("slug conflict for namespace %s: display name %q already used as slug by namespace %s (keep old value)", nsName, nsName, owner))
				plans[nsName] = keepCurrent()
				continue
			}
			urlKeyOwners[nsName] = nsName
			plans[nsName] = nsSlugPlan{urlKey: nsName, setSlug: true}
		}
	}

	// 2.6 仓库 md 文件 → 页面（D7 URL key 语义：URL 用 slug）。
	// path 首段 = plans[dir].urlKey（slug，降级=显示名）；source_path 恒存
	// 仓库真实路径（去 .md，保留大小写/Unicode），与 URL 解耦。
	wanted := make([]wantedPage, 0, len(files))
	wantedByPath := make(map[string]wantedPage, len(files))
	for _, f := range files {
		rel := strings.TrimSuffix(f.Path, ".md")
		norm, ok := ValidatePath(rel)
		if !ok {
			// 根级 README/CONTRIBUTING 等非 wiki 页面：合法仓库文件但非法页面路径，跳过。
			slog.Debug("wiki sync: skip non-page md", "path", rel)
			continue
		}
		dir := NamespaceOf(norm)
		urlKey := plans[dir].urlKey
		title, order, _, _, body := parseMarkdownFile(f)
		wp := wantedPage{
			path:        urlKey + norm[len(dir):], // 首段替换为 URL key
			sourcePath:  rel,
			namespace:   urlKey,
			displayName: dir,
			title:       title,
			order:       order,
			body:        body,
		}
		wanted = append(wanted, wp)
		wantedByPath[wp.path] = wp
	}

	// 2.7 既有页面 path 迁移：URL key 变化（slug 变更/首回填）时，把该命名空间
	// 全部页面（含软删）的 path 首段与 namespace 列迁移到新 URL key，并刷新
	// byPath，保证后续 upsert 幂等命中（不误走 create 新建）。
	// 无 slug 的中文目录：urlKey=显示名，与存量 path 首段一致 → 零迁移。
	for nsName, plan := range plans {
		ns := wikiNamespaces.GetByName(nsName)
		if ns.Id == 0 {
			continue // 首次同步：页面 upsert 阶段创建命名空间行
		}
		curKey := ns.SlugOrEmpty()
		if curKey == "" {
			curKey = nsName
		}
		if curKey == plan.urlKey {
			continue
		}
		for _, p := range listAllUnscoped() {
			if NamespaceOf(p.Path) != curKey {
				continue
			}
			newPath := plan.urlKey + strings.TrimPrefix(p.Path, curKey)
			if err := migratePagePath(p, newPath, plan.urlKey); err != nil {
				errs = append(errs, fmt.Sprintf("migrate path %s: %v", p.Path, err))
				continue
			}
			delete(byPath, p.Path)
			p.Path = newPath
			p.Namespace = plan.urlKey
			byPath[newPath] = p
			slog.Info("wiki sync: page path migrated for URL key change",
				"namespace", nsName, "oldKey", curKey, "newKey", plan.urlKey, "page", newPath)
		}
	}

	// 3. 逐页 upsert（每页独立事务）。单页失败聚合为整体失败：DB 与仓库
	//    部分偏离时 run 必须标记 failed，而不是向运维报告 success。
	for _, wp := range wanted {
		// 仓库顶层目录 = namespace 显示名；不存在则自动创建（名字=显示名，
		// 与 URL key 解耦）。放在 upsert 循环内，覆盖「目录曾被 D5 删除、
		// 页面重新出现走恢复路径」的场景（恢复路径不经过 createPageFromRepo，
		// 命名空间行必须在此重建）。
		if !wikiNamespaces.Exists(wp.displayName) {
			if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: wp.displayName}); err != nil {
				errs = append(errs, fmt.Sprintf("create namespace %s: %v", wp.displayName, err))
				continue
			}
		}
		existingPage, ok := byPath[wp.path]
		if !ok {
			// review L5：命名空间删除→重建且 URL key 变化时（如目录删除后
			// index.md 新增 slug），旧软删页面按 path 无法命中，但 source_path
			// （仓库相对路径）稳定 → 找回复用（迁移 path + 恢复），避免新建
			// 重复 page/topic 并遗留孤儿软删页。
			if orphan := wikiPages.GetBySourcePathUnscoped(wp.sourcePath); orphan.Id != 0 {
				if err := migratePagePath(&orphan, wp.path, wp.namespace); err != nil {
					errs = append(errs, fmt.Sprintf("adopt %s: %v", wp.path, err))
					continue
				}
				// 页面行恢复 + 清空 hash 强制走 updatePageFromRepo：收养页面对应的
				// topic 在命名空间删除时被同步软删，内容未变时幂等判断会跳过更新，
				// topic 生命周期将永远留在 USER_DELETED（updatePageFromRepo 内
				// 负责恢复 topic 生命周期）。
				if err := wikiPages.RestoreSoftDeleted(orphan.Id); err != nil {
					errs = append(errs, fmt.Sprintf("restore %s: %v", wp.path, err))
					continue
				}
				orphan.Path = wp.path
				orphan.Namespace = wp.namespace
				orphan.DeletedAt = gorm.DeletedAt{}
				orphan.ContentHash = ""
				byPath[wp.path] = &orphan
				existingPage = &orphan
				slog.Info("wiki sync: adopted orphaned page after namespace recreate",
					"sourcePath", wp.sourcePath, "path", wp.path)
			} else {
				if err := createPageFromRepo(cfg, wp); err != nil {
					errs = append(errs, fmt.Sprintf("create %s: %v", wp.path, err))
					continue
				}
				result.PagesAdded++
				continue
			}
		}
		curHash := sha256Hex(wp.body)
		restored := existingPage.DeletedAt.Valid
		// 仓库重新出现已删页面 → 先恢复页面行（复用原 topic/评论/点赞/订阅），
		// 恢复必须优先于幂等判断：否则内容未变的软删页面永远无法解除 deleted_at。
		if restored {
			if err := wikiPages.RestoreSoftDeleted(existingPage.Id); err != nil {
				errs = append(errs, fmt.Sprintf("restore %s: %v", wp.path, err))
				continue
			}
		}
		// 幂等：正文 hash、frontmatter title/order、GitHub 源路径均未变且无需恢复
		// → 零变更。只改 frontmatter（标题/排序）的提交也必须触发更新，否则投影
		// 的标题/导航顺序永久陈旧。
		if existingPage.ContentHash == curHash &&
			existingPage.Title == wp.title &&
			existingPage.SortOrder == wp.order &&
			existingPage.SourcePath == wp.sourcePath &&
			!restored {
			continue
		}
		if err := updatePageFromRepo(cfg, existingPage, wp, curHash); err != nil {
			errs = append(errs, fmt.Sprintf("update %s: %v", wp.path, err))
			continue
		}
		result.PagesUpdated++
	}

	// 4. 删除：仓库中不存在的已发布页面 → 软删（保留评论/互动）。
	//    重新扫描最新 path（2.7 可能迁移过 URL key），按 URL key 匹配 wanted。
	for _, p := range listAllUnscoped() {
		if _, ok := wantedByPath[p.Path]; ok {
			continue
		}
		if p.DeletedAt.Valid {
			continue
		}
		if err := softDeleteWikiPage(p); err != nil {
			errs = append(errs, fmt.Sprintf("delete %s: %v", p.Path, err))
			continue
		}
		result.PagesDeleted++
	}

	// 5. 命名空间元数据同步（D4/D7）：按 2.6 定稿的 plans 应用
	//    description/order（仅 index.md 携带时）+ slug 列。
	//    幂等：描述/排序/slug 都未变时跳过（CompareAndSwap 语义）。
	for nsName, plan := range plans {
		ns := wikiNamespaces.GetByName(nsName)
		if ns.Id == 0 {
			continue // 页面 upsert 阶段会创建；此处仅更新已存在的
		}
		meta := namespaceMeta[nsName]
		curSlug := ns.SlugOrEmpty()
		wantSlug := ""
		if plan.slug != nil {
			wantSlug = *plan.slug
		}
		// 仅 index.md 实际携带 description/order 时应用（review L1：缺失或
		// frontmatter 字段被删时，不得用零值清空已同步的命名空间元数据）。
		metaChanged := meta.carried && (ns.Description != meta.description || ns.SortOrder != meta.order)
		if !metaChanged && curSlug == wantSlug {
			continue
		}
		if meta.carried {
			ns.Description = meta.description
			ns.SortOrder = meta.order
		}
		if plan.setSlug {
			ns.Slug = plan.slug
		}
		if err := wikiNamespaces.Save(&ns); err != nil {
			errs = append(errs, fmt.Sprintf("update namespace meta %s: %v", nsName, err))
		}
	}

	// 6. 命名空间删除（D5）：仓库顶层目录（显示名）消失 → 自动删除命名空间。
	// 仓库中实际存在的顶层目录 = 全部 wanted 页面 displayName。
	// 同步驱动可绕过 hasPages 守卫：仓库是唯一真实源，目录删除即权威删除信号；
	// 页面已在第 4 步软删（topic 一并转入 USER_DELETED，评论/互动保留）。
	repoNamespaces := make(map[string]struct{}, len(wanted))
	for _, wp := range wanted {
		repoNamespaces[wp.displayName] = struct{}{}
	}
	for _, ns := range wikiNamespaces.List() {
		if _, ok := repoNamespaces[ns.Name]; ok {
			continue
		}
		// 仍有未软删页面（页面删除失败保护）→ 不删除命名空间。
		// namespace 列 = URL key，按当前 key 计数。
		var activePages int64
		if err := dbconnect.Connect().Table("wiki_pages").
			Where("namespace = ? AND deleted_at IS NULL", namespaceURLKey(ns)).
			Count(&activePages).Error; err != nil {
			errs = append(errs, fmt.Sprintf("count pages for namespace %s: %v", ns.Name, err))
			continue
		}
		if activePages > 0 {
			continue
		}
		if err := DeleteNamespace(ns.Name); err != nil {
			errs = append(errs, fmt.Sprintf("delete namespace %s: %v", ns.Name, err))
			continue
		}
		result.NamespacesDeleted++
		slog.Info("wiki sync: namespace removed (top-level dir gone from repo)",
			"namespace", ns.Name, "headSha", result.HeadSha)
	}

	if len(errs) > 0 {
		return fmt.Errorf("wiki sync: %d page(s) failed: %s", len(errs), strings.Join(errs, "; "))
	}
	return nil
}

// listAllUnscoped 返回全部页面（含软删，供恢复检测）。
func listAllUnscoped() []*wikiPages.Entity {
	var entities []*wikiPages.Entity
	dbconnect.Connect().Table("wiki_pages").Unscoped().Find(&entities)
	return entities
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// createPageFromRepo 从仓库文件新建 wiki 页面（topic + 首楼 + wiki_pages 投影）。
// path/namespace 列均存 URL key（slug，降级=显示名）；命名空间行由 upsert
// 循环按显示名创建（此处只做防御性校验）。
func createPageFromRepo(cfg GitConfig, wp wantedPage) error {
	ns := wp.namespace
	if ns == "" {
		return fmt.Errorf("empty namespace for path %s", wp.path)
	}
	if wp.displayName == "" {
		return fmt.Errorf("empty display name for path %s", wp.path)
	}
	// 嵌套路径：parent_id 关联父页面。
	parentID := uint64(0)
	if segments := strings.Split(wp.path, "/"); len(segments) > 2 {
		parentPath := strings.Join(segments[:len(segments)-1], "/")
		if parent := wikiPages.GetByPath(parentPath); parent.Id != 0 {
			parentID = parent.Id
		}
	}

	rendered := markdown2html.PostMarkdownToHTML(wp.body)
	toc := encodeTOCOrEmpty(markdown2html.ExtractHeadings(wp.body))

	topic := topics.Entity{
		UserId:           wikiSystemUserID,
		Title:            wp.title,
		Status:           1,
		ProcessStatus:    topics.ProcessStatusNormal,
		TopicType:        topics.TopicTypeWiki,
		Excerpt:          markdown2html.ExtractDescription(wp.body, 200),
		FirstImageURL:    markdown2html.ExtractFirstImageURL(wp.body),
		VisibilityStatus: topics.VisibilityActive,
		RetentionStatus:  topics.RetentionNormal,
	}
	var firstPost posts.Entity
	var page wikiPages.Entity
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		if err := topics.CreateTx(tx, &topic); err != nil {
			return err
		}
		firstPost = posts.Entity{
			TopicId:          topic.Id,
			PostNo:           1,
			UserId:           wikiSystemUserID,
			Content:          wp.body,
			RenderedHTML:     rendered,
			RenderedVersion:  markdown2html.GetPostVersion(),
			ProcessStatus:    posts.ProcessStatusNormal,
			VisibilityStatus: posts.VisibilityActive,
			RetentionStatus:  posts.RetentionNormal,
		}
		if err := posts.CreateTx(tx, &firstPost); err != nil {
			return err
		}
		topic.FirstPostId = firstPost.Id
		topic.LastPostId = firstPost.Id
		topic.PostSeq = 1
		if err := topics.SaveTx(tx, &topic); err != nil {
			return err
		}
		page = wikiPages.Entity{
			TopicId:             topic.Id,
			Namespace:           ns,
			Path:                wp.path,
			SourcePath:          wp.sourcePath,
			ParentId:            parentID,
			SortOrder:           wp.order,
			Title:               wp.title,
			Content:             wp.body,
			RenderedHTML:        rendered,
			Toc:                 toc,
			ContentHash:         sha256Hex(wp.body),
			PublishedRevisionNo: 1,
		}
		return wikiPages.CreateTx(tx, &page)
	})
	if err != nil {
		return err
	}
	// 提交后副作用：文件引用 + 搜索索引 + 发布事件 + git 溯源快照。
	fileusageservice.ReplaceTopic(topic.Id, wikiSystemUserID, wp.body)
	if _, err := searchservice.BuildSingleTopicSearchDocument(&topic, &firstPost); err != nil {
		slog.Warn("wiki sync: search index failed", "topicId", topic.Id, "error", err)
	}
	eventbus.Publish(context.Background(), &eventhandlers.TopicPublishedEvent{Topic: &topic, FirstPost: &firstPost})
	updateGitTrace(cfg, page.Id, wp.sourcePath)
	return nil
}

// updatePageFromRepo 更新已存在页面的投影（内容/标题/渲染/哈希 + topic/post 物化）。
func updatePageFromRepo(cfg GitConfig, page *wikiPages.Entity, wp wantedPage, curHash string) error {
	rendered := markdown2html.PostMarkdownToHTML(wp.body)
	toc := encodeTOCOrEmpty(markdown2html.ExtractHeadings(wp.body))

	topic := topics.UnscopedGet(page.TopicId)
	if topic.Id == 0 {
		return fmt.Errorf("topic %d not found for page %d", page.TopicId, page.Id)
	}
	firstPost := posts.UnscopedGet(topic.FirstPostId)
	if firstPost.Id == 0 {
		return fmt.Errorf("first post %d not found for topic %d", topic.FirstPostId, topic.Id)
	}

	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		// 页面恢复场景：topic 处于软删（USER_DELETED）时先恢复生命周期，
		// 与 softDeleteWikiPage 的删除语义对称（Unscoped 写，不受 GORM 软删 scope 影响）。
		if topic.DeletedAt.Valid {
			if err := tx.Table("topics").Unscoped().Where("id = ?", topic.Id).Updates(map[string]any{
				"deleted_at":        gorm.Expr("NULL"),
				"visibility_status": topics.VisibilityActive,
				"retention_status":  topics.RetentionNormal,
				"deleted_by":        0,
				"delete_reason":     "",
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Table("wiki_pages").Where("id = ?", page.Id).Updates(map[string]any{
			"title":         wp.title,
			"content":       wp.body,
			"rendered_html": rendered,
			"toc":           toc,
			"content_hash":  curHash,
			"sort_order":    wp.order,
			"source_path":   wp.sourcePath,
			"updated_at":    time.Now(),
		}).Error; err != nil {
			return err
		}
		// topic 物化（只更新 wiki 派生列，避免整行 Save 回写并发统计字段）。
		topic.Title = wp.title
		topic.Excerpt = markdown2html.ExtractDescription(wp.body, 200)
		topic.FirstImageURL = markdown2html.ExtractFirstImageURL(wp.body)
		if err := topics.UpdateWikiSyncedMetaTx(tx, &topic); err != nil {
			return err
		}
		// post 物化。
		firstPost.Content = wp.body
		firstPost.RenderedHTML = rendered
		firstPost.RenderedVersion = markdown2html.GetPostVersion()
		firstPost.ProcessStatus = posts.ProcessStatusNormal
		return posts.UpdateWikiSyncedContentTx(tx, &firstPost)
	})
	if err != nil {
		return err
	}
	// 提交后副作用：文件引用 + 搜索 + git 溯源快照 + watcher 通知。
	fileusageservice.ReplaceTopic(topic.Id, wikiSystemUserID, wp.body)
	if _, err := searchservice.BuildSingleTopicSearchDocument(&topic, &firstPost); err != nil {
		slog.Warn("wiki sync: search index failed", "topicId", topic.Id, "error", err)
	}
	updateGitTrace(cfg, page.Id, wp.sourcePath)
	notifyWatchersThrottled(page.TopicId, page.Path, wp.title, wikiSystemUserID)
	return nil
}

// softDeleteWikiPage 仓库中已移除的页面 → 论坛软删（保留互动，走删除生命周期）。
func softDeleteWikiPage(page *wikiPages.Entity) error {
	return dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("topics").Unscoped().Where("id = ?", page.TopicId).Updates(map[string]any{
			"deleted_at":        time.Now(),
			"visibility_status": topics.VisibilityUserDeleted,
			// 仓库移除 ≠ 用户删除：wiki 页面不进用户恢复/自动清除路径。
			// RECOVERABLE 会在 30 天后被 retention 清扫永久清除；恢复由同步器
			// 在页面重新出现时执行（updatePageFromRepo 恢复 ACTIVE/NORMAL）。
			"retention_status": topics.RetentionNormal,
		}).Error; err != nil {
			return err
		}
		return tx.Table("wiki_pages").Where("id = ?", page.Id).Delete(&wikiPages.Entity{}).Error
	})
}

// updateGitTrace 同步后更新页面的 git 溯源列（贡献者快照 + 最后提交 SHA/时间）。
// 失败仅记日志，不阻断同步（贡献者为空时页面仍可读）。
func updateGitTrace(cfg GitConfig, pageID uint64, relPath string) {
	contributors := buildContributorsSnapshot(cfg.CloneDir, relPath)
	commitSha := ""
	var commitAt time.Time
	if out, err := runGit(cfg.CloneDir, "log", "-1", "--format=%H%n%cI", "--", relPath); err == nil {
		lines := strings.SplitN(strings.TrimSpace(out), "\n", 2)
		if len(lines) > 0 {
			commitSha = strings.TrimSpace(lines[0])
		}
		if len(lines) > 1 {
			if t, err := time.Parse(time.RFC3339, strings.TrimSpace(lines[1])); err == nil {
				commitAt = t
			}
		}
	}
	updates := map[string]any{
		"contributors_json": contributors,
		"last_commit_sha":   commitSha,
	}
	if !commitAt.IsZero() {
		updates["last_commit_at"] = commitAt
	}
	if err := wikiPages.UpdateGitTrace(pageID, updates); err != nil {
		slog.Warn("wiki sync: update git trace failed", "pageId", pageID, "error", err)
	}
}

// encodeTOCOrEmpty 编码 TOC，失败返回空串（不阻断同步）。
func encodeTOCOrEmpty(items []markdown2html.HeadingItem) string {
	data, err := json.Marshal(items)
	if err != nil {
		return ""
	}
	return string(data)
}

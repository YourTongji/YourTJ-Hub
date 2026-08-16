package wikiservice

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
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
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiSyncRuns"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"go.yaml.in/yaml/v3"
	"gorm.io/gorm"
)

// wikiSystemUserID GitHub 同步创建的页面 topic/post 占位 user_id。
// 不创建真实用户行：0 仅作系统占位（作者展示为空，不影响互动/搜索）。
const wikiSystemUserID uint64 = 0

// ---------- 配置 ----------

// GitConfig GitHub wiki 仓库同步配置（config.toml [wiki.git] section）。
type GitConfig struct {
	Enable     bool   // 必须显式启用，默认 false
	AllowEmpty bool   // 是否允许空源删除全部现有页面，默认 false
	Repo       string // 仓库地址（https://github.com/YourTongji/YourTJ-Wiki.git）
	Branch     string // 默认分支（main）
	CloneDir   string // 本地 clone 目录（默认 ./storage/wiki-repo）
	Schedule   string // 定时同步 cron spec（默认 "30 3 * * *"）
}

// LoadGitConfig 读取 [wiki.git] 配置。
func LoadGitConfig() GitConfig {
	return GitConfig{
		Enable:     preferences.GetBool("wiki.git.enabled", false),
		AllowEmpty: preferences.GetBool("wiki.git.allow_empty", false),
		Repo:       strings.TrimSpace(preferences.GetString("wiki.git.repo", "")),
		Branch:     strings.TrimSpace(preferences.GetString("wiki.git.branch", "main")),
		CloneDir:   strings.TrimSpace(preferences.GetString("wiki.git.clone_dir", "./storage/wiki-repo")),
		Schedule:   strings.TrimSpace(preferences.GetString("wiki.git.schedule", "30 3 * * *")),
	}
}

// Enabled 返回 wiki git 同步是否启用（repo 配置非空）。
func (c GitConfig) Enabled() bool {
	return c.Enable && c.Repo != ""
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

// EditURL 返回某页面的 GitHub 编辑外链（{repo}/edit/{branch}/{path}.md）。
func (c GitConfig) EditURL(pagePath string) string {
	if !c.Enabled() {
		return ""
	}
	if repo := c.RepoPath(); repo != "" {
		return "https://github.com/" + repo + "/edit/" + c.Branch + "/" + pagePath + ".md"
	}
	return ""
}

// HistoryURL 返回某页面的 GitHub 历史外链（{repo}/commits/{branch}/{path}.md）。
func (c GitConfig) HistoryURL(pagePath string) string {
	if !c.Enabled() {
		return ""
	}
	if repo := c.RepoPath(); repo != "" {
		return "https://github.com/" + repo + "/commits/" + c.Branch + "/" + pagePath + ".md"
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

// ---------- git 操作 ----------

// ensureClone 确保本地 clone 存在且 remote 与配置一致（无则 clone，有则 fetch + reset --hard）。
// 返回当前 head SHA。本地工作区永不手动修改，reset --hard 比 pull 更确定。
// 配置变更（repo 换源）时丢弃缓存 clone 重建，避免投影到错误仓库。
const (
	gitCommandTimeout = 2 * time.Minute
	maxWikiPages      = 2000
	maxWikiPageBytes  = 2 << 20
	maxWikiTotalBytes = 64 << 20
)

func ensureClone(ctx context.Context, cfg GitConfig) (string, error) {
	if cfg.Repo == "" {
		return "", fmt.Errorf("wiki git repo not configured")
	}
	if cfg.CloneDir == "" {
		cfg.CloneDir = "./storage/wiki-repo"
	}
	cloneDir, err := safeCloneDir(cfg.CloneDir)
	if err != nil {
		return "", err
	}
	cfg.CloneDir = cloneDir
	if _, err := os.Stat(filepath.Join(cfg.CloneDir, ".git")); err == nil {
		out, err := runGit(ctx, cfg.CloneDir, "remote", "get-url", "origin")
		if err != nil {
			return "", fmt.Errorf("read cached clone remote: %w", err)
		}
		if !sameRemote(strings.TrimSpace(out), cfg.Repo) {
			return "", fmt.Errorf("wiki git clone_dir origin does not match configured repository")
		}
		top, err := runGit(ctx, cfg.CloneDir, "rev-parse", "--show-toplevel")
		if err != nil {
			return "", fmt.Errorf("resolve cached clone root: %w", err)
		}
		if !samePath(strings.TrimSpace(top), cfg.CloneDir) {
			return "", fmt.Errorf("wiki git clone_dir is not the repository root")
		}
		if out, err := runGit(ctx, cfg.CloneDir, "fetch", "origin", cfg.Branch); err != nil {
			return "", fmt.Errorf("git fetch: %v: %s", err, out)
		}
		if out, err := runGit(ctx, cfg.CloneDir, "reset", "--hard", "origin/"+cfg.Branch); err != nil {
			return "", fmt.Errorf("git reset: %v: %s", err, out)
		}
	} else {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat clone dir: %w", err)
		}
		if entries, readErr := os.ReadDir(cfg.CloneDir); readErr == nil && len(entries) != 0 {
			return "", fmt.Errorf("wiki git clone_dir is non-empty and is not a git repository")
		} else if readErr != nil && !os.IsNotExist(readErr) {
			return "", fmt.Errorf("read clone dir: %w", readErr)
		}
		if err := os.MkdirAll(cfg.CloneDir, 0o755); err != nil {
			return "", fmt.Errorf("mkdir clone dir: %w", err)
		}
		// 新 clone：--depth=1 只拉默认分支最新（同步只关心 head）。
		if out, err := runGit(ctx, "", "clone", "--depth=1", "--branch", cfg.Branch, cfg.Repo, cfg.CloneDir); err != nil {
			return "", fmt.Errorf("git clone: %v: %s", err, out)
		}
	}
	out, err := runGit(ctx, cfg.CloneDir, "rev-parse", "HEAD")
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

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, gitCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
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

func safeCloneDir(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("wiki git clone_dir must not be empty")
	}
	dir, err := filepath.Abs(filepath.Clean(value))
	if err != nil {
		return "", fmt.Errorf("resolve clone_dir: %w", err)
	}
	root := filepath.VolumeName(dir) + string(filepath.Separator)
	if samePath(dir, root) {
		return "", errors.New("wiki git clone_dir must not be a filesystem root")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	if samePath(dir, cwd) || pathContains(dir, cwd) {
		return "", errors.New("wiki git clone_dir must not be the working directory or its ancestor")
	}
	if home, err := os.UserHomeDir(); err == nil && samePath(dir, home) {
		return "", errors.New("wiki git clone_dir must not be the user home directory")
	}
	if info, err := os.Lstat(dir); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("wiki git clone_dir must not be a symbolic link")
	}
	return dir, nil
}

func samePath(a, b string) bool {
	return canonicalPath(a) == canonicalPath(b)
}

func pathContains(parent, child string) bool {
	rel, err := filepath.Rel(canonicalPath(parent), canonicalPath(child))
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func canonicalPath(value string) string {
	clean := filepath.Clean(value)
	if resolved, err := filepath.EvalSymlinks(clean); err == nil {
		return resolved
	}
	return clean
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
	var totalBytes int64
	err := filepath.WalkDir(cloneDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic links are not allowed in wiki repository: %s", path)
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > maxWikiPageBytes {
			return fmt.Errorf("wiki markdown exceeds %d bytes: %s", maxWikiPageBytes, path)
		}
		totalBytes += info.Size()
		if totalBytes > maxWikiTotalBytes {
			return fmt.Errorf("wiki markdown total exceeds %d bytes", maxWikiTotalBytes)
		}
		if len(files) >= maxWikiPages {
			return fmt.Errorf("wiki repository exceeds %d pages", maxWikiPages)
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
func buildContributorsSnapshot(ctx context.Context, cloneDir, relPath string) string {
	out, err := runGit(ctx, cloneDir, "log", "--pretty=format:%an", "--", relPath)
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
}

// parseMarkdownFile 解析 md：frontmatter + 正文（去掉 frontmatter 块）。
// title 缺失时用文件名（去 .md）兜底。
func parseMarkdownFile(f repoFile) (title string, order int, description string, body string) {
	content := string(f.Content)
	title = strings.TrimSuffix(filepath.Base(f.Path), filepath.Ext(f.Path))
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
	HeadSha      string `json:"headSha"`
	PagesAdded   int    `json:"pagesAdded"`
	PagesUpdated int    `json:"pagesUpdated"`
	PagesDeleted int    `json:"pagesDeleted"`
}

// SyncAccepted 手动触发同步的立即响应（同步异步执行，进度由 status/runs 轮询）。
type SyncAccepted struct {
	Accepted bool `json:"accepted"`
}

// Sync 执行一次 GitHub → 论坛投影同步（webhook / 定时 / 手动共用）。
// 幂等：内容 hash 不变则跳过（重复同步零变更）。
func Sync(trigger string) (*SyncResult, error) {
	return SyncContext(context.Background(), trigger)
}

func SyncContext(ctx context.Context, trigger string) (*SyncResult, error) {
	return SyncWithConfigContext(ctx, LoadGitConfig(), trigger)
}

// SyncWithConfig 使用显式配置执行一次同步（测试注入本地仓库用；
// 生产路径 Sync 内部读取 [wiki.git] 配置）。
func SyncWithConfig(cfg GitConfig, trigger string) (*SyncResult, error) {
	return SyncWithConfigContext(context.Background(), cfg, trigger)
}

func SyncWithConfigContext(ctx context.Context, cfg GitConfig, trigger string) (*SyncResult, error) {
	if !cfg.Enabled() {
		return nil, fmt.Errorf("wiki git sync disabled or not configured")
	}
	if !TryAcquireSyncLock() {
		syncPending.Store(true)
		return nil, ErrSyncAlreadyRunning
	}
	defer func() {
		ReleaseSyncLock()
		// 运行期间到达的 webhook push 合并补跑一次（只补一次，避免链式同步）。
		if syncPending.CompareAndSwap(true, false) {
			if _, err := syncOnce(ctx, cfg, "webhook"); err != nil {
				slog.Warn("wiki sync: pending rerun failed", "error", err)
			}
		}
	}()
	return syncOnce(ctx, cfg, trigger)
}

// syncOnce 执行一次同步主体（调用方持有同步锁；不重入锁）。
func syncOnce(ctx context.Context, cfg GitConfig, trigger string) (*SyncResult, error) {
	run := wikiSyncRuns.Entity{Trigger: trigger, Status: wikiSyncRuns.StatusRunning}
	if err := wikiSyncRuns.Create(&run); err != nil {
		return nil, fmt.Errorf("create sync run: %w", err)
	}

	head, err := ensureClone(ctx, cfg)
	if err != nil {
		_ = wikiSyncRuns.MarkFinished(run.Id, wikiSyncRuns.StatusFailed, "", 0, 0, 0, err.Error())
		return nil, err
	}

	result := &SyncResult{HeadSha: head}
	if err := applyRepoToDBContext(ctx, cfg, result); err != nil {
		_ = wikiSyncRuns.MarkFinished(run.Id, wikiSyncRuns.StatusFailed, head, result.PagesAdded, result.PagesUpdated, result.PagesDeleted, err.Error())
		return result, err
	}
	_ = wikiSyncRuns.MarkFinished(run.Id, wikiSyncRuns.StatusSuccess, head, result.PagesAdded, result.PagesUpdated, result.PagesDeleted, "")
	return result, nil
}

// wantedPage 仓库 md 解析后的目标页面。
type wantedPage struct {
	path       string
	sourcePath string // 仓库原始相对路径（去 .md，保留大小写）：GitHub 外链拼接用
	namespace  string
	title      string
	order      int
	body       string
}

// applyRepoToDB 把仓库当前文件树投影到 DB（核心幂等 diff）。
func applyRepoToDB(cfg GitConfig, result *SyncResult) error {
	return applyRepoToDBContext(context.Background(), cfg, result)
}

func applyRepoToDBContext(ctx context.Context, cfg GitConfig, result *SyncResult) error {
	files, err := scanRepoFiles(cfg.CloneDir)
	if err != nil {
		return fmt.Errorf("scan repo files: %w", err)
	}

	// 仓库 md 文件 -> 页面路径。扫描和解析全部发生在事务前；任何错误都不会
	// 触碰当前可读投影。
	wanted := make([]wantedPage, 0, len(files))
	wantedByPath := make(map[string]wantedPage, len(files))
	for _, f := range files {
		rel := strings.TrimSuffix(strings.ToLower(f.Path), ".md")
		norm, ok := ValidatePath(rel)
		if !ok {
			// 根级 README/CONTRIBUTING 等非 wiki 页面：合法仓库文件但非法页面路径，跳过。
			slog.Debug("wiki sync: skip non-page md", "path", rel)
			continue
		}
		title, order, _, body := parseMarkdownFile(f)
		wp := wantedPage{
			path:       norm,
			sourcePath: strings.TrimSuffix(f.Path, filepath.Ext(f.Path)),
			namespace:  NamespaceOf(norm),
			title:      title,
			order:      order,
			body:       body,
		}
		if _, duplicate := wantedByPath[norm]; duplicate {
			return fmt.Errorf("multiple source files normalize to wiki path %s", norm)
		}
		wanted = append(wanted, wp)
		wantedByPath[norm] = wp
	}
	if len(wanted) == 0 && !cfg.AllowEmpty && len(wikiPages.ListAll()) > 0 {
		return errors.New("refusing to delete all wiki pages from an empty source; set wiki.git.allow_empty=true to confirm")
	}
	sort.Slice(wanted, func(i, j int) bool { return wanted[i].path < wanted[j].path })

	stagedResult := SyncResult{HeadSha: result.HeadSha}
	var effects []projectionEffect
	err = dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		existing, err := wikiPages.ListAllUnscopedTx(tx)
		if err != nil {
			return fmt.Errorf("list existing wiki pages: %w", err)
		}
		assignments, matched := matchExistingPages(wantedByPath, existing)
		if err := ensureNamespacesTx(tx, wanted); err != nil {
			return err
		}
		if err := stageConflictingPathsTx(tx, wantedByPath, assignments, existing); err != nil {
			return err
		}

		projectedByPath := make(map[string]*wikiPages.Entity, len(wanted))
		for _, wp := range wanted {
			page := assignments[wp.path]
			if page == nil {
				effect, err := createPageFromRepoTx(tx, wp)
				if err != nil {
					return fmt.Errorf("create %s: %w", wp.path, err)
				}
				stagedResult.PagesAdded++
				effects = append(effects, effect)
				projectedByPath[wp.path] = effect.page
				continue
			}
			changed := page.DeletedAt.Valid || page.Path != wp.path || page.Namespace != wp.namespace ||
				page.ContentHash != sha256Hex(wp.body) || page.Title != wp.title ||
				page.SortOrder != wp.order || page.SourcePath != wp.sourcePath
			if !changed {
				projectedByPath[wp.path] = page
				continue
			}
			effect, err := updatePageFromRepoTx(tx, page, wp)
			if err != nil {
				return fmt.Errorf("update %s: %w", wp.path, err)
			}
			stagedResult.PagesUpdated++
			effects = append(effects, effect)
			projectedByPath[wp.path] = effect.page
		}

		for _, page := range existing {
			if matched[page.Id] || page.DeletedAt.Valid {
				continue
			}
			if err := softDeleteWikiPageTx(tx, page); err != nil {
				return fmt.Errorf("delete %s: %w", page.Path, err)
			}
			if err := enqueueWikiProjectionTaskTx(tx, result.HeadSha, page.Id, page.TopicId, false); err != nil {
				return fmt.Errorf("enqueue delete side effects for %s: %w", page.Path, err)
			}
			stagedResult.PagesDeleted++
		}

		// parent_id 取最终路径图计算，避免文件遍历顺序或 rename chain 产生旧父级。
		for _, wp := range wanted {
			page := projectedByPath[wp.path]
			parentID := uint64(0)
			if parent := projectedByPath[parentWikiPath(wp.path)]; parent != nil {
				parentID = parent.Id
			}
			if page.ParentId != parentID {
				if err := tx.Table("wiki_pages").Unscoped().Where("id = ?", page.Id).Update("parent_id", parentID).Error; err != nil {
					return fmt.Errorf("set parent for %s: %w", wp.path, err)
				}
				page.ParentId = parentID
			}
		}
		for _, effect := range effects {
			if err := enqueueWikiProjectionTaskTx(tx, result.HeadSha, effect.page.Id, effect.topic.Id, effect.updated); err != nil {
				return fmt.Errorf("enqueue projection side effects for %s: %w", effect.wp.path, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	*result = stagedResult
	// TopicPublishedEvent is part of the established creation contract: it updates
	// daily topic stats, notification hooks, and the LLMS cache. Publish only after
	// the projection transaction commits so consumers never observe rolled-back rows.
	for _, effect := range effects {
		if effect.created {
			eventbus.Publish(context.Background(), &eventhandlers.TopicPublishedEvent{Topic: effect.topic, FirstPost: effect.firstPost})
		}
	}
	refreshGitTrace(ctx, cfg, effects)
	return nil
}

func matchExistingPages(wanted map[string]wantedPage, existing []*wikiPages.Entity) (map[string]*wikiPages.Entity, map[uint64]bool) {
	assignments := make(map[string]*wikiPages.Entity, len(wanted))
	matched := make(map[uint64]bool, len(existing))
	activeByPath := make(map[string]*wikiPages.Entity, len(existing))
	deletedByPath := make(map[string]*wikiPages.Entity, len(existing))
	oldByHash := make(map[string][]*wikiPages.Entity)
	newByHash := make(map[string][]string)
	for _, page := range existing {
		if page.DeletedAt.Valid {
			deletedByPath[page.Path] = page
			continue
		}
		activeByPath[page.Path] = page
		if page.ContentHash != "" {
			oldByHash[page.ContentHash] = append(oldByHash[page.ContentHash], page)
		}
	}
	for pathValue, wp := range wanted {
		newByHash[sha256Hex(wp.body)] = append(newByHash[sha256Hex(wp.body)], pathValue)
	}
	// 唯一 hash 是 rename 身份，必须先于路径匹配；这样 swap/chain 不会把目标路径
	// 上原有页面错误消费掉。重复正文不猜测身份，回退到路径。
	for hash, newPaths := range newByHash {
		oldPages := oldByHash[hash]
		if len(newPaths) == 1 && len(oldPages) == 1 {
			assignments[newPaths[0]] = oldPages[0]
			matched[oldPages[0].Id] = true
		}
	}
	for pathValue := range wanted {
		if assignments[pathValue] != nil {
			continue
		}
		if page := activeByPath[pathValue]; page != nil && !matched[page.Id] {
			assignments[pathValue] = page
			matched[page.Id] = true
		}
	}
	for pathValue := range wanted {
		if assignments[pathValue] == nil {
			if page := deletedByPath[pathValue]; page != nil {
				assignments[pathValue] = page
				matched[page.Id] = true
			}
		}
	}
	return assignments, matched
}

func stageConflictingPathsTx(tx *gorm.DB, wanted map[string]wantedPage, assignments map[string]*wikiPages.Entity, existing []*wikiPages.Entity) error {
	byPath := make(map[string]*wikiPages.Entity, len(existing))
	for _, page := range existing {
		byPath[page.Path] = page
	}
	toStage := make(map[uint64]*wikiPages.Entity)
	for pathValue := range wanted {
		assigned, occupant := assignments[pathValue], byPath[pathValue]
		if assigned != nil && assigned.Path != pathValue {
			toStage[assigned.Id] = assigned
		}
		if occupant != nil && (assigned == nil || occupant.Id != assigned.Id) {
			toStage[occupant.Id] = occupant
		}
	}
	ids := make([]uint64, 0, len(toStage))
	for id := range toStage {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		page := toStage[id]
		oldPath := page.Path
		stagingPath := fmt.Sprintf("%s/sync-staging-%d", page.Namespace, page.Id)
		for suffix := 1; byPath[stagingPath] != nil; suffix++ {
			stagingPath = fmt.Sprintf("%s/sync-staging-%d-%d", page.Namespace, page.Id, suffix)
		}
		if err := wikiPages.MovePathTx(tx, page.Id, stagingPath); err != nil {
			return fmt.Errorf("stage occupied path %s: %w", oldPath, err)
		}
		delete(byPath, oldPath)
		byPath[stagingPath] = page
		page.Path = stagingPath
	}
	return nil
}

func ensureNamespacesTx(tx *gorm.DB, wanted []wantedPage) error {
	existing, err := wikiNamespaces.ListTx(tx)
	if err != nil {
		return fmt.Errorf("list namespaces: %w", err)
	}
	seen := make(map[string]bool, len(existing))
	for _, namespace := range existing {
		seen[namespace.Name] = true
	}
	for _, wp := range wanted {
		if seen[wp.namespace] {
			continue
		}
		if err := wikiNamespaces.CreateTx(tx, &wikiNamespaces.Entity{Name: wp.namespace}); err != nil {
			return fmt.Errorf("create namespace %s: %w", wp.namespace, err)
		}
		seen[wp.namespace] = true
	}
	return nil
}

func parentWikiPath(pathValue string) string {
	index := strings.LastIndex(pathValue, "/")
	if index <= 0 || !strings.Contains(pathValue[:index], "/") {
		return ""
	}
	return pathValue[:index]
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

type projectionEffect struct {
	page      *wikiPages.Entity
	topic     *topics.Entity
	firstPost *posts.Entity
	wp        wantedPage
	created   bool
	updated   bool
}

// createPageFromRepoTx 从仓库文件新建 wiki 页面（topic + 首楼 + wiki_pages 投影）。
func createPageFromRepoTx(tx *gorm.DB, wp wantedPage) (projectionEffect, error) {
	ns := wp.namespace
	if ns == "" {
		return projectionEffect{}, fmt.Errorf("empty namespace for path %s", wp.path)
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
	if err := topics.CreateTx(tx, &topic); err != nil {
		return projectionEffect{}, err
	}
	firstPost := posts.Entity{
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
		return projectionEffect{}, err
	}
	topic.FirstPostId = firstPost.Id
	topic.LastPostId = firstPost.Id
	topic.PostSeq = 1
	if err := topics.SaveTx(tx, &topic); err != nil {
		return projectionEffect{}, err
	}
	page := wikiPages.Entity{
		TopicId:             topic.Id,
		Namespace:           ns,
		Path:                wp.path,
		SourcePath:          wp.sourcePath,
		ParentId:            0,
		SortOrder:           wp.order,
		Title:               wp.title,
		Content:             wp.body,
		RenderedHTML:        rendered,
		Toc:                 toc,
		ContentHash:         sha256Hex(wp.body),
		PublishedRevisionNo: 1,
	}
	if err := wikiPages.CreateTx(tx, &page); err != nil {
		return projectionEffect{}, err
	}
	return projectionEffect{page: &page, topic: &topic, firstPost: &firstPost, wp: wp, created: true}, nil
}

// updatePageFromRepoTx 更新已存在页面的投影（内容/标题/渲染/哈希 + topic/post 物化）。
func updatePageFromRepoTx(tx *gorm.DB, page *wikiPages.Entity, wp wantedPage) (projectionEffect, error) {
	curHash := sha256Hex(wp.body)
	rendered := markdown2html.PostMarkdownToHTML(wp.body)
	toc := encodeTOCOrEmpty(markdown2html.ExtractHeadings(wp.body))

	var topic topics.Entity
	if err := tx.Table("topics").Unscoped().First(&topic, page.TopicId).Error; err != nil {
		return projectionEffect{}, fmt.Errorf("topic %d not found for page %d: %w", page.TopicId, page.Id, err)
	}
	var firstPost posts.Entity
	if err := tx.Table("posts").Unscoped().First(&firstPost, topic.FirstPostId).Error; err != nil {
		return projectionEffect{}, fmt.Errorf("first post %d not found for topic %d: %w", topic.FirstPostId, topic.Id, err)
	}

	if topic.DeletedAt.Valid {
		if err := tx.Table("topics").Unscoped().Where("id = ?", topic.Id).Updates(map[string]any{
			"deleted_at":        gorm.Expr("NULL"),
			"visibility_status": topics.VisibilityActive,
			"retention_status":  topics.RetentionNormal,
			"deleted_by":        0,
			"delete_reason":     "",
		}).Error; err != nil {
			return projectionEffect{}, err
		}
	}
	if err := tx.Table("wiki_pages").Unscoped().Where("id = ?", page.Id).Updates(map[string]any{
		"deleted_at":    gorm.Expr("NULL"),
		"namespace":     wp.namespace,
		"path":          wp.path,
		"title":         wp.title,
		"content":       wp.body,
		"rendered_html": rendered,
		"toc":           toc,
		"content_hash":  curHash,
		"sort_order":    wp.order,
		"source_path":   wp.sourcePath,
		"updated_at":    time.Now(),
	}).Error; err != nil {
		return projectionEffect{}, err
	}
	topic.Title = wp.title
	topic.Excerpt = markdown2html.ExtractDescription(wp.body, 200)
	topic.FirstImageURL = markdown2html.ExtractFirstImageURL(wp.body)
	if err := topics.UpdateWikiSyncedMetaTx(tx, &topic); err != nil {
		return projectionEffect{}, err
	}
	firstPost.Content = wp.body
	firstPost.RenderedHTML = rendered
	firstPost.RenderedVersion = markdown2html.GetPostVersion()
	firstPost.ProcessStatus = posts.ProcessStatusNormal
	if err := posts.UpdateWikiSyncedContentTx(tx, &firstPost); err != nil {
		return projectionEffect{}, err
	}
	page.Namespace = wp.namespace
	page.Path = wp.path
	page.SourcePath = wp.sourcePath
	page.Title = wp.title
	page.Content = wp.body
	page.RenderedHTML = rendered
	page.Toc = toc
	page.ContentHash = curHash
	page.SortOrder = wp.order
	page.DeletedAt = gorm.DeletedAt{}
	return projectionEffect{page: page, topic: &topic, firstPost: &firstPost, wp: wp, updated: true}, nil
}

// softDeleteWikiPageTx 仓库中已移除的页面 -> 论坛软删（保留互动）。
func softDeleteWikiPageTx(tx *gorm.DB, page *wikiPages.Entity) error {
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
}

func refreshGitTrace(ctx context.Context, cfg GitConfig, effects []projectionEffect) {
	for _, effect := range effects {
		updateGitTrace(ctx, cfg, effect.page.Id, effect.wp.sourcePath)
	}
}

// updateGitTrace 同步后更新页面的 git 溯源列（贡献者快照 + 最后提交 SHA/时间）。
// 失败仅记日志，不阻断同步（贡献者为空时页面仍可读）。
func updateGitTrace(ctx context.Context, cfg GitConfig, pageID uint64, relPath string) {
	contributors := buildContributorsSnapshot(ctx, cfg.CloneDir, relPath)
	commitSha := ""
	var commitAt time.Time
	if out, err := runGit(ctx, cfg.CloneDir, "log", "-1", "--format=%H%n%cI", "--", relPath); err == nil {
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

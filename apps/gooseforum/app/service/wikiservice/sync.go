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

// EditURL 返回某页面的 GitHub 编辑外链（{repo}/edit/{branch}/{path}.md）。
func (c GitConfig) EditURL(pagePath string) string {
	if repo := c.RepoPath(); repo != "" {
		return "https://github.com/" + repo + "/edit/" + c.Branch + "/" + pagePath + ".md"
	}
	return ""
}

// HistoryURL 返回某页面的 GitHub 历史外链（{repo}/commits/{branch}/{path}.md）。
func (c GitConfig) HistoryURL(pagePath string) string {
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
}

// parseMarkdownFile 解析 md：frontmatter + 正文（去掉 frontmatter 块）。
// title 缺失时用文件名（去 .md）兜底。
func parseMarkdownFile(f repoFile) (title string, order int, description string, body string) {
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
	defer func() {
		ReleaseSyncLock()
		// 运行期间到达的 webhook push 合并补跑一次（只补一次，避免链式同步）。
		if syncPending.CompareAndSwap(true, false) {
			if _, err := syncOnce(cfg, "webhook"); err != nil {
				slog.Warn("wiki sync: pending rerun failed", "error", err)
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
		_ = wikiSyncRuns.MarkFinished(run.Id, wikiSyncRuns.StatusFailed, "", 0, 0, 0, err.Error())
		return nil, err
	}

	result := &SyncResult{HeadSha: head}
	if err := applyRepoToDB(cfg, result); err != nil {
		_ = wikiSyncRuns.MarkFinished(run.Id, wikiSyncRuns.StatusFailed, head, result.PagesAdded, result.PagesUpdated, result.PagesDeleted, err.Error())
		return result, err
	}
	_ = wikiSyncRuns.MarkFinished(run.Id, wikiSyncRuns.StatusSuccess, head, result.PagesAdded, result.PagesUpdated, result.PagesDeleted, "")
	if result.NamespacesDeleted > 0 {
		slog.Info("wiki sync: namespaces deleted", "count", result.NamespacesDeleted, "headSha", head)
	}
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

	// 2. 仓库 md 文件 → 页面路径（去掉 .md 后缀，保留原始大小写/Unicode）。
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
		title, order, _, body := parseMarkdownFile(f)
		wp := wantedPage{
			path:       norm,
			sourcePath: strings.TrimSuffix(f.Path, ".md"),
			namespace:  NamespaceOf(norm),
			title:      title,
			order:      order,
			body:       body,
		}
		wanted = append(wanted, wp)
		wantedByPath[norm] = wp
	}

	// 2.5 顶层目录 index.md → 命名空间元数据（D4：description/sort_order 跟随
	// frontmatter）。仅当 index.md 实际携带 description/order 时才应用，
	// 避免无 frontmatter 的 index.md 清空命名空间描述。
	type nsMeta struct {
		description string
		order       int
	}
	namespaceMeta := map[string]nsMeta{}
	for _, f := range files {
		parts := strings.Split(f.Path, "/")
		if len(parts) < 2 || parts[len(parts)-1] != "index.md" {
			continue
		}
		_, order, description, _ := parseMarkdownFile(f)
		if description == "" && order == 0 {
			continue
		}
		namespaceMeta[parts[0]] = nsMeta{description: description, order: order}
	}

	// 3. 逐页 upsert（每页独立事务）。单页失败聚合为整体失败：DB 与仓库
	//    部分偏离时 run 必须标记 failed，而不是向运维报告 success。
	var errs []string
	for _, wp := range wanted {
		// 仓库顶层目录 = namespace；不存在则自动创建。放在 upsert 循环内，
		// 覆盖「目录曾被 D5 删除、页面重新出现走恢复路径」的场景（恢复路径
		// 不经过 createPageFromRepo，命名空间行必须在此重建）。
		if !wikiNamespaces.Exists(wp.namespace) {
			if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: wp.namespace}); err != nil {
				errs = append(errs, fmt.Sprintf("create namespace %s: %v", wp.namespace, err))
				continue
			}
		}
		existingPage, ok := byPath[wp.path]
		if !ok {
			if err := createPageFromRepo(cfg, wp); err != nil {
				errs = append(errs, fmt.Sprintf("create %s: %v", wp.path, err))
				continue
			}
			result.PagesAdded++
			continue
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
	for _, p := range existing {
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

	// 5. 命名空间元数据同步（D4）：顶层目录 index.md 的 frontmatter
	// description/order → wiki_namespaces.description/sort_order。
	// 幂等：描述与排序都未变时跳过（CompareAndSwap 语义）。
	for nsName, meta := range namespaceMeta {
		ns := wikiNamespaces.GetByName(nsName)
		if ns.Id == 0 {
			continue // 页面 upsert 阶段会创建；此处仅更新已存在的
		}
		if ns.Description == meta.description && ns.SortOrder == meta.order {
			continue
		}
		ns.Description = meta.description
		ns.SortOrder = meta.order
		if err := wikiNamespaces.Save(&ns); err != nil {
			errs = append(errs, fmt.Sprintf("update namespace meta %s: %v", nsName, err))
		}
	}

	// 6. 命名空间删除（D5）：仓库顶层目录消失 → 自动删除命名空间。
	// 仓库中实际存在的顶层目录 = 全部 wanted 页面 path 的首段。
	// 同步驱动可绕过 hasPages 守卫：仓库是唯一真实源，目录删除即权威删除信号；
	// 页面已在第 4 步软删（topic 一并转入 USER_DELETED，评论/互动保留）。
	repoNamespaces := make(map[string]struct{}, len(wanted))
	for _, wp := range wanted {
		repoNamespaces[wp.namespace] = struct{}{}
	}
	for _, ns := range wikiNamespaces.List() {
		if _, ok := repoNamespaces[ns.Name]; ok {
			continue
		}
		// 仍有未软删页面（页面删除失败保护）→ 不删除命名空间。
		var activePages int64
		if err := dbconnect.Connect().Table("wiki_pages").
			Where("namespace = ? AND deleted_at IS NULL", ns.Name).
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
func createPageFromRepo(cfg GitConfig, wp wantedPage) error {
	ns := wp.namespace
	if ns == "" {
		return fmt.Errorf("empty namespace for path %s", wp.path)
	}
	// 仓库顶层目录 = namespace；不存在则自动创建。
	if !wikiNamespaces.Exists(ns) {
		if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: ns}); err != nil {
			return fmt.Errorf("create namespace %s: %w", ns, err)
		}
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

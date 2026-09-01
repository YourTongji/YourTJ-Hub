package wikiservice

import (
	"bytes"
	"cmp"
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
	"slices"
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
// 自愈（issue #290）：锁空闲 = 本进程无在途同步，running 行只可能来自
// 崩溃/写失败遗留 → 回收，避免管理端 lastRun.status=running 永久禁用
// 手动同步（页面刷新即恢复可用）。锁被占用时跳过（同步真在运行）。
// 页面/命名空间计数失败必须上抛：面板必须区分 DB 故障与真实零页面（issue #287）。
func BuildSyncStatus() (SyncStatus, error) {
	ReconcileStaleRuns()
	cfg := LoadGitConfig()
	status := SyncStatus{
		Enabled: cfg.Enabled(),
		Repo:    cfg.Repo,
		Branch:  cfg.Branch,
	}
	if latest, err := wikiSyncRuns.Latest(); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return SyncStatus{}, fmt.Errorf("load latest wiki sync run: %w", err)
		}
	} else if latest.Id != 0 {
		status.HeadSha = latest.HeadSha
		last := ToRunView(latest)
		status.LastRun = &last
	}
	recentRuns, err := wikiSyncRuns.ListRecent(10)
	if err != nil {
		return SyncStatus{}, fmt.Errorf("list recent wiki sync runs: %w", err)
	}
	for _, r := range recentRuns {
		status.RecentRuns = append(status.RecentRuns, ToRunView(r))
	}
	if err := dbconnect.Connect().Table("wiki_pages").Count(&status.Pages.Total).Error; err != nil {
		return SyncStatus{}, fmt.Errorf("count wiki pages: %w", err)
	}
	if err := dbconnect.Connect().Table("wiki_namespaces").Count(&status.Pages.Namespaces).Error; err != nil {
		return SyncStatus{}, fmt.Errorf("count wiki namespaces: %w", err)
	}
	return status, nil
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

// syncHandoffMu 关闭「最终排空检查与解锁之间」的丢触发窗口（review P1）：
// 到达方在 handoffMu 临界区内执行「TryLock 失败 → 置 pending」，持锁方在
// handoffMu 临界区内执行「排空循环 + 释放锁」。二者互斥后，pending 的置位
// 要么发生在持锁方最终检查之前（随后被排空循环消费），要么发生在锁释放
// 之后（此时 TryLock 必然成功，触发直接进入正常同步）——不存在「已置位
// 却无人消费」的窗口。锁序：到达方 handoffMu → syncMu.TryLock（非阻塞）；
// 持锁方 syncMu（已持有）→ handoffMu。到达方临界区极短，补跑期间到达方
// 短暂阻塞在 handoffMu 上等待补跑完成，不丢触发。
var syncHandoffMu sync.Mutex

// syncPending 运行期间到达的 webhook push 合并标记：当前同步持锁排空补跑
// （SyncWithConfig defer 内循环 CAS 消费），避免投影停留在旧 head（并发
// 丢弃 push 会 stale 到下次定时同步）。
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

// unshallowMarkerFile 浅克隆→全量升级的「待重建 git trace」持久化标记
// （review P1）：unshallow 成功后写入 .git/ 内（fetch/reset/扫描均不触碰），
// rebuildGitTraces 完成后删除。投影失败/进程崩溃后标记保留，下次同步的
// ensureClone 仍会检测到并重建存量页面贡献者缓存——否则 .git/shallow 已被
// git 删除，仅靠局部变量会永久丢失升级机会，depth-1 快照残留。
const unshallowMarkerFile = "wiki-trace-rebuild"

// ensureClone 确保本地 clone 存在且 remote 与配置一致（无则 clone，有则 fetch + reset --hard）。
// 返回当前 head SHA，以及本次是否需要重建全部页面的 git 溯源缓存（unshallowed）：
// 触发条件 = 本次发生浅克隆→全量升级，或存在未消费的升级标记（上次升级后投影
// 失败/崩溃，重建未完成）。存量 depth-1 缓存只有最后一位作者，必须全量重建。
// 本地工作区永不手动修改，reset --hard 比 pull 更确定。
// 配置变更（repo 换源）时丢弃缓存 clone 重建，避免投影到错误仓库。
func ensureClone(cfg GitConfig) (head string, unshallowed bool, err error) {
	if cfg.Repo == "" {
		return "", false, fmt.Errorf("wiki git repo not configured")
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
					return "", false, fmt.Errorf("remove stale clone: %w", err)
				}
			}
		}
	}
	if _, err := os.Stat(filepath.Join(cfg.CloneDir, ".git")); err == nil {
		markerPath := filepath.Join(cfg.CloneDir, ".git", unshallowMarkerFile)
		// 未消费的升级标记：上次 unshallow 后投影失败/崩溃，全量重建未完成
		// → 本次仍须重建（review P1：.git/shallow 已删除，无法再靠它检测）。
		if _, err := os.Stat(markerPath); err == nil {
			unshallowed = true
		}
		// 存量浅克隆（v1 用 --depth=1 建立）→ 补全历史：贡献者统计依赖完整 git log。
		// 升级后必须重建全部页面的贡献者缓存（P1：幂等同步不会触碰未变化页面）。
		if _, err := os.Stat(filepath.Join(cfg.CloneDir, ".git", "shallow")); err == nil {
			if out, err := runGit(cfg.CloneDir, "fetch", "--unshallow", "origin", cfg.Branch); err != nil {
				return "", false, fmt.Errorf("git fetch --unshallow: %w: %s", err, out)
			}
			unshallowed = true
			// 持久化待重建标记：本次投影失败时，下次同步仍会重建
			// （review P1：unshallow 后 shallow 文件已删除，状态必须落盘）。
			if err := os.WriteFile(markerPath, []byte("1"), 0o644); err != nil {
				return "", false, fmt.Errorf("write git trace rebuild marker: %w", err)
			}
		}
		if out, err := runGit(cfg.CloneDir, "fetch", "origin", cfg.Branch); err != nil {
			return "", unshallowed, fmt.Errorf("git fetch: %w: %s", err, out)
		}
		if out, err := runGit(cfg.CloneDir, "reset", "--hard", "origin/"+cfg.Branch); err != nil {
			return "", unshallowed, fmt.Errorf("git reset: %w: %s", err, out)
		}
	} else {
		if err := os.MkdirAll(cfg.CloneDir, 0o755); err != nil {
			return "", false, fmt.Errorf("mkdir clone dir: %w", err)
		}
		// 全量 clone（不用 --depth=1，贡献者统计依赖完整 git log 历史）+
		// --single-branch（只取配置分支，避免拉取无关长驻分支的冗余对象）。
		if out, err := runGit("", "clone", "--single-branch", "--branch", cfg.Branch, cfg.Repo, cfg.CloneDir); err != nil {
			return "", false, fmt.Errorf("git clone: %w: %s", err, out)
		}
	}
	out, err := runGit(cfg.CloneDir, "rev-parse", "HEAD")
	if err != nil {
		return "", unshallowed, fmt.Errorf("git rev-parse: %w: %s", err, out)
	}
	return strings.TrimSpace(out), unshallowed, nil
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

// maxWikiPageBytes 单页源文件大小上限（review F2）：仓库恶意/意外的大文件
// （如符号链接指向 /dev/zero）会撑爆内存并卡死同步，超限直接报错。
const maxWikiPageBytes = 4 << 20 // 4 MiB

// scanRepoFiles 递归扫描 clone 目录下的 .md 文件（排除 .git、隐藏目录）。
// 只接受普通文件（lstat IsRegular）：Git 克隆会把仓库符号链接物化为 symlink，
// 若跟随读取可把服务器任意文件（如 ../../config.toml）投影为公开 wiki 页
// （review F2），或读 /dev/zero 类目标永久阻塞同步。符号链接一律拒绝。
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
		// review F2：拒绝符号链接（lstat 语义，不跟随）。
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("wiki page %s is a symlink (not allowed)", path)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("wiki page %s is not a regular file", path)
		}
		if info.Size() > maxWikiPageBytes {
			return fmt.Errorf("wiki page %s exceeds %d bytes", path, maxWikiPageBytes)
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
	slices.SortFunc(files, func(a, b repoFile) int {
		return cmp.Compare(a.Path, b.Path)
	})
	return files, nil
}

// gitContributor 仓库 git log 贡献者快照条目（BuildContributors 数据源）。
// 注意：email 仅作内存聚合键，不持久化（P2：原始邮箱属个人数据，contributors_json
// 会进入 DB/备份；BuildContributors 也不需要 email）。Username 由 GitHub noreply
// 邮箱解析，供前端拼头像与主页外链；自定义邮箱贡献者 username 为空（无链接降级）。
type gitContributor struct {
	Name     string `json:"name"`
	Username string `json:"username,omitempty"`
	Count    int    `json:"count"`
}

// gitLogPath 把页面 source_path（仓库相对路径，去 .md 后缀）规范化为 git pathspec
// 可精确匹配的仓库文件路径（补 .md）：git log pathspec 是精确匹配，
// `git log -- guide/start` 匹配不到仓库中的 guide/start.md。
func gitLogPath(sourcePath string) string {
	if strings.HasSuffix(sourcePath, ".md") {
		return sourcePath
	}
	return sourcePath + ".md"
}

// buildContributorsSnapshot 从 git log 统计某文件贡献者（同步时写入页面缓存）。
// 公开仓库无鉴权；失败返回空快照（不阻断同步）。
// 聚合键优先级：username（GitHub noreply 可解析，合并新旧邮箱格式同人）→
// email（自定义邮箱）→ name（匿名提交兜底）。email 仅内存聚合键，不序列化。
// 展示名取该聚合键最近一次提交的 name。
func buildContributorsSnapshot(cloneDir, relPath string) string {
	// --follow：贡献者统计跨 Git 重命名历史（issue #288 收养的页面在新路径下
	// 仍能归因旧路径提交；无 --follow 时 git log -- new.md 只返回重命名后的提交）。
	out, err := runGit(cloneDir, "log", "--follow", "--pretty=format:%an%x1f%ae", "--", gitLogPath(relPath))
	if err != nil || strings.TrimSpace(out) == "" {
		return ""
	}
	type agg struct {
		name     string
		email    string
		username string
		count    int
	}
	byKey := make(map[string]*agg)
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "\x1f", 2)
		name := strings.TrimSpace(parts[0])
		if name == "" {
			continue
		}
		email := ""
		if len(parts) > 1 {
			email = strings.TrimSpace(parts[1])
		}
		// 聚合键优先级：username（GitHub noreply 可解析——合并新旧邮箱格式同人）→
		// email（自定义邮箱）→ name（匿名提交兜底）。注意：同人混用 noreply 与
		// 自定义邮箱时无可靠身份源（无 mailmap）可合并，会按不同键拆分为两人。
		key := githubUsernameFromEmail(email)
		if key == "" {
			key = email
		}
		if key == "" {
			key = name
		}
		item := byKey[key]
		if item == nil {
			// git log 新→旧：首个出现的提交即该贡献者最近一次提交，
			// name 取首次（最新）值，后续旧提交不覆盖显示名。
			item = &agg{name: name, email: email, username: githubUsernameFromEmail(email)}
			byKey[key] = item
		}
		item.count++
	}
	items := make([]gitContributor, 0, len(byKey))
	for _, item := range byKey {
		items = append(items, gitContributor{
			Name:     item.name,
			Username: item.username,
			Count:    item.count,
		})
	}
	slices.SortFunc(items, func(a, b gitContributor) int {
		return cmp.Compare(b.Count, a.Count)
	})
	data, err := json.Marshal(items)
	if err != nil {
		return ""
	}
	return string(data)
}

// githubNoReplySuffix GitHub 隐私邮箱后缀（公开仓库贡献者默认开启邮箱隐私）。
const githubNoReplySuffix = "@users.noreply.github.com"

// githubUsernameFromEmail 从 GitHub noreply 隐私邮箱解析用户名：
//   - 新版格式 {id}+{username}@users.noreply.github.com（2021+）
//   - 旧版格式 {username}@users.noreply.github.com（2017-2021）
//   - 自定义邮箱 / 无法解析 → 返回空（前端降级为首字母占位、无外链）。
func githubUsernameFromEmail(email string) string {
	email = strings.TrimSpace(email)
	local, ok := strings.CutSuffix(email, githubNoReplySuffix)
	if !ok || local == "" {
		return ""
	}
	// 新版 {id}+{username}：取最后一个 + 之后的部分（id 全数字）。
	if i := strings.LastIndex(local, "+"); i >= 0 {
		local = local[i+1:]
	}
	if !validGithubUsername(local) {
		return ""
	}
	return local
}

// validGithubUsername GitHub 用户名宽松校验：字母数字连字符，不以连字符开头/结尾。
func validGithubUsername(name string) bool {
	if name == "" || len(name) > 39 || name[0] == '-' || name[len(name)-1] == '-' {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' {
			continue
		}
		return false
	}
	return true
}

// githubAvatarURL GitHub 官方动态头像直链（无需 API/token；size 控制分辨率）。
func githubAvatarURL(username string) string {
	return "https://github.com/" + username + ".png?size=56"
}

// githubProfileURL GitHub 用户主页外链。
func githubProfileURL(username string) string {
	return "https://github.com/" + username
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
// 返回 (title, order, description, body)。
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
	return SyncContext(context.Background(), trigger)
}

// SyncContext runs a sync with an explicit lifecycle context. The context is
// checked before acquiring the process/database lock so detached jobs can be
// abandoned without starting new work after their owner has gone away.
func SyncContext(ctx context.Context, trigger string) (*SyncResult, error) {
	return SyncWithConfigContext(ctx, LoadGitConfig(), trigger)
}

// SyncWithConfig 使用显式配置执行一次同步（测试注入本地仓库用；
// 生产路径 Sync 内部读取 [wiki.git] 配置）。
func SyncWithConfig(cfg GitConfig, trigger string) (*SyncResult, error) {
	return SyncWithConfigContext(context.Background(), cfg, trigger)
}

// SyncWithConfigContext is the cancellable form used by request-launched jobs
// and tests that inject a local repository configuration.
func SyncWithConfigContext(ctx context.Context, cfg GitConfig, trigger string) (*SyncResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !cfg.Enabled() {
		return nil, fmt.Errorf("wiki git sync not configured ([wiki.git].repo empty)")
	}
	if !TryAcquireSyncLock() {
		// review P1（状态交接竞态）：TryLock 失败不能直接置 pending 返回——
		// 持锁方可能正处于「最终检查与解锁之间」。在 handoffMu 临界区内
		// 重试：若持锁方已完成交接（锁已释放）则 TryLock 成功，直接进入
		// 正常同步（不丢触发）；若仍持锁则置 pending，持锁方的下一次临界区
		// 检查必然消费它。任何情况下都不存在「已置位却无人消费」的窗口。
		syncHandoffMu.Lock()
		if !TryAcquireSyncLock() {
			syncPending.Store(true)
			syncHandoffMu.Unlock()
			return nil, ErrSyncAlreadyRunning
		}
		syncHandoffMu.Unlock()
	}
	// 崩溃恢复（issue #290）：此刻锁空闲刚被本调用持有 = 本进程无其他在途
	// 同步，库中 running 行必为崩溃/重启/MarkFinished 失败残留 → 回收，
	// 避免管理端 lastRun.status=running 永久禁用手动同步。锁已持有，
	// markAbandonedRuns 不再重入锁（不调用 ReconcileStaleRuns）。
	markAbandonedRuns()
	// 补跑必须持有同步锁执行（syncOnce 假定调用方持锁）。修复前 defer 先
	// ReleaseSyncLock 再补跑：补跑期间到达的第三个触发会成功获取锁并并发
	// 进入 syncOnce，对同一 clone fetch/reset、对同一投影并发 upsert（#285）。
	// 循环排空 pending：补跑期间新到达的触发合并为下一次补跑——不丢失、不
	// 并发；每次补跑都由一次新的到达触发（非自触发），到达停止即终止。
	// 最终交接（review P1）：每次 pending 检查都在 handoffMu 临界区内，
	// 与到达方的「TryLock 失败 → 置 pending」互斥——pending 置位要么被
	// 本次/下一次检查消费，要么发生在锁释放之后（到达方随后 TryLock
	// 成功直接同步），不存在无人消费的窗口。
	defer func() {
		for {
			syncHandoffMu.Lock()
			if !syncPending.CompareAndSwap(true, false) {
				if h := syncReleaseHook.Load(); h != nil {
					(*h)() // 测试钩子：仍持 handoffMu+syncMu，即将释放
				}
				ReleaseSyncLock()
				syncHandoffMu.Unlock()
				return
			}
			syncHandoffMu.Unlock()
			runPendingSync(cfg)
		}
	}()
	return syncOnce(cfg, trigger)
}

// runPendingSync 持锁补跑一次合并的同步触发（调用方持有同步锁）。
// review MEDIUM：补跑在调用方 goroutine 内执行，panic 同样会终止进程
// （webhook/manual 外层已有 Recover，此处再兜底，覆盖 startup/cron 之外
// 的直接调用方）。
func runPendingSync(cfg GitConfig) {
	defer recovery.Recover("wiki_sync_pending_rerun")
	if _, err := syncOnce(cfg, "webhook"); err != nil {
		slog.Warn("wiki sync: pending rerun failed", "error", err)
	}
}

// syncOnceEnterHook 测试专用钩子：每次 syncOnce 进入主体时调用（生产为 nil）。
// 并发测试用它确定性阻塞 syncOnce，构造「主运行/补跑」重叠窗口验证锁串行化。
// 原子装载：测试安装/清理与 syncOnce 并发读取之间无数据竞争（ci-backend-race
// 在 -race 下运行该包）。
var syncOnceEnterHook atomic.Pointer[func()]

// syncReleaseHook 测试专用钩子：持锁方完成最终 pending 检查（确认无遗留）、
// 即将 ReleaseSyncLock 时调用（此时仍持有 syncMu + syncHandoffMu）。并发
// 测试用它确定性阻塞「最终交接」窗口，构造「检查已过、锁未放」的状态，
// 验证该窗口内到达的触发被接管为正常同步而非丢弃（review P1 回归测试）。
// 原子装载：与并发读取之间无数据竞争（ci-backend-race 在 -race 下运行）。
var syncReleaseHook atomic.Pointer[func()]

// syncOnce 执行一次同步主体（调用方持有同步锁；不重入锁）。
func syncOnce(cfg GitConfig, trigger string) (*SyncResult, error) {
	if h := syncOnceEnterHook.Load(); h != nil {
		(*h)()
	}
	run := wikiSyncRuns.Entity{Trigger: trigger, Status: wikiSyncRuns.StatusRunning}
	if err := wikiSyncRuns.Create(&run); err != nil {
		return nil, fmt.Errorf("create sync run: %w", err)
	}

	head, unshallowed, err := ensureClone(cfg)
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
	// 浅克隆→全量升级（review P1）：幂等同步跳过未变化页面，此处全量重建
	// 贡献者缓存，否则存量 depth-1 页面的 contributors_json 永远停留最后一位作者。
	if unshallowed {
		rebuildGitTraces(cfg)
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
// path = 仓库相对路径（去 .md，保留大小写/Unicode）；namespace = 仓库顶层
// 目录名（显示名，即 URL 首段）；sourcePath 与 path 一致（GitHub 外链用）。
type wantedPage struct {
	path         string
	sourcePath   string
	namespace    string
	displayName  string
	title        string
	order        int
	body         string
	renderedHTML string
	paraAnchors  []ParaAnchor
}

// migratePagePath 迁移单页 path/namespace（收养时按 wanted 目标值迁移）。
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

// adoptDisappearedPage 尝试把「仓库中消失的页面行」收养为 wanted 页面
// （Git 重命名/移动/命名空间重建）：迁移 path/namespace + 恢复软删 + 清空
// content_hash，返回被收养的页面（调用方随后走 updatePageFromRepo，恢复
// topic 生命周期并刷新内容/source_path；清空 hash 强制跳过幂等判断）。
// 返回 nil 表示没有唯一可收养候选（调用方走 createPageFromRepo 新建）。
//
// 匹配优先级（每行最多被收养一次，adoptedIDs 防止同次同步重复收养）：
//  1. source_path 精确匹配（review L5：命名空间删除→重建且 URL key 变化时，
//     旧软删页面 path 首段已是旧 key，但仓库相对路径稳定）；
//  2. content_hash 唯一匹配（issue #288：内容未变的文件换了路径；同 hash
//     多候选、或同 hash 多个 wanted（复制）均视为歧义，fail-safe 不猜测
//     合并，保持新建+软删的旧行为）。
func adoptDisappearedPage(wp wantedPage, curHash string, disappearedByHash map[string][]*wikiPages.Entity, wantedCountByHash map[string]int, adoptedIDs map[uint64]struct{}) (*wikiPages.Entity, error) {
	if orphan := wikiPages.GetBySourcePathUnscoped(wp.sourcePath); orphan.Id != 0 {
		if _, done := adoptedIDs[orphan.Id]; !done {
			oldPath := orphan.Path
			if err := adoptWikiPage(&orphan, wp); err != nil {
				return nil, err
			}
			adoptedIDs[orphan.Id] = struct{}{}
			slog.Info("wiki sync: adopted orphaned page after namespace recreate",
				"sourcePath", wp.sourcePath, "oldPath", oldPath, "path", wp.path)
			return &orphan, nil
		}
	}
	// 内容哈希收养仅在无歧义时生效：同 hash 的 wanted 多于 1 个（复制/
	// 合并场景）或消失候选多于 1 个（多页同内容）都不猜测。
	if wantedCountByHash[curHash] > 1 {
		return nil, nil
	}
	var candidates []*wikiPages.Entity
	for _, c := range disappearedByHash[curHash] {
		if _, done := adoptedIDs[c.Id]; done {
			continue
		}
		candidates = append(candidates, c)
	}
	if len(candidates) == 1 {
		adopted := candidates[0]
		oldPath := adopted.Path
		if err := adoptWikiPage(adopted, wp); err != nil {
			return nil, err
		}
		adoptedIDs[adopted.Id] = struct{}{}
		slog.Info("wiki sync: adopted renamed/moved page",
			"oldPath", oldPath, "path", wp.path, "sourcePath", wp.sourcePath)
		return adopted, nil
	}
	return nil, nil
}

// adoptWikiPage 执行收养落库：迁移 path/namespace、重算 parent_id、恢复软删、
// 清空 content_hash，并同步更新内存实体（供调用方复用）。
func adoptWikiPage(page *wikiPages.Entity, wp wantedPage) error {
	if err := migratePagePath(page, wp.path, wp.namespace); err != nil {
		return err
	}
	// 嵌套路径：parent_id 关联新路径的父页面（与 createPageFromRepo 同规则）。
	parentID := uint64(0)
	if segments := strings.Split(wp.path, "/"); len(segments) > 2 {
		parentPath := strings.Join(segments[:len(segments)-1], "/")
		if parent := wikiPages.GetByPath(parentPath); parent.Id != 0 {
			parentID = parent.Id
		}
	}
	if parentID != page.ParentId {
		if err := dbconnect.Connect().Table("wiki_pages").Where("id = ?", page.Id).
			Update("parent_id", parentID).Error; err != nil {
			return err
		}
		page.ParentId = parentID
	}
	// 页面行恢复 + 清空 hash 强制走 updatePageFromRepo：收养页面对应的
	// topic 在软删时被同步软删，内容未变时幂等判断会跳过更新，topic 生命
	// 周期将永远留在 USER_DELETED（updatePageFromRepo 内负责恢复 topic）。
	if err := wikiPages.RestoreSoftDeleted(page.Id); err != nil {
		return err
	}
	page.Path = wp.path
	page.Namespace = wp.namespace
	page.DeletedAt = gorm.DeletedAt{}
	page.ContentHash = ""
	return nil
}

// applyRepoToDB 把仓库当前文件树投影到 DB（核心幂等 diff）。
func applyRepoToDB(cfg GitConfig, result *SyncResult) error {
	files, err := scanRepoFiles(cfg.CloneDir)
	if err != nil {
		return fmt.Errorf("scan repo files: %w", err)
	}

	// 1. 收集现有页面（含软删，用于恢复）。
	existing, err := wikiPages.ListAll()
	if err != nil {
		return fmt.Errorf("list existing wiki pages: %w", err)
	}
	byPath := make(map[string]*wikiPages.Entity, len(existing))
	for _, p := range existing {
		byPath[p.Path] = p
	}
	unscoped, err := listAllUnscoped()
	if err != nil {
		return fmt.Errorf("list unscoped wiki pages: %w", err)
	}
	for _, p := range unscoped {
		if _, ok := byPath[p.Path]; !ok {
			byPath[p.Path] = p
		}
	}

	// 2. 顶层目录 index.md → 命名空间元数据（description/sort_order 跟随 frontmatter）。
	// 仅当 index.md 实际携带 description/order 时才应用，
	// 避免无 frontmatter 的 index.md 清空命名空间元数据。
	type nsMeta struct {
		description string
		order       int
		carried     bool // index.md 实际携带元数据（review L1：缺失时不得用零值覆盖）
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
		namespaceMeta[parts[0]] = nsMeta{description: description, order: order, carried: true}
	}

	var errs []string
	// review M1：引用校验/渲染失败的页面跳过（保留旧版本），聚合为 run 错误。
	skipWanted := make(map[string]bool, 0)

	// 2.6 仓库 md 文件 → 页面（URL 语义 = 仓库目录名）。
	// path = 仓库相对路径（去 .md，保留大小写/Unicode）；namespace = 顶层目录名；
	// source_path 与 path 一致（GitHub 外链即仓库真实路径）。
	wanted := make([]wantedPage, 0, len(files))
	wantedByPath := make(map[string]wantedPage, len(files))
	var invalidPaths []string
	for _, f := range files {
		rel := strings.TrimSuffix(f.Path, ".md")
		norm, err := ValidatePathError(rel)
		if err != nil {
			if strings.Contains(rel, "/") {
				// 正式内容路径非法 → fail-fast：run 标记 failed，绝不静默
				// 跳过并报告成功（issue #283）。
				invalidPaths = append(invalidPaths, fmt.Sprintf("%s: %v", rel, err))
				continue
			}
			// 仓库根级 README/CONTRIBUTING/LICENSE 等元文件：显式排除
			// （合法仓库文件但非页面路径），不阻断同步。
			slog.Debug("wiki sync: skip root-level non-page md", "path", rel)
			continue
		}
		dir := NamespaceOf(norm)
		if dir == "_assets" {
			return fmt.Errorf("wiki namespace %q is reserved for repository assets", dir)
		}
		title, order, _, body := parseMarkdownFile(f)
		wp := wantedPage{
			path:        norm,
			sourcePath:  rel,
			namespace:   dir,
			displayName: dir,
			title:       title,
			order:       order,
			body:        body,
		}
		wanted = append(wanted, wp)
		wantedByPath[wp.path] = wp
	}
	if len(invalidPaths) > 0 {
		// fail-fast 必须在任何 DB 写入（upsert/软删/命名空间删除）之前返回：
		// 非法路径未修正时同步整体失败，避免部分投影与误删。
		return fmt.Errorf("wiki sync: %d invalid page path(s): %s", len(invalidPaths), strings.Join(invalidPaths, "; "))
	}

	resolver := newWikiReferenceResolver(cfg, wanted, hotdataserve.GetWikiSyncSettingsConfigCache().AssetCDN)
	// review M1：单页引用校验/渲染失败 → 跳过该页并聚合告警（其余页面继续），
	// 避免一个坏链接冻结整个 wiki 的更新与删除。但安全类错误（仓库根逃逸/
	// 符号链接越界）必须整体失败：恶意链接不允许通过「坏页跳过」绕过校验。
	for i := range wanted {
		wp := &wanted[i]
		if err := resolver.Validate(*wp); err != nil {
			if errors.Is(err, errWikiRefEscapesRepo) {
				return err
			}
			errs = append(errs, fmt.Sprintf("skip %s: %v", wp.path, err))
			skipWanted[wp.path] = true
			continue
		}
		rendered, err := resolver.Render(*wp)
		if err != nil {
			if errors.Is(err, errWikiRefEscapesRepo) {
				return err
			}
			errs = append(errs, fmt.Sprintf("skip %s: %v", wp.path, err))
			skipWanted[wp.path] = true
			continue
		}
		wp.renderedHTML = rendered.HTML
		wp.paraAnchors = rendered.ParaAnchors
	}

	// 2.8 计算「仓库中消失的页面行」索引（issue #288：重命名/移动收养用）。
	// 消失 = 当前 path 不在 wanted 中的页面行（含软删——先删除
	// 后重建的两步 rename 需要软删候选）。按 content_hash 分组；同 hash 多
	// 候选时由收养逻辑判定歧义（fail-safe，不猜测合并）。
	disappearedByHash := make(map[string][]*wikiPages.Entity)
	unscopedPages, err := listAllUnscoped()
	if err != nil {
		return fmt.Errorf("list unscoped wiki pages for adoption: %w", err)
	}
	for _, p := range unscopedPages {
		if _, ok := wantedByPath[p.Path]; ok {
			continue
		}
		if p.ContentHash == "" {
			continue
		}
		disappearedByHash[p.ContentHash] = append(disappearedByHash[p.ContentHash], p)
	}
	adoptedIDs := make(map[uint64]struct{})

	// 2.85 每 content_hash 的 wanted 页面计数（内容哈希收养的歧义判定：
	// 同 hash 多个 wanted（复制场景）时禁止收养，fail-safe 不猜测合并）。
	wantedCountByHash := make(map[string]int, len(wanted))
	for _, wp := range wanted {
		wantedCountByHash[sha256Hex(wp.body)]++
	}

	// 3. 逐页 upsert（每页独立事务）。单页失败聚合为整体失败：DB 与仓库
	//    部分偏离时 run 必须标记 failed，而不是向运维报告 success。
	for _, wp := range wanted {
		if skipWanted[wp.path] {
			// review M1：引用校验/渲染失败的页面保留 DB 旧版本（不写空
			// renderedHTML 覆盖），仅聚合告警；其余页面正常 upsert。
			continue
		}
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
		curHash := sha256Hex(wp.body)
		if !ok {
			// 收养「仓库中消失的页面行」：
			//  - review L5：source_path 精确匹配（命名空间删除→重建且 URL key 变化）；
			//  - issue #288：content_hash 唯一匹配（Git 重命名/移动——同内容不同路径）。
			// 收养 = 迁移 path/namespace + 恢复软删 + 清空 hash 强制走
			// updatePageFromRepo（topic 生命周期恢复 + source_path 刷新），
			// 复用原 topic，回复/点赞/收藏/订阅全部保留。
			adopted, adoptErr := adoptDisappearedPage(wp, curHash, disappearedByHash, wantedCountByHash, adoptedIDs)
			if adoptErr != nil {
				errs = append(errs, fmt.Sprintf("adopt %s: %v", wp.path, adoptErr))
				continue
			}
			if adopted != nil {
				byPath[wp.path] = adopted
				existingPage = adopted
			} else {
				if err := createPageFromRepo(cfg, wp); err != nil {
					errs = append(errs, fmt.Sprintf("create %s: %v", wp.path, err))
					continue
				}
				result.PagesAdded++
				continue
			}
		}
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
			existingPage.RenderedHTML == wp.renderedHTML &&
			!restored {
			continue
		}
		// 纯渲染变化（CDN 切换等）不打扰 watcher：正文/标题/排序/源路径
		// 均未变时内容语义未变，仅资源 URL 形态变化，通知会误导订阅者
		// （review P2）。
		contentChanged := existingPage.ContentHash != curHash ||
			existingPage.Title != wp.title ||
			existingPage.SortOrder != wp.order ||
			existingPage.SourcePath != wp.sourcePath ||
			restored
		if err := updatePageFromRepo(cfg, existingPage, wp, curHash, contentChanged); err != nil {
			errs = append(errs, fmt.Sprintf("update %s: %v", wp.path, err))
			continue
		}
		result.PagesUpdated++
	}

	// 4. 删除：仓库中不存在的已发布页面 → 软删（保留评论/互动）。
	//    重新扫描最新 path（收养可能迁移过），按 path 匹配 wanted。
	unscopedPages, err = listAllUnscoped()
	if err != nil {
		return fmt.Errorf("list unscoped wiki pages for deletion pass: %w", err)
	}
	for _, p := range unscopedPages {
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

	// 4.5 Rebuild page-to-index links after every create/update/delete pass.
	// Navigation itself is path-derived so directories without index.md remain
	// valid; parent_id is a reconciled cache for pages with an ancestor index.
	if err := reconcileWikiPageParents(); err != nil {
		errs = append(errs, fmt.Sprintf("reconcile wiki page parents: %v", err))
	}

	// 5. 命名空间元数据同步：应用 description/order（仅 index.md 携带时）。
	//    幂等：描述/排序都未变时跳过（CompareAndSwap 语义）。
	for nsName, meta := range namespaceMeta {
		ns := wikiNamespaces.GetByName(nsName)
		if ns.Id == 0 {
			continue // 页面 upsert 阶段会创建；此处仅更新已存在的
		}
		// 仅 index.md 实际携带 description/order 时应用（review L1：缺失或
		// frontmatter 字段被删时，不得用零值清空已同步的命名空间元数据）。
		metaChanged := meta.carried && (ns.Description != meta.description || ns.SortOrder != meta.order)
		if !metaChanged {
			continue
		}
		ns.Description = meta.description
		ns.SortOrder = meta.order
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
	namespaces, err := wikiNamespaces.List()
	if err != nil {
		return fmt.Errorf("list wiki namespaces for deletion pass: %w", err)
	}
	for _, ns := range namespaces {
		if _, ok := repoNamespaces[ns.Name]; ok {
			continue
		}
		// 仍有未软删页面（页面删除失败保护）→ 不删除命名空间。
		// namespace 列 = 仓库目录名，按目录名计数。
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

// reconcileWikiPageParents rebuilds parent_id from the nearest ancestor index
// page. Root index pages and pages below directories without index.md attach to
// the namespace root (0). Derived parent paths are always shorter, so a
// successful reconciliation removes stale links and cannot create a cycle.
func reconcileWikiPageParents() error {
	pages, err := wikiPages.ListAll()
	if err != nil {
		return err
	}
	byPath := make(map[string]*wikiPages.Entity, len(pages))
	for _, page := range pages {
		byPath[page.Path] = page
	}
	return dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		for _, page := range pages {
			parentID := parentIndexID(page, byPath)
			if page.ParentId == parentID {
				continue
			}
			if err := tx.Table("wiki_pages").Where("id = ?", page.Id).Update("parent_id", parentID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func parentIndexID(page *wikiPages.Entity, byPath map[string]*wikiPages.Entity) uint64 {
	parts := strings.Split(page.Path, "/")
	for end := len(parts) - 1; end >= 1; end-- {
		candidate := strings.Join(parts[:end], "/") + "/index"
		if candidate == page.Path {
			continue
		}
		if parent, ok := byPath[candidate]; ok {
			return parent.Id
		}
	}
	return 0
}

// listAllUnscoped 返回全部页面（含软删，供恢复检测）。
// 查询失败必须上抛：删除 pass 依赖全量页面清单，吞错会漏删并谎报同步成功（issue #287）。
func listAllUnscoped() ([]*wikiPages.Entity, error) {
	var entities []*wikiPages.Entity
	err := dbconnect.Connect().Table("wiki_pages").Unscoped().Find(&entities).Error
	return entities, err
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// createPageFromRepo 从仓库文件新建 wiki 页面（topic + 首楼 + wiki_pages 投影）。
// path/namespace 列均存仓库相对路径/顶层目录名；命名空间行由 upsert
// 循环按显示名创建（此处只做防御性校验）。
func createPageFromRepo(cfg GitConfig, wp wantedPage) error {
	ns := wp.namespace
	if ns == "" {
		return fmt.Errorf("empty namespace for path %s", wp.path)
	}
	if wp.displayName == "" {
		return fmt.Errorf("empty display name for path %s", wp.path)
	}
	// parent_id 由同步结束后的 reconcileWikiPageParents 统一重算（PR #303），
	// 不在单页 upsert 内推导；renderedHTML 由引用解析器预渲染（review M1）。
	rendered := wp.renderedHTML
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
			TopicId:    topic.Id,
			Namespace:  ns,
			Path:       wp.path,
			SourcePath: wp.sourcePath,
			// parent_id is rebuilt after the complete repository projection. A
			// single upsert cannot safely derive it when ancestors appear later,
			// move, delete, or restore in the same sync.
			ParentId:            0,
			SortOrder:           wp.order,
			Title:               wp.title,
			Content:             wp.body,
			RenderedHTML:        rendered,
			Toc:                 toc,
			ParaAnchors:         encodeParaAnchorsOrEmpty(wp.paraAnchors),
			ContentHash:         sha256Hex(wp.body),
			PublishedRevisionNo: 1,
		}
		if err := wikiPages.CreateTx(tx, &page); err != nil {
			return err
		}
		return searchservice.EnqueueTopicSearchTask(tx, topic.Id)
	})
	if err != nil {
		return err
	}
	// 提交后副作用：文件引用 + 搜索索引（话题索引 + wiki 段落索引） + 发布事件 + git 溯源快照。
	fileusageservice.ReplaceTopic(topic.Id, wikiSystemUserID, wp.body)
	if err := searchservice.IndexWikiPageDocuments(page.Id); err != nil {
		slog.Warn("wiki sync: wiki page index failed", "pageId", page.Id, "error", err)
	}
	eventbus.Publish(context.Background(), &eventhandlers.TopicPublishedEvent{Topic: &topic, FirstPost: &firstPost})
	updateGitTrace(cfg, page.Id, wp.sourcePath)
	return nil
}

// updatePageFromRepo 更新已存在页面的投影（内容/标题/渲染/哈希 + topic/post 物化）。
// contentChanged 表示正文/元数据/恢复事件是否变化：仅 CDN 切换导致的纯渲染
// 变化（false）不发送 watcher 通知。
func updatePageFromRepo(cfg GitConfig, page *wikiPages.Entity, wp wantedPage, curHash string, contentChanged bool) error {
	rendered := wp.renderedHTML
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
			"para_anchors":  encodeParaAnchorsOrEmpty(wp.paraAnchors),
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
		if err := posts.UpdateWikiSyncedContentTx(tx, &firstPost); err != nil {
			return err
		}
		return searchservice.EnqueueTopicSearchTask(tx, topic.Id)
	})
	if err != nil {
		return err
	}
	// 提交后副作用：文件引用 + 搜索（话题索引 + wiki 段落索引） + git 溯源快照 + watcher 通知。
	fileusageservice.ReplaceTopic(topic.Id, wikiSystemUserID, wp.body)
	if err := searchservice.IndexWikiPageDocuments(page.Id); err != nil {
		slog.Warn("wiki sync: wiki page index failed", "pageId", page.Id, "error", err)
	}
	updateGitTrace(cfg, page.Id, wp.sourcePath)
	if contentChanged {
		notifyWatchersThrottled(page.TopicId, page.Path, wp.title, wikiSystemUserID)
	}
	return nil
}

// softDeleteWikiPage 仓库中已移除的页面 → 论坛软删（保留互动，走删除生命周期）。
func softDeleteWikiPage(page *wikiPages.Entity) error {
	if err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
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
	}); err != nil {
		return err
	}
	// 数据库软删提交后立即清理段落索引，避免已移除页面继续出现在搜索结果中。
	if err := searchservice.DeleteWikiPageDocuments(page.Id); err != nil {
		return fmt.Errorf("清理 wiki 页面 %d 搜索索引: %w", page.Id, err)
	}
	return nil
}

// updateGitTrace 同步后更新页面的 git 溯源列（贡献者快照 + 最后提交 SHA/时间）。
// 失败仅记日志，不阻断同步（贡献者为空时页面仍可读）。
func updateGitTrace(cfg GitConfig, pageID uint64, relPath string) {
	contributors := buildContributorsSnapshot(cfg.CloneDir, relPath)
	commitSha := ""
	var commitAt time.Time
	if out, err := runGit(cfg.CloneDir, "log", "--follow", "-1", "--format=%H%n%cI", "--", gitLogPath(relPath)); err == nil {
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

// rebuildGitTraces 浅克隆→全量升级后重建全部页面的 git 溯源缓存
// （review P1）：applyRepoToDB 幂等跳过未变化页面，若不在此全量重建，
// 存量 depth-1 页面的 contributors_json 永远停留在最后一位作者。
// 重建完成后删除持久化升级标记（unshallowMarkerFile）——标记由 ensureClone
// 在 unshallow 成功时写入，投影失败/崩溃时保留，本次成功消费后才清除：
// 保证「unshallow 后首次投影失败、修复后重试」仍会刷新未变化页面（review P1）。
// 单页失败仅记日志，不阻断整体重建；标记清除失败仅告警（下次同步仍会重建）。
func rebuildGitTraces(cfg GitConfig) {
	pages, err := wikiPages.ListAll()
	if err != nil {
		slog.Warn("wiki sync: rebuild git traces failed to list pages", "error", err)
		return
	}
	for _, p := range pages {
		if p.SourcePath == "" {
			continue
		}
		updateGitTrace(cfg, p.Id, p.SourcePath)
	}
	markerPath := filepath.Join(cfg.CloneDir, ".git", unshallowMarkerFile)
	if err := os.Remove(markerPath); err != nil && !os.IsNotExist(err) {
		slog.Warn("wiki sync: remove git trace rebuild marker failed", "error", err)
	}
	slog.Info("wiki sync: rebuilt git traces after unshallow upgrade", "pages", len(pages))
}

// encodeTOCOrEmpty 编码 TOC，失败返回空串（不阻断同步）。
func encodeTOCOrEmpty(items []markdown2html.HeadingItem) string {
	data, err := json.Marshal(items)
	if err != nil {
		return ""
	}
	return string(data)
}

// encodeParaAnchorsOrEmpty 编码段落锚点索引，失败/为空返回空串（不阻断同步）。
func encodeParaAnchorsOrEmpty(anchors []ParaAnchor) string {
	if len(anchors) == 0 {
		return ""
	}
	data, err := json.Marshal(anchors)
	if err != nil {
		return ""
	}
	return string(data)
}

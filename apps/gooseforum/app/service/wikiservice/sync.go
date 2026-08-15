package wikiservice

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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

// TryAcquireSyncLock 尝试获取同步锁（防重入；webhook/定时/手动并发时只跑一个）。
func TryAcquireSyncLock() bool {
	return syncMu.TryLock()
}

// ReleaseSyncLock 释放同步锁。
func ReleaseSyncLock() {
	syncMu.Unlock()
}

// ---------- git 操作 ----------

// ensureClone 确保本地 clone 存在（无则 clone，有则 fetch + reset --hard）。
// 返回当前 head SHA。本地工作区永不手动修改，reset --hard 比 pull 更确定。
func ensureClone(cfg GitConfig) (string, error) {
	if cfg.Repo == "" {
		return "", fmt.Errorf("wiki git repo not configured")
	}
	if cfg.CloneDir == "" {
		cfg.CloneDir = "./storage/wiki-repo"
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
	HeadSha      string `json:"headSha"`
	PagesAdded   int    `json:"pagesAdded"`
	PagesUpdated int    `json:"pagesUpdated"`
	PagesDeleted int    `json:"pagesDeleted"`
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
		return nil, fmt.Errorf("wiki sync already running")
	}
	defer ReleaseSyncLock()

	run := wikiSyncRuns.Entity{Trigger: trigger, Status: wikiSyncRuns.StatusRunning}
	if err := wikiSyncRuns.Create(&run); err != nil {
		return nil, fmt.Errorf("create sync run: %w", err)
	}

	head, err := ensureClone(cfg)
	if err != nil {
		_ = wikiSyncRuns.MarkFinished(run.Id, wikiSyncRuns.StatusFailed, 0, 0, 0, err.Error())
		return nil, err
	}

	result := &SyncResult{HeadSha: head}
	if err := applyRepoToDB(cfg, result); err != nil {
		_ = wikiSyncRuns.MarkFinished(run.Id, wikiSyncRuns.StatusFailed, result.PagesAdded, result.PagesUpdated, result.PagesDeleted, err.Error())
		return result, err
	}
	_ = wikiSyncRuns.MarkFinished(run.Id, wikiSyncRuns.StatusSuccess, result.PagesAdded, result.PagesUpdated, result.PagesDeleted, "")
	return result, nil
}

// wantedPage 仓库 md 解析后的目标页面。
type wantedPage struct {
	path      string
	namespace string
	title     string
	order     int
	body      string
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

	// 2. 仓库 md 文件 → 页面路径（去掉 .md 后缀，规范化小写 slug）。
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
			path:      norm,
			namespace: NamespaceOf(norm),
			title:     title,
			order:     order,
			body:      body,
		}
		wanted = append(wanted, wp)
		wantedByPath[norm] = wp
	}

	// 3. 逐页 upsert（每页独立事务：单页失败不阻断整批，记日志继续）。
	for _, wp := range wanted {
		existingPage, ok := byPath[wp.path]
		if !ok {
			if err := createPageFromRepo(wp); err != nil {
				slog.Error("wiki sync: create page failed", "path", wp.path, "error", err)
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
				slog.Error("wiki sync: restore page failed", "path", wp.path, "error", err)
				continue
			}
		}
		if existingPage.ContentHash == curHash && !restored {
			continue // 幂等：内容未变且无需恢复，零变更。
		}
		if err := updatePageFromRepo(existingPage, wp, curHash); err != nil {
			slog.Error("wiki sync: update page failed", "path", wp.path, "error", err)
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
			slog.Error("wiki sync: delete page failed", "path", p.Path, "error", err)
			continue
		}
		result.PagesDeleted++
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
func createPageFromRepo(wp wantedPage) error {
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
		page := wikiPages.Entity{
			TopicId:             topic.Id,
			Namespace:           ns,
			Path:                wp.path,
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
	// 提交后副作用：搜索索引 + 发布事件。
	if _, err := searchservice.BuildSingleTopicSearchDocument(&topic, &firstPost); err != nil {
		slog.Warn("wiki sync: search index failed", "topicId", topic.Id, "error", err)
	}
	eventbus.Publish(context.Background(), &eventhandlers.TopicPublishedEvent{Topic: &topic, FirstPost: &firstPost})
	return nil
}

// updatePageFromRepo 更新已存在页面的投影（内容/标题/渲染/哈希 + topic/post 物化）。
func updatePageFromRepo(page *wikiPages.Entity, wp wantedPage, curHash string) error {
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
	// 提交后副作用：文件引用 + 搜索。
	fileusageservice.ReplaceTopic(topic.Id, wikiSystemUserID, wp.body)
	if _, err := searchservice.BuildSingleTopicSearchDocument(&topic, &firstPost); err != nil {
		slog.Warn("wiki sync: search index failed", "topicId", topic.Id, "error", err)
	}
	return nil
}

// softDeleteWikiPage 仓库中已移除的页面 → 论坛软删（保留互动，走删除生命周期）。
func softDeleteWikiPage(page *wikiPages.Entity) error {
	return dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("topics").Unscoped().Where("id = ?", page.TopicId).Updates(map[string]any{
			"deleted_at":        time.Now(),
			"visibility_status": topics.VisibilityUserDeleted,
			"retention_status":  topics.RetentionRecoverable,
		}).Error; err != nil {
			return err
		}
		return tx.Table("wiki_pages").Where("id = ?", page.Id).Delete(&wikiPages.Entity{}).Error
	})
}

// encodeTOCOrEmpty 编码 TOC，失败返回空串（不阻断同步）。
func encodeTOCOrEmpty(items []markdown2html.HeadingItem) string {
	data, err := json.Marshal(items)
	if err != nil {
		return ""
	}
	return string(data)
}

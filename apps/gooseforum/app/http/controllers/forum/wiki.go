package forum

import (
	"log/slog"
	"math"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/topicviewservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/wikiservice"
	"github.com/gin-gonic/gin"
)

// WikiHomeProps wiki 首页 props。
type WikiHomeProps struct {
	Namespaces []wikiservice.NamespaceSummary `json:"namespaces"`
	Recent     []wikiservice.RecentPage       `json:"recent"`
	CanManage  bool                           `json:"canManage"`
}

// WikiHome 渲染 wiki 首页（PageComponent: wiki.home）。
func WikiHome(c *gin.Context) {
	loginUser := component.GetLoginUser(c)
	home, err := wikiservice.BuildHome()
	if err != nil {
		slog.Error("wiki home build failed", "error", err)
		renderInternalError(c)
		return
	}
	props := WikiHomeProps{
		Namespaces: home.Namespaces,
		Recent:     home.Recent,
		CanManage:  loginUser != nil && wikiservice.HasPageManagerPermission(loginUser.UserId),
	}
	payload := PagePayload{
		Component: PageComponentWikiHome,
		Props:     props,
		Meta: PageMeta{
			Title:       pageTitle("Wiki"),
			Description: "YourTJ Wiki 知识库",
			Canonical:   component.GetBaseUri(c) + "/wiki",
		},
		Layout:  buildLayout(c, "wiki"),
		URL:     buildPageURL(c),
		Version: payloadVersion,
	}
	payload.Layout.Sidebar.Mode = "wiki"
	tree, err := wikiTreePayload("")
	if err != nil {
		slog.Error("wiki home tree build failed", "error", err)
		renderInternalError(c)
		return
	}
	payload.Layout.Sidebar.WikiTree = tree
	renderPage(c, "wiki.gohtml", payload)
}

// WikiDetailProps wiki 详情页 props。
type WikiDetailProps struct {
	Page         wikiservice.PageDetail    `json:"page"`
	Contributors []wikiservice.Contributor `json:"contributors"`
	HotTopics    []TopicPayload            `json:"hotTopics"`
}

// WikiDetail 渲染 wiki 详情页（PageComponent: wiki.detail）。
func WikiDetail(c *gin.Context) {
	path := c.Param("path")
	if path == "" {
		path = c.Param("wildcard")
	}
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	if assetPath, ok := strings.CutPrefix(path, "_assets/"); ok {
		wikiAsset(c, assetPath)
		return
	}
	// D7 URL 语义（URL 用 slug）：path 首段 = URL key（slug，降级=显示名）。
	// ResolvePageByURLPath 先直查 slug 路径，未命中时回退按显示名解析重建
	// （兼容中文目录声明 slug 前的旧链接 / 直接访问中文显示名 URL）。
	// gin 已解码 URL 段，按原样查询。
	if path == "" || strings.Contains(path, "//") {
		renderNotFound(c)
		return
	}
	page := wikiservice.ResolvePageByURLPath(path)
	if page.Id == 0 {
		renderNotFound(c)
		return
	}
	topic := topics.Get(page.TopicId)
	if topic.Id == 0 {
		renderNotFound(c)
		return
	}
	if !canViewTopic(&topic, component.LoginUserId(c)) {
		renderNotFound(c)
		return
	}
	loginUser := component.GetLoginUser(c)
	loginUserID := uint64(0)
	if loginUser != nil {
		loginUserID = loginUser.UserId
	}

	detail, err := wikiservice.LoadPageDetail(&page, &topic)
	if err != nil {
		slog.Warn("wiki detail load failed", "path", path, "error", err)
		renderNotFound(c)
		return
	}
	props := WikiDetailProps{
		Page:         detail,
		Contributors: wikiservice.BuildContributors(page.Id),
		HotTopics:    buildTopicHotTopics(topic.Id),
	}
	// 互动状态（点赞/收藏/订阅）复用 topicUserAction。
	if loginUserID > 0 {
		action := topicUserAction.GetByTopicId(loginUserID, topic.Id)
		props.Page.Liked = action.LikedAt != nil
		props.Page.Bookmarked = action.BookmarkedAt != nil
		props.Page.Watched = action.WatchedAt != nil
	}
	// GitHub SSOT：编辑/历史走仓库外链（公开 fork + PR），站内无编辑。
	// D7：外链必须用仓库真实路径 source_path（path 首段已是 URL key=slug，
	// 与仓库目录名解耦，不能再用于 GitHub 文件定位）。
	// 存量页面 source_path 可能为空（v23 回填前/同步失败窗口），降级用 path
	// 保证外链可点（review MEDIUM：管理端已有同款回退，SSR 此处补齐）。
	cfg := wikiservice.LoadGitConfig()
	repoPath := page.SourcePath
	if repoPath == "" {
		repoPath = page.Path
	}
	props.Page.CanEdit = cfg.Enabled()
	props.Page.EditUrl = cfg.EditURL(repoPath)
	props.Page.HistoryUrl = cfg.HistoryURL(repoPath)

	payload := PagePayload{
		Component: PageComponentWikiDetail,
		Props:     props,
		Meta: PageMeta{
			Title:       pageTitle(detail.Title),
			Description: detail.Title,
			Canonical:   component.GetBaseUri(c) + "/wiki/" + page.Path,
		},
		Layout:  buildLayout(c, "wiki"),
		URL:     buildPageURL(c),
		Version: payloadVersion,
	}
	payload.Layout.Sidebar.Mode = "wiki"
	tree, err := wikiTreePayload(page.Path)
	if err != nil {
		slog.Error("wiki detail tree build failed", "path", path, "error", err)
		renderInternalError(c)
		return
	}
	payload.Layout.Sidebar.WikiTree = tree
	renderPage(c, "wiki.gohtml", payload)
	// 计一次浏览（review P2：TopicDetail 已记录，wiki 详情此前漏记）。
	if shouldCountTopicView(&topic) {
		topicviewservice.RecordView(topic.Id)
	}
}

// wikiAssetType 资源扩展名 → Content-Type 白名单（review H1）：
// 只允许惰性内容类型；HTML/SVG/XML/JS/WASM/无扩展名等可执行文档一律拒绝
// 内联渲染（否则公开 PR 合并一个 .html/.svg 即成为论坛同源脚本执行原语）。
// 返回 (contentType, ok)；ok=false 时调用方以 octet-stream + attachment 兜底。
func wikiAssetType(name string) (string, bool) {
	ext := strings.ToLower(path.Ext(name))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg", true
	case ".png":
		return "image/png", true
	case ".gif":
		return "image/gif", true
	case ".webp":
		return "image/webp", true
	case ".avif":
		return "image/avif", true
	case ".bmp":
		return "image/bmp", true
	case ".ico":
		return "image/x-icon", true
	case ".pdf":
		return "application/pdf", true
	case ".doc", ".docx":
		return "application/msword", true
	case ".xls", ".xlsx":
		return "application/vnd.ms-excel", true
	case ".ppt", ".pptx":
		return "application/vnd.ms-powerpoint", true
	case ".zip", ".gz", ".7z", ".rar", ".tar":
		return "application/octet-stream", true
	case ".txt", ".csv":
		return "text/plain", true
	case ".md", ".markdown", ".mdown", ".mkd":
		return "", false // Markdown 源文件不得作为资产提供
	default:
		return "", false
	}
}

// wikiAsset serves a validated repository asset through the existing Wiki
// catch-all route, which avoids introducing a conflicting Gin wildcard route.
// 安全（review H1）：仅白名单类型内联渲染；其余一律
// application/octet-stream + Content-Disposition: attachment 下载，并加
// Content-Security-Policy: sandbox 兜底（即使类型绕过也无法执行脚本）。
func wikiAsset(c *gin.Context, assetPath string) {
	cfg := wikiservice.LoadGitConfig()
	if !cfg.Enabled() {
		renderNotFound(c)
		return
	}
	// review F3：匿名文件端点限流（per-IP 固定配额，60s 窗口 120 次）。
	// 资产是公开只读端点，不需要配置化；防止未认证调用方反复拉大文件
	// 造成带宽/磁盘 I/O 耗尽。wiki 页面浏览不受影响（限流只在资产分派内）。
	store := ratelimit.Default()
	key := "wiki.asset:ip:" + c.ClientIP()
	if ok, retryAfter, _ := store.Allow(key, 120, time.Minute); !ok {
		c.Header("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limited"})
		return
	}
	file, info, err := wikiservice.OpenWikiAsset(cfg.CloneDir, assetPath)
	if err != nil {
		slog.Warn("wiki asset unavailable", "path", assetPath, "error", err)
		renderNotFound(c)
		return
	}
	defer func() { _ = file.Close() }()
	contentType, safe := wikiAssetType(info.Name())
	if !safe {
		// 未知/危险类型：强制下载，绝不内联（nosniff + CSP sandbox 双保险）。
		contentType = "application/octet-stream"
		c.Header("Content-Disposition", "attachment; filename=\""+strings.ReplaceAll(info.Name(), `"`, "")+"\"")
	}
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Content-Type-Options", "nosniff")
	// CSP sandbox：即使类型/内容被绕过，文档上下文内也不执行脚本。
	c.Header("Content-Security-Policy", "sandbox")
	c.Header("Content-Type", contentType)
	http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), file)
}

func wikiTreePayload(activePath string) ([]WikiTreeNamespacePayload, error) {
	tree, err := wikiservice.BuildTree(activePath)
	if err != nil {
		return nil, err
	}
	result := make([]WikiTreeNamespacePayload, 0, len(tree))
	for _, ns := range tree {
		result = append(result, WikiTreeNamespacePayload{
			Name:  ns.Name,
			Label: ns.Label,
			Slug:  ns.Slug,
			Nodes: wikiTreeNodesPayload(ns.Nodes),
		})
	}
	return result, nil
}

func wikiTreeNodesPayload(nodes []wikiservice.TreeNode) []WikiTreeNodePayload {
	result := make([]WikiTreeNodePayload, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, WikiTreeNodePayload{
			Kind: node.Kind, PageId: node.PageId, Path: node.Path, Title: node.Title,
			Active: node.Active, Children: wikiTreeNodesPayload(node.Children),
		})
	}
	return result
}

// WikiSearchJSONReq wiki 站内搜索请求（前端局内搜索面板调用）。
type WikiSearchJSONReq struct {
	Q     string `form:"q"`
	Limit int    `form:"limit"`
}

// WikiSearchJSONResp wiki 站内搜索响应。
type WikiSearchJSONResp struct {
	Query             string                        `json:"query"`
	Total             int64                         `json:"total"`
	Items             []wikiservice.PageSearchResult `json:"items"`
	SearchUnavailable bool                          `json:"searchUnavailable"`
}

// WikiSearchJSON 提供 wiki 站内局内搜索 JSON API（复用段落级 Meilisearch 索引，
// 聚合为页面级结果；搜索不可用时降级返回空结果并标记 searchUnavailable）。
func WikiSearchJSON(req component.BetterRequest[WikiSearchJSONReq]) component.Response {
	query := strings.TrimSpace(req.Params.Q)
	limit := req.Params.Limit
	if limit <= 0 || limit > 20 {
		limit = 12
	}
	resp := &WikiSearchJSONResp{Query: query, Items: []wikiservice.PageSearchResult{}}
	if query == "" {
		return component.SuccessResponse(resp)
	}
	props, err := wikiservice.SearchPages(query, limit)
	if err != nil {
		slog.Warn("wiki search unavailable", "query", query, "error", err)
		resp.SearchUnavailable = true
		return component.SuccessResponse(resp)
	}
	resp.Total = props.Total
	resp.Items = props.Items
	return component.SuccessResponse(resp)
}

package forum

import (
	"log/slog"
	"strings"

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
		slog.Error("wiki home load failed", "error", err)
		renderInternalError(c)
		return
	}
	tree, err := wikiTreePayload("")
	if err != nil {
		slog.Error("wiki tree load failed", "error", err)
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

	tree, err := wikiTreePayload(page.Path)
	if err != nil {
		slog.Error("wiki tree load failed", "path", path, "error", err)
		renderInternalError(c)
		return
	}
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
	payload.Layout.Sidebar.WikiTree = tree
	renderPage(c, "wiki.gohtml", payload)
	// 计一次浏览（review P2：TopicDetail 已记录，wiki 详情此前漏记）。
	if shouldCountTopicView(&topic) {
		topicviewservice.RecordView(topic.Id)
	}
}

func wikiTreePayload(activePath string) ([]WikiTreeNamespacePayload, error) {
	tree, err := wikiservice.BuildTree(activePath)
	if err != nil {
		return nil, err
	}
	result := make([]WikiTreeNamespacePayload, 0, len(tree))
	for _, ns := range tree {
		pages := make([]WikiTreePagePayload, 0, len(ns.Pages))
		for _, p := range ns.Pages {
			pages = append(pages, WikiTreePagePayload{
				PageId: p.PageId,
				Path:   p.Path,
				Title:  p.Title,
				Active: p.Active,
			})
		}
		result = append(result, WikiTreeNamespacePayload{
			Name:  ns.Name,
			Label: ns.Label,
			Slug:  ns.Slug,
			Pages: pages,
		})
	}
	return result, nil
}

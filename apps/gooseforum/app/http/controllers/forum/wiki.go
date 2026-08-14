package forum

import (
	"log/slog"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
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
	home := wikiservice.BuildHome()
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
	payload.Layout.Sidebar.WikiTree = wikiTreePayload("")
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
	// 路径一律小写归一：path 存库为小写 slug，大写 URL 直接查询会 404（review）。
	path = strings.ToLower(path)
	if path == "" || strings.Contains(path, "//") {
		renderNotFound(c)
		return
	}
	page := wikiPages.GetByPath(path)
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
	props.Page.CanEdit = wikiservice.CanEditPage(loginUserID, &page, &topic)

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
	payload.Layout.Sidebar.WikiTree = wikiTreePayload(page.Path)
	renderPage(c, "wiki.gohtml", payload)
	// 计一次浏览（review P2：TopicDetail 已记录，wiki 详情此前漏记）。
	if shouldCountTopicView(&topic) {
		topicviewservice.RecordView(topic.Id)
	}
}

func wikiTreePayload(activePath string) []WikiTreeNamespacePayload {
	tree := wikiservice.BuildTree(activePath)
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
			Pages: pages,
		})
	}
	return result
}

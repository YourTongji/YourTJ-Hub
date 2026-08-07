package forum

import (
	"math"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/i18n"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
)

func Search(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	scope := c.Query("scope")
	page := parsePositiveInt(c.DefaultQuery("page", "1"), 1)
	props := buildSearchPageProps(query, scope, page)
	payload := PagePayload{
		Component: PageComponentSearch,
		Props:     props,
		Meta:      buildSearchMeta(c, query),
		Layout:    buildLayout(c, "search"),
		URL:       buildPageURL(c),
		Version:   payloadVersion,
	}
	renderAppShell(c, payload)
}

type SearchJSONReq struct {
	Q     string `form:"q"`
	Scope string `form:"scope"`
	Page  int    `form:"page"`
}

// SearchJSON 提供搜索的 JSON API（复用 buildSearchPageProps 的分页逻辑），
// 供前端异步搜索与移动端体验使用；SSR 页面渲染链路保持不变。
func SearchJSON(req component.BetterRequest[SearchJSONReq]) component.Response {
	page := req.Params.Page
	if page < 1 {
		page = 1
	}
	props := buildSearchPageProps(strings.TrimSpace(req.Params.Q), req.Params.Scope, page)
	return component.SuccessResponse(props)
}

func buildSearchMeta(c *gin.Context, query string) PageMeta {
	lang := requestLang(c)
	title := i18n.T(lang, "search")
	if query != "" {
		title = query + " - " + title
	}
	return PageMeta{
		Title:     pageTitle(title),
		Canonical: component.GetBaseUri(c) + buildSearchURL(query, c.Query("scope"), 1),
		Robots:    "noindex,follow",
	}
}

func totalPages(total int64, pageSize int) int {
	if total <= 0 || pageSize <= 0 {
		return 0
	}
	return int(math.Ceil(float64(total) / float64(pageSize)))
}

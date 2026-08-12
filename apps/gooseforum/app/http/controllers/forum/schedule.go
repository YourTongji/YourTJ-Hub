package forum

import (
	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
)

// Schedule 排课器 SSR 页面（/schedule）。
// 排课器为纯客户端交互工具，数据全部走 /api/pk/* JSON API 异步加载（issue #187），
// SSR 仅提供空壳（Props 为空结构），复用 app_shell 渲染链路，无需额外 gohtml。
func Schedule(c *gin.Context) {
	payload := PagePayload{
		Component: PageComponentSchedule,
		Props:     ScheduleProps{},
		Meta:      buildScheduleMeta(c),
		Layout:    buildLayout(c, "schedule"),
		URL:       buildPageURL(c),
		Version:   payloadVersion,
	}
	renderAppShell(c, payload)
}

// buildScheduleMeta 排课器页 meta（Title/Description/Canonical）。
func buildScheduleMeta(c *gin.Context) PageMeta {
	lang := requestLang(c)
	return PageMeta{
		Title:       pageTitle(i18n.T(lang, "schedule")),
		Description: i18n.T(lang, "meta.scheduleDesc", "site", siteTitle()),
		Canonical:   component.GetBaseUri(c) + c.Request.URL.Path,
	}
}

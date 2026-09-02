package forum

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/gin-gonic/gin"
)

// Schedule 排课器 SSR 页面（/schedule）。
// 排课器为纯客户端交互工具，数据全部走 /api/pk/* JSON API 异步加载（issue #187），
// SSR 仅提供空壳（Props 为空结构），复用 app_shell 渲染链路，无需额外 gohtml。
func Schedule(c *gin.Context) {
	payload := PagePayload{
		Component: PageComponentSchedule,
		Props: ScheduleProps{
			// 节次作息直读 DB（与 admin GET /schedule-settings 同一口径）：
			// 该值由管理员低频热改，单行 page_config 查询开销可忽略，5s localcache
			// 反而引入「保存后最多 5s 不生效」的陈旧窗口（且多进程部署下清缓存只对
			// 处理保存的进程生效）。管理员保存即调用 ClearScheduleSettingsConfigCache，
			// 保留该缓存清空调用以兼容其它读方。
			SectionTimes: pageConfig.GetConfigByPageType(pageConfig.ScheduleSettings, defaultconfig.GetDefaultScheduleSettingsConfig()).SectionTimes,
		},
		Meta:    buildScheduleMeta(c),
		Layout:  buildLayout(c, "schedule"),
		URL:     buildPageURL(c),
		Version: payloadVersion,
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

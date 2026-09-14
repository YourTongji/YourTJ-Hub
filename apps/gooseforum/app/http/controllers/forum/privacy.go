package forum

import (
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/setting"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/gin-gonic/gin"
)

const umamiPrivacyDisclosureMarker = "自建 Umami"

const legacyInsightFlarePrivacyDisclosure = `## 事件观测与性能数据

- **公共页面会加载自建 InsightFlare 事件观测服务**：用于统计页面访问、站内路由切换、出站链接和页面性能。当前统计站点为 https://f.yourtj.de，观测服务为 https://ana.yourtj.de。
- **采集字段**：事件可能包含访问页面的 hostname、pathname、URL query string、URL hash、页面标题、来源页面、语言、时区、屏幕尺寸、匿名访问者/会话标识，以及浏览器提供的有限设备信息；性能观测可能包含 TTFB、FCP、LCP、CLS、INP 等 Web Vitals。请勿把密码、Token、身份证件号或其他敏感信息放入 URL 的 query 或 hash。
- **浏览器隐私信号**：当前统计站点配置为不遵循浏览器 Do Not Track（DNT）信号；站点未提供应用内统计开关。如不希望参与统计，请阻止统计脚本或其采集请求。
- **保存与用途**：上述观测数据仅用于站点运行分析、性能改进与安全运营，由自建 InsightFlare 服务处理；具体保存期限以该服务的站点设置和归档策略为准。
`

const umamiPrivacyDisclosure = `## 访问统计与会话回放

- **公共页面会加载自建 Umami**：统计脚本与会话记录器均由 https://umi.yourtj.de 提供，用于分析页面访问、站内导航、设备与来源概况，以及改进页面交互体验。
- **访问统计**：Umami 会处理页面浏览、来源页面、浏览器、操作系统、设备类型和大致国家/地区等匿名统计信息，不使用统计 Cookie，也不以这些统计信息直接识别个人。
- **会话回放**：启用记录的会话可能包含鼠标移动、点击、滚动、页面导航和表单交互；采样比例、输入/文本遮罩级别、最长记录时长与排除区域由 Umami 站点配置控制。请勿在公开 URL、帖子或其他非敏感输入区域提交密码、Token、证件号等敏感信息。
- **保存与用途**：上述数据仅用于站点运行分析、体验改进与故障排查，由自建 Umami 服务处理；具体保存期限以服务端配置为准。
`

// Privacy 隐私政策页面
func Privacy(c *gin.Context) {
	config := hotdataserve.GetPrivacyPolicyConfigCache()
	payload := PagePayload{
		Component: PageComponentPrivacy,
		Props:     buildPrivacyPageProps(config),
		Meta:      buildPrivacyMeta(c),
		Layout:    buildLayout(c, "privacy"),
		URL:       buildPageURL(c),
		Version:   payloadVersion,
	}
	renderPage(c, "privacy.gohtml", payload)
}

// PrivacyPageProps 隐私政策页面数据
type PrivacyPageProps struct {
	Enabled     bool   `json:"enabled"`
	ContentHTML string `json:"contentHtml"`
}

func buildPrivacyPageProps(config pageConfig.PrivacyPolicyConfig) PrivacyPageProps {
	content := strings.ReplaceAll(config.Content, legacyInsightFlarePrivacyDisclosure, umamiPrivacyDisclosure)
	contentHTML := markdown2html.MarkdownToHTML(content)
	// A persisted custom policy can outlive the repository default. Append the
	// production disclosure at render time so collection is never enabled
	// against an online policy that omits the current Umami data scope.
	if setting.IsProduction() && config.Enabled && !strings.Contains(content, umamiPrivacyDisclosureMarker) {
		contentHTML += markdown2html.MarkdownToHTML(umamiPrivacyDisclosure)
	}
	return PrivacyPageProps{
		Enabled:     config.Enabled,
		ContentHTML: contentHTML,
	}
}

func buildPrivacyMeta(c *gin.Context) PageMeta {
	lang := requestLang(c)
	return PageMeta{
		Title:       pageTitle(i18n.T(lang, "privacy")),
		Description: i18n.T(lang, "meta.privacyDesc", "site", siteTitle()),
		Canonical:   component.GetBaseUri(c) + "/privacy",
	}
}

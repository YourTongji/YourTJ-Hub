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
const legacyInsightFlarePrivacyDisclosureMarker = "## 事件观测与性能数据"

const umamiPrivacyDisclosure = `## 访问统计与会话回放

- **公共页面会加载自建 Umami**：统计脚本与会话记录器均由 https://umi.yourtj.de 提供，用于分析页面访问、站内导航、设备与来源概况、页面性能，以及改进页面交互体验。
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
	content := replaceLegacyInsightFlareDisclosure(config.Content)
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

func replaceLegacyInsightFlareDisclosure(content string) string {
	start := strings.Index(content, legacyInsightFlarePrivacyDisclosureMarker)
	if start < 0 {
		return content
	}

	end := len(content)
	if next := strings.Index(content[start+len(legacyInsightFlarePrivacyDisclosureMarker):], "\n## "); next >= 0 {
		end = start + len(legacyInsightFlarePrivacyDisclosureMarker) + next
	}
	legacySection := content[start:end]
	if !strings.Contains(legacySection, "InsightFlare") && !strings.Contains(legacySection, "https://ana.yourtj.de") {
		return content
	}

	return content[:start] + umamiPrivacyDisclosure + content[end:]
}

func buildPrivacyMeta(c *gin.Context) PageMeta {
	lang := requestLang(c)
	return PageMeta{
		Title:       pageTitle(i18n.T(lang, "privacy")),
		Description: i18n.T(lang, "meta.privacyDesc", "site", siteTitle()),
		Canonical:   component.GetBaseUri(c) + "/privacy",
	}
}

package forum

import (
	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/i18n"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
)

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
	return PrivacyPageProps{
		Enabled:     config.Enabled,
		ContentHTML: config.GetHtmlContent(),
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

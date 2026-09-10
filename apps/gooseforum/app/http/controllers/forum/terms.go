package forum

import (
	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// Terms 服务条款页面
func Terms(c *gin.Context) {
	config := hotdataserve.GetTermsOfServiceConfigCache()
	payload := PagePayload{
		Component: PageComponentTerms,
		Props:     buildTermsPageProps(config),
		Meta:      buildTermsMeta(c),
		Layout:    buildLayout(c, "terms"),
		URL:       buildPageURL(c),
		Version:   payloadVersion,
	}
	renderPage(c, "terms.gohtml", payload)
}

// TermsPageProps 服务条款页面数据
type TermsPageProps struct {
	Enabled     bool   `json:"enabled"`
	ContentHTML string `json:"contentHtml"`
}

func buildTermsPageProps(config pageConfig.TermsOfServiceConfig) TermsPageProps {
	return TermsPageProps{
		Enabled:     config.Enabled,
		ContentHTML: config.GetHtmlContent(),
	}
}

func buildTermsMeta(c *gin.Context) PageMeta {
	lang := requestLang(c)
	return PageMeta{
		Title:       pageTitle(i18n.T(lang, "terms")),
		Description: i18n.T(lang, "meta.termsDesc", "site", siteTitle()),
		Canonical:   component.GetBaseUri(c) + "/terms",
	}
}

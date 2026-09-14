package forum

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/gin-gonic/gin"
)

// The shell loads immediately; provider latency belongs to the public status API.
func Status(c *gin.Context) {
	lang := requestLang(c)
	renderAppShell(c, PagePayload{
		Component: PageComponentStatus,
		Props:     struct{}{},
		Meta:      PageMeta{Title: pageTitle(i18n.T(lang, "status.title")), Description: i18n.T(lang, "status.description"), Canonical: component.GetBaseUri(c) + "/status"},
		Layout:    buildLayout(c, "status"), URL: buildPageURL(c), Version: payloadVersion,
	})
}

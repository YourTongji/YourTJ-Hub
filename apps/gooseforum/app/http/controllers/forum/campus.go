package forum

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/gin-gonic/gin"
)

// All private data is fetched after session validation. SSR contains no school records.
func Campus(c *gin.Context) {
	renderAppShell(c, PagePayload{Component: PageComponentCampus, Props: struct{}{}, Meta: PageMeta{Title: pageTitle("我的校园"), Description: "你的同济课表、学业记录与校园消息。", Canonical: component.GetBaseUri(c) + "/campus"}, Layout: campusLayout(buildLayout(c, "campus")), URL: buildPageURL(c), Version: payloadVersion})
}

// Private school records never run analytics, replay or administrator-injected scripts.
func campusLayout(layout LayoutPayload) LayoutPayload {
	layout.UmamiEnabled = false
	layout.Site.ExternalLinks = ""
	return layout
}

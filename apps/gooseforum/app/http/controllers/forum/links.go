package forum

import (
	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

func Links(c *gin.Context) {
	payload := PagePayload{
		Component: PageComponentLinks,
		Props:     buildLinksPageProps(hotdataserve.GetFriendLinksConfigCache()),
		Meta:      buildLinksMeta(c),
		Layout:    buildLayout(c, "links"),
		URL:       buildPageURL(c),
		Version:   payloadVersion,
	}
	renderPage(c, "links.gohtml", payload)
}

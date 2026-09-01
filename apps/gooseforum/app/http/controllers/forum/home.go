package forum

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

func Home(c *gin.Context) {
	sort := c.Query("sort")
	if sort == "" {
		sort = "latest"
	}
	page := cast.ToInt(c.Query("page"))
	if page <= 0 {
		page = 1
	}

	topicPage := hotdataserve.GetLatestTopicsSimpleVoPaginated(page, sort)
	payload := PagePayload{
		Component: PageComponentHome,
		Props:     buildHomeProps(component.LoginUserId(c), page, sort, topicPage.Topics, topicPage.HasNext),
		Meta:      buildHomeMeta(c, page, sort, topicPage.HasNext),
		Layout:    buildLayout(c, activeKeyForHome(sort)),
		URL:       buildPageURL(c),
		Version:   payloadVersion,
	}

	renderPage(c, "home.gohtml", payload)
}

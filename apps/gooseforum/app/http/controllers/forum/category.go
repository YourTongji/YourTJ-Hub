package forum

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

func Category(c *gin.Context) {
	id := cast.ToUint64(c.Param("id"))
	category := hotdataserve.GetCleanCategoryById(id)
	if category == nil {
		renderNotFoundWithMessage(c, component.MessagePageNotFound)
		return
	}

	sort := c.Param("sort")
	if sort == "" {
		sort = "latest"
	}
	page := cast.ToInt(c.Query("page"))
	if page <= 0 {
		page = 1
	}
	topicPage := hotdataserve.GetTopicsByCategorySimpleVo(id, sort, page)

	payload := PagePayload{
		Component: PageComponentCategory,
		Props:     buildCategoryPageProps(component.LoginUserId(c), category, page, sort, topicPage.Topics, topicPage.HasNext),
		Meta:      buildCategoryMeta(c, category, page, sort, topicPage.HasNext),
		Layout:    buildLayout(c, "category_"+cast.ToString(category.Id)),
		URL:       buildPageURL(c),
		Version:   payloadVersion,
	}
	renderPage(c, "category.gohtml", payload)
}

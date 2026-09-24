package forum

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/transform"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/homefeedservice"
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

	if sort == "following" {
		followingHome(c, page)
		return
	}
	topicPage := hotdataserve.GetLatestTopicsSimpleVoPaginated(page, sort)
	payload := PagePayload{
		Component: PageComponentHome,
		Props:     buildHomeProps(c, page, sort, topicPage.Topics, topicPage.HasNext),
		Meta:      buildHomeMeta(c, page, sort, topicPage.HasNext),
		Layout:    buildLayout(c, activeKeyForHome(sort)),
		URL:       buildPageURL(c),
		Version:   payloadVersion,
	}

	renderPage(c, "home.gohtml", payload)
}

func followingLoginURL() string {
	return "/login?" + url.Values{"redirect": {"/?sort=following"}}.Encode()
}

func followingHome(c *gin.Context, page int) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Vary", "X-Goose-Page, Accept")
	viewerID := component.LoginUserId(c)
	if viewerID == 0 {
		if isPageRequest(c) {
			c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
		} else {
			c.Redirect(http.StatusFound, followingLoginURL())
		}
		return
	}
	cursor := c.Query("cursor")
	if page > 1 && cursor == "" {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestInvalidParams, nil))
		return
	}
	result, err := homefeedservice.Following(c.Request.Context(), viewerID, cursor)
	if errors.Is(err, homefeedservice.ErrInvalidCursor) {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestInvalidParams, nil))
		return
	}
	if err != nil {
		renderInternalError(c)
		return
	}
	entities := make([]*topics.Entity, len(result.Topics))
	for i := range result.Topics {
		entities[i] = &result.Topics[i]
	}
	props := buildHomeProps(c, page, "following", transform.Topics2Vo(entities, hotdataserve.CategoryMap()), result.NextCursor != "")
	if result.NextCursor != "" {
		next, _ := url.Parse(props.Pagination.NextURL)
		query := next.Query()
		query.Set("cursor", result.NextCursor)
		next.RawQuery = query.Encode()
		props.Pagination.NextURL = next.String()
	}
	meta := buildHomeMeta(c, page, "following", false)
	meta.Robots = "noindex, nofollow"
	meta.Canonical = ""
	// Keyset pagination cannot reconstruct a previous page from a page number.
	// A refresh/restart always requests the first page without a cursor.
	meta.PrevURL = ""
	meta.NextURL = props.Pagination.NextURL
	renderPage(c, "home.gohtml", PagePayload{
		Component: PageComponentHome,
		Props:     props,
		Meta:      meta,
		Layout:    buildLayout(c, activeKeyForHome("following")),
		URL:       buildPageURL(c),
		Version:   payloadVersion,
	})
}

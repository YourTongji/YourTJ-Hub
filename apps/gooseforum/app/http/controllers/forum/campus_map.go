package forum

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/gin-gonic/gin"
)

// CampusMap is a public read-only atlas. Map data ships as versioned frontend
// assets; this page does not create a second identity, database, or map service.
func CampusMap(c *gin.Context) {
	privateCourses := c.Query("mine") == "1"
	layout := buildLayout(c, "campusMap")
	if privateCourses {
		c.Header("Cache-Control", "private, no-store")
		layout = campusLayout(layout)
	}
	payload := PagePayload{
		Component: PageComponentCampusMap,
		Props:     struct{}{},
		Meta: PageMeta{
			Title:       pageTitle("校园地图"),
			Description: "探索同济大学各校区的建筑、体育场地与校园生活设施。",
			Canonical:   component.GetBaseUri(c) + "/map",
		},
		Layout:  layout,
		URL:     buildPageURL(c),
		Version: payloadVersion,
	}
	renderAppShell(c, payload)
}

package forum

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/transform"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/feedservice"
	"github.com/gin-gonic/gin"
)

func feedCapable(c *gin.Context) bool {
	return !isPageRequest(c) || c.GetHeader("X-Goose-Feed-Version") == "2"
}
func foregroundPage(c *gin.Context) bool {
	return c.GetHeader("X-Goose-Prefetch") != "1" && c.GetHeader("Sec-Purpose") != "prefetch"
}

func personalizeDefault(c *gin.Context, sort string) string {
	uid := component.LoginUserId(c)
	if uid > 0 {
		c.Header("Cache-Control", "private, no-store")
		c.Header("Vary", "X-Goose-Page, X-Goose-Feed-Version, Accept")
		if foregroundPage(c) {
			feedservice.CaptureActivity(uid)
		}
	}
	if !feedCapable(c) {
		if sort == "for_you" {
			return "latest"
		}
		return sort
	}
	if uid > 0 && foregroundPage(c) {
		treatment, variant, err := feedservice.AssignDefault(c.Request.Context(), uid)
		if err == nil {
			c.Set("feed.entryVariant", variant)
			if treatment && sort == "" {
				return "for_you"
			}
		}
	}
	if sort == "for_you" && !feedconfig.Current().Enabled {
		c.Set("feed.degradeReason", "disabled")
		return "latest"
	}
	return sort
}

func decorateFeedProps(c *gin.Context, props *HomeProps, items []feedservice.Candidate, hash string, versions ...feedconfig.Config) {
	if !feedCapable(c) {
		if foregroundPage(c) {
			feedservice.CaptureLegacyServe(component.LoginUserId(c), props.Sort, items)
		}
		return
	}
	props.ActualSort = props.Sort
	if reason, ok := c.Get("feed.degradeReason"); ok {
		props.DegradeReason, _ = reason.(string)
	}
	for i := range props.Tabs {
		if props.Tabs[i].Key == "latest" {
			props.Tabs[i].URL = "/?sort=latest"
		}
	}
	if component.LoginUserId(c) > 0 && feedconfig.Current().Enabled && feedconfig.RankReady() {
		props.Tabs = append([]TabPayload{{Key: "for_you", Label: i18n.T(requestLang(c), "forYouFeed"), URL: "/?sort=for_you", Active: props.Sort == "for_you"}}, props.Tabs...)
	}
	if props.Sort == "latest" && props.Pagination.NextURL != "" {
		u, _ := url.Parse(props.Pagination.NextURL)
		q := u.Query()
		q.Set("sort", "latest")
		u.RawQuery = q.Encode()
		props.Pagination.NextURL = u.String()
	}
	if !foregroundPage(c) {
		return
	}
	variant := "unassigned"
	if v, ok := c.Get("feed.entryVariant"); ok {
		variant, _ = v.(string)
	}
	props.FeedTrace = feedservice.CaptureServeVersion(component.LoginUserId(c), props.Sort, items, variant, "v2", hash, versions...)
	if props.FeedTrace != "" {
		positions := map[uint64]int{}
		for i, item := range items {
			positions[item.ID] = i
		}
		for i := range props.Topics {
			if pos, ok := positions[props.Topics[i].ID]; ok {
				p := pos
				props.Topics[i].FeedTrace = props.FeedTrace
				props.Topics[i].FeedPosition = &p
			}
		}
	}
}

func forYouHome(c *gin.Context, page int) {
	uid := component.LoginUserId(c)
	if uid == 0 {
		if isPageRequest(c) {
			c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
		} else {
			c.Redirect(http.StatusFound, "/login?"+url.Values{"redirect": {"/?sort=for_you"}}.Encode())
		}
		return
	}
	result, err := feedservice.ForYou(c.Request.Context(), uid, c.Query("cursor"))
	if errors.Is(err, feedservice.ErrSnapshotExpired) {
		c.JSON(http.StatusConflict, gin.H{"code": 1, "errorCode": "feed_snapshot_expired", "restartUrl": "/?sort=for_you"})
		return
	}
	if errors.Is(err, feedservice.ErrInvalidCursor) {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestInvalidParams, nil))
		return
	}
	degraded := false
	if err != nil {
		c.Set("feed.degradeReason", "warming_or_busy")
		result, err = feedservice.Fallback(c.Request.Context(), uid)
		degraded = true
		if err != nil {
			renderInternalError(c)
			return
		}
	}
	entities := make([]*topics.Entity, len(result.Topics))
	for i := range result.Topics {
		entities[i] = &result.Topics[i]
	}
	props := buildHomeProps(c, page, "for_you", transform.Topics2Vo(entities, hotdataserve.CategoryMap()), result.NextCursor != "")
	if result.NextCursor != "" {
		props.Pagination.NextURL = "/?" + url.Values{"sort": {"for_you"}, "page": {strconv.Itoa(page + 1)}, "cursor": {result.NextCursor}}.Encode()
	}
	if degraded {
		props.Sort = "latest"
		props.Pagination.NextURL = "/?sort=latest"
		props.Pagination.HasNext = false
	}
	if result.EntryVariant != "" {
		c.Set("feed.entryVariant", result.EntryVariant)
	}
	decorateFeedProps(c, &props, result.Items, result.Hash, result.Config)
	reasons := map[uint64]string{}
	for _, i := range result.Items {
		reasons[i.ID] = i.Reason
	}
	for i := range props.Topics {
		props.Topics[i].FeedReason = reasons[props.Topics[i].ID]
	}
	meta := buildHomeMeta(c, page, "for_you", false)
	meta.Robots = "noindex, nofollow"
	meta.Canonical = ""
	meta.PrevURL = ""
	meta.NextURL = props.Pagination.NextURL
	renderPage(c, "home.gohtml", PagePayload{Component: PageComponentHome, Props: props, Meta: meta, Layout: buildLayout(c, activeKeyForHome("for_you")), URL: buildPageURL(c), Version: payloadVersion})
}

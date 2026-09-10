package forum

import (
	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/spf13/cast"
)

const (
	userProfileSectionSummary   = "summary"
	userProfileSectionActivity  = "activity"
	userProfileSectionBadges    = "badges"
	userProfileSectionBookmarks = "bookmarks"

	userProfileActivityTimeline  = "timeline"
	userProfileActivityTopics    = "topics"
	userProfileActivityLikes     = "likes"
	userProfileActivityBookmarks = "bookmarks"
	userProfileActivityFollowing = "following"
	userProfileActivityFollowers = "followers"
)

func UserProfile(c *gin.Context) {
	userID := cast.ToUint64(c.Param("userId"))
	user, err := users.Get(userID)
	if err != nil || user.Id == 0 {
		RenderNotFoundPage(c, component.MessagePageNotFound)
		return
	}

	// 收藏列表仅对本人可见：他人（含匿名）访问 /u/:id/bookmarks 一律 404，
	// 避免泄露他人收藏内容与收藏时间。
	section := resolveUserProfileSection(c.Param("section"))
	if section == userProfileSectionBookmarks && component.LoginUserId(c) != user.Id {
		RenderNotFoundPage(c, component.MessagePageNotFound)
		return
	}

	props := buildUserProfileProps(c, user, section, resolveUserProfileActivitySection(c.Param("subsection")))
	payload := PagePayload{
		Component: PageComponentUser,
		Props:     props,
		Meta:      buildUserMeta(c, props.User),
		Layout:    buildLayout(c, "user"),
		URL:       buildPageURL(c),
		Version:   payloadVersion,
	}
	renderPage(c, "user.gohtml", payload)
}

func resolveUserProfileSection(raw string) string {
	switch raw {
	case userProfileSectionActivity, userProfileSectionBadges, userProfileSectionBookmarks:
		return raw
	default:
		return userProfileSectionSummary
	}
}

func resolveUserProfileActivitySection(raw string) string {
	switch raw {
	case userProfileActivityTopics, userProfileActivityLikes, userProfileActivityBookmarks, userProfileActivityFollowing, userProfileActivityFollowers:
		return raw
	default:
		return userProfileActivityTimeline
	}
}

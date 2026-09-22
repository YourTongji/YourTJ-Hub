package forum

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

const (
	userProfileSectionSummary   = "summary"
	userProfileSectionActivity  = "activity"
	userProfileSectionBadges    = "badges"
	userProfileSectionBookmarks = "bookmarks"
	userProfileSectionFollowing = "following"
	userProfileSectionFollowers = "followers"

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
	activitySection := resolveUserProfileActivitySection(c.Param("subsection"))
	if section == userProfileSectionActivity && isUserConnectionSection(activitySection) {
		// Keep old activity URLs working while exposing the same canonical props as
		// /u/:id/following and /u/:id/followers.
		section = activitySection
	}
	if isUserConnectionSection(section) {
		activitySection = section
	}
	if section == userProfileSectionBookmarks && component.LoginUserId(c) != user.Id {
		RenderNotFoundPage(c, component.MessagePageNotFound)
		return
	}

	props := buildUserProfileProps(c, user, section, activitySection)
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
	case userProfileSectionActivity, userProfileSectionBadges, userProfileSectionBookmarks, userProfileSectionFollowing, userProfileSectionFollowers:
		return raw
	default:
		return userProfileSectionSummary
	}
}

func isUserConnectionSection(section string) bool {
	return section == userProfileSectionFollowing || section == userProfileSectionFollowers
}

func resolveUserProfileActivitySection(raw string) string {
	switch raw {
	case userProfileActivityTopics, userProfileActivityLikes, userProfileActivityBookmarks, userProfileActivityFollowing, userProfileActivityFollowers:
		return raw
	default:
		return userProfileActivityTimeline
	}
}

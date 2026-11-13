package vo

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/decorationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice/view"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/badgeservice"
)

// UserInfoShow is the shared account-service result.
type UserInfoShow = view.UserInfoShow

// UserDetailedVo is the editable profile payload used by account settings.
type UserDetailedVo struct {
	Decoration          *decorationservice.Display `json:"decoration,omitempty"`
	Id                  uint64                     `json:"id"`
	Username            string                     `json:"username"`
	Email               string                     `json:"email"`
	PendingEmail        string                     `json:"pendingEmail"` // 两阶段换绑暂存邮箱（issue #678）；空 = 无进行中的换绑
	Nickname            string                     `json:"nickname"`
	AvatarUrl           string                     `json:"avatarUrl"`
	ProfileCoverUrl     string                     `json:"profileCoverUrl"`
	Bio                 string                     `json:"bio"`
	Signature           string                     `json:"signature"`
	WebsiteName         string                     `json:"websiteName"`
	Website             string                     `json:"website"`
	Locale              string                     `json:"locale"`
	ExternalInformation users.ExternalInformation  `json:"externalInformation"`
	Prestige            int64                      `json:"prestige"`
	WornBadgeCode       string                     `json:"wornBadgeCode"`
	DisplayBadges       []badgeservice.UserBadge   `json:"displayBadges"`
	Badges              []badgeservice.UserBadge   `json:"badges"`
	WearableBadges      []badgeservice.UserBadge   `json:"wearableBadges"`
	WornBadge           *badgeservice.UserBadge    `json:"wornBadge,omitempty"`
	CreatedAt           time.Time                  `json:"createdAt"`
}

// Package transform maps model entities to API/view payload structs.
package transform

import (
	"context"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/decorationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice/view"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/vo"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/badgeservice"
)

// User2userShow preserves the HTTP adapter API using the account-owned projection.
func User2userShow(user users.EntityComplete) *vo.UserInfoShow {
	return view.BuildUserShow(user)
}

// User2UserDetailedVo maps a user entity to the detailed profile payload.
// PendingEmail 只暴露仍在占用窗口内的换绑暂存（issue #678）；过期暂存视为
// 已放弃，不向设置页展示。
func User2UserDetailedVo(user users.EntityComplete) *vo.UserDetailedVo {
	userBadges := badgeservice.GetUserBadges(user.Id)
	return &vo.UserDetailedVo{
		Decoration:          decorationservice.Effective(context.Background(), user.Id),
		Id:                  user.Id,
		Username:            user.Username,
		Email:               user.Email,
		PendingEmail:        user.FreshPendingEmail(time.Now()),
		Nickname:            user.Nickname,
		AvatarUrl:           user.GetWebAvatarUrl(),
		ProfileCoverUrl:     user.ProfileCoverUrl,
		Bio:                 user.Bio,
		Signature:           user.Signature,
		WebsiteName:         user.WebsiteName,
		Website:             user.Website,
		Locale:              user.Locale,
		ExternalInformation: user.ExternalInformation,
		Prestige:            user.Prestige,
		WornBadgeCode:       user.WornBadgeCode,
		Badges:              userBadges,
		DisplayBadges:       badgeservice.DisplayBadgesFromList(userBadges, user.DisplayBadgeCodes),
		WearableBadges:      badgeservice.WearableBadgesFromList(userBadges),
		WornBadge:           badgeservice.WornBadgeFromList(userBadges, user.WornBadgeCode),
		CreatedAt:           user.CreatedAt,
	}
}

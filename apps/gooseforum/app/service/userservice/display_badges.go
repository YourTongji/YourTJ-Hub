package userservice

import (
	"encoding/json"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/badgeservice"
)

func SetDisplayBadges(userID uint64, codes []string) bool {
	if userID == 0 || !badgeservice.ValidDisplayBadges(badgeservice.GetUserBadges(userID), codes) {
		return false
	}
	if codes == nil {
		codes = []string{}
	}
	encoded, err := json.Marshal(codes)
	if err != nil {
		return false
	}
	if err := users.UpdateFields(userID, map[string]any{"display_badge_codes": string(encoded)}); err != nil {
		return false
	}
	InvalidateUserInfoCache(userID)
	InvalidateUserPublicProfileCache(userID)
	return true
}

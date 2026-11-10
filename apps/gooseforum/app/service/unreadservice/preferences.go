package unreadservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/realtimeservice"
)

type Preferences = users.UnreadDisplayPreferences

func GetPreferences(userID uint64) (Preferences, error) {
	return users.GetUnreadDisplayPreferences(userID)
}

func UpdatePreferences(userID uint64, style *string, categories *[]string) (Preferences, error) {
	if err := users.PatchUnreadDisplayPreferences(userID, style, categories); err != nil {
		return Preferences{}, err
	}
	Invalidate(userID)
	// Existing private invalidation hints refresh other sessions. They contain
	// neither counts nor content and do not create a notification or push task.
	realtimeservice.PublishNotificationsChanged(userID, "preferences")
	return GetPreferences(userID)
}

package unreadservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/realtimeservice"
)

type Preferences struct {
	BadgeStyle string `json:"badgeStyle"`
}

func GetPreferences(userID uint64) (Preferences, error) {
	style, err := users.GetUnreadBadgeStyle(userID)
	return Preferences{BadgeStyle: style}, err
}

func UpdatePreferences(userID uint64, style string) (Preferences, error) {
	if err := users.SetUnreadBadgeStyle(userID, style); err != nil {
		return Preferences{}, err
	}
	Invalidate(userID)
	// Existing private invalidation hints refresh other sessions. They contain
	// neither counts nor content and do not create a notification or push task.
	realtimeservice.PublishNotificationsChanged(userID, "preferences")
	return Preferences{BadgeStyle: style}, nil
}

package userservice

import "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"

// Blocking stops direct contact in both directions; existing public content and
// conversation history remain available for context and reporting.
func SetBlockedUser(ownerID, targetID uint64, blocked bool) error {
	return users.SetBlockedUser(ownerID, targetID, blocked)
}
func ListBlockedUsers(ownerID uint64) ([]users.BlockedUser, error) {
	return users.ListBlockedUsers(ownerID)
}

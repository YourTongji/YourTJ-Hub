package userservice

import (
	"context"
	"errors"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

// CheckFreshSessionVersion bypasses the profile cache so a revoked-all or
// account-close operation takes effect on an already-open event stream.
func CheckFreshSessionVersion(ctx context.Context, userID, version uint64) (bool, error) {
	currentVersion, actorType, err := users.GetSessionStateWithContext(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return currentVersion == version && actorType != users.ActorTypeBot, nil
}

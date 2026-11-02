package userservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func SaveUser(userEntity *users.EntityComplete) error {
	if err := users.Save(userEntity); err != nil {
		return err
	}
	RefreshUserCaches(userEntity)
	return nil
}

func RefreshUserCaches(userEntity *users.EntityComplete) {
	if userEntity == nil || userEntity.Id == 0 {
		return
	}
	refreshUserInfo(*userEntity)
}

func UpdateUserFields(userID uint64, fields map[string]any) error {
	if err := users.UpdateFields(userID, fields); err != nil {
		return err
	}
	InvalidateUserFields(userID)
	return nil
}

// UpdateUserFieldsTx updates the fields inside the caller's transaction, so an
// edit and its side records commit together. The caller must call
// InvalidateUserFields after the transaction commits.
func UpdateUserFieldsTx(tx *gorm.DB, userID uint64, fields map[string]any) error {
	return users.UpdateFieldsTx(tx, userID, fields)
}

// InvalidateUserFields drops the per-user caches refreshed by UpdateUserFields.
func InvalidateUserFields(userID uint64) {
	InvalidateUserInfoCache(userID)
	InvalidateUserPublicProfileCache(userID)
}

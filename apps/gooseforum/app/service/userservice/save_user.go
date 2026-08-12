package userservice

import "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"

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

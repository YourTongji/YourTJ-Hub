package userservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pointservice"
	"gorm.io/gorm"
)

// CreateVerifiedAccountTx initializes an ordinary human account without a local
// password. The caller must authenticate the identity and validate signup policy.
// No first-user administrator grant is made through external authentication.
func CreateVerifiedAccountTx(tx *gorm.DB, username, email, locale string, maxDaily int) (*users.EntityComplete, error) {
	user := &users.EntityComplete{Username: username, Nickname: username, Email: email, Locale: normalizeUserLocale(locale), AvatarUrl: users.RandAvatarUrl()}
	user.Activate()
	if err := users.CreateVerifiedAccountTx(tx, user, maxDaily); err != nil {
		return nil, err
	}
	if err := pointservice.InitUserPointsTx(tx, user.Id, 100); err != nil {
		return nil, err
	}
	if err := userStatistics.CreateTx(tx, user.Id); err != nil {
		return nil, err
	}
	return user, nil
}

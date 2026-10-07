package userservice

import (
	"context"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/feedservice"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agenteventservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/appleauthservice"
	"gorm.io/gorm"
)

// CloseAccount commits the private sticker-library fence and account deletion
// together. A failure leaves the still-active account's memberships usable.
// Asset rows and file references deliberately survive for shared history.
func CloseAccount(ctx context.Context, userID uint64) error {
	err := dbconnect.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := sticker.CloseLibraryTx(tx, userID); err != nil {
			return err
		}
		if err := appleauthservice.RevokeAndDeleteTx(ctx, tx, userID); err != nil {
			return err
		}
		if err := feed.CloseTx(tx, userID); err != nil {
			return err
		}
		if err := users.CloseAccountTx(tx, userID); err != nil {
			return err
		}
		return agenteventservice.WithdrawActorTx(tx, userID)
	})
	if err == nil {
		feedservice.InvalidateAccount(userID)
	}
	return err
}

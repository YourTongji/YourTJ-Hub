package appleauthservice

import (
	"context"
	"errors"
	"strconv"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userOAuth"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

const Provider = "apple"

var ErrNoBinding = errors.New("Apple account must be connected in account settings first")
var ErrAlreadyBound = errors.New("Apple account already connected")
var ErrAccountUnavailable = errors.New("account unavailable for Apple authentication")

func refreshPurpose(userID uint64) string {
	return "yourtj-apple-revocation:" + strconv.FormatUint(userID, 10)
}

// Exchange logs into an existing sub binding (ownerID=0), or explicitly binds
// the authenticated owner. Email is neither requested nor used to merge users.
func Exchange(ctx context.Context, code, token, nonce string, ownerID uint64) (*users.EntityComplete, error) {
	client := currentClient.Load()
	if client == nil {
		return nil, ErrUnavailable
	}
	proof, err := client.Authenticate(ctx, code, token, nonce)
	if err != nil {
		return nil, err
	}
	user, err := storeIdentity(ctx, dbconnect.Connect(), proof, ownerID)
	if err != nil {
		// Do not retain an orphaned grant for a refused binding or unregistered login.
		// Revocation is best effort here; no local account/session was created.
		_ = client.Revoke(ctx, proof.RefreshToken)
	}
	return user, err
}

func storeIdentity(ctx context.Context, conn *gorm.DB, proof identity, ownerID uint64) (*users.EntityComplete, error) {
	if proof.Subject == "" || proof.RefreshToken == "" {
		return nil, ErrInvalidCredential
	}
	bindingMode := ownerID != 0
	if !bindingMode {
		binding, err := userOAuth.GetByIdentityTx(conn.WithContext(ctx), Provider, proof.Subject, false)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrNoBinding
			}
			return nil, err
		}
		ownerID = binding.UserId
	}
	var user users.EntityComplete
	err := conn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Bind, login, unlink and closure acquire the owner row before OAuth rows.
		var err error
		user, err = users.GetForAuthenticationTx(tx, ownerID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAccountUnavailable
			}
			return err
		}
		if user.IsBot() || user.IsFrozen == users.StatusFrozen {
			return ErrAccountUnavailable
		}
		binding, err := userOAuth.GetByIdentityTx(tx, Provider, proof.Subject, true)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if binding.Id != 0 && binding.UserId != ownerID {
			return ErrAlreadyBound
		}
		if binding.Id == 0 {
			if !bindingMode {
				return ErrNoBinding
			}
			existing, err := userOAuth.GetByOwnerTx(tx, ownerID, Provider)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if existing.Id != 0 {
				return ErrAlreadyBound
			}
		}
		encrypted, err := securestore.EncryptPurpose(proof.RefreshToken, refreshPurpose(ownerID))
		if err != nil {
			return err
		}
		binding.UserId, binding.Provider, binding.ProviderUid = ownerID, Provider, proof.Subject
		return userOAuth.SaveAppleBindingTx(tx, &binding, encrypted)
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// RevokeAndDeleteTx is called before account closure commits, or by explicit
// unlink. A transient Apple error leaves the account and encrypted grant intact
// so the owner can retry. Apple revocation is idempotent across local retries.
func RevokeAndDeleteTx(ctx context.Context, tx *gorm.DB, userID uint64) error {
	if _, err := users.GetForAuthenticationTx(tx, userID); err != nil {
		return err
	}
	binding, err := userOAuth.GetByOwnerTx(tx, userID, Provider)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	client := currentClient.Load()
	if client == nil {
		return ErrUnavailable
	}
	refresh, err := securestore.DecryptPurpose(binding.AppleRefreshToken, refreshPurpose(userID))
	if err != nil {
		return ErrUnavailable
	}
	if err := client.Revoke(ctx, refresh); err != nil {
		return err
	}
	return userOAuth.DeleteTx(tx, &binding)
}

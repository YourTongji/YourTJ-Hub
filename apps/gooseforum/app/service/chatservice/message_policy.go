package chatservice

import (
	"errors"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

var ErrMessagePermissionDenied = errors.New("private message permission denied")

// Call after the participant locks, committed-retry lookup and block check.
// Config rows alone never establish a prior conversation: a message must exist.
func checkNewConversationPolicy(tx *gorm.DB, senderID, peerID uint64) error {
	settings, err := users.GetPrivacySettingsWithDB(tx, peerID)
	if err != nil {
		return err
	}
	if settings.ChatMessagePolicy == users.ChatMessageEveryone {
		return nil
	}
	conversations := tx.Model(&imUserChatConfigs.Entity{}).Select("conv_id").Where(
		"(user_id = ? AND peer_id = ?) OR (user_id = ? AND peer_id = ?)", senderID, peerID, peerID, senderID)
	var previous messages.Entity
	err = tx.Select("id").Where("conv_id IN (?) AND sender_id IN ?", conversations, []uint64{senderID, peerID}).Take(&previous).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if settings.ChatMessagePolicy == users.ChatMessageMutual {
		outgoing, incoming, err := userFollow.MutualStateWithDB(tx, senderID, peerID)
		if err != nil {
			return err
		}
		if outgoing && incoming {
			return nil
		}
	}
	return ErrMessagePermissionDenied
}

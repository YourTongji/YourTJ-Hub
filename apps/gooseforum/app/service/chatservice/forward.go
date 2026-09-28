package chatservice

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/realtimeservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/unreadservice"
	"gorm.io/gorm"
)

type ForwardRequest struct {
	ConvID          uint64   `json:"convId" validate:"required"`
	PeerID          uint64   `json:"peerId" validate:"required"`
	MessageIDs      []uint64 `json:"messageIds" validate:"required,min=1,max=50,dive,required"`
	Mode            string   `json:"mode" validate:"oneof=individual merged"`
	ClientForwardID string   `json:"clientForwardId" validate:"required,max=64"`
}
type ForwardResult struct {
	ConvID     uint64   `json:"convId"`
	MessageIDs []uint64 `json:"messageIds"`
}

func forwardedPayload(message messages.Entity) *messages.ForwardedBundle {
	if message.MsgType == messages.ForwardType {
		return messages.ParseForward(message.Content)
	}
	return nil
}

// ForwardMessages atomically delivers one recipient's batch. The client may
// select several recipients, but each is acknowledged/retried independently.
func ForwardMessages(actor uint64, request ForwardRequest) (*ForwardResult, error) {
	ids := slices.Clone(request.MessageIDs)
	if actor == 0 || actor == request.PeerID || request.PeerID == 0 || request.ConvID == 0 || len(ids) == 0 || len(ids) > messages.MaxForwardMessages || !clientMessageKey.MatchString(request.ClientForwardID) || (request.Mode != "individual" && request.Mode != "merged") || (request.Mode == "individual" && len(ids) > 10) {
		return nil, errors.New("invalid forward request")
	}
	slices.Sort(ids)
	for i, id := range ids {
		if id == 0 || (i > 0 && ids[i-1] == id) {
			return nil, errors.New("invalid forward message IDs")
		}
	}
	request.MessageIDs = ids
	outgoingCount := len(ids)
	if request.Mode == "merged" {
		outgoingCount = 1
	}
	keys := make([]string, outgoingCount)
	for i := range keys {
		key, err := forwardMessageKey(actor, request, i)
		if err != nil {
			return nil, err
		}
		keys[i] = key
	}
	security := hotdataserve.GetSecuritySettingsConfigCache()
	for attempt := 0; attempt < 3; attempt++ {
		result := &ForwardResult{MessageIDs: make([]uint64, 0, len(ids))}
		changed := false
		err := db.Connect().Transaction(func(tx *gorm.DB) error {
			// Direct, uncached membership check: IDs from another conversation never
			// become a lookup oracle or a source for this copy.
			var member imUserChatConfigs.Entity
			if err := tx.Where("user_id = ? AND conv_id = ? AND is_deleted = 0", actor, request.ConvID).First(&member).Error; err != nil {
				return errors.New("conversation not found")
			}
			var source []messages.Entity
			if err := tx.Where("conv_id = ? AND id IN ?", request.ConvID, ids).Order("id").Find(&source).Error; err != nil {
				return err
			}
			if len(source) != len(ids) {
				return errors.New("conversation not found")
			}
			if err := users.LockInteractionUsers(tx, actor, request.PeerID); err != nil {
				return err
			}
			// Resolve an acknowledged batch before rebuilding display-only metadata.
			// A profile update must never turn a successful delivery into a retry failure.
			previous := make([]messages.Entity, 0, outgoingCount)
			for i := range outgoingCount {
				var stored messages.Entity
				err := tx.Where("sender_id = ? AND client_message_id = ?", actor, keys[i]).First(&stored).Error
				if err == nil {
					previous = append(previous, stored)
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
			if len(previous) > 0 {
				if len(previous) != outgoingCount {
					return errors.New("forward identity conflict")
				}
				for i, stored := range previous {
					convID, err := sendMessageWithEffects(tx, actor, request.PeerID, stored.Content, stored.MsgType, false, keys[i])
					if err != nil {
						return err
					}
					result.ConvID = convID
					result.MessageIDs = append(result.MessageIDs, stored.Id)
				}
				return nil
			}
			outgoing := source
			if request.Mode == "merged" {
				encoded, err := mergedForwardContent(tx, source)
				if err != nil {
					return err
				}
				outgoing = []messages.Entity{{Content: encoded, MsgType: messages.ForwardType}}
			} else {
				for _, message := range source {
					if message.MsgType == messages.ForwardType {
						if forwardedPayload(message) == nil {
							return errors.New("unsupported forwarded message")
						}
					} else if message.MsgType < 1 || message.MsgType > 3 {
						return errors.New("unsupported forwarded message")
					}
				}
			}
			for i, message := range outgoing {
				key := keys[i]
				if len(moderationservice.FindSensitiveWordsWithConfig(messages.DisplayContent(message.Content, message.MsgType), security)) > 0 {
					return errors.New("forwarded content blocked")
				}
				changed = true
				convID, err := sendMessageWithEffects(tx, actor, request.PeerID, message.Content, message.MsgType, false, key)
				if err != nil {
					return err
				}
				result.ConvID = convID
				var stored messages.Entity
				if err := tx.Select("id").Where("sender_id = ? AND client_message_id = ?", actor, key).First(&stored).Error; err != nil {
					return err
				}
				result.MessageIDs = append(result.MessageIDs, stored.Id)
			}
			return nil
		})
		if err == nil {
			if changed {
				imUserChatConfigs.InvalidateConversationAccess(actor, result.ConvID)
				imUserChatConfigs.InvalidateConversationAccess(request.PeerID, result.ConvID)
				unreadservice.Invalidate(request.PeerID)
				realtimeservice.PublishChatChanged(actor, result.ConvID, "sent")
				realtimeservice.PublishChatChanged(request.PeerID, result.ConvID, "received")
			}
			return result, nil
		}
		if attempt == 2 || !retryableChatWrite(err) {
			return nil, err
		}
		time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
	}
	return nil, errors.New("forward retry exhausted")
}

// Only merged forwarding creates a snapshot. Individual copies keep each
// original body and must not inherit the aggregate snapshot byte limit.
func mergedForwardContent(tx *gorm.DB, source []messages.Entity) (string, error) {
	// Current profile names are display-only; private per-viewer notes are never copied.
	senderIDs := make([]uint64, 0, len(source))
	for _, message := range source {
		senderIDs = append(senderIDs, message.SenderId)
	}
	identities, err := users.ChatIdentitiesByIDs(tx, senderIDs)
	if err != nil {
		return "", err
	}
	bundle := &messages.ForwardedBundle{Version: 1}
	totalBytes := 0
	for _, message := range source {
		totalBytes += len(message.Content)
		if totalBytes > messages.MaxForwardBytes {
			return "", errors.New("forward content too large")
		}
		name := "Unknown user"
		identity := identities[message.SenderId]
		if identity.Name != "" {
			name = identity.Name
		}
		entry := messages.ForwardedEntry{SenderName: name, AvatarURL: identity.AvatarURL, Content: message.Content, CreatedAt: message.CreatedAt.Format(time.RFC3339), MsgType: message.MsgType}
		if nested := forwardedPayload(message); nested != nil {
			entry.Forwarded = nested
			entry.Content = nested.Text() // Readable fallback for older clients.
		} else if message.MsgType < 1 || message.MsgType > 3 {
			return "", errors.New("unsupported forwarded message")
		}
		bundle.Messages = append(bundle.Messages, entry)
	}
	return bundle.Encode()
}

func forwardMessageKey(actor uint64, request ForwardRequest, index int) (string, error) {
	identity, err := json.Marshal(struct {
		Actor   uint64
		Request ForwardRequest
		Index   int
	}{actor, request, index})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(identity)
	return hex.EncodeToString(digest[:]), nil
}

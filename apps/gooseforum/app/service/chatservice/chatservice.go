package chatservice

import (
	"errors"
	"slices"
	"strings"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/vo"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imConversations"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/realtimeservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/unreadservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/urlconfig"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultMessageLimit = 30
	maxMessageLimit     = 100
	maxVisibleReadIDs   = 100
)

type MessageCursorResult struct {
	List          []*vo.MessageVo `json:"list"`
	HasMoreBefore bool            `json:"hasMoreBefore"`
	HasMoreAfter  bool            `json:"hasMoreAfter"`
	NextBeforeId  uint64          `json:"nextBeforeId"`
	LatestId      uint64          `json:"latestId"`
}

// SendMessage creates or updates a direct conversation and stores a message.
func SendMessage(senderId, peerId uint64, content string, msgType int8) (uint64, error) {
	return sendMessage(db.Connect(), senderId, peerId, content, msgType)
}

func sendMessage(conn *gorm.DB, senderId, peerId uint64, content string, msgType int8) (uint64, error) {
	if senderId == peerId {
		return 0, errors.New("cannot send message to yourself")
	}
	// A conversation row serializes message ID allocation, summary updates, and
	// visible-read commands. New-conversation uniqueness races roll back their
	// entire transaction before a bounded retry finds the winner's config.
	for attempt := 0; attempt < 3; attempt++ {
		var convId uint64
		err := conn.Transaction(func(tx *gorm.DB) error {
			var senderConfig imUserChatConfigs.Entity
			findErr := tx.Where("user_id = ? AND peer_id = ?", senderId, peerId).First(&senderConfig).Error
			if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
				return findErr
			}
			if senderConfig.Id == 0 {
				var reverse imUserChatConfigs.Entity
				reverseErr := tx.Where("user_id = ? AND peer_id = ?", peerId, senderId).First(&reverse).Error
				if reverseErr != nil && !errors.Is(reverseErr, gorm.ErrRecordNotFound) {
					return reverseErr
				}
				if reverse.Id != 0 {
					convId = reverse.ConvId
				} else {
					conv := &imConversations.Entity{Type: 1, LastMsgTime: time.Now()}
					if err := tx.Create(conv).Error; err != nil {
						return err
					}
					convId = conv.Id
				}
			} else {
				convId = senderConfig.ConvId
			}
			if err := lockConversation(tx, convId); err != nil {
				return err
			}
			if senderConfig.Id == 0 {
				senderConfig = imUserChatConfigs.Entity{UserId: senderId, PeerId: peerId, ConvId: convId}
				if err := tx.Create(&senderConfig).Error; err != nil {
					return err
				}
			} else if senderConfig.ConvId != convId {
				return errors.New("conversation membership changed")
			}
			var peerConfig imUserChatConfigs.Entity
			peerErr := tx.Where("user_id = ? AND peer_id = ?", peerId, senderId).First(&peerConfig).Error
			if peerErr != nil && !errors.Is(peerErr, gorm.ErrRecordNotFound) {
				return peerErr
			}
			if peerConfig.Id == 0 {
				peerConfig = imUserChatConfigs.Entity{UserId: peerId, PeerId: senderId, ConvId: convId}
				if err := tx.Create(&peerConfig).Error; err != nil {
					return err
				}
			} else if peerConfig.ConvId != convId {
				return errors.New("conversation membership changed")
			}
			now := time.Now()
			if err := tx.Create(&messages.Entity{
				ConvId: convId, SenderId: senderId, Content: content,
				MsgType: msgType, IsRead: 0, CreatedAt: now,
			}).Error; err != nil {
				return err
			}
			if err := tx.Model(&imConversations.Entity{}).Where("id = ?", convId).
				Updates(map[string]any{"last_msg_content": content, "last_msg_time": now}).Error; err != nil {
				return err
			}
			if err := tx.Model(&imUserChatConfigs.Entity{}).Where("id = ?", senderConfig.Id).
				Updates(map[string]any{"updated_at": now, "is_deleted": 0}).Error; err != nil {
				return err
			}
			return tx.Model(&imUserChatConfigs.Entity{}).Where("id = ?", peerConfig.Id).
				Updates(map[string]any{"unread_count": gorm.Expr("unread_count + 1"), "updated_at": now, "is_deleted": 0}).Error
		})
		if err == nil {
			imUserChatConfigs.InvalidateConversationAccess(senderId, convId)
			imUserChatConfigs.InvalidateConversationAccess(peerId, convId)
			unreadservice.Invalidate(peerId)
			realtimeservice.PublishChatChanged(senderId, convId, "sent")
			realtimeservice.PublishChatChanged(peerId, convId, "received")
			return convId, nil
		}
		if attempt == 2 || !retryableChatWrite(err) {
			return 0, err
		}
		time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
	}
	return 0, errors.New("chat send retry exhausted")
}

// GetChatList returns the current user's conversations.
func GetChatList(userId uint64) ([]*vo.ChatItemVo, error) {
	configs := imUserChatConfigs.GetUserConfigs(userId)
	if len(configs) == 0 {
		return []*vo.ChatItemVo{}, nil
	}

	peerIds := lo.Map(configs, func(cfg imUserChatConfigs.Entity, _ int) uint64 {
		return cfg.PeerId
	})
	convIds := lo.Map(configs, func(cfg imUserChatConfigs.Entity, _ int) uint64 {
		return cfg.ConvId
	})

	peers := users.GetMapByIds(peerIds)
	convs := imConversations.GetMapByIds(convIds)

	list := lo.Map(configs, func(cfg imUserChatConfigs.Entity, _ int) *vo.ChatItemVo {
		peer := peers[cfg.PeerId]
		conv := convs[cfg.ConvId]

		chatItem := &vo.ChatItemVo{
			Id:          cfg.Id,
			PeerId:      cfg.PeerId,
			UnreadCount: cfg.UnreadCount,
			ConvId:      cfg.ConvId,
			PeerUrl:     urlconfig.User(cfg.PeerId),
		}

		if peer != nil {
			chatItem.PeerUsername = peer.Username
			chatItem.PeerAvatar = peer.GetWebAvatarUrl()
		} else {
			chatItem.PeerUsername = "Unknown User"
		}

		if conv != nil {
			chatItem.LastMsg = conv.LastMsgContent
			chatItem.LastMsgTime = conv.LastMsgTime.Format(time.RFC3339)
		}

		return chatItem
	})

	return list, nil
}

// GetMessages returns cursor-paginated messages for a conversation.
func GetMessages(userId, convId uint64, beforeId, afterId uint64, limit int) (*MessageCursorResult, error) {
	if !imUserChatConfigs.CanAccessConversation(userId, convId) {
		return nil, errors.New("conversation not found")
	}
	if beforeId > 0 && afterId > 0 {
		return nil, errors.New("beforeId and afterId cannot be used together")
	}
	if limit <= 0 {
		limit = defaultMessageLimit
	}
	if limit > maxMessageLimit {
		limit = maxMessageLimit
	}

	queryLimit := limit + 1
	var msgs []messages.Entity
	switch {
	case beforeId > 0:
		msgs = messages.GetBeforeId(convId, beforeId, queryLimit)
	case afterId > 0:
		msgs = messages.GetAfterId(convId, afterId, queryLimit)
	default:
		msgs = messages.GetLatestByConvId(convId, queryLimit)
	}

	hasMore := len(msgs) > limit
	if hasMore {
		msgs = msgs[:limit]
	}
	if afterId == 0 {
		slices.Reverse(msgs)
	}

	list := lo.Map(msgs, func(m messages.Entity, _ int) *vo.MessageVo {
		return &vo.MessageVo{
			Id:        m.Id,
			SenderId:  m.SenderId,
			Content:   m.Content,
			MsgType:   m.MsgType,
			IsRead:    m.IsRead,
			CreatedAt: m.CreatedAt.Format(time.RFC3339),
			IsSelf:    m.SenderId == userId,
		}
	})

	result := &MessageCursorResult{
		List: list,
	}
	if len(list) > 0 {
		result.NextBeforeId = list[0].Id
		result.LatestId = list[len(list)-1].Id
	}
	if afterId > 0 {
		result.HasMoreAfter = hasMore
	} else {
		result.HasMoreBefore = hasMore
	}
	return result, nil
}

// MarkRead 清除指定会话的未读状态。
//
// 必须先校验调用方是否为该会话成员，否则任意已认证用户可枚举连续的 convId
// 越权翻转他人私聊会话的已读状态（issue #111，CWE-639）。校验失败时返回
// 与 GetMessages 一致的 "conversation not found" 错误语义，且不触碰任何状态。
func MarkRead(userId, convId uint64) error {
	var peerID uint64
	err := db.Connect().Transaction(func(tx *gorm.DB) error {
		config, err := lockedMember(tx, userId, convId)
		if err != nil {
			return err
		}
		peerID = config.PeerId
		if err := tx.Model(&messages.Entity{}).
			Where("conv_id = ? AND sender_id != ? AND is_read = 0", convId, userId).
			Update("is_read", 1).Error; err != nil {
			return err
		}
		return tx.Model(config).Update("unread_count", 0).Error
	})
	if err != nil {
		return err
	}
	unreadservice.Invalidate(userId)
	realtimeservice.PublishChatChanged(userId, convId, "read")
	realtimeservice.PublishChatChanged(peerID, convId, "read")
	return nil
}

type VisibleReadResult struct {
	ConvId                 uint64   `json:"convId"`
	AcknowledgedMessageIds []uint64 `json:"acknowledgedMessageIds"`
	UnreadCount            uint     `json:"unreadCount"`
}

type MessageReadState struct {
	Id     uint64 `json:"id"`
	IsRead int    `json:"isRead"`
}

type MessageReadStatesResult struct {
	Items       []MessageReadState `json:"items"`
	UnreadCount uint               `json:"unreadCount"`
}

// MarkVisibleRead only acknowledges the incoming IDs actually displayed by a
// client. A higher message ID never implies earlier messages were seen.
func MarkVisibleRead(userId, convId uint64, ids []uint64) (*VisibleReadResult, error) {
	return markVisibleRead(db.Connect(), userId, convId, ids)
}

func markVisibleRead(conn *gorm.DB, userId, convId uint64, ids []uint64) (*VisibleReadResult, error) {
	unique, err := validMessageIDs(ids)
	if err != nil || userId == 0 || convId == 0 {
		return nil, errors.New("invalid visible message IDs")
	}
	result := &VisibleReadResult{ConvId: convId, AcknowledgedMessageIds: unique}
	var peerID uint64
	err = conn.Transaction(func(tx *gorm.DB) error {
		config, err := lockedMember(tx, userId, convId)
		if err != nil {
			return err
		}
		peerID = config.PeerId
		var owned []messages.Entity
		if err := tx.Select("id").Where("conv_id = ? AND sender_id != ? AND id IN ?", convId, userId, unique).
			Find(&owned).Error; err != nil {
			return err
		}
		if len(owned) != len(unique) {
			return errors.New("conversation not found")
		}
		updated := tx.Model(&messages.Entity{}).
			Where("conv_id = ? AND sender_id != ? AND id IN ? AND is_read = 0", convId, userId, unique).
			Update("is_read", 1)
		if updated.Error != nil {
			return updated.Error
		}
		// Only newly acknowledged rows decrement the counter. The conversation
		// lock also serializes sends; clamp legacy counter drift without wrapping.
		unread := config.UnreadCount
		if changed := uint(updated.RowsAffected); changed < unread {
			unread -= changed
		} else {
			unread = 0
		}
		result.UnreadCount = unread
		return tx.Model(config).Update("unread_count", unread).Error
	})
	if err != nil {
		return nil, err
	}
	unreadservice.Invalidate(userId)
	realtimeservice.PublishChatChanged(userId, convId, "read")
	realtimeservice.PublishChatChanged(peerID, convId, "read")
	return result, nil
}

// GetMessageReadStates refreshes a loaded message window without downloading
// message bodies. Both incoming and outgoing IDs are permitted for members.
func GetMessageReadStates(userId, convId uint64, ids []uint64) (*MessageReadStatesResult, error) {
	return getMessageReadStates(db.Connect(), userId, convId, ids)
}

func getMessageReadStates(conn *gorm.DB, userId, convId uint64, ids []uint64) (*MessageReadStatesResult, error) {
	unique, err := validMessageIDs(ids)
	if err != nil || userId == 0 || convId == 0 {
		return nil, errors.New("invalid message IDs")
	}
	result := &MessageReadStatesResult{Items: make([]MessageReadState, 0, len(unique))}
	err = conn.Transaction(func(tx *gorm.DB) error {
		// READ COMMITTED otherwise permits separate statements to observe
		// different commits. Share the bounded conversation lock with mutations.
		config, err := lockedMember(tx, userId, convId)
		if err != nil {
			return err
		}
		var found []messages.Entity
		if err := tx.Select("id", "is_read").Where("conv_id = ? AND id IN ?", convId, unique).
			Find(&found).Error; err != nil {
			return err
		}
		if len(found) != len(unique) {
			return errors.New("conversation not found")
		}
		byID := make(map[uint64]int, len(found))
		for _, item := range found {
			byID[item.Id] = item.IsRead
		}
		for _, id := range unique {
			result.Items = append(result.Items, MessageReadState{Id: id, IsRead: byID[id]})
		}
		result.UnreadCount = config.UnreadCount
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func validMessageIDs(ids []uint64) ([]uint64, error) {
	if len(ids) == 0 || len(ids) > maxVisibleReadIDs {
		return nil, errors.New("invalid message IDs")
	}
	seen := make(map[uint64]struct{}, len(ids))
	unique := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			return nil, errors.New("invalid message ID")
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			unique = append(unique, id)
		}
	}
	return unique, nil
}

func lockConversation(tx *gorm.DB, convId uint64) error {
	var conv imConversations.Entity
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").Where("id = ?", convId).First(&conv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("conversation not found")
		}
		return err
	}
	return nil
}

func lockedMember(tx *gorm.DB, userId, convId uint64) (*imUserChatConfigs.Entity, error) {
	if err := lockConversation(tx, convId); err != nil {
		return nil, err
	}
	var config imUserChatConfigs.Entity
	if err := tx.Where("user_id = ? AND conv_id = ? AND is_deleted = 0", userId, convId).
		First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("conversation not found")
		}
		return nil, err
	}
	return &config, nil
}

func retryableChatWrite(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint") ||
		strings.Contains(message, "duplicate key") ||
		strings.Contains(message, "database is locked") ||
		strings.Contains(message, "sqlite_busy") ||
		strings.Contains(message, "deadlock detected")
}

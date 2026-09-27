package api

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/chatservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
)

// SendMessageReq 发送私信请求
type SendMessageReq struct {
	ClientMessageID string `json:"clientMessageId" validate:"omitempty,max=64"`
	PeerId          uint64 `json:"peerId" validate:"required"`
	Content         string `json:"content" validate:"required"`
	MsgType         int8   `json:"msgType" validate:"oneof=1 2 3"` // 1: Text, 2: Image, 3: Voice
}

// SendMessage 发送私信
func SendMessage(req component.BetterRequest[SendMessageReq]) component.Response {
	// 敏感词检查：私信仅支持直接拦截（无状态字段，不适合延迟可见）
	if words := moderationservice.FindSensitiveWords(req.Params.Content); len(words) > 0 {
		word := words[0]
		moderationservice.SensitiveContentBlocked(req.UserId, "chat", 0, word, truncateExcerpt(req.Params.Content))
		return component.FailResponseCode(
			component.MessageChatSensitiveBlocked,

			component.MessageParams{"word": word, "words": words})

	}
	// Set default msg type to text if not provided or 0 (though validate should handle it if required, let's assume default 1)
	msgType := req.Params.MsgType
	if msgType == 0 {
		msgType = 1
	}
	convId, err := chatservice.SendMessage(req.UserId, req.Params.PeerId, req.Params.Content, msgType, req.Params.ClientMessageID)
	if err != nil {
		// Database errors can contain private values. Record diagnostic categories,
		// never message bodies or raw driver details in the response or this log.
		var driverError interface{ SQLState() string }
		var sqlState string
		if errors.As(err, &driverError) {
			sqlState = driverError.SQLState()
		}
		slog.Warn("chat send failed", "senderID", req.UserId, "peerID", req.Params.PeerId, "errorType", fmt.Sprintf("%T", err), "sqlState", sqlState)
		return component.FailResponseCode(component.MessageChatSendFailed, nil)
	}
	return successDataMap("convId", convId)
}

// GetMessagesReq 获取消息记录请求
type GetMessagesReq struct {
	ConvId   uint64 `json:"convId" validate:"required"`
	BeforeId uint64 `json:"beforeId"`
	AfterId  uint64 `json:"afterId"`
	Limit    int    `json:"limit" validate:"omitempty,min=1,max=100"`
}

// GetMessages 获取消息记录
func GetMessages(req component.BetterRequest[GetMessagesReq]) component.Response {
	result, err := chatservice.GetMessages(req.UserId, req.Params.ConvId, req.Params.BeforeId, req.Params.AfterId, req.Params.Limit)
	if err != nil {
		return component.FailResponseCode(component.MessageChatGetMessagesFailed, nil)
	}
	return component.SuccessResponse(result)
}

// MarkReadReq 标记已读请求
type MarkReadReq struct {
	ConvId uint64 `json:"convId" validate:"required"`
}

// MarkChatRead 标记已读
func MarkChatRead(req component.BetterRequest[MarkReadReq]) component.Response {
	err := chatservice.MarkRead(req.UserId, req.Params.ConvId)
	if err != nil {
		return component.FailResponseCode(component.MessageChatMarkReadFailed, nil)
	}
	return component.SuccessResponse(nil)
}

type ChatMessageIDsReq struct {
	ConvId     uint64   `json:"convId" validate:"required"`
	MessageIds []uint64 `json:"messageIds" validate:"required,min=1,max=100,dive,required"`
}

// MarkChatVisibleRead acknowledges only messages the client actually displayed.
func MarkChatVisibleRead(req component.BetterRequest[ChatMessageIDsReq]) component.Response {
	result, err := chatservice.MarkVisibleRead(req.UserId, req.Params.ConvId, req.Params.MessageIds)
	if err != nil {
		return component.FailResponseCode(component.MessageChatMarkReadFailed, nil)
	}
	return component.SuccessResponse(result)
}

// GetChatMessageReadStates refreshes read flags without message bodies.
func GetChatMessageReadStates(req component.BetterRequest[ChatMessageIDsReq]) component.Response {
	result, err := chatservice.GetMessageReadStates(req.UserId, req.Params.ConvId, req.Params.MessageIds)
	if err != nil {
		return component.FailResponseCode(component.MessageChatGetMessagesFailed, nil)
	}
	return component.SuccessResponse(result)
}

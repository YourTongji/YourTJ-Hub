package api

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/chatservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
)

// SendMessageReq 发送私信请求
type SendMessageReq struct {
	PeerId  uint64 `json:"peerId" validate:"required"`
	Content string `json:"content" validate:"required"`
	MsgType int8   `json:"msgType" validate:"oneof=1 2 3"` // 1: Text, 2: Image, 3: Voice
}

// SendMessage 发送私信
func SendMessage(req component.BetterRequest[SendMessageReq]) component.Response {
	// 敏感词检查：私信仅支持直接拦截（无状态字段，不适合延迟可见）
	if hit, word := moderationservice.CheckContentAllowed(req.Params.Content); hit {
		moderationservice.SensitiveContentBlocked(req.UserId, "chat", 0, word, truncateExcerpt(req.Params.Content))
		return component.FailResponseCode(
			component.MessageChatSensitiveBlocked,

			component.MessageParams{"word": word})

	}
	// Set default msg type to text if not provided or 0 (though validate should handle it if required, let's assume default 1)
	msgType := req.Params.MsgType
	if msgType == 0 {
		msgType = 1
	}
	convId, err := chatservice.SendMessage(req.UserId, req.Params.PeerId, req.Params.Content, msgType)
	if err != nil {
		return component.FailResponseCode(
			component.MessageChatSendFailed,

			component.MessageParams{"error": err.Error()})

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

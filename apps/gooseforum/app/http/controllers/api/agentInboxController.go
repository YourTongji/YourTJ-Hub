package api

import (
	"errors"

	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/agentInbox"
	"gorm.io/gorm"
)

// AgentInboxItem is the Agent-facing view of one inbox row. The delivery
// state is read from the inbox row itself (single source of truth).
type AgentInboxItem struct {
	Id             uint64 `json:"id"`
	TopicId        uint64 `json:"topicId"`
	PostId         uint64 `json:"postId"`
	EventType      string `json:"eventType"`
	ActorId        uint64 `json:"actorId"`
	ContentPreview string `json:"contentPreview"`
	Status         uint8  `json:"status"`
	DeliveryStatus uint8  `json:"deliveryStatus"`
	Attempts       uint8  `json:"attempts"`
	LastError      string `json:"lastError"`
	ReadAt         *int64 `json:"readAt,omitempty"`
	CreatedAt      int64  `json:"createdAt"`
	UpdatedAt      int64  `json:"updatedAt"`
}

func toAgentInboxItem(entity agentInbox.Entity) AgentInboxItem {
	var readAt *int64
	if entity.ReadAt != nil {
		value := entity.ReadAt.UnixMilli()
		readAt = &value
	}
	return AgentInboxItem{
		Id:             entity.Id,
		TopicId:        entity.TopicId,
		PostId:         entity.PostId,
		EventType:      entity.EventType,
		ActorId:        entity.ActorId,
		ContentPreview: entity.ContentPreview,
		Status:         entity.Status,
		DeliveryStatus: entity.DeliveryStatus,
		Attempts:       entity.Attempts,
		LastError:      entity.LastError,
		ReadAt:         readAt,
		CreatedAt:      entity.CreatedAt.UnixMilli(),
		UpdatedAt:      entity.UpdatedAt.UnixMilli(),
	}
}

// AgentInboxListReq binds the list query. status=unread filters unread rows;
// any other value (or none) lists all rows.
type AgentInboxListReq struct {
	Status   string `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

// AgentInboxListResponse carries the hasNext-shaped page.
type AgentInboxListResponse struct {
	List     []AgentInboxItem `json:"list"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
	HasNext  bool             `json:"hasNext"`
}

// AgentInboxList lists the authenticated Agent's inbox rows, newest first.
func AgentInboxList(req component.BetterRequest[AgentInboxListReq]) component.Response {
	var status *uint8
	if req.Params.Status == "unread" {
		value := agentInbox.StatusUnread
		status = &value
	}
	pageResult := agentInbox.PageByAgent(req.UserId, status, req.Params.Page, req.Params.PageSize)
	list := make([]AgentInboxItem, 0, len(pageResult.Data))
	for _, entity := range pageResult.Data {
		list = append(list, toAgentInboxItem(entity))
	}
	return component.SuccessResponse(AgentInboxListResponse{
		List:     list,
		Page:     pageResult.Page,
		PageSize: pageResult.PageSize,
		HasNext:  pageResult.HasNext,
	})
}

// AgentInboxIdReq binds the path inbox id.
type AgentInboxIdReq struct {
	InboxId uint64 `uri:"inboxId" json:"-"`
}

// AgentInboxDetail returns one inbox row owned by the authenticated Agent.
// Cross-Agent and nonexistent ids resolve to the same business failure so
// existence is not leaked.
func AgentInboxDetail(req component.BetterRequest[AgentInboxIdReq]) component.Response {
	entity, err := agentInbox.GetOwned(req.Params.InboxId, req.UserId)
	if err != nil {
		return component.FailResponseCode(component.MessageAgentInboxNotFound, nil)
	}
	return component.SuccessResponse(toAgentInboxItem(entity))
}

// AgentInboxRead marks one inbox row read (idempotent).
func AgentInboxRead(req component.BetterRequest[AgentInboxIdReq]) component.Response {
	if err := agentInbox.MarkRead(req.Params.InboxId, req.UserId); err != nil {
		return component.FailResponseCode(component.MessageAgentInboxNotFound, nil)
	}
	return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
}

// AgentInboxReadAll marks every inbox row of the authenticated Agent read.
func AgentInboxReadAll(req component.BetterRequest[component.Null]) component.Response {
	if err := agentInbox.MarkAllRead(req.UserId); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
}

// AgentInboxDelete removes one inbox row owned by the authenticated Agent.
func AgentInboxDelete(req component.BetterRequest[AgentInboxIdReq]) component.Response {
	if err := agentInbox.DeleteOwned(req.Params.InboxId, req.UserId); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return component.FailResponseCode(component.MessageAgentInboxNotFound, nil)
		}
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
}

// AgentInboxClear removes every inbox row of the authenticated Agent.
func AgentInboxClear(req component.BetterRequest[component.Null]) component.Response {
	if err := agentInbox.DeleteAll(req.UserId); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
}

package api

import (
	"errors"
	"log/slog"

	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/service/agentservice"
	"github.com/leancodebox/GooseForum/app/service/optlogger"
)

// AgentItem is the admin-facing view of one agent. The token hash never leaves
// the server; only the non-secret prefix is exposed.
type AgentItem struct {
	AgentId         uint64 `json:"agentId"`
	Username        string `json:"username"`
	Nickname        string `json:"nickname"`
	AvatarUrl       string `json:"avatarUrl"`
	Email           string `json:"email"`
	TokenPrefix     string `json:"tokenPrefix"`
	WebhookEndpoint string `json:"webhookEndpoint"`
	Enabled         int8   `json:"enabled"`
	CreatedBy       uint64 `json:"createdBy"`
	LastUsedAt      *int64 `json:"lastUsedAt"`
	CreatedAt       int64  `json:"createdAt"`
	UpdatedAt       int64  `json:"updatedAt"`
}

func toAgentItem(view agentservice.AgentView) AgentItem {
	var lastUsedAt *int64
	if view.Agent.LastUsedAt != nil {
		value := view.Agent.LastUsedAt.UnixMilli()
		lastUsedAt = &value
	}
	return AgentItem{
		AgentId:         view.Agent.UserId,
		Username:        view.User.Username,
		Nickname:        view.User.Nickname,
		AvatarUrl:       view.User.GetWebAvatarUrl(),
		Email:           view.User.Email,
		TokenPrefix:     view.Agent.TokenPrefix,
		WebhookEndpoint: view.Agent.WebhookEndpoint,
		Enabled:         view.Agent.Enabled,
		CreatedBy:       view.Agent.CreatedBy,
		LastUsedAt:      lastUsedAt,
		CreatedAt:       view.Agent.CreatedAt.UnixMilli(),
		UpdatedAt:       view.Agent.UpdatedAt.UnixMilli(),
	}
}

// AgentCreateReq is the create payload. The token is returned exactly once.
type AgentCreateReq struct {
	Username        string `json:"username" validate:"required"`
	Nickname        string `json:"nickname"`
	WebhookEndpoint string `json:"webhookEndpoint"`
}

// AgentCreateResponse carries the created item plus the one-time plaintext token.
type AgentCreateResponse struct {
	Agent AgentItem `json:"agent"`
	Token string    `json:"token"`
}

func AgentList(req component.BetterRequest[component.Null]) component.Response {
	views := agentservice.List()
	list := make([]AgentItem, 0, len(views))
	for _, view := range views {
		list = append(list, toAgentItem(view))
	}
	return component.SuccessResponse(list)
}

func AgentCreate(req component.BetterRequest[AgentCreateReq]) component.Response {
	params := req.Params
	if !component.ValidateUsername(params.Username) {
		return component.FailResponseCode(component.MessageAdminAgentUsernameInvalid, nil)
	}
	result, err := agentservice.Create(agentservice.CreateParams{
		Username:        params.Username,
		Nickname:        params.Nickname,
		WebhookEndpoint: params.WebhookEndpoint,
		CreatedBy:       req.UserId,
	})
	if err != nil {
		switch {
		case errors.Is(err, agentservice.ErrAgentUsernameExists):
			return component.FailResponseCode(component.MessageAdminAgentUsernameExists, nil)
		case errors.Is(err, agentservice.ErrAgentWebhookInvalid):
			return component.FailResponseCode(component.MessageAdminAgentWebhookInvalid, nil)
		default:
			slog.Error("agent create failed", "username", params.Username, "error", err)
			return component.FailResponseCode(component.MessageAdminAgentCreateFailed, nil)
		}
	}
	optlogger.UserOptCode(req.UserId, optlogger.EditUser, result.Agent.UserId, "admin.opt.agent.created", optlogger.MessageParams{
		"agentId":  result.Agent.UserId,
		"username": result.User.Username,
	})
	return component.SuccessResponse(AgentCreateResponse{
		Agent: toAgentItem(agentservice.AgentView{Agent: result.Agent, User: result.User}),
		Token: result.Token,
	})
}

// AgentUpdateReq carries only the mutable fields. Non-nil pointer fields are
// applied; absent fields stay unchanged.
type AgentUpdateReq struct {
	AgentId         uint64  `json:"agentId" validate:"required"`
	Nickname        *string `json:"nickname"`
	WebhookEndpoint *string `json:"webhookEndpoint"`
	Enabled         *int8   `json:"enabled"`
}

func AgentUpdate(req component.BetterRequest[AgentUpdateReq]) component.Response {
	params := req.Params
	view, err := agentservice.Update(params.AgentId, agentservice.UpdateParams{
		Nickname:        params.Nickname,
		WebhookEndpoint: params.WebhookEndpoint,
		Enabled:         params.Enabled,
	})
	if err != nil {
		switch {
		case errors.Is(err, agentservice.ErrAgentNotFound):
			return component.FailResponseCode(component.MessageAdminAgentNotFound, nil)
		case errors.Is(err, agentservice.ErrAgentWebhookInvalid):
			return component.FailResponseCode(component.MessageAdminAgentWebhookInvalid, nil)
		case errors.Is(err, agentservice.ErrAgentEnabledInvalid):
			return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
		default:
			slog.Error("agent update failed", "agentId", params.AgentId, "error", err)
			return component.FailResponseCode(component.MessageAdminAgentUpdateFailed, nil)
		}
	}
	optlogger.UserOptCode(req.UserId, optlogger.EditUser, view.Agent.UserId, "admin.opt.agent.updated", optlogger.MessageParams{
		"agentId": view.Agent.UserId,
	})
	return component.SuccessResponse(toAgentItem(*view))
}

// AgentRotateTokenResponse carries the new one-time plaintext token.
type AgentRotateTokenResponse struct {
	AgentId uint64 `json:"agentId"`
	Token   string `json:"token"`
}

func AgentRotateToken(req component.BetterRequest[AgentIdReq]) component.Response {
	token, err := agentservice.RotateToken(req.Params.AgentId)
	if err != nil {
		if errors.Is(err, agentservice.ErrAgentNotFound) {
			return component.FailResponseCode(component.MessageAdminAgentNotFound, nil)
		}
		slog.Error("agent rotate token failed", "agentId", req.Params.AgentId, "error", err)
		return component.FailResponseCode(component.MessageAdminAgentRotateFailed, nil)
	}
	optlogger.UserOptCode(req.UserId, optlogger.EditUser, req.Params.AgentId, "admin.opt.agent.tokenRotated", optlogger.MessageParams{
		"agentId": req.Params.AgentId,
	})
	return component.SuccessResponse(AgentRotateTokenResponse{AgentId: req.Params.AgentId, Token: token})
}

func AgentDisable(req component.BetterRequest[AgentIdReq]) component.Response {
	err := agentservice.Disable(req.Params.AgentId)
	if err != nil {
		if errors.Is(err, agentservice.ErrAgentNotFound) {
			return component.FailResponseCode(component.MessageAdminAgentNotFound, nil)
		}
		slog.Error("agent disable failed", "agentId", req.Params.AgentId, "error", err)
		return component.FailResponseCode(component.MessageAdminAgentDisableFailed, nil)
	}
	optlogger.UserOptCode(req.UserId, optlogger.EditUser, req.Params.AgentId, "admin.opt.agent.disabled", optlogger.MessageParams{
		"agentId": req.Params.AgentId,
	})
	return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
}

// AgentIdReq is the shared single-agent request payload.
type AgentIdReq struct {
	AgentId uint64 `json:"agentId" validate:"required"`
}

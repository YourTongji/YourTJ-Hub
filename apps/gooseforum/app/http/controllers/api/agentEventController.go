package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentWebhook"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agenteventservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentwebhookservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/optlogger"
)

type AgentEventsReq struct {
	After string `form:"after"`
	Limit int    `form:"limit"`
}
type AgentEventReq struct {
	EventID string `uri:"eventId" validate:"required,max=80"`
}
type AgentAckEventsReq struct {
	EventIDs []string `json:"eventIds" validate:"required,min=1,max=100,dive,required,max=80"`
}

func agentEventFailure(err error) component.Response {
	var e *agenteventservice.Error
	if errors.As(err, &e) {
		params := component.MessageParams{}
		if e.ReplayFloor != "" {
			params["replayFloor"] = e.ReplayFloor
		}
		return component.FailResponseCode(component.MessageCode("agent.events."+e.Code), params)
	}
	slog.Error("agent event operation failed", "error", err)
	return component.FailResponseCode(component.MessageOperationFailed, nil)
}
func AgentEvents(req component.BetterRequest[AgentEventsReq]) component.Response {
	if req.Params.Limit < 0 || req.Params.Limit > 100 || len(req.Params.After) > 2048 {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	page, err := agenteventservice.ListWithCredential(req.UserId, req.Params.After, req.Params.Limit, agentCredentialHash(req))
	if err != nil {
		return agentEventFailure(err)
	}
	return component.SuccessResponse(page)
}
func AgentEvent(req component.BetterRequest[AgentEventReq]) component.Response {
	if len(req.Params.EventID) == 0 || len(req.Params.EventID) > 80 {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	e, err := agenteventservice.GetWithCredential(req.UserId, req.Params.EventID, agentCredentialHash(req))
	if err != nil {
		return agentEventFailure(err)
	}
	return component.SuccessResponse(e)
}
func AgentAckEvents(req component.BetterRequest[AgentAckEventsReq]) component.Response {
	if len(req.Params.EventIDs) == 0 || len(req.Params.EventIDs) > 100 {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	for _, id := range req.Params.EventIDs {
		if len(id) == 0 || len(id) > 80 {
			return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
		}
	}
	if err := agenteventservice.AckWithCredential(req.UserId, req.Params.EventIDs, agentCredentialHash(req)); err != nil {
		return agentEventFailure(err)
	}
	return component.SuccessResponse(true)
}

type MentionTargetsReq struct {
	Q     string `form:"q"`
	Limit int    `form:"limit"`
}
type MentionTargetItem struct {
	UserID    uint64 `json:"userId"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
	ActorType string `json:"actorType"`
}

func MentionTargets(req component.BetterRequest[MentionTargetsReq]) component.Response {
	if len([]rune(req.Params.Q)) > 64 || req.Params.Limit < 0 || req.Params.Limit > 20 {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	rows, err := users.ListMentionTargets(req.Params.Q, req.Params.Limit)
	if err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	list := make([]MentionTargetItem, 0, len(rows))
	for _, r := range rows {
		actorType := "human"
		if r.ActorType == users.ActorTypeBot {
			actorType = "bot"
		}
		u := users.EntityComplete{AvatarUrl: r.AvatarURL}
		list = append(list, MentionTargetItem{r.UserID, r.Username, r.Nickname, u.GetWebAvatarUrl(), actorType})
	}
	return component.SuccessResponse(list)
}

type AgentWebhookConfigReq struct {
	AgentID         uint64   `json:"agentId" validate:"required"`
	ConfigVersion   uint64   `json:"configVersion"`
	EventsEnabled   bool     `json:"eventsEnabled"`
	EventTypes      []string `json:"eventTypes" validate:"max=3"`
	WebhookEnabled  bool     `json:"webhookEnabled"`
	WebhookEndpoint string   `json:"webhookEndpoint" validate:"max=512"`
}
type AgentWebhookRotateReq struct {
	AgentID       uint64 `json:"agentId" validate:"required"`
	ConfigVersion uint64 `json:"configVersion"`
	Emergency     bool   `json:"emergency"`
}
type AgentWebhookPageReq struct {
	AgentID  uint64 `json:"agentId" validate:"required"`
	Page     int    `json:"page" validate:"omitempty,min=1,max=10000"`
	PageSize int    `json:"pageSize" validate:"omitempty,min=1,max=100"`
}
type AgentWebhookRedeliverReq struct {
	AgentID    uint64 `json:"agentId" validate:"required"`
	DeliveryID uint64 `json:"deliveryId" validate:"required"`
}
type AgentIntentReplayReq struct {
	AgentID  uint64 `json:"agentId" validate:"required"`
	IntentID string `json:"intentId" validate:"required,max=64"`
}

func agentWebhookFailure(err error) component.Response {
	code := ""
	switch {
	case errors.Is(err, agentwebhookservice.ErrConfigConflict):
		code = "configConflict"
	case errors.Is(err, agentwebhookservice.ErrInvalidConfig):
		code = "invalidTarget"
	case errors.Is(err, agentwebhookservice.ErrSecretRequired):
		code = "secretRequired"
	case errors.Is(err, agentwebhookservice.ErrUnavailable):
		code = "unavailable"
	case errors.Is(err, agentwebhookservice.ErrNotReplayable):
		code = "expired"
	default:
		slog.Error("agent webhook admin operation failed", "error", err)
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.FailResponseCode(component.MessageCode("agent.webhook."+code), nil)
}
func auditAgentWebhook(adminID, agentID uint64, action string) {
	optlogger.UserOptCode(adminID, optlogger.EditUser, agentID, "admin.opt.agent."+action, optlogger.MessageParams{"agentId": agentID})
}
func AgentWebhookConfigure(req component.BetterRequest[AgentWebhookConfigReq]) component.Response {
	p := req.Params
	_, err := agentwebhookservice.Configure(p.AgentID, p.ConfigVersion, agentwebhookservice.ConfigParams{EventsEnabled: p.EventsEnabled, EventTypes: p.EventTypes, WebhookEnabled: p.WebhookEnabled, WebhookEndpoint: p.WebhookEndpoint})
	if err != nil {
		return agentWebhookFailure(err)
	}
	auditAgentWebhook(req.UserId, p.AgentID, "webhookConfigured")
	view, err := agentservice.Get(p.AgentID)
	if err != nil {
		return agentWebhookFailure(err)
	}
	return component.SuccessResponse(toAgentItem(*view))
}
func AgentWebhookRotateSecret(req component.BetterRequest[AgentWebhookRotateReq]) component.Response {
	result, err := agentwebhookservice.RotateSecret(req.Params.AgentID, req.Params.ConfigVersion, req.Params.Emergency)
	if err != nil {
		return agentWebhookFailure(err)
	}
	if req.GinContext != nil {
		req.GinContext.Header("Cache-Control", "no-store")
	}
	auditAgentWebhook(req.UserId, req.Params.AgentID, "webhookSecretRotated")
	return component.SuccessResponse(result)
}
func AgentWebhookTest(req component.BetterRequest[AgentIdReq]) component.Response {
	result, err := agentwebhookservice.Test(req.Params.AgentId, req.UserId)
	if err != nil {
		return agentWebhookFailure(err)
	}
	auditAgentWebhook(req.UserId, req.Params.AgentId, "webhookTestQueued")
	return component.SuccessResponse(result)
}
func AgentWebhookDeliveries(req component.BetterRequest[AgentWebhookPageReq]) component.Response {
	page, err := agentwebhookservice.ListDeliveries(req.Params.AgentID, req.Params.Page, req.Params.PageSize)
	if err != nil {
		return agentWebhookFailure(err)
	}
	type item struct {
		agentWebhook.Delivery
		Attempts []agentWebhook.Attempt `json:"attempts"`
	}
	list := make([]item, 0, len(page.Items))
	for _, d := range page.Items {
		attempts, err := agentwebhookservice.GetAttempts(req.Params.AgentID, d.ID)
		if err != nil {
			return agentWebhookFailure(err)
		}
		list = append(list, item{d, attempts})
	}
	return component.SuccessResponse(map[string]any{"list": list, "total": page.Total, "page": page.Page, "pageSize": page.Size})
}
func AgentWebhookRedeliver(req component.BetterRequest[AgentWebhookRedeliverReq]) component.Response {
	if err := agentwebhookservice.Redeliver(req.Params.AgentID, req.Params.DeliveryID, req.UserId); err != nil {
		return agentWebhookFailure(err)
	}
	auditAgentWebhook(req.UserId, req.Params.AgentID, "webhookRedelivered")
	return component.SuccessResponse(true)
}
func AgentInteractionIntents(req component.BetterRequest[AgentWebhookPageReq]) component.Response {
	// Service owns recipient filtering and paging; the HTTP layer never queries
	// task or event tables to reconstruct another domain's recovery state.
	page, err := agenteventservice.ListAgentIntents(req.Params.AgentID, req.Params.Page, req.Params.PageSize)
	if err != nil {
		return agentEventFailure(err)
	}
	return component.SuccessResponse(page)
}
func AgentInteractionReplay(req component.BetterRequest[AgentIntentReplayReq]) component.Response {
	if err := agenteventservice.ReplayAgentIntent(req.Params.AgentID, req.Params.IntentID); err != nil {
		return agentEventFailure(err)
	}
	auditAgentWebhook(req.UserId, req.Params.AgentID, "interactionReplayed")
	return component.SuccessResponse(true)
}

// Credential errors deliberately preserve the canonical Agent 401 envelope.
func agentWriteAuthFailure() component.Response {
	return component.BuildResponse(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
}

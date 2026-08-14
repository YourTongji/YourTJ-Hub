package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/optlogger"
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
		case errors.Is(err, agentservice.ErrAgentNicknameInvalid):
			return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
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
		case errors.Is(err, agentservice.ErrAgentNicknameInvalid):
			return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
		case errors.Is(err, agentservice.ErrAgentWebhookInvalid):
			return component.FailResponseCode(component.MessageAdminAgentWebhookInvalid, nil)
		case errors.Is(err, agentservice.ErrAgentEnabledInvalid):
			return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
		case errors.Is(err, agentservice.ErrAgentNeedsRotate):
			return component.FailResponseCode(component.MessageAdminAgentNeedsRotate, nil)
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
		switch {
		case errors.Is(err, agentservice.ErrAgentNotFound):
			return component.FailResponseCode(component.MessageAdminAgentNotFound, nil)
		case errors.Is(err, agentservice.ErrAgentRotateConflict):
			return component.FailResponseCode(component.MessageAdminAgentRotateConflict, nil)
		default:
			slog.Error("agent rotate token failed", "agentId", req.Params.AgentId, "error", err)
			return component.FailResponseCode(component.MessageAdminAgentRotateFailed, nil)
		}
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

// AgentMeResponse is the authenticated Agent's own view. Only the non-secret
// token prefix is exposed; the token and its hash never leave the server.
type AgentMeResponse struct {
	AgentId     uint64 `json:"agentId"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname"`
	AvatarUrl   string `json:"avatarUrl"`
	TokenPrefix string `json:"tokenPrefix"`
	Enabled     int8   `json:"enabled"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
}

// AgentMe returns the Agent's own profile. The bearer middleware already
// resolved the credential; a miss here means the row vanished in between,
// which resolves to the same 401 envelope as any other failed resolution.
func AgentMe(req component.BetterRequest[component.Null]) component.Response {
	view, err := agentservice.Get(req.UserId)
	if err != nil {
		return component.BuildResponse(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
	}
	return component.SuccessResponse(AgentMeResponse{
		AgentId:     view.Agent.UserId,
		Username:    view.User.Username,
		Nickname:    view.User.Nickname,
		AvatarUrl:   view.User.GetWebAvatarUrl(),
		TokenPrefix: view.Agent.TokenPrefix,
		Enabled:     view.Agent.Enabled,
		CreatedAt:   view.Agent.CreatedAt.UnixMilli(),
		UpdatedAt:   view.Agent.UpdatedAt.UnixMilli(),
	})
}

type AgentTopicListReq struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"pageSize"`
	Sort       string `form:"sort"`
	CategoryId uint64 `form:"categoryId"`
}

// AgentTopicItem is the published-topic view for Agents.
type AgentTopicItem struct {
	Id            uint64   `json:"id"`
	Title         string   `json:"title"`
	Excerpt       string   `json:"excerpt"`
	CategoryIds   []uint64 `json:"categoryIds"`
	UserId        uint64   `json:"userId"`
	Status        int8     `json:"status"`
	ProcessStatus int8     `json:"processStatus"`
	ReplyCount    uint64   `json:"replyCount"`
	ViewCount     uint64   `json:"viewCount"`
	PostCount     uint64   `json:"postCount"`
	LastPostedAt  *int64   `json:"lastPostedAt,omitempty"`
	CreatedAt     int64    `json:"createdAt"`
	UpdatedAt     int64    `json:"updatedAt"`
}

type AgentTopicListResponse struct {
	List     []AgentTopicItem `json:"list"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
	HasNext  bool             `json:"hasNext"`
}

// AgentTopicList lists published (status=1, process_status=0) topics with the
// same pagination, sort, and category filter as the forum topic page.
func AgentTopicList(req component.BetterRequest[AgentTopicListReq]) component.Response {
	pageResult := topics.Page(topics.PageQuery{
		Page:         req.Params.Page,
		PageSize:     req.Params.PageSize,
		FilterStatus: true,
		CategoryId:   req.Params.CategoryId,
		Sort:         req.Params.Sort,
		TopicType:    topics.TopicTypePtr(topics.TopicTypeForum),
	})
	list := make([]AgentTopicItem, 0, len(pageResult.Data))
	for _, entity := range pageResult.Data {
		list = append(list, toAgentTopicItem(entity))
	}
	return component.SuccessResponse(AgentTopicListResponse{
		List:     list,
		Page:     pageResult.Page,
		PageSize: pageResult.PageSize,
		HasNext:  pageResult.HasNext,
	})
}

func toAgentTopicItem(entity topics.Entity) AgentTopicItem {
	var lastPostedAt *int64
	if entity.LastPostedAt != nil {
		value := entity.LastPostedAt.UnixMilli()
		lastPostedAt = &value
	}
	return AgentTopicItem{
		Id:            entity.Id,
		Title:         entity.Title,
		Excerpt:       entity.Excerpt,
		CategoryIds:   entity.CategoryIds,
		UserId:        entity.UserId,
		Status:        entity.Status,
		ProcessStatus: entity.ProcessStatus,
		ReplyCount:    entity.ReplyCount,
		ViewCount:     entity.ViewCount,
		PostCount:     entity.PostCount,
		LastPostedAt:  lastPostedAt,
		CreatedAt:     entity.CreatedAt.UnixMilli(),
		UpdatedAt:     entity.UpdatedAt.UnixMilli(),
	}
}

// AgentWriteTopicReq is the Agent topic payload. Agent DTOs deliberately omit
// the website/captcha fields of the human endpoint.
type AgentWriteTopicReq struct {
	Title      string   `json:"title" validate:"required"`
	Content    string   `json:"content" validate:"required"`
	CategoryId []uint64 `json:"categoryId" validate:"min=1,max=3"`
}

// AgentWriteTopic creates a published topic (topicStatus=1) owned by the
// authenticated Agent. The shared write core skips browser-only gates
// (honeypot, captcha, new-user cooldown) while keeping every other rule.
func AgentWriteTopic(req component.BetterRequest[AgentWriteTopicReq]) component.Response {
	return writeTopic(component.BetterRequest[WriteTopicReq]{
		Params: WriteTopicReq{
			Title:       req.Params.Title,
			Content:     req.Params.Content,
			CategoryId:  req.Params.CategoryId,
			TopicStatus: 1,
		},
		UserId:     req.UserId,
		GinContext: req.GinContext,
	}, true)
}

// AgentPostListReq binds the path topicId plus the same window query
// parameters as the forum PostWindow endpoint.
type AgentPostListReq struct {
	TopicId      uint64 `uri:"topicId" json:"-"`
	AnchorPostId uint64 `form:"anchorPostId"`
	AnchorPostNo uint64 `form:"anchorPostNo"`
	BeforePostNo uint64 `form:"beforePostNo"`
	AfterPostNo  uint64 `form:"afterPostNo"`
	Limit        int    `form:"limit"`
}

// AgentPostList reuses the forum PostWindow behavior for Agent readers.
func AgentPostList(req component.BetterRequest[AgentPostListReq]) component.Response {
	return forum.PostWindow(component.BetterRequest[forum.PostWindowReq]{
		Params: forum.PostWindowReq{
			TopicID:      req.Params.TopicId,
			AnchorPostID: req.Params.AnchorPostId,
			AnchorPostNo: req.Params.AnchorPostNo,
			BeforePostNo: req.Params.BeforePostNo,
			AfterPostNo:  req.Params.AfterPostNo,
			Limit:        req.Params.Limit,
		},
		UserId:     req.UserId,
		GinContext: req.GinContext,
	})
}

// AgentCreatePostReq binds the path topicId and the reply payload. The topic
// id in the path is authoritative.
type AgentCreatePostReq struct {
	TopicId       uint64 `uri:"topicId" json:"-"`
	Content       string `json:"content"`
	ReplyToPostId uint64 `json:"replyToPostId"`
}

// AgentCreatePost appends a post to the topic from the path. The shared write
// core skips browser-only gates while keeping every other rule and side
// effect.
func AgentCreatePost(req component.BetterRequest[AgentCreatePostReq]) component.Response {
	return createPost(component.BetterRequest[CreatePostReq]{
		Params: CreatePostReq{
			TopicId:       req.Params.TopicId,
			Content:       req.Params.Content,
			ReplyToPostId: req.Params.ReplyToPostId,
		},
		UserId:     req.UserId,
		GinContext: req.GinContext,
	}, true)
}

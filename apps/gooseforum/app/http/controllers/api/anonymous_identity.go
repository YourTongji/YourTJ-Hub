package api

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/anonymousidentityservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
)

type AnonymousBatchReq struct {
	Day        string `json:"day" validate:"required"`
	RequestKey string `json:"requestKey" validate:"required,min=8,max=128"`
}
type AnonymousConfirmReq struct {
	BatchID string `json:"batchId" validate:"required"`
	Index   int    `json:"index" validate:"min=0,max=9"`
}
type AnonymousDisabledReq struct {
	Disabled bool `json:"disabled"`
}
type AnonymousRevealReq struct {
	PublicUID string `json:"publicUid" validate:"required"`
	Reason    string `json:"reason" validate:"required,max=512"`
}

func anonymousResponse(reqContext component.BetterRequest[component.Null], value any, err error) component.Response {
	reqContext.GinContext.Header("Cache-Control", "private, no-store")
	if err != nil {
		return component.FailResponseCode(component.MessageCode(anonymousidentityservice.ErrorCode(err)), nil)
	}
	return component.SuccessResponse(value)
}
func AnonymousState(req component.BetterRequest[component.Null]) component.Response {
	value, err := anonymousidentityservice.Default(req.GinContext.Request.Context()).State(req.UserId)
	return anonymousResponse(req, value, err)
}
func AnonymousGenerate(req component.BetterRequest[AnonymousBatchReq]) component.Response {
	req.GinContext.Header("Cache-Control", "private, no-store")
	value, err := anonymousidentityservice.Default(req.GinContext.Request.Context()).Generate(req.UserId, req.Params.Day, req.Params.RequestKey)
	if err != nil {
		return component.FailResponseCode(component.MessageCode(anonymousidentityservice.ErrorCode(err)), nil)
	}
	return component.SuccessResponse(value)
}
func AnonymousConfirm(req component.BetterRequest[AnonymousConfirmReq]) component.Response {
	req.GinContext.Header("Cache-Control", "private, no-store")
	value, err := anonymousidentityservice.Default(req.GinContext.Request.Context()).Confirm(req.UserId, req.Params.BatchID, req.Params.Index)
	if err != nil {
		return component.FailResponseCode(component.MessageCode(anonymousidentityservice.ErrorCode(err)), nil)
	}
	return component.SuccessResponse(value)
}
func AnonymousDisable(req component.BetterRequest[AnonymousDisabledReq]) component.Response {
	req.GinContext.Header("Cache-Control", "private, no-store")
	err := anonymousidentityservice.Default(req.GinContext.Request.Context()).SetDisabled(req.UserId, req.Params.Disabled)
	if err != nil {
		return component.FailResponseCode(component.MessageCode(anonymousidentityservice.ErrorCode(err)), nil)
	}
	return component.SuccessResponse(true)
}
func AnonymousReveal(req component.BetterRequest[AnonymousRevealReq]) component.Response {
	req.GinContext.Header("Cache-Control", "private, no-store")
	req.GinContext.Header("Pragma", "no-cache")
	value, err := anonymousidentityservice.Default(req.GinContext.Request.Context()).Reveal(req.UserId, req.Params.PublicUID, req.Params.Reason, req.GinContext.GetHeader("X-Request-ID"))
	if err != nil {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	return component.SuccessResponse(value)
}

type AnonymousGovernReq struct {
	PostID   uint64 `json:"postId" validate:"required"`
	Disabled bool   `json:"disabled"`
	Reason   string `json:"reason" validate:"required,max=512"`
}

func AnonymousGovern(req component.BetterRequest[AnonymousGovernReq]) component.Response {
	req.GinContext.Header("Cache-Control", "private, no-store")
	post := posts.GetMapByIdsUnscoped([]uint64{req.Params.PostID})[req.Params.PostID]
	if post == nil || post.PersonaUID == "" {
		return component.FailResponseCode(component.MessagePostNotFound, nil)
	}
	topic := topics.UnscopedGet(post.TopicId)
	if topic.Id == 0 || !moderationservice.CanModerateAnyCategory(req.UserId, topic.CategoryIds) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	if err := anonymousidentityservice.Default(req.GinContext.Request.Context()).Govern(req.UserId, post.PersonaUID, req.Params.Reason, req.Params.Disabled); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	userservice.InvalidateUserInfoCache(post.UserId)
	return component.SuccessResponse(true)
}

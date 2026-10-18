package api

import (
	"errors"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
	"gorm.io/gorm"
)

type SubmitAppealReq struct {
	Kind       string `json:"kind" validate:"required,oneof=topic post"`
	Id         uint64 `json:"id" validate:"required"`
	RevisionId uint64 `json:"revisionId" validate:"required"`
	Reason     string `json:"reason" validate:"required,max=512"`
}
type OwnerAppealsReq struct {
	Kind string `form:"kind" validate:"required,oneof=topic post"`
	Id   uint64 `form:"id" validate:"required"`
}
type ModerationAppealsReq struct {
	Page     int    `json:"page" validate:"omitempty,min=1,max=1000000"`
	PageSize int    `json:"pageSize" validate:"omitempty,min=1,max=50"`
	Status   string `json:"status" validate:"omitempty,oneof=pending approved rejected stale"`
}
type AppealActionReq struct {
	Id      uint64 `json:"id" validate:"required"`
	Approve bool   `json:"approve"`
	Reason  string `json:"reason" validate:"max=512"`
}

// submitAppealFailure tells the author what happened and what to do next.
func submitAppealFailure(err error) component.Response {
	switch {
	case errors.Is(err, publicationservice.ErrAppealReason):
		return component.FailResponseCode(component.MessageContentAppealReasonInvalid, nil)
	case errors.Is(err, publicationservice.ErrAppealFrozen):
		return component.FailResponseCode(component.MessageContentAppealFrozen, nil)
	case errors.Is(err, publicationservice.ErrAppealDuplicate), errors.Is(err, gorm.ErrDuplicatedKey):
		return component.FailResponseCode(component.MessageContentAppealDuplicate, nil)
	case errors.Is(err, publicationservice.ErrAppealPermission), errors.Is(err, gorm.ErrRecordNotFound):
		return component.FailResponseCode(component.MessageContentAppealNotFound, nil)
	case errors.Is(err, publicationservice.ErrUnavailable):
		return component.FailResponseCode(component.MessageContentAppealIneligible, nil)
	}
	return component.FailResponseCode(component.MessageOperationFailed, nil)
}

// reviewAppealFailure tells the reviewer why the decision did not apply.
func reviewAppealFailure(err error) component.Response {
	switch {
	case errors.Is(err, publicationservice.ErrAppealReason):
		return component.FailResponseCode(component.MessageModerationAppealReason, nil)
	case errors.Is(err, publicationservice.ErrAppealStale):
		return component.FailResponseCode(component.MessageModerationAppealStale, nil)
	case errors.Is(err, publicationservice.ErrAppealAuthorFrozen):
		return component.FailResponseCode(component.MessageModerationAppealAuthorFrozen, nil)
	case errors.Is(err, publicationservice.ErrAppealPermission), errors.Is(err, gorm.ErrRecordNotFound):
		return component.FailResponseCode(component.MessageModerationAppealNotFound, nil)
	case errors.Is(err, publicationservice.ErrUnavailable):
		return component.FailResponseCode(component.MessageModerationAppealProcessed, nil)
	}
	return component.FailResponseCode(component.MessageOperationFailed, nil)
}
func SubmitContentAppeal(req component.BetterRequest[SubmitAppealReq]) component.Response {
	if req.GinContext != nil {
		req.GinContext.Header("Cache-Control", "no-store")
	}
	appeal, err := publicationservice.SubmitAppeal(requestContext(req.GinContext), req.UserId, req.Params.Kind, req.Params.Id, req.Params.RevisionId, req.Params.Reason)
	if err != nil {
		return submitAppealFailure(err)
	}
	return component.SuccessResponse(appeal)
}
func ContentAppeals(req component.BetterRequest[OwnerAppealsReq]) component.Response {
	if req.GinContext != nil {
		req.GinContext.Header("Cache-Control", "no-store")
	}
	items, err := publicationservice.OwnerAppeals(requestContext(req.GinContext), req.UserId, req.Params.Kind, req.Params.Id)
	if err != nil {
		return submitAppealFailure(err)
	}
	return component.SuccessResponse(component.DataMap{"items": items})
}
func ModerationAppeals(req component.BetterRequest[ModerationAppealsReq]) component.Response {
	if !moderationservice.CanAccessModeration(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	if req.GinContext != nil {
		req.GinContext.Header("Cache-Control", "no-store")
	}
	page, size, status := req.Params.Page, req.Params.PageSize, req.Params.Status
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 20
	}
	if status == "" {
		status = "pending"
	}
	items, total, err := publicationservice.ModeratorAppeals(requestContext(req.GinContext), req.UserId, page, size, status)
	if err != nil {
		return reviewAppealFailure(err)
	}
	return component.SuccessResponse(component.DataMap{"items": items, "total": total, "page": page, "pageSize": size})
}
func ModerationAppealAction(req component.BetterRequest[AppealActionReq]) component.Response {
	if !moderationservice.CanAccessModeration(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	if req.GinContext != nil {
		req.GinContext.Header("Cache-Control", "no-store")
	}
	if err := publicationservice.ReviewAppeal(requestContext(req.GinContext), req.Params.Id, req.UserId, req.Params.Approve, req.Params.Reason); err != nil {
		return reviewAppealFailure(err)
	}
	return component.SuccessResponseCode(true, component.MessageOperationSuccess, nil)
}

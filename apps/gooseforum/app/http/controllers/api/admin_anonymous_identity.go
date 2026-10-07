package api

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/anonymousidentityservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
)

func AdminAnonymousList(req component.BetterRequest[anonymousidentityservice.AdminListQuery]) component.Response {
	value, err := anonymousidentityservice.Default(req.GinContext.Request.Context()).ListAdmin(req.UserId, req.Params)
	if err != nil {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	return component.SuccessResponse(value)
}

func AdminAnonymousGovern(req component.BetterRequest[AdminAnonymousGovernReq]) component.Response {
	owner, err := anonymousidentityservice.Default(req.GinContext.Request.Context()).GovernAdmin(req.UserId, req.Params.PublicUID, req.Params.Reason, *req.Params.Disabled)
	if err != nil {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	// Public cards cache account publishing eligibility. No private relation is
	// returned to the browser, stored in an operation record or exported.
	userservice.InvalidateUserInfoCache(owner)
	return component.SuccessResponse(true)
}

type AdminAnonymousGovernReq struct {
	PublicUID string `json:"publicUid" validate:"required,len=32"`
	Reason    string `json:"reason" validate:"required,max=512"`
	Disabled  *bool  `json:"disabled" validate:"required"`
}

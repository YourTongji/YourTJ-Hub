package api

import (
	"errors"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/decorationservice"
	"gorm.io/gorm"
)

func decorationResponse(value any, err error) component.Response {
	if err == nil {
		return component.SuccessResponse(value)
	}
	code := component.MessageOperationFailed
	switch {
	case errors.Is(err, decorationservice.ErrInvalid), errors.Is(err, decorationservice.ErrConflict):
		code = component.MessageRequestInvalidParams
	case errors.Is(err, decorationservice.ErrUnavailable):
		code = component.MessagePermissionDenied
	case errors.Is(err, gorm.ErrRecordNotFound):
		code = component.MessageUserNotFound
	}
	return component.FailResponseCode(code, nil)
}

func MyDecorations(req component.BetterRequest[component.Null]) component.Response {
	req.GinContext.Header("Cache-Control", "private, no-store")
	state, err := decorationservice.Current(req.GinContext.Request.Context(), req.UserId)
	return decorationResponse(state, err)
}

type EquipDecorationsInput struct {
	AvatarFrame   *string `json:"avatarFrame" validate:"required,max=64"`
	NicknameStyle *string `json:"nicknameStyle" validate:"required,max=64"`
}

func EquipDecorations(req component.BetterRequest[EquipDecorationsInput]) component.Response {
	req.GinContext.Header("Cache-Control", "private, no-store")
	state, err := decorationservice.Equip(req.GinContext.Request.Context(), req.UserId, decorationservice.Equipped{AvatarFrame: *req.Params.AvatarFrame, NicknameStyle: *req.Params.NicknameStyle})
	return decorationResponse(state, err)
}
func DecorationList(req component.BetterRequest[component.Null]) component.Response {
	styles, err := decorationservice.Catalog(req.GinContext.Request.Context())
	return decorationResponse(styles, err)
}
func SaveDecoration(req component.BetterRequest[decorationservice.Style]) component.Response {
	style, err := decorationservice.SaveStyle(req.GinContext.Request.Context(), req.UserId, req.Params)
	return decorationResponse(style, err)
}

type UserDecorationsQuery struct {
	UserID uint64 `form:"userId" validate:"required"`
}

func UserDecorations(req component.BetterRequest[UserDecorationsQuery]) component.Response {
	req.GinContext.Header("Cache-Control", "private, no-store")
	state, err := decorationservice.AdminCurrent(req.GinContext.Request.Context(), req.Params.UserID)
	return decorationResponse(state, err)
}
func GrantDecoration(req component.BetterRequest[decorationservice.Adjustment]) component.Response {
	state, err := decorationservice.Adjust(req.GinContext.Request.Context(), req.UserId, req.Params, false)
	return decorationResponse(state, err)
}
func RevokeDecoration(req component.BetterRequest[decorationservice.Adjustment]) component.Response {
	state, err := decorationservice.Adjust(req.GinContext.Request.Context(), req.UserId, req.Params, true)
	return decorationResponse(state, err)
}

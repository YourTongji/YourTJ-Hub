package api

import (
	"errors"
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
)

type UserBlockRequest struct {
	TargetUserID uint64 `json:"targetUserId" binding:"required"`
	Blocked      *bool  `json:"blocked" binding:"required"`
}
type UserBlocksPayload struct {
	OwnerID uint64              `json:"ownerId"`
	Blocks  []users.BlockedUser `json:"blocks"`
}

func GetUserBlocks(req component.BetterRequest[component.Null]) component.Response {
	req.GinContext.Header("Cache-Control", "private, no-store")
	blocks, err := userservice.ListBlockedUsers(req.UserId)
	if err != nil {
		response := component.FailResponseCode(component.MessageOperationFailed, nil)
		response.Code = http.StatusServiceUnavailable
		return response
	}
	return component.SuccessResponse(UserBlocksPayload{OwnerID: req.UserId, Blocks: blocks})
}
func SetUserBlock(req component.BetterRequest[UserBlockRequest]) component.Response {
	req.GinContext.Header("Cache-Control", "private, no-store")
	err := userservice.SetBlockedUser(req.UserId, req.Params.TargetUserID, *req.Params.Blocked)
	if err != nil {
		response := component.FailResponseCode(component.MessageRequestInvalidParams, nil)
		switch {
		case errors.Is(err, users.ErrBlockTarget):
			response.Code = http.StatusBadRequest
		case errors.Is(err, users.ErrBlockLimit):
			response.Code = http.StatusConflict
		default:
			response = component.FailResponseCode(component.MessageOperationFailed, nil)
			response.Code = http.StatusServiceUnavailable
		}
		return response
	}
	return component.SuccessResponse(true)
}

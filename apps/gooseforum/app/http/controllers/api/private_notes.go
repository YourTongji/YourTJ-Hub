package api

import (
	"errors"
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
)

type PrivateNoteRequest struct {
	TargetUserID uint64  `json:"targetUserId" binding:"required"`
	Note         *string `json:"note" binding:"required"`
}
type PrivateNotesPayload struct {
	OwnerID uint64              `json:"ownerId"`
	Notes   []users.PrivateNote `json:"notes"`
}

func GetPrivateNotes(req component.BetterRequest[component.Null]) component.Response {
	req.GinContext.Header("Cache-Control", "private, no-store")
	notes, err := userservice.ListPrivateNotes(req.UserId)
	if err != nil {
		response := component.FailResponseCode(component.MessageOperationFailed, nil)
		response.Code = http.StatusServiceUnavailable
		return response
	}
	return component.SuccessResponse(PrivateNotesPayload{OwnerID: req.UserId, Notes: notes})
}
func SetPrivateNote(req component.BetterRequest[PrivateNoteRequest]) component.Response {
	req.GinContext.Header("Cache-Control", "private, no-store")
	err := userservice.SetPrivateNote(req.UserId, req.Params.TargetUserID, *req.Params.Note)
	if err != nil {
		response := component.FailResponseCode(component.MessageRequestInvalidParams, nil)
		switch {
		case errors.Is(err, userservice.ErrInvalidPrivateNote), errors.Is(err, users.ErrPrivateNoteTarget):
			response.Code = http.StatusBadRequest
		case errors.Is(err, users.ErrPrivateNoteLimit):
			response.Code = http.StatusConflict
		default:
			response = component.FailResponseCode(component.MessageOperationFailed, nil)
			response.Code = http.StatusServiceUnavailable
		}
		return response
	}
	return component.SuccessResponse(true)
}

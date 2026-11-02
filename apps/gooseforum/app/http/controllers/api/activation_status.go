package api

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/emailactivationservice"
	"net/http"
	"time"
)

func GetActivationStatus(req component.BetterRequest[component.Null]) component.Response {
	req.GinContext.Header("Cache-Control", "private, no-store")
	user, err := req.GetUser()
	if err != nil || user.Id == 0 {
		return component.FailResponseCode(component.MessageUserFetchFailed, nil)
	}
	status, err := emailactivationservice.GetStatus(user, time.Now())
	if err != nil {
		response := component.FailResponseCode(component.MessageOperationFailed, nil)
		response.Code = http.StatusServiceUnavailable
		return response
	}
	return component.SuccessResponse(status)
}

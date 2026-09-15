package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
)

func GetSearchMaintenance(req component.BetterRequest[component.Null]) component.Response {
	ctx := context.Background()
	if req.GinContext != nil {
		ctx = req.GinContext.Request.Context()
	}
	status, err := searchservice.GetIndexMaintenanceStatus(ctx)
	if err != nil {
		slog.Error("admin search status failed", "error", err)
		return component.BuildResponse(http.StatusServiceUnavailable, component.FailDataCode(component.MessageOperationFailed, nil))
	}
	return component.SuccessResponse(status)
}

func CreateSearchMaintenance(req component.BetterRequest[searchservice.MaintenanceRequest]) component.Response {
	ctx := context.Background()
	if req.GinContext != nil {
		ctx = req.GinContext.Request.Context()
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	submission, err := searchservice.CreateIndexMaintenance(ctx, req.Params, req.UserId)
	if err != nil {
		slog.Error("admin search maintenance enqueue failed", "error", err)
		return component.BuildResponse(http.StatusServiceUnavailable, component.FailDataCode(component.MessageOperationFailed, nil))
	}
	return component.SuccessResponse(submission)
}

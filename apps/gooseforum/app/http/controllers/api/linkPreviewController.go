package api

import (
	"log/slog"
	"sync"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/linkpreviewservice"
)

type ResolveLinkPreviewsRequest struct {
	URLs []string `json:"urls" validate:"required,min=1,max=5,dive,min=1,max=2048"`
}

func ResolveLinkPreviews(req component.BetterRequest[ResolveLinkPreviewsRequest]) component.Response {
	slog.Info("link_preview_resolve_request", "urlCount", len(req.Params.URLs), "authenticated", req.UserId != 0)
	previews := make([]linkpreviewservice.Preview, len(req.Params.URLs))
	resolver := linkpreviewservice.Default()
	ctx := req.GinContext.Request.Context()

	var wg sync.WaitGroup
	for index, rawURL := range req.Params.URLs {
		wg.Go(func() {
			previews[index] = resolver.Resolve(ctx, req.UserId, rawURL)
		})
	}
	wg.Wait()

	return component.SuccessResponse(previews)
}

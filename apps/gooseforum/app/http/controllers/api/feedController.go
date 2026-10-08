package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/feedservice"
	"github.com/gin-gonic/gin"
)

func FeedEvents(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32<<10)
	var req struct {
		Patches     []feedservice.Patch     `json:"patches"`
		SeenPatches []feedservice.SeenPatch `json:"seenPatches"`
	}
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestInvalidParams, nil))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 200*time.Millisecond)
	defer cancel()
	confirmed, err := feedservice.CaptureEvents(ctx, component.LoginUserId(c), req.Patches, req.SeenPatches)
	if errors.Is(err, feedservice.ErrRateLimited) || errors.Is(err, feedservice.ErrQueueFull) {
		c.Header("Retry-After", "5")
		c.JSON(http.StatusTooManyRequests, component.FailDataCode(component.MessageOperationFailed, nil))
		return
	}
	if err != nil {
		if !errors.Is(err, feedservice.ErrInvalidTrace) {
			c.JSON(http.StatusServiceUnavailable, component.FailDataCode(component.MessageOperationFailed, nil))
			return
		}
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestInvalidParams, nil))
		return
	}
	if len(req.SeenPatches) > 0 {
		c.JSON(http.StatusOK, component.SuccessData(gin.H{"seenConfirmed": confirmed}))
		return
	}
	c.JSON(http.StatusOK, component.SuccessData(true))
}

func FeedSummary(req component.BetterRequest[component.Null]) component.Response {
	summary, err := feedservice.GetSummary(betterRequestContext(req))
	if err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(summary)
}

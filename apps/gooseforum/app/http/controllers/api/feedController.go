package api

import (
	"errors"
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/feedservice"
	"github.com/gin-gonic/gin"
)

func FeedEvents(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32<<10)
	var req struct {
		Patches []feedservice.Patch `json:"patches"`
	}
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestInvalidParams, nil))
		return
	}
	err := feedservice.CapturePatches(component.LoginUserId(c), req.Patches)
	if errors.Is(err, feedservice.ErrRateLimited) || errors.Is(err, feedservice.ErrQueueFull) {
		c.Header("Retry-After", "5")
		c.JSON(http.StatusTooManyRequests, component.FailDataCode(component.MessageOperationFailed, nil))
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestInvalidParams, nil))
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

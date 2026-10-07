package api

import (
	"context"
	"net/http"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/gin-gonic/gin"
)

// finishAIModeration is used only by shadow observations after a normal write.
func finishAIModeration(check *moderationservice.AIModeration, subjectID uint64) {
	check.Finish(subjectID)
}

// applyAIModeration routes moderated submissions to the durable pending path.
// The returned check is used only for shadow observations after a normal write.
func applyAIModeration(ctx context.Context, input moderationservice.AIContentInput) (*moderationservice.AIModeration, bool) {
	if moderationservice.NeedsSubmissionReview(ctx, input) {
		return nil, true
	}
	return moderationservice.PrepareAIModeration(ctx, input), false
}

// extendWriteDeadline covers explicit administrator provider connection tests.
func extendWriteDeadline(c *gin.Context, budget time.Duration) {
	if c == nil || c.Writer == nil || budget <= 0 {
		return
	}
	_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(budget + 10*time.Second))
}

// publishSuccess acknowledges a durable save. Clients retain the checking code
// to render an owner-only pending entry, while the toast simply says sent.
func publishSuccess(data any, pendingReview bool) component.Response {
	if pendingReview {
		return component.SuccessResponseCode(data, component.MessageContentModerationChecking, nil)
	}
	return component.SuccessResponse(data)
}

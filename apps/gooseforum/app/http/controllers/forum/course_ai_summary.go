package forum

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
)

// GetCourseSummaryReq 课程 AI 总结请求参数（URI courseId + 可选 refresh）。
type GetCourseSummaryReq struct {
	CourseId uint64 `uri:"courseId" validate:"required"`
	Refresh  bool   `form:"refresh"`
}

// GetCourseSummary 课程 AI 总结（公开只读，B7 issue #181）：
// 返回 status=cached/generated/insufficient_data/disabled（200）或 404/429/500。
// 429 带 Retry-After header（前端据此展示限流倒计时）。
func GetCourseSummary(req component.BetterRequest[GetCourseSummaryReq]) component.Response {
	if req.Params.CourseId == 0 {
		return component.BuildResponse(http.StatusBadRequest,
			component.FailDataCode(component.MessageRequestInvalidParams, nil))
	}
	result, err := courseservice.GetAiSummary(req.Params.CourseId, req.Params.Refresh)
	if err != nil {
		switch {
		case errors.Is(err, courseservice.ErrAiSummaryCourseNotFound):
			return component.BuildResponse(http.StatusNotFound,
				component.FailDataCode(component.MessagePageNotFound, nil))
		default:
			var rateErr *courseservice.AiSummaryRateLimitError
			if errors.As(err, &rateErr) {
				seconds := int(rateErr.RetryAfter.Seconds())
				if seconds < 1 {
					seconds = 1
				}
				setRetryAfterHeader(req, seconds)
				return component.BuildResponse(http.StatusTooManyRequests,
					component.FailDataCode(component.MessageRateLimited,
						component.MessageParams{
							"action":            "course.summary",
							"retryAfterSeconds": seconds,
						}))
			}
			slog.Error("course_ai_summary_failed", "courseId", req.Params.CourseId, "error", err)
			return component.BuildResponse(http.StatusInternalServerError,
				component.FailDataCode(component.MessageCourseSummaryFailed, nil))
		}
	}
	return component.SuccessResponse(result)
}

// setRetryAfterHeader 回写 Retry-After（前端 api.ts 读取该 header 计算限流秒数）。
func setRetryAfterHeader(req component.BetterRequest[GetCourseSummaryReq], seconds int) {
	if req.GinContext == nil {
		return
	}
	req.GinContext.Header("Retry-After", strconv.Itoa(seconds))
}

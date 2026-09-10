package forum

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
)

// BookmarkCourseReq 课程收藏请求体（对齐 topics/bookmark 的 BookmarkTopicReq）。
type BookmarkCourseReq struct {
	CourseId uint64 `json:"courseId" validate:"required"`
	Action   int    `json:"action" validate:"min=1,max=2"` // 1 收藏，2 取消
}

// BookmarkCourse 登录用户收藏/取消课程（幂等）。课程不存在或已隐藏 → 404。
func BookmarkCourse(req component.BetterRequest[BookmarkCourseReq]) component.Response {
	if req.Params.CourseId == 0 {
		return component.BuildResponse(http.StatusBadRequest,
			component.FailDataCode(component.MessageRequestInvalidParams, nil))
	}
	_, err := courseservice.BookmarkCourse(req.UserId, req.Params.CourseId, req.Params.Action)
	if err != nil {
		if errors.Is(err, courseservice.ErrCourseNotFound) {
			return component.BuildResponse(http.StatusNotFound,
				component.FailDataCode(component.MessageCourseNotFound, nil))
		}
		slog.Error("course_bookmark_failed", "userId", req.UserId, "courseId", req.Params.CourseId, "error", err)
		return component.BuildResponse(http.StatusInternalServerError,
			component.FailDataCode(component.MessageOperationFailed, nil))
	}
	return component.SuccessResponse(true)
}

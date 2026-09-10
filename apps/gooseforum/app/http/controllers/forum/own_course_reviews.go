package forum

import (
	"net/http"
	"strconv"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
)

type OwnCourseReviewsReq struct {
	Cursor   string `form:"cursor"`
	PageSize *int   `form:"pageSize"`
}

func OwnCourseReviews(req component.BetterRequest[OwnCourseReviewsReq]) component.Response {
	var before uint64
	var err error
	if req.Params.Cursor != "" {
		before, err = strconv.ParseUint(req.Params.Cursor, 10, 64)
	}
	pageSize := courseservice.DefaultReviewPageSize
	if req.Params.PageSize != nil {
		pageSize = *req.Params.PageSize
	}
	if err != nil || pageSize < 1 || pageSize > courseservice.MaxReviewPageSize {
		return component.BuildResponse(http.StatusBadRequest, component.FailDataCode(component.MessageRequestInvalidParams, nil))
	}
	page, err := courseservice.ListOwnReviews(req.UserId, before, pageSize)
	if err != nil {
		return component.BuildResponse(http.StatusInternalServerError, component.FailDataCode(component.MessageReviewListFailed, nil))
	}
	return component.SuccessResponse(page)
}

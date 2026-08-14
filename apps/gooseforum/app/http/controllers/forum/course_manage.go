package forum

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/optlogger"
)

// ---- 课程管理（CourseManager/Admin） ----

// AdminCourseListReq 管理端课程检索请求。
type AdminCourseListReq struct {
	Keyword    string `json:"keyword"`
	Department string `json:"department"`
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
}

// AdminCourseList 返回管理端课程分页（含隐藏课程）。
func AdminCourseList(req component.BetterRequest[AdminCourseListReq]) component.Response {
	if !canModerateCourseReviews(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	pageData, err := courseservice.AdminCourseList(courseservice.AdminCourseQuery{
		Keyword:    req.Params.Keyword,
		Department: req.Params.Department,
		Page:       req.Params.Page,
		Size:       req.Params.PageSize,
	})
	if err != nil {
		slog.Error("course_manage_list_failed", "error", err)
		return component.BuildResponse(http.StatusInternalServerError,
			component.FailDataCode(component.MessageCourseListFailed, nil))
	}
	return component.SuccessResponse(pageData)
}

// AdminCourseCreateReq 新增课程请求体。
type AdminCourseCreateReq struct {
	PrimaryCode string   `json:"primaryCode" validate:"required"`
	Name        string   `json:"name" validate:"required"`
	Department  string   `json:"department"`
	CreditX10   int      `json:"creditX10"`
	Aliases     []string `json:"aliases"`
	Instructors []string `json:"instructors"`
}

// AdminCourseCreate 新增课程。
func AdminCourseCreate(req component.BetterRequest[AdminCourseCreateReq]) component.Response {
	if !canModerateCourseReviews(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	item, err := courseservice.CreateCourse(courseservice.CourseCreateInput{
		PrimaryCode: req.Params.PrimaryCode,
		Name:        req.Params.Name,
		Department:  req.Params.Department,
		CreditX10:   req.Params.CreditX10,
		Aliases:     req.Params.Aliases,
		Instructors: req.Params.Instructors,
	})
	if err != nil {
		return courseManageErrorResponse(err)
	}
	optlogger.UserOptCode(req.UserId, optlogger.CreateCourse, item.Id, "course.created", optlogger.MessageParams{
		"courseId":    item.Id,
		"primaryCode": item.PrimaryCode,
		"name":        item.Name,
	})
	return component.SuccessResponse(item)
}

// AdminCourseUpdateReq 编辑课程请求体。
type AdminCourseUpdateReq struct {
	CourseId    uint64    `json:"courseId" validate:"required"`
	PrimaryCode *string   `json:"primaryCode"`
	Name        *string   `json:"name"`
	Department  *string   `json:"department"`
	CreditX10   *int      `json:"creditX10"`
	Aliases     *[]string `json:"aliases"`
	Instructors *[]string `json:"instructors"`
}

// AdminCourseUpdate 编辑课程。
func AdminCourseUpdate(req component.BetterRequest[AdminCourseUpdateReq]) component.Response {
	if !canModerateCourseReviews(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	item, err := courseservice.UpdateCourse(req.Params.CourseId, courseservice.CourseUpdateInput{
		PrimaryCode: req.Params.PrimaryCode,
		Name:        req.Params.Name,
		Department:  req.Params.Department,
		CreditX10:   req.Params.CreditX10,
		Aliases:     req.Params.Aliases,
		Instructors: req.Params.Instructors,
	})
	if err != nil {
		return courseManageErrorResponse(err)
	}
	optlogger.UserOptCode(req.UserId, optlogger.UpdateCourse, item.Id, "course.updated", optlogger.MessageParams{
		"courseId":    item.Id,
		"primaryCode": item.PrimaryCode,
		"name":        item.Name,
	})
	return component.SuccessResponse(item)
}

// AdminCourseDeleteReq 删除课程请求体。
type AdminCourseDeleteReq struct {
	CourseId uint64 `json:"courseId" validate:"required"`
}

// AdminCourseDelete 级联删除课程（课程+评价+统计投影一并移除），并写审计日志。
func AdminCourseDelete(req component.BetterRequest[AdminCourseDeleteReq]) component.Response {
	if !canModerateCourseReviews(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	info, err := courseservice.DeleteCourse(req.Params.CourseId)
	if err != nil {
		return courseManageErrorResponse(err)
	}
	optlogger.UserOptCode(req.UserId, optlogger.DeleteCourse, info.Id, "course.deleted", optlogger.MessageParams{
		"courseId":      info.Id,
		"primaryCode":   info.PrimaryCode,
		"name":          info.Name,
		"offeringCount": info.OfferingCount,
		"reviewCount":   info.ReviewCount,
	})
	return component.SuccessResponse(true)
}

// ---- 评价管理（CourseManager/Admin） ----

// AdminReviewListReq 管理端评价检索请求。
type AdminReviewListReq struct {
	Keyword  string `json:"keyword"`
	Status   *int8  `json:"status"` // -1 全部；0/1/2 按状态过滤
	Cursor   uint64 `json:"cursor"`
	PageSize int    `json:"pageSize"`
}

// AdminReviewList 返回管理端评价分页（含隐藏/删除，跨课程名/课号/正文搜索）。
func AdminReviewList(req component.BetterRequest[AdminReviewListReq]) component.Response {
	if !canModerateCourseReviews(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	status := int8(-1)
	if req.Params.Status != nil {
		status = *req.Params.Status
	}
	pageData, err := courseservice.AdminReviewList(courseservice.AdminReviewQuery{
		Keyword:  req.Params.Keyword,
		Status:   status,
		Cursor:   req.Params.Cursor,
		PageSize: req.Params.PageSize,
	})
	if err != nil {
		slog.Error("course_manage_review_list_failed", "error", err)
		return component.BuildResponse(http.StatusInternalServerError,
			component.FailDataCode(component.MessageReviewListFailed, nil))
	}
	return component.SuccessResponse(pageData)
}

// AdminReviewUpdateReq 管理端编辑评价请求体。
type AdminReviewUpdateReq struct {
	ReviewId uint64  `json:"reviewId" validate:"required"`
	Rating   *int    `json:"rating"`
	Content  *string `json:"content"`
}

// AdminReviewUpdate 管理端编辑评价。
func AdminReviewUpdate(req component.BetterRequest[AdminReviewUpdateReq]) component.Response {
	if !canModerateCourseReviews(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	payload, err := courseservice.AdminUpdateReview(req.Params.ReviewId, courseservice.AdminReviewUpdateInput{
		Rating:  req.Params.Rating,
		Content: req.Params.Content,
	})
	if err != nil {
		return courseManageErrorResponse(err)
	}
	optlogger.UserOptCode(req.UserId, optlogger.UpdateReview, payload.Id, "review.updated", optlogger.MessageParams{
		"reviewId":   payload.Id,
		"offeringId": payload.OfferingId,
		"rating":     payload.Rating,
	})
	return component.SuccessResponse(payload)
}

// AdminReviewDeleteReq 管理端删除评价请求体。
type AdminReviewDeleteReq struct {
	ReviewId uint64 `json:"reviewId" validate:"required"`
}

// AdminReviewDelete 管理端硬删除评价。
func AdminReviewDelete(req component.BetterRequest[AdminReviewDeleteReq]) component.Response {
	if !canModerateCourseReviews(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	info, err := courseservice.AdminDeleteReview(req.Params.ReviewId)
	if err != nil {
		return courseManageErrorResponse(err)
	}
	optlogger.UserOptCode(req.UserId, optlogger.DeleteReview, info.Id, "review.deleted", optlogger.MessageParams{
		"reviewId":   info.Id,
		"offeringId": info.OfferingId,
		"rating":     info.Rating,
		"status":     info.Status,
	})
	return component.SuccessResponse(true)
}

// ---- 统计重建 ----

// AdminCourseStatsRebuild 入队一次全量课程统计重建（后台任务替换 rebuild-course-stats CLI）。
func AdminCourseStatsRebuild(req component.BetterRequest[component.Null]) component.Response {
	if !canModerateCourseReviews(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	if err := courseservice.EnqueueCourseStatsRebuildTask(); err != nil {
		slog.Error("course_stats_rebuild_enqueue_failed", "error", err)
		return component.BuildResponse(http.StatusInternalServerError,
			component.FailDataCode(component.MessageCourseStatsRebuildFailed, nil))
	}
	return component.SuccessResponseCode(true, component.MessageCourseStatsRebuildQueued, nil)
}

// ---- SSR 页面 ----

// CourseManagement 课程/评价管理页面（/moderation/courses）。
// 独立 CourseManager 权限，数据全部走 JSON API 异步加载。
// activeKey 与前端 AppShell.vue 侧边栏菜单 key（courseManage）保持一致，避免高亮错位。
func CourseManagement(c *gin.Context) {
	if !canModerateCourseReviews(component.LoginUserId(c)) {
		renderNotFound(c)
		return
	}
	payload := PagePayload{
		Component: PageComponentCourseManagement,
		Props:     struct{}{},
		Meta: PageMeta{
			Title:       pageTitle(i18n.T(requestLang(c), "meta.courseManagement")),
			Description: i18n.T(requestLang(c), "meta.courseManagementDesc"),
		},
		Layout:  buildLayout(c, "courseManage"),
		URL:     buildPageURL(c),
		Version: payloadVersion,
	}
	renderAppShell(c, payload)
}

// ---- 辅助 ----

// courseManageErrorResponse 把课程/评价管理 service 的稳定 sentinel 映射为语义 HTTP 状态。
func courseManageErrorResponse(err error) component.Response {
	switch {
	case errors.Is(err, courseservice.ErrCourseNotFound):
		return component.BuildResponse(http.StatusNotFound,
			component.FailDataCode(component.MessageCourseNotFound, nil))
	case errors.Is(err, courseservice.ErrCourseCodeRequired):
		return component.BuildResponse(http.StatusBadRequest,
			component.FailDataCode(component.MessageCourseCodeRequired, nil))
	case errors.Is(err, courseservice.ErrCourseNameRequired):
		return component.BuildResponse(http.StatusBadRequest,
			component.FailDataCode(component.MessageCourseNameRequired, nil))
	case errors.Is(err, courseservice.ErrCourseCodeConflict):
		return component.BuildResponse(http.StatusConflict,
			component.FailDataCode(component.MessageCourseCodeConflict, nil))
	case errors.Is(err, courseservice.ErrCourseCreditInvalid):
		return component.BuildResponse(http.StatusBadRequest,
			component.FailDataCode(component.MessageCourseCreditInvalid, nil))
	case errors.Is(err, courseservice.ErrReviewNotFound):
		return component.BuildResponse(http.StatusNotFound,
			component.FailDataCode(component.MessageReviewNotFound, nil))
	case errors.Is(err, courseservice.ErrRatingOutOfRange):
		return component.BuildResponse(http.StatusBadRequest,
			component.FailDataCode(component.MessageReviewRatingInvalid, nil))
	case errors.Is(err, courseservice.ErrReviewContentEmpty):
		return component.BuildResponse(http.StatusBadRequest,
			component.FailDataCode(component.MessageReviewContentEmpty, nil))
	case errors.Is(err, courseservice.ErrReviewContentTooLong):
		return component.BuildResponse(http.StatusBadRequest,
			component.FailDataCode(component.MessageReviewContentTooLong,
				component.MessageParams{"maxLength": courseservice.ReviewContentMaxLength}))
	default:
		slog.Error("course_manage_write_failed", "error", err)
		return component.BuildResponse(http.StatusInternalServerError,
			component.FailDataCode(component.MessageOperationFailed, nil))
	}
}

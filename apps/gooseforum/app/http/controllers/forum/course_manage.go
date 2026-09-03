package forum

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/optlogger"
	"github.com/gin-gonic/gin"
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
	ReviewScope *string   `json:"reviewScope"`
	TeamKey     *string   `json:"teamKey"`
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
		ReviewScope: req.Params.ReviewScope,
		TeamKey:     req.Params.TeamKey,
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

// ---- 课程沿革（course_relations 审核）----

// AdminCourseRelationListReq 沿革候选列表请求。
type AdminCourseRelationListReq struct {
	Status       string `json:"status"`       // 空 = 全部；pending/approved/ignored/merged
	RelationType string `json:"relationType"` // 空 = 全部；EQUIVALENT/RENAMED_FROM/SPLIT_FROM/MERGED_FROM/RELATED
	Page         int    `json:"page"`
	PageSize     int    `json:"pageSize"`
}

// AdminCourseRelationList 返回沿革候选分页（含证据快照）。
func AdminCourseRelationList(req component.BetterRequest[AdminCourseRelationListReq]) component.Response {
	if !canModerateCourseReviews(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	pageData, err := courseservice.AdminRelationList(course.RelationQuery{
		Status:       req.Params.Status,
		RelationType: req.Params.RelationType,
		Page:         req.Params.Page,
		Size:         req.Params.PageSize,
	})
	if err != nil {
		slog.Error("course_relation_list_failed", "error", err)
		return component.BuildResponse(http.StatusInternalServerError,
			component.FailDataCode(component.MessageCourseRelationListFailed, nil))
	}
	return component.SuccessResponse(pageData)
}

// AdminCourseRelationApproveReq 批准非合并沿革候选请求。
type AdminCourseRelationApproveReq struct {
	RelationId uint64 `json:"relationId" validate:"required"`
}

// AdminCourseRelationApprove 批准拆分/相关等非合并关系（SPLIT_FROM/MERGED_FROM/RELATED → approved）。
// EQUIVALENT/RENAMED_FROM 必须走合并（AdminCourseMerge），误传返回语义错误。
func AdminCourseRelationApprove(req component.BetterRequest[AdminCourseRelationApproveReq]) component.Response {
	if !canModerateCourseReviews(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	entity, err := courseservice.AdminRelationApprove(req.Params.RelationId)
	if err != nil {
		return courseManageErrorResponse(err)
	}
	optlogger.UserOptCode(req.UserId, optlogger.UpdateCourse, entity.Id, "course.relation.approved", optlogger.MessageParams{
		"relationId":   entity.Id,
		"fromCourseId": entity.FromCourseId,
		"toCourseId":   entity.ToCourseId,
		"relationType": entity.RelationType,
	})
	return component.SuccessResponse(entity)
}

// AdminCourseRelationIgnore 忽略沿革候选（pending → ignored）。
func AdminCourseRelationIgnore(req component.BetterRequest[AdminCourseRelationApproveReq]) component.Response {
	if !canModerateCourseReviews(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	entity, err := courseservice.AdminRelationIgnore(req.Params.RelationId)
	if err != nil {
		return courseManageErrorResponse(err)
	}
	optlogger.UserOptCode(req.UserId, optlogger.UpdateCourse, entity.Id, "course.relation.ignored", optlogger.MessageParams{
		"relationId":   entity.Id,
		"fromCourseId": entity.FromCourseId,
		"toCourseId":   entity.ToCourseId,
		"relationType": entity.RelationType,
	})
	return component.SuccessResponse(entity)
}

// AdminCourseRelationReset 撤回人工处理决定（approved/ignored → pending），让候选回到待审核队列。
func AdminCourseRelationReset(req component.BetterRequest[AdminCourseRelationApproveReq]) component.Response {
	if !canModerateCourseReviews(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	entity, err := courseservice.AdminRelationReset(req.Params.RelationId)
	if err != nil {
		return courseManageErrorResponse(err)
	}
	optlogger.UserOptCode(req.UserId, optlogger.UpdateCourse, entity.Id, "course.relation.resetted", optlogger.MessageParams{
		"relationId":   entity.Id,
		"fromCourseId": entity.FromCourseId,
		"toCourseId":   entity.ToCourseId,
		"relationType": entity.RelationType,
	})
	return component.SuccessResponse(entity)
}

// AdminCourseRelationCreateReq 手动创建沿革关系请求。
type AdminCourseRelationCreateReq struct {
	FromCourseId uint64  `json:"fromCourseId" validate:"required"`
	ToCourseId   uint64  `json:"toCourseId" validate:"required"`
	RelationType string  `json:"relationType" validate:"required"`
	Evidence     string  `json:"evidence"`
	Confidence   float64 `json:"confidence"`
}

// AdminCourseRelationCreate 手动创建沿革关系（source=manual；幂等返回已存在行）。
func AdminCourseRelationCreate(req component.BetterRequest[AdminCourseRelationCreateReq]) component.Response {
	if !canModerateCourseReviews(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	entity, err := courseservice.AdminRelationCreate(courseservice.AdminRelationCreateInput{
		FromCourseId: req.Params.FromCourseId,
		ToCourseId:   req.Params.ToCourseId,
		RelationType: req.Params.RelationType,
		Evidence:     req.Params.Evidence,
		Confidence:   req.Params.Confidence,
	})
	if err != nil {
		return courseManageErrorResponse(err)
	}
	optlogger.UserOptCode(req.UserId, optlogger.CreateCourse, entity.Id, "course.relation.created", optlogger.MessageParams{
		"relationId":   entity.Id,
		"fromCourseId": entity.FromCourseId,
		"toCourseId":   entity.ToCourseId,
		"relationType": entity.RelationType,
	})
	return component.SuccessResponse(entity)
}

// AdminCourseMerge 人工确认等价后物理合并（EQUIVALENT/RENAMED_FROM）。
func AdminCourseMerge(req component.BetterRequest[AdminCourseRelationApproveReq]) component.Response {
	if !canModerateCourseReviews(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	result, err := courseservice.MergeCourses(req.Params.RelationId)
	if err != nil {
		return courseManageErrorResponse(err)
	}
	optlogger.UserOptCode(req.UserId, optlogger.UpdateCourse, result.ToCourseId, "course.merged", optlogger.MessageParams{
		"relationId":      result.RelationId,
		"fromCourseId":    result.FromCourseId,
		"toCourseId":      result.ToCourseId,
		"fromName":        result.FromName,
		"toName":          result.ToName,
		"movedOfferings":  result.MovedOfferings,
		"migratedAliases": result.MigratedAliases,
		"skippedAliases":  result.SkippedAliases,
	})
	return component.SuccessResponse(result)
}

// AdminCourseMergeUndo 撤销已合并的沿革关系（offering/alias 迁回、from 卡恢复）。
func AdminCourseMergeUndo(req component.BetterRequest[AdminCourseRelationApproveReq]) component.Response {
	if !canModerateCourseReviews(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	result, err := courseservice.UndoMergeCourse(req.Params.RelationId)
	if err != nil {
		return courseManageErrorResponse(err)
	}
	optlogger.UserOptCode(req.UserId, optlogger.UpdateCourse, result.ToCourseId, "course.mergeUndone", optlogger.MessageParams{
		"relationId":     result.RelationId,
		"fromCourseId":   result.FromCourseId,
		"toCourseId":     result.ToCourseId,
		"fromName":       result.FromName,
		"toName":         result.ToName,
		"movedOfferings": result.MovedOfferings,
	})
	return component.SuccessResponse(result)
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
	case errors.Is(err, courseservice.ErrRelationNotFound):
		return component.BuildResponse(http.StatusNotFound,
			component.FailDataCode(component.MessageCourseRelationNotFound, nil))
	case errors.Is(err, courseservice.ErrRelationNotMergeable):
		return component.BuildResponse(http.StatusConflict,
			component.FailDataCode(component.MessageCourseRelationNotMerge, nil))
	case errors.Is(err, courseservice.ErrRelationAlreadyMerged):
		return component.BuildResponse(http.StatusConflict,
			component.FailDataCode(component.MessageCourseRelationMerged, nil))
	case errors.Is(err, courseservice.ErrMergeConflict):
		return component.BuildResponse(http.StatusConflict,
			component.FailDataCode(component.MessageCourseRelationConflict, nil))
	case errors.Is(err, courseservice.ErrMergeTargetHidden):
		return component.BuildResponse(http.StatusConflict,
			component.FailDataCode(component.MessageCourseMergeTargetHidden, nil))
	case errors.Is(err, courseservice.ErrReviewScopeInvalid):
		return component.BuildResponse(http.StatusBadRequest,
			component.FailDataCode(component.MessageCourseReviewScopeInvalid, nil))
	case errors.Is(err, courseservice.ErrRelationConfidenceInvalid):
		return component.BuildResponse(http.StatusBadRequest,
			component.FailDataCode(component.MessageCourseRelationConfidenceInvalid, nil))
	case errors.Is(err, courseservice.ErrRelationStateNotResettable):
		return component.BuildResponse(http.StatusConflict,
			component.FailDataCode(component.MessageCourseRelationNotResettable, nil))
	default:
		slog.Error("course_manage_write_failed", "error", err)
		return component.BuildResponse(http.StatusInternalServerError,
			component.FailDataCode(component.MessageOperationFailed, nil))
	}
}

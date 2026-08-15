package forum

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/optlogger"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/gin-gonic/gin"
)

// ---- 写评 / 编辑 / 删除 / helpful / 举报 ----

// CreateCourseReviewReq 创建评价请求体。
type CreateCourseReviewReq struct {
	OfferingId  uint64 `json:"offeringId" validate:"required"`
	Rating      int    `json:"rating" validate:"required,min=1,max=5"`
	Content     string `json:"content" validate:"required"`
	IsAnonymous bool   `json:"isAnonymous"`
}

// CreateCourseReview 登录用户为 offering 写评价。
func CreateCourseReview(req component.BetterRequest[CreateCourseReviewReq]) component.Response {
	payload, err := courseservice.CreateReview(req.UserId, courseservice.CreateReviewInput{
		OfferingId:  req.Params.OfferingId,
		Rating:      req.Params.Rating,
		Content:     req.Params.Content,
		IsAnonymous: req.Params.IsAnonymous,
	})
	if err != nil {
		return reviewErrorResponse(err)
	}
	return component.SuccessResponse(payload)
}

// UpdateCourseReviewReq 更新评价请求体（URI reviewId + 可选字段）。
// Content 为指针以区分"缺省（保留正文）"与"显式空串（清空正文）"，匹配 PATCH 部分更新语义。
type UpdateCourseReviewReq struct {
	ReviewId    uint64  `uri:"reviewId" validate:"required"`
	Rating      *int    `json:"rating"`
	Content     *string `json:"content"`
	IsAnonymous *bool   `json:"isAnonymous"`
}

// UpdateCourseReview 作者更新自己的评价。
func UpdateCourseReview(req component.BetterRequest[UpdateCourseReviewReq]) component.Response {
	payload, err := courseservice.UpdateReview(req.UserId, req.Params.ReviewId, courseservice.UpdateReviewInput{
		Rating:      req.Params.Rating,
		Content:     req.Params.Content,
		IsAnonymous: req.Params.IsAnonymous,
	})
	if err != nil {
		return reviewErrorResponse(err)
	}
	return component.SuccessResponse(payload)
}

// DeleteCourseReviewReq 删除评价请求体（URI reviewId）。
type DeleteCourseReviewReq struct {
	ReviewId uint64 `uri:"reviewId" validate:"required"`
}

// DeleteCourseReview 作者删除评价（进入隔离窗口）。
func DeleteCourseReview(req component.BetterRequest[DeleteCourseReviewReq]) component.Response {
	if err := courseservice.DeleteReview(req.UserId, req.Params.ReviewId); err != nil {
		return reviewErrorResponse(err)
	}
	return component.SuccessResponse(true)
}

// ReviewHelpfulReq 标记/取消 helpful 请求体（URI reviewId）。
type ReviewHelpfulReq struct {
	ReviewId uint64 `uri:"reviewId" validate:"required"`
}

// MarkReviewHelpful 幂等标记 helpful。
func MarkReviewHelpful(req component.BetterRequest[ReviewHelpfulReq]) component.Response {
	if err := courseservice.SetReviewHelpful(req.UserId, req.Params.ReviewId, true); err != nil {
		return reviewErrorResponse(err)
	}
	return component.SuccessResponse(true)
}

// UnmarkReviewHelpful 幂等取消 helpful。
func UnmarkReviewHelpful(req component.BetterRequest[ReviewHelpfulReq]) component.Response {
	if err := courseservice.SetReviewHelpful(req.UserId, req.Params.ReviewId, false); err != nil {
		return reviewErrorResponse(err)
	}
	return component.SuccessResponse(true)
}

// ReportCourseReviewReq 举报评价请求体（URI reviewId + reason/note）。
type ReportCourseReviewReq struct {
	ReviewId uint64 `json:"-" uri:"reviewId" validate:"required"`
	Reason   string `json:"reason" validate:"required,oneof=spam abuse illegal irrelevant other"`
	Note     string `json:"note"`
}

// ReportCourseReview 举报他人评价（登录；不能举报自己的内容）。
func ReportCourseReview(req component.BetterRequest[ReportCourseReviewReq]) component.Response {
	target, ok := reportTargetInfo(reports.TargetCourseReview, req.Params.ReviewId, req.UserId)
	if !ok {
		return component.FailResponseCode(component.MessageReportTargetInvalid, nil)
	}
	if target.UserID == req.UserId {
		return component.FailResponseCode(component.MessageReportOwnContent, nil)
	}
	_, created, err := reports.CreateOpen(reports.Entity{
		TargetType: reports.TargetCourseReview,
		TargetId:   req.Params.ReviewId,
		ReporterId: req.UserId,
		Reason:     req.Params.Reason,
		Note:       trimReportNote(req.Params.Note),
	})
	if err != nil {
		return component.FailResponseCode(component.MessageReportCreateFailed, nil)
	}
	if !created {
		return component.FailResponseCode(component.MessageReportDuplicate, nil)
	}
	return component.SuccessResponse(true)
}

// ---- 公开读：课程评价列表 ----

// ListCourseReviewsReq 课程评价列表（URI courseId；可选 offeringId 过滤；
// B2: cursor/pageSize 分页, issue #174）。
type ListCourseReviewsReq struct {
	CourseId   uint64 `uri:"courseId" validate:"required"`
	OfferingId uint64 `form:"offeringId"`
	Cursor     string `form:"cursor"`
	PageSize   int    `form:"pageSize"`
}

// ListCourseReviews 返回课程（或指定 offering）的可见评价分页；可选 JWT 提供 viewer 状态。
// offeringId 路径必须属于 courseId 且可见，否则 404（防止列出隐藏 offering 的评价，
// 或跨课程指定无归属的 offering）。非法 cursor 或 pageSize 超限 → 400（B2, issue #174）。
func ListCourseReviews(req component.BetterRequest[ListCourseReviewsReq]) component.Response {
	// B2：pageSize 上限 50；非法 cursor 格式 → 400 友好提示（非 500）。
	if req.Params.PageSize > courseservice.MaxReviewPageSize {
		return component.BuildResponse(http.StatusBadRequest,
			component.FailDataCode(component.MessageRequestInvalidParams, nil))
	}
	cursor, err := courseservice.DecodeCursor(req.Params.Cursor)
	if err != nil {
		return component.BuildResponse(http.StatusBadRequest,
			component.FailDataCode(component.MessageRequestInvalidParams, nil))
	}
	// offering 过滤时 cursor 只用 reviewId 段（忽略 offering 段）。
	if req.Params.OfferingId > 0 {
		cursor.OfferingId = 0
	}
	if req.Params.OfferingId > 0 {
		offering, gErr := course.GetOffering(req.Params.OfferingId)
		if gErr != nil || offering.Id == 0 ||
			offering.CourseId != req.Params.CourseId ||
			offering.Status != course.OfferingStatusVisible {
			return component.BuildResponse(http.StatusNotFound,
				component.FailDataCode(component.MessageReviewOfferingNotFound, nil))
		}
	}
	page, err := courseservice.ListReviewsPage(req.Params.CourseId, req.Params.OfferingId, req.UserId, cursor, req.Params.PageSize)
	if err != nil {
		slog.Error("course_review_list_failed", "courseId", req.Params.CourseId, "error", err)
		return component.BuildResponse(http.StatusInternalServerError,
			component.FailDataCode(component.MessageReviewListFailed, nil))
	}
	return component.SuccessResponse(page)
}

// ---- 审核：隐藏/恢复、举报队列、身份揭示 ----

// ModerationCourseReviewStatusReq 审核隐藏/恢复评价请求体。
type ModerationCourseReviewStatusReq struct {
	ReviewId uint64 `json:"reviewId" validate:"required"`
	Action   string `json:"action" validate:"required,oneof=hide show"`
}

// ModerationCourseReviewStatus 独立 course moderator 隐藏/恢复评价（CourseManager）。
func ModerationCourseReviewStatus(req component.BetterRequest[ModerationCourseReviewStatusReq]) component.Response {
	if !canModerateCourseReviews(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	hidden := req.Params.Action == "hide"
	if err := courseservice.SetReviewVisibility(req.Params.ReviewId, hidden); err != nil {
		if errors.Is(err, courseservice.ErrReviewNotFound) {
			return component.FailResponseCode(component.MessageReviewNotFound, nil)
		}
		slog.Error("course_review_moderation_status_failed", "reviewId", req.Params.ReviewId, "error", err)
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	moderationservice.ReviewStatusChanged(req.UserId, req.Params.ReviewId, hidden)
	return component.SuccessResponse(true)
}

// ModerationCourseReviewReportListReq 课评举报队列查询。
type ModerationCourseReviewReportListReq struct {
	Status   string `json:"status" validate:"omitempty,oneof=open resolved rejected"`
	Cursor   uint64 `json:"cursor"`
	PageSize int    `json:"pageSize"`
}

// ModerationCourseReviewReportItem 课评举报项：匿名作者身份永不出现。
type ModerationCourseReviewReportItem struct {
	ID          uint64             `json:"id"`
	ReviewId    uint64             `json:"reviewId"`
	Reason      string             `json:"reason"`
	Note        string             `json:"note"`
	Status      string             `json:"status"`
	Resolution  string             `json:"resolution"`
	Excerpt     string             `json:"excerpt"`
	Reporter    TopicAuthorPayload `json:"reporter"`
	Handler     TopicAuthorPayload `json:"handler"`
	CreatedAt   string             `json:"createdAt"`
	HandledAt   string             `json:"handledAt,omitempty"`
	ReportCount int                `json:"reportCount"`
}

// ModerationCourseReviewReportListResponse 课评举报队列分页响应。
type ModerationCourseReviewReportListResponse struct {
	Items      []ModerationCourseReviewReportItem `json:"items"`
	NextCursor uint64                             `json:"nextCursor"`
	HasNext    bool                               `json:"hasNext"`
}

// ModerationCourseReviewReportList 课评举报队列（CourseManager；不泄露匿名作者身份）。
func ModerationCourseReviewReportList(req component.BetterRequest[ModerationCourseReviewReportListReq]) component.Response {
	if !canModerateCourseReviews(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	pageSize := component.BoundPageSizeWithRange(req.Params.PageSize, 10, 50)
	status := req.Params.Status
	if status == "" {
		status = reports.StatusOpen
	}
	records := reports.CursorPage(reports.CursorPageQuery{
		TargetType: reports.TargetCourseReview,
		Status:     status,
		Cursor:     req.Params.Cursor,
		PageSize:   uint64(pageSize + 1),
	})
	hasNext := len(records) > pageSize
	pageRecords := records
	if hasNext {
		pageRecords = records[:pageSize]
	}
	items := buildModerationCourseReviewReportItems(pageRecords)
	nextCursor := uint64(0)
	if hasNext && len(pageRecords) > 0 {
		nextCursor = pageRecords[len(pageRecords)-1].Id
	}
	return component.SuccessResponse(ModerationCourseReviewReportListResponse{
		Items:      items,
		NextCursor: nextCursor,
		HasNext:    hasNext,
	})
}

// ModerationCourseReviewRevealReq 身份揭示请求体（Admin；必须填写理由）。
type ModerationCourseReviewRevealReq struct {
	ReviewId uint64 `json:"reviewId" validate:"required"`
	Reason   string `json:"reason" validate:"required"`
}

// CourseReviewAuthorRevealPayload Admin 专用的匿名作者身份揭示 DTO。
type CourseReviewAuthorRevealPayload struct {
	ReviewId     uint64 `json:"reviewId"`
	AuthorUserId uint64 `json:"authorUserId,omitempty"`
	Username     string `json:"username,omitempty"`
	Nickname     string `json:"nickname,omitempty"`
	IsAnonymous  bool   `json:"isAnonymous"`
	Source       string `json:"source"`
}

// ModerationCourseReviewReveal 仅 Admin 可查看匿名作者真实身份；必须填写理由并产生审计记录。
func ModerationCourseReviewReveal(req component.BetterRequest[ModerationCourseReviewRevealReq]) component.Response {
	// 身份揭示权限不随 CourseManager 自动获得；v1 仅 Admin。
	if !moderationservice.IsAdmin(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	entity, err := course.GetReview(req.Params.ReviewId)
	if err != nil || entity.Id == 0 {
		return component.FailResponseCode(component.MessageReviewNotFound, nil)
	}
	payload := CourseReviewAuthorRevealPayload{
		ReviewId:     entity.Id,
		AuthorUserId: entity.AuthorID(),
		IsAnonymous:  entity.IsAnonymous,
		Source:       entity.Source,
	}
	if entity.AuthorID() > 0 {
		if user, ok := userservice.GetUserInfo(entity.AuthorID()); ok {
			payload.Username = user.Username
			payload.Nickname = user.Nickname
		}
	}
	// 受限审计：actor、review、reason、request trace 进入 opt_record，不写普通应用日志。
	optlogger.UserOptCode(req.UserId, optlogger.RevealCourseReviewAuthor, entity.Id,
		"review.identityRevealed", optlogger.MessageParams{
			"reviewId": entity.Id,
			"authorId": entity.AuthorID(),
			"reason":   req.Params.Reason,
		})
	return component.SuccessResponse(payload)
}

// ---- 审核 SSR 页面 ----

// CourseReviewModeration 课评审核页面（/moderation/course-reviews）。
// 独立 CourseManager 权限，与论坛版主工作台分开；数据全部走 JSON API 异步加载。
func CourseReviewModeration(c *gin.Context) {
	if !canModerateCourseReviews(component.LoginUserId(c)) {
		renderNotFound(c)
		return
	}
	payload := PagePayload{
		Component: PageComponentCourseReviewModeration,
		Props:     struct{}{},
		Meta: PageMeta{
			Title:       pageTitle(i18n.T(requestLang(c), "meta.courseReviewModeration")),
			Description: i18n.T(requestLang(c), "meta.courseReviewModerationDesc"),
		},
		Layout:  buildLayout(c, "courseReviews"),
		URL:     buildPageURL(c),
		Version: payloadVersion,
	}
	renderAppShell(c, payload)
}

// ---- 辅助 ----

// canModerateCourseReviews 判断用户是否拥有独立课程审核权限（CourseManager，Admin 通过 CheckRole 隐含包含）。
func canModerateCourseReviews(userID uint64) bool {
	if userID == 0 {
		return false
	}
	roleID, ok := userservice.GetUserRoleId(userID)
	return ok && permission.CheckRole(roleID, permission.CourseManager)
}

// reviewErrorResponse 把 course service 的稳定 sentinel 映射为语义 HTTP 状态。
func reviewErrorResponse(err error) component.Response {
	switch {
	case errors.Is(err, courseservice.ErrReviewNotFound):
		return component.BuildResponse(http.StatusNotFound,
			component.FailDataCode(component.MessageReviewNotFound, nil))
	case errors.Is(err, courseservice.ErrReviewNotOwned):
		return component.BuildResponse(http.StatusForbidden,
			component.FailDataCode(component.MessageReviewNotOwned, nil))
	case errors.Is(err, courseservice.ErrReviewDuplicate):
		return component.BuildResponse(http.StatusConflict,
			component.FailDataCode(component.MessageReviewDuplicate, nil))
	case errors.Is(err, courseservice.ErrOfferingNotFound):
		return component.BuildResponse(http.StatusNotFound,
			component.FailDataCode(component.MessageReviewOfferingNotFound, nil))
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
		slog.Error("course_review_write_failed", "error", err)
		return component.BuildResponse(http.StatusInternalServerError,
			component.FailDataCode(component.MessageOperationFailed, nil))
	}
}

// buildModerationCourseReviewReportItems 批量构建课评举报项（不泄露匿名作者身份）。
// reportCount 为该 review 累计举报总数（跨全部状态），不限于当前分页。
func buildModerationCourseReviewReportItems(records []reports.Entity) []ModerationCourseReviewReportItem {
	if len(records) == 0 {
		return []ModerationCourseReviewReportItem{}
	}
	reviewIDs := make([]uint64, 0, len(records))
	userIDs := make([]uint64, 0, len(records)*2)
	for _, record := range records {
		reviewIDs = appendUniqueUint64(reviewIDs, record.TargetId)
		userIDs = appendUniqueUint64(userIDs, record.ReporterId)
		userIDs = appendUniqueUint64(userIDs, record.HandlerId)
	}
	reportCountByReview := reports.CountByTargetIds(reports.TargetCourseReview, reviewIDs)
	reviewMap := course.GetReviewMapByIds(reviewIDs)
	userMap := users.GetMapByIds(userIDs)
	items := make([]ModerationCourseReviewReportItem, 0, len(records))
	for _, record := range records {
		item := ModerationCourseReviewReportItem{
			ID:          record.Id,
			ReviewId:    record.TargetId,
			Reason:      record.Reason,
			Note:        record.Note,
			Status:      record.Status,
			Resolution:  record.Resolution,
			Reporter:    userPayload(record.ReporterId, userMap),
			Handler:     userPayload(record.HandlerId, userMap),
			CreatedAt:   record.CreatedAt.Format(time.RFC3339),
			ReportCount: reportCountByReview[record.TargetId],
		}
		if record.HandledAt != nil {
			item.HandledAt = record.HandledAt.Format(time.RFC3339)
		}
		if review, ok := reviewMap[record.TargetId]; ok {
			item.Excerpt = moderationExcerpt(review.Content)
		}
		if item.Excerpt == "" {
			item.Excerpt = fmt.Sprintf("#%d", record.TargetId)
		}
		items = append(items, item)
	}
	return items
}

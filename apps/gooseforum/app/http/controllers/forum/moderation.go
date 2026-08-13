package forum

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/transform"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/contentDeleteEvent"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/llmsservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/urlconfig"
	"gorm.io/gorm"
)

const moderationPageSize = 20

func Moderation(c *gin.Context) {
	userID := component.LoginUserId(c)
	if !moderationservice.CanAccessModeration(userID) {
		renderNotFound(c)
		return
	}
	page := moderationPage(c)
	global, categoryIDs := moderationservice.ScopeForUser(userID)
	if !global && len(categoryIDs) == 0 {
		renderNotFound(c)
		return
	}
	availableCategories := moderationCategories(global, categoryIDs)
	if len(availableCategories) == 0 {
		renderNotFound(c)
		return
	}
	categoryID := moderationCategoryID(c, availableCategories)

	pageData := topics.PageForModeration(topics.ModerationPageQuery{
		Page:                page,
		PageSize:            moderationPageSize,
		FilterProcessStatus: true,
		ProcessStatus:       1,
		CategoryIDs:         []uint64{categoryID},
	})
	hasNext := pageData.HasNext
	nextPage := 0
	nextURL := ""
	if hasNext {
		nextPage = pageData.Page + 1
		nextURL = buildModerationPageURL(categoryID, nextPage)
	}
	payload := PagePayload{
		Component: PageComponentModeration,
		Props: ModerationPageProps{
			CategoryTabs: buildModerationCategoryTabs(availableCategories, categoryID),
			Topics:       buildTopicPayloads(transform.Topics2Vo(moderationEntityPointers(pageData.Data), hotdataserve.CategoryMap())),
			Pagination: PaginationPayload{
				Page:     pageData.Page,
				NextPage: nextPage,
				HasNext:  hasNext,
				NextURL:  nextURL,
			},
		},
		Meta: PageMeta{
			Title:       pageTitle(i18n.T(requestLang(c), "meta.moderation")),
			Description: i18n.T(requestLang(c), "meta.moderationDesc"),
		},
		Layout:  buildLayout(c, "moderation"),
		URL:     buildPageURL(c),
		Version: payloadVersion,
	}
	renderAppShell(c, payload)
}

func moderationPage(c *gin.Context) int {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

type ModerationTopicStatusReq struct {
	TopicId uint64 `json:"topicId" validate:"required"`
	Action  string `json:"action" validate:"oneof=ban unban"`
}

type ModerationLogListReq struct {
	Cursor   uint64 `json:"cursor"`
	PageSize int    `json:"pageSize"`
}

type ModerationLogListResponse struct {
	Items      []ModerationLogItem `json:"items"`
	NextCursor uint64              `json:"nextCursor"`
	HasNext    bool                `json:"hasNext"`
}

type CreateReportReq struct {
	TargetType string `json:"targetType" validate:"required,oneof=topic post"`
	TargetId   uint64 `json:"targetId" validate:"required"`
	Reason     string `json:"reason" validate:"required,oneof=spam abuse illegal irrelevant other"`
	Note       string `json:"note"`
}

type ModerationReportListReq struct {
	Status   string `json:"status" validate:"omitempty,oneof=open closed resolved rejected"`
	Cursor   uint64 `json:"cursor"`
	PageSize int    `json:"pageSize"`
	Category uint64 `json:"category"`
}

type ModerationReportStatusReq struct {
	Id     uint64 `json:"id" validate:"required"`
	Action string `json:"action" validate:"oneof=ban resolve reject"`
}

type ModerationPostStatusReq struct {
	PostId uint64 `json:"postId" validate:"required"`
	Action string `json:"action" validate:"oneof=ban unban"`
}

type ModerationReportListResponse struct {
	Items      []ModerationReportItem `json:"items"`
	NextCursor uint64                 `json:"nextCursor"`
	HasNext    bool                   `json:"hasNext"`
}

type ModerationReportItem struct {
	ID         uint64                 `json:"id"`
	TargetType string                 `json:"targetType"`
	TargetID   uint64                 `json:"targetId"`
	TargetURL  string                 `json:"targetUrl"`
	Title      string                 `json:"title"`
	Excerpt    string                 `json:"excerpt"`
	Reason     string                 `json:"reason"`
	Note       string                 `json:"note"`
	Status     string                 `json:"status"`
	Resolution string                 `json:"resolution"`
	Reporter   TopicAuthorPayload     `json:"reporter"`
	Handler    TopicAuthorPayload     `json:"handler"`
	Categories []TopicCategoryPayload `json:"categories"`
	CreatedAt  string                 `json:"createdAt"`
	HandledAt  string                 `json:"handledAt,omitempty"`
	// TargetDeleted 标记目标已被作者/管理端删除，前端展示"该内容已被删除"，
	// 审核仍基于举报时刻快照进行（R6）。
	TargetDeleted bool `json:"targetDeleted,omitempty"`
}

type ModerationLogItem struct {
	ID          uint64                 `json:"id"`
	Action      string                 `json:"action"`
	Actor       TopicAuthorPayload     `json:"actor"`
	Subject     ModerationLogSubject   `json:"subject"`
	Categories  []TopicCategoryPayload `json:"categories"`
	MessageCode string                 `json:"messageCode"`
	Params      map[string]any         `json:"params"`
	CreatedAt   string                 `json:"createdAt"`
}

type ModerationLogSubject struct {
	Type    string `json:"type"`
	ID      uint64 `json:"id"`
	Title   string `json:"title"`
	URL     string `json:"url,omitempty"`
	Excerpt string `json:"excerpt,omitempty"`
}

func UpdateModerationTopicStatus(req component.BetterRequest[ModerationTopicStatusReq]) component.Response {
	topic := topics.Get(req.Params.TopicId)
	if topic.Id == 0 {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}
	if !moderationservice.CanModerateAnyCategory(req.UserId, topic.CategoryIds) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	nextStatus := moderationTargetStatus(req.Params.Action)
	if topic.ProcessStatus == nextStatus {
		return component.SuccessResponse(true)
	}
	if err := topics.UpdateProcessStatus(topic.Id, nextStatus); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	topic.ProcessStatus = nextStatus
	hotdataserve.ClearTopicListCache()
	// 审核封禁/解封不发布事件，同步清理 LLMS 投影缓存，避免封禁内容在 10s 窗口内继续导出。
	llmsservice.ClearCache()
	firstPost := posts.Get(topic.FirstPostId)
	if firstPost.Id == 0 {
		firstPost, _ = posts.GetByTopicPostNoAtOrAfter(topic.Id, 1)
	}
	if _, err := searchservice.BuildSingleTopicSearchDocument(&topic, &firstPost); err != nil {
		slog.Error("failed to rebuild topic search document", "topicId", topic.Id, "err", err)
	}
	moderationservice.TopicStatusChanged(req.UserId, topic.Id, topic.Title, nextStatus == 1)
	return component.SuccessResponse(true)
}

func CreateReport(req component.BetterRequest[CreateReportReq]) component.Response {
	target, ok := reportTargetInfo(req.Params.TargetType, req.Params.TargetId, req.UserId)
	if !ok {
		return component.FailResponseCode(component.MessageReportTargetInvalid, nil)
	}
	if target.UserID == req.UserId {
		return component.FailResponseCode(component.MessageReportOwnContent, nil)
	}
	report, created, err := reports.CreateOpen(reports.Entity{
		TargetType:       req.Params.TargetType,
		TargetId:         req.Params.TargetId,
		TopicId:          target.TopicID,
		ReporterId:       req.UserId,
		Reason:           req.Params.Reason,
		Note:             trimReportNote(req.Params.Note),
		EvidenceSnapshot: buildReportEvidenceSnapshot(req.Params.TargetType, req.Params.TargetId, target.TopicID, target.UserID),
	})
	if err != nil {
		return component.FailResponseCode(component.MessageReportCreateFailed, nil)
	}
	if !created {
		return component.FailResponseCode(component.MessageReportDuplicate, nil)
	}
	moderationservice.InvalidateTopic(target.TopicID)
	eventbus.Publish(context.Background(), &eventhandlers.ReportCreatedEvent{
		ReportId:   report.Id,
		TargetType: report.TargetType,
		TargetId:   report.TargetId,
		TopicId:    report.TopicId,
		ReporterId: report.ReporterId,
		Reason:     report.Reason,
	})
	return component.SuccessResponse(true)
}

// buildReportEvidenceSnapshot 在举报创建时刻定格目标内容（Issue #94 R6）。
// 目标随后被作者删除时，审核仍能基于快照继续，避免"删帖逃罚"。
func buildReportEvidenceSnapshot(targetType string, targetID uint64, topicID uint64, authorID uint64) reports.EvidenceSnapshotData {
	snapshot := reports.EvidenceSnapshotData{
		TargetType: targetType,
		TargetID:   targetID,
		TopicID:    topicID,
		AuthorID:   authorID,
		CreatedAt:  time.Now(),
	}
	if authorID > 0 {
		if author, err := users.Get(authorID); err == nil && author.Id > 0 {
			snapshot.AuthorName = author.Username
		}
	}
	switch targetType {
	case reports.TargetTopic:
		topic := topics.GetSimple(targetID)
		if topic.Id > 0 {
			snapshot.Title = topic.Title
			snapshot.Excerpt = moderationExcerpt(topic.Excerpt)
			snapshot.CategoryIDs = topic.CategoryIds
			snapshot.TopicID = topic.Id
			snapshot.TargetURL = urlconfig.PostDetail(topic.Id)
		}
	case reports.TargetPost:
		post := posts.Get(targetID)
		if post.Id > 0 {
			snapshot.Title = fmt.Sprintf("回复 #%d", post.PostNo)
			snapshot.Excerpt = moderationExcerpt(post.Content)
			snapshot.TopicID = post.TopicId
			snapshot.TargetURL = fmt.Sprintf("%s#post-%d", urlconfig.PostDetail(post.TopicId), post.Id)
			topic := topics.GetSimple(post.TopicId)
			if topic.Id > 0 {
				snapshot.CategoryIDs = topic.CategoryIds
				if snapshot.Title == "" {
					snapshot.Title = topic.Title
				}
			}
		}
	}
	return snapshot
}

func UpdateModerationPostStatus(req component.BetterRequest[ModerationPostStatusReq]) component.Response {
	post := posts.Get(req.Params.PostId)
	if post.Id == 0 {
		return component.FailResponseCode(component.MessagePostNotFound, nil)
	}
	topic := topics.GetSimple(post.TopicId)
	if topic.Id == 0 || !moderationservice.CanModerateAnyCategory(req.UserId, topic.CategoryIds) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	nextStatus := moderationTargetStatus(req.Params.Action)
	if post.ProcessStatus == nextStatus {
		return component.SuccessResponse(true)
	}
	if err := posts.UpdateProcessStatus(post.Id, nextStatus); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	// 审核封禁/解封回复不发布事件，同步清理 LLMS 投影缓存。
	llmsservice.ClearCache()
	userMap := users.GetMapByIds([]uint64{post.UserId})
	author := userPayload(post.UserId, userMap)
	moderationservice.PostStatusChanged(req.UserId, moderationservice.PostSnapshot{
		PostId:       post.Id,
		TopicId:      post.TopicId,
		TopicTitle:   topic.Title,
		PostNo:       post.PostNo,
		PostAuthorId: post.UserId,
		PostAuthor:   author.Username,
		Excerpt:      moderationExcerpt(post.Content),
	}, nextStatus == 1)
	return component.SuccessResponse(true)
}

func ModerationReportList(req component.BetterRequest[ModerationReportListReq]) component.Response {
	if !moderationservice.CanAccessModeration(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	pageSize := component.BoundPageSizeWithRange(req.Params.PageSize, 10, 50)
	status := req.Params.Status
	if status == "" {
		status = reports.StatusOpen
	}
	items, nextCursor, hasNext := moderationReportPage(req.UserId, status, req.Params.Category, req.Params.Cursor, pageSize)
	return component.SuccessResponse(ModerationReportListResponse{
		Items:      items,
		NextCursor: nextCursor,
		HasNext:    hasNext,
	})
}

func UpdateModerationReportStatus(req component.BetterRequest[ModerationReportStatusReq]) component.Response {
	report := reports.Get(req.Params.Id)
	if report.Id == 0 {
		return component.FailResponseCode(component.MessageReportNotFound, nil)
	}
	if !canModerateReportTarget(req.UserId, report.TargetType, report.TargetId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	nextStatus := reports.StatusResolved
	resolution := reports.ResolutionBanned
	switch req.Params.Action {
	case "reject":
		nextStatus = reports.StatusRejected
		resolution = reports.ResolutionIgnored
	case "resolve":
		resolution = ""
	}
	if err := reports.UpdateStatus(report.Id, nextStatus, resolution, req.UserId); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	moderationservice.ReportStatusChanged(req.UserId, buildReportLogSnapshot(report, resolution), nextStatus)
	moderationservice.InvalidateTopic(reportTopicID(report))
	return component.SuccessResponse(true)
}

func ModerationLogList(req component.BetterRequest[ModerationLogListReq]) component.Response {
	if !moderationservice.CanAccessModeration(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	pageSize := component.BoundPageSizeWithRange(req.Params.PageSize, 10, 50)
	global, categoryIDs := moderationservice.ScopeForUser(req.UserId)
	if !global && len(categoryIDs) == 0 {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	if global {
		categoryIDs = nil
	}
	records := moderationLog.CursorPage(moderationLog.CursorPageQuery{
		Cursor:           req.Params.Cursor,
		PageSize:         uint64(pageSize + 1),
		ScopeCategoryIDs: categoryIDs,
	})
	hasNext := len(records) > pageSize
	if hasNext {
		records = records[:pageSize]
	}
	nextCursor := uint64(0)
	if hasNext && len(records) > 0 {
		nextCursor = records[len(records)-1].Id
	}
	return component.SuccessResponse(ModerationLogListResponse{
		Items:      buildModerationLogItems(records),
		NextCursor: nextCursor,
		HasNext:    hasNext,
	})
}

// ViewDeletedContentReq 查看已删除内容原文的请求（PRD R7）。
// 查看必须提供理由，每次查看都会写入 moderation_log（EvidenceViewed）与埋点，
// 满足"查看已删内容需理由 + 审计"的最小权限要求。
type ViewDeletedContentReq struct {
	ContentType string `json:"contentType" validate:"required,oneof=topic post"`
	ContentID   uint64 `json:"contentId" validate:"required"`
	Reason      string `json:"reason" validate:"required,min=1,max=300"`
}

// ModerationDeletedContentView 已删除内容原文视图（仅版主查看，含删除元数据）。
type ModerationDeletedContentView struct {
	ContentType  string                 `json:"contentType"`
	ContentID    uint64                 `json:"contentId"`
	TopicID      uint64                 `json:"topicId,omitempty"`
	Title        string                 `json:"title"`
	Content      string                 `json:"content"`
	AuthorID     uint64                 `json:"authorId"`
	AuthorName   string                 `json:"authorName"`
	Categories   []TopicCategoryPayload `json:"categories"`
	DeletedBy    uint64                 `json:"deletedBy"`
	DeletedByWho string                 `json:"deletedByWho"`
	DeletedAt    string                 `json:"deletedAt"`
	DeleteReason string                 `json:"deleteReason"`
	TargetURL    string                 `json:"targetUrl"`
}

// ViewDeletedContent 版主查看已删除内容原文：必须提供理由，写审计日志（R7）。
// 全局版主/管理员可查看其作用域内的已删内容；分类版主仅可查看其分类内的内容。
func ViewDeletedContent(req component.BetterRequest[ViewDeletedContentReq]) component.Response {
	if !moderationservice.CanAccessModeration(req.UserId) {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	reason := strings.TrimSpace(req.Params.Reason)
	if reason == "" {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}

	var view ModerationDeletedContentView
	switch req.Params.ContentType {
	case reports.TargetTopic:
		topic := topics.UnscopedGet(req.Params.ContentID)
		if topic.Id == 0 || topic.VisibilityStatus == topics.VisibilityActive || topic.RetentionStatus == topics.RetentionPurged {
			return component.FailResponseCode(component.MessageTopicNotFound, nil)
		}
		if !moderationservice.CanModerateAnyCategory(req.UserId, topic.CategoryIds) {
			return component.FailResponseCode(component.MessagePermissionDenied, nil)
		}
		view = ModerationDeletedContentView{
			ContentType:  reports.TargetTopic,
			ContentID:    topic.Id,
			TopicID:      topic.Id,
			Title:        topic.Title,
			AuthorID:     topic.UserId,
			DeletedBy:    topic.DeletedBy,
			DeletedAt:    formatDeletedTime(topic.DeletedAt),
			DeleteReason: topic.DeleteReason,
			TargetURL:    urlconfig.PostDetail(topic.Id),
			Categories:   categoryPayloads(topic.CategoryIds),
		}
		firstPost := posts.UnscopedGet(topic.FirstPostId)
		if firstPost.Id > 0 {
			view.Content = firstPost.Content
		}
	case reports.TargetPost:
		post := posts.UnscopedGet(req.Params.ContentID)
		if post.Id == 0 || post.VisibilityStatus == posts.VisibilityActive || post.RetentionStatus == posts.RetentionPurged {
			return component.FailResponseCode(component.MessagePostNotFound, nil)
		}
		topic := topics.UnscopedGet(post.TopicId)
		if topic.Id == 0 || !moderationservice.CanModerateAnyCategory(req.UserId, topic.CategoryIds) {
			return component.FailResponseCode(component.MessagePermissionDenied, nil)
		}
		view = ModerationDeletedContentView{
			ContentType:  reports.TargetPost,
			ContentID:    post.Id,
			TopicID:      post.TopicId,
			Title:        fmt.Sprintf("回复 #%d", post.PostNo),
			Content:      post.Content,
			AuthorID:     post.UserId,
			DeletedBy:    post.DeletedBy,
			DeletedAt:    formatDeletedTime(post.DeletedAt),
			DeleteReason: post.DeleteReason,
			TargetURL:    fmt.Sprintf("%s#post-%d", urlconfig.PostDetail(post.TopicId), post.Id),
			Categories:   categoryPayloads(topic.CategoryIds),
		}
	default:
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}

	if author, err := users.Get(view.AuthorID); err == nil && author.Id > 0 {
		view.AuthorName = author.Username
	}
	if view.DeletedBy > 0 {
		if operator, err := users.Get(view.DeletedBy); err == nil && operator.Id > 0 {
			view.DeletedByWho = operator.Username
		}
	}

	moderationservice.EvidenceViewed(req.UserId, view.ContentType, view.ContentID, view.Title, reason)
	if err := contentDeleteEvent.Record(contentDeleteEvent.Entity{
		EventType:   string(contentDeleteEvent.EventModerationViewed),
		ContentType: view.ContentType,
		ContentID:   view.ContentID,
		TopicID:     view.TopicID,
		ActorID:     req.UserId,
	}); err != nil {
		slog.Error("record moderation deleted content viewed failed", "contentType", view.ContentType, "contentId", view.ContentID, "err", err)
	}
	return component.SuccessResponse(view)
}

func formatDeletedTime(d gorm.DeletedAt) string {
	if !d.Valid || d.Time.IsZero() {
		return ""
	}
	return d.Time.Format(time.DateTime)
}

func moderationTargetStatus(actionType string) int8 {
	if actionType == "unban" {
		return 0
	}
	return 1
}

type reportTargetInfoData struct {
	UserID    uint64
	ArticleID uint64
	TopicID   uint64
}

func reportTargetInfo(targetType string, targetID uint64, userID uint64) (reportTargetInfoData, bool) {
	switch targetType {
	case reports.TargetTopic:
		topic := topics.GetSimple(targetID)
		if topic.Id == 0 || !CanViewTopicSimple(&topic, userID) {
			return reportTargetInfoData{}, false
		}
		return reportTargetInfoData{UserID: topic.UserId, ArticleID: topic.Id, TopicID: topic.Id}, true
	case reports.TargetPost:
		post := posts.Get(targetID)
		if post.Id == 0 || post.ProcessStatus != 0 {
			return reportTargetInfoData{}, false
		}
		topic := topics.GetSimple(post.TopicId)
		if topic.Id == 0 || !CanViewTopicSimple(&topic, userID) {
			return reportTargetInfoData{}, false
		}
		return reportTargetInfoData{UserID: post.UserId, ArticleID: topic.Id, TopicID: topic.Id}, true
	case reports.TargetCourseReview:
		review, err := course.GetReview(targetID)
		if err != nil || review.Id == 0 || review.Status != course.ReviewStatusVisible {
			return reportTargetInfoData{}, false
		}
		return reportTargetInfoData{UserID: review.AuthorID()}, true
	default:
		return reportTargetInfoData{}, false
	}
}

func reportTopicID(record reports.Entity) uint64 {
	if record.TopicId > 0 {
		return record.TopicId
	}
	if record.TargetType == reports.TargetTopic {
		return record.TargetId
	}
	if record.TargetType == reports.TargetPost {
		return posts.Get(record.TargetId).TopicId
	}
	return 0
}

func trimReportNote(note string) string {
	runes := []rune(strings.TrimSpace(note))
	if len(runes) > 300 {
		runes = runes[:300]
	}
	return string(runes)
}

func moderationExcerpt(content string) string {
	runes := []rune(strings.TrimSpace(content))
	if len(runes) > 120 {
		runes = runes[:120]
	}
	return string(runes)
}

func buildReportLogSnapshot(record reports.Entity, resolution string) moderationservice.ReportSnapshot {
	userMap := users.GetMapByIds([]uint64{record.ReporterId})
	reporter := userPayload(record.ReporterId, userMap)
	snapshot := moderationservice.ReportSnapshot{
		ReportId:   record.Id,
		TargetType: record.TargetType,
		TargetId:   record.TargetId,
		Reason:     record.Reason,
		Resolution: resolution,
		ReporterId: record.ReporterId,
		Reporter:   reporter.Username,
	}
	switch record.TargetType {
	case reports.TargetTopic:
		topic := topics.Get(record.TargetId)
		if topic.Id > 0 {
			snapshot.TopicId = topic.Id
			snapshot.TopicTitle = topic.Title
			snapshot.TargetURL = urlconfig.PostDetail(topic.Id)
			snapshot.Excerpt = moderationExcerpt(topic.Excerpt)
		}
	case reports.TargetPost:
		post := posts.Get(record.TargetId)
		if post.Id > 0 {
			topic := topics.Get(post.TopicId)
			snapshot.TopicId = post.TopicId
			snapshot.TopicTitle = topic.Title
			snapshot.PostNo = post.PostNo
			snapshot.TargetURL = fmt.Sprintf("%s#post-%d", urlconfig.PostDetail(post.TopicId), post.Id)
			snapshot.Excerpt = moderationExcerpt(post.Content)
		}
	case reports.TargetCourseReview:
		review, err := course.GetReview(record.TargetId)
		if err == nil && review.Id > 0 {
			snapshot.TargetURL = fmt.Sprintf("/courses/reviews/%d", review.Id)
			snapshot.Excerpt = moderationExcerpt(review.Content)
		}
	}
	return snapshot
}

func canModerateReportTarget(userID uint64, targetType string, targetID uint64) bool {
	// 课评举报由独立 CourseManager 权限处理，不走 forum category scope。
	if targetType == reports.TargetCourseReview {
		return canModerateCourseReviews(userID)
	}
	categoryIDs, ok := reportTargetCategories(targetType, targetID)
	return ok && moderationservice.CanModerateAnyCategory(userID, categoryIDs)
}

func reportTargetCategories(targetType string, targetID uint64) ([]uint64, bool) {
	switch targetType {
	case reports.TargetTopic:
		topic := topics.Get(targetID)
		if topic.Id == 0 {
			// 目标已删：回退 Unscoped，让有作用域权限的版主仍能基于快照审核（R6）。
			topic = topics.UnscopedGet(targetID)
		}
		return topic.CategoryIds, topic.Id > 0
	case reports.TargetPost:
		post := posts.Get(targetID)
		if post.Id == 0 {
			post = posts.UnscopedGet(targetID)
		}
		if post.Id == 0 {
			return nil, false
		}
		topic := topics.Get(post.TopicId)
		if topic.Id == 0 {
			topic = topics.UnscopedGet(post.TopicId)
		}
		return topic.CategoryIds, topic.Id > 0
	default:
		return nil, false
	}
}

func moderationCategoryID(c *gin.Context, categories []TopicCategoryPayload) uint64 {
	requested, err := strconv.ParseUint(c.Query("category"), 10, 64)
	if err == nil && requested > 0 {
		for _, category := range categories {
			if category.ID == requested {
				return requested
			}
		}
	}
	return categories[0].ID
}

func buildModerationCategoryTabs(categories []TopicCategoryPayload, activeID uint64) []TabPayload {
	tabs := make([]TabPayload, 0, len(categories))
	for _, category := range categories {
		tabs = append(tabs, TabPayload{
			Key:    strconv.FormatUint(category.ID, 10),
			Label:  category.Name,
			URL:    buildModerationPageURL(category.ID, 1),
			Active: category.ID == activeID,
		})
	}
	return tabs
}

func buildModerationPageURL(categoryID uint64, page int) string {
	values := url.Values{}
	if categoryID > 0 {
		values.Set("category", strconv.FormatUint(categoryID, 10))
	}
	if page > 1 {
		values.Set("page", strconv.Itoa(page))
	}
	if encoded := values.Encode(); encoded != "" {
		return "/moderation?" + encoded
	}
	return "/moderation"
}

func moderationCategories(global bool, categoryIDs []uint64) []TopicCategoryPayload {
	categories := hotdataserve.GetCategory()
	allowed := map[uint64]bool{}
	if global {
		for _, category := range categories {
			if category != nil {
				allowed[category.Id] = true
			}
		}
	} else {
		for _, categoryID := range categoryIDs {
			allowed[categoryID] = true
		}
	}
	res := make([]TopicCategoryPayload, 0, len(allowed))
	for _, category := range categories {
		if category == nil || !allowed[category.Id] {
			continue
		}
		res = append(res, TopicCategoryPayload{
			ID:    category.Id,
			Name:  category.Name,
			URL:   categoryURL(category),
			Color: category.Color,
		})
	}
	return res
}

func moderationEntityPointers(data []topics.Entity) []*topics.Entity {
	res := make([]*topics.Entity, 0, len(data))
	for i := range data {
		res = append(res, &data[i])
	}
	return res
}

func buildModerationLogItems(records []moderationLog.Entity) []ModerationLogItem {
	if len(records) == 0 {
		return []ModerationLogItem{}
	}
	actorIDs := make([]uint64, 0, len(records))
	topicIDs := make([]uint64, 0, len(records))
	for _, record := range records {
		actorIDs = appendUniqueUint64(actorIDs, record.ActorUserId)
		if record.Payload.Params != nil {
			topicIDs = appendUniqueUint64(topicIDs, uint64FromParam(record.Payload.Params["topicId"]))
		}
		if record.SubjectType == moderationLog.SubjectTopic {
			topicIDs = appendUniqueUint64(topicIDs, record.SubjectId)
		}
	}
	userMap := users.GetMapByIds(actorIDs)
	topicMap := topics.GetPointerMapByIds(topicIDs)
	items := make([]ModerationLogItem, 0, len(records))
	for _, record := range records {
		payload := record.Payload
		if payload.Params == nil {
			payload.Params = map[string]any{}
		}
		subject := moderationLogSubject(record, payload.Params, topicMap)
		items = append(items, ModerationLogItem{
			ID:          record.Id,
			Action:      record.Action,
			Actor:       userPayload(record.ActorUserId, userMap),
			Subject:     subject,
			Categories:  moderationLogCategories(record, payload.Params, subject.ID, topicMap),
			MessageCode: payload.MessageCode,
			Params:      payload.Params,
			CreatedAt:   record.CreatedAt.Format(time.DateTime),
		})
	}
	return items
}

func moderationLogSubject(record moderationLog.Entity, params map[string]any, topicMap map[uint64]*topics.Entity) ModerationLogSubject {
	switch record.SubjectType {
	case moderationLog.SubjectTopic:
		subject := ModerationLogSubject{Type: record.SubjectType, ID: record.SubjectId, Title: fmt.Sprint(params["title"])}
		if topic := topicMap[record.SubjectId]; topic != nil {
			subject.Title = topic.Title
			subject.URL = urlconfig.PostDetail(topic.Id)
			subject.Excerpt = topic.Excerpt
		} else if subject.ID > 0 {
			subject.URL = urlconfig.PostDetail(subject.ID)
		}
		if subject.Title == "" || subject.Title == "<nil>" {
			subject.Title = fmt.Sprintf("#%d", subject.ID)
		}
		return subject
	case moderationLog.SubjectCategory:
		title := fmt.Sprint(params["categoryName"])
		if title == "" || title == "<nil>" {
			title = fmt.Sprintf("#%d", record.SubjectId)
		}
		return ModerationLogSubject{Type: record.SubjectType, ID: record.SubjectId, Title: title}
	case moderationLog.SubjectPost:
		title := fmt.Sprint(params["title"])
		topicID := uint64FromParam(params["topicId"])
		postNo := uint64FromParam(params["postNo"])
		excerpt := fmt.Sprint(params["excerpt"])
		if (title == "" || title == "<nil>") && topicID > 0 {
			if topic := topicMap[topicID]; topic != nil {
				title = topic.Title
			}
		}
		if title == "" || title == "<nil>" {
			title = fmt.Sprintf("#%d", record.SubjectId)
		}
		subject := ModerationLogSubject{Type: record.SubjectType, ID: record.SubjectId, Title: postLogTitle(title, postNo), Excerpt: excerpt}
		if topicID > 0 {
			subject.URL = fmt.Sprintf("%s#post-%d", urlconfig.PostDetail(topicID), record.SubjectId)
		}
		return subject
	case moderationLog.SubjectReport:
		title := fmt.Sprint(params["title"])
		if title == "" || title == "<nil>" {
			title = fmt.Sprintf("举报 #%d", record.SubjectId)
		}
		postNo := uint64FromParam(params["postNo"])
		if postNo > 0 {
			title = postLogTitle(title, postNo)
		}
		subject := ModerationLogSubject{Type: record.SubjectType, ID: record.SubjectId, Title: title, Excerpt: fmt.Sprint(params["excerpt"])}
		if targetURL := fmt.Sprint(params["targetUrl"]); targetURL != "" && targetURL != "<nil>" {
			subject.URL = targetURL
		}
		return subject
	case moderationLog.SubjectUser:
		title := fmt.Sprint(params["username"])
		if title == "" || title == "<nil>" {
			title = fmt.Sprintf("#%d", record.SubjectId)
		}
		return ModerationLogSubject{Type: record.SubjectType, ID: record.SubjectId, Title: title}
	default:
		return ModerationLogSubject{Type: moderationLog.SubjectSystem, ID: record.SubjectId, Title: fmt.Sprintf("#%d", record.SubjectId)}
	}
}

func postLogTitle(topicTitle string, postNo uint64) string {
	if postNo == 0 {
		return topicTitle
	}
	return fmt.Sprintf("%s #%d", topicTitle, postNo)
}

func uint64FromParam(value any) uint64 {
	switch v := value.(type) {
	case uint64:
		return v
	case uint:
		return uint64(v)
	case int:
		return uint64(v)
	case int64:
		return uint64(v)
	case float64:
		return uint64(v)
	default:
		return 0
	}
}

func moderationLogCategories(record moderationLog.Entity, params map[string]any, subjectID uint64, topicMap map[uint64]*topics.Entity) []TopicCategoryPayload {
	topicID := subjectID
	if record.SubjectType == moderationLog.SubjectPost || record.SubjectType == moderationLog.SubjectReport {
		topicID = uint64FromParam(params["topicId"])
	}
	topic := topicMap[topicID]
	if topic == nil {
		return []TopicCategoryPayload{}
	}
	return categoryPayloads(topic.CategoryIds)
}

func moderationReportPage(userID uint64, status string, categoryID uint64, cursor uint64, pageSize int) ([]ModerationReportItem, uint64, bool) {
	scopeCategoryIDs, ok := reportScopeCategoryIDs(userID, categoryID)
	if !ok {
		return []ModerationReportItem{}, 0, false
	}
	query := reports.CursorPageQuery{
		Cursor:           cursor,
		PageSize:         uint64(pageSize + 1),
		ScopeCategoryIDs: scopeCategoryIDs,
	}
	if status == "closed" {
		query.Statuses = []string{reports.StatusResolved, reports.StatusRejected}
	} else {
		query.Status = status
	}
	records := reports.CursorPage(query)
	hasNext := len(records) > pageSize
	pageRecords := records
	if hasNext {
		pageRecords = records[:pageSize]
	}
	batchMaps := reportBatchMaps(pageRecords)
	items := make([]ModerationReportItem, 0, len(pageRecords))
	for _, record := range pageRecords {
		item, ok := buildModerationReportItem(userID, categoryID, record, batchMaps)
		if ok {
			items = append(items, item)
		}
	}
	nextCursor := uint64(0)
	if hasNext && len(pageRecords) > 0 {
		nextCursor = pageRecords[len(pageRecords)-1].Id
	}
	return items, nextCursor, hasNext
}

func reportScopeCategoryIDs(userID uint64, categoryID uint64) ([]uint64, bool) {
	global, categoryIDs := moderationservice.ScopeForUser(userID)
	if global {
		if categoryID > 0 {
			return []uint64{categoryID}, true
		}
		return nil, true
	}
	if categoryID == 0 {
		return categoryIDs, len(categoryIDs) > 0
	}
	if slices.Contains(categoryIDs, categoryID) {
		return []uint64{categoryID}, true
	}
	return nil, false
}

type moderationReportBatchMaps struct {
	TopicMap map[uint64]topics.Entity
	PostMap  map[uint64]*posts.Entity
	UserMap  map[uint64]*users.EntityComplete
}

func reportBatchMaps(records []reports.Entity) moderationReportBatchMaps {
	topicIDs := make([]uint64, 0, len(records))
	postIDs := make([]uint64, 0, len(records))
	userIDs := make([]uint64, 0, len(records)*2)
	for _, record := range records {
		userIDs = appendUniqueUint64(userIDs, record.ReporterId)
		userIDs = appendUniqueUint64(userIDs, record.HandlerId)
		topicIDs = appendUniqueUint64(topicIDs, record.TopicId)
		if record.TargetType == reports.TargetTopic {
			topicIDs = appendUniqueUint64(topicIDs, record.TargetId)
		}
		if record.TargetType == reports.TargetPost {
			postIDs = appendUniqueUint64(postIDs, record.TargetId)
		}
	}
	// 举报目标可能已被作者删除（软删）。为了仍能基于快照审核并保持分类作用域
	// 过滤，这里用 Unscoped 加载含已删行在内的目标数据（Issue #94 R6）。
	postMap := posts.GetMapByIdsUnscoped(postIDs)
	for _, post := range postMap {
		topicIDs = appendUniqueUint64(topicIDs, post.TopicId)
	}
	return moderationReportBatchMaps{
		TopicMap: topics.GetMapByIdsUnscoped(topicIDs),
		PostMap:  postMap,
		UserMap:  users.GetMapByIds(userIDs),
	}
}

func reportCategoriesFromMaps(record reports.Entity, batchMaps moderationReportBatchMaps) ([]uint64, bool) {
	switch record.TargetType {
	case reports.TargetTopic:
		topic := batchMaps.TopicMap[record.TargetId]
		if topic.Id == 0 {
			// 目标已删：回退到举报时刻快照中的分类，保持分类作用域过滤可用（R6）。
			return record.EvidenceSnapshot.CategoryIDs, len(record.EvidenceSnapshot.CategoryIDs) > 0
		}
		return topic.CategoryIds, true
	case reports.TargetPost:
		topicID := record.TopicId
		if topicID == 0 {
			post := batchMaps.PostMap[record.TargetId]
			if post == nil {
				return record.EvidenceSnapshot.CategoryIDs, len(record.EvidenceSnapshot.CategoryIDs) > 0
			}
			topicID = post.TopicId
		}
		topic := batchMaps.TopicMap[topicID]
		if topic.Id == 0 {
			// 目标所在话题已删：同样回退到快照分类。
			return record.EvidenceSnapshot.CategoryIDs, len(record.EvidenceSnapshot.CategoryIDs) > 0
		}
		return topic.CategoryIds, true
	default:
		return nil, false
	}
}

func buildModerationReportItem(userID uint64, categoryID uint64, record reports.Entity, batchMaps moderationReportBatchMaps) (ModerationReportItem, bool) {
	categoryIDs, ok := reportCategoriesFromMaps(record, batchMaps)
	if !ok || !moderationservice.CanModerateAnyCategory(userID, categoryIDs) {
		return ModerationReportItem{}, false
	}
	if categoryID > 0 && !slices.Contains(categoryIDs, categoryID) {
		return ModerationReportItem{}, false
	}
	resolution := record.Resolution
	if resolution == "" && record.Status == reports.StatusRejected {
		resolution = reports.ResolutionIgnored
	}
	item := ModerationReportItem{
		ID:         record.Id,
		TargetType: record.TargetType,
		TargetID:   record.TargetId,
		Reason:     record.Reason,
		Note:       record.Note,
		Status:     record.Status,
		Resolution: resolution,
		Reporter:   userPayload(record.ReporterId, batchMaps.UserMap),
		Handler:    userPayload(record.HandlerId, batchMaps.UserMap),
		Categories: categoryPayloads(categoryIDs),
		CreatedAt:  record.CreatedAt.Format(time.DateTime),
	}
	if record.HandledAt != nil {
		item.HandledAt = record.HandledAt.Format(time.DateTime)
	}
	switch record.TargetType {
	case reports.TargetTopic:
		topic := batchMaps.TopicMap[record.TargetId]
		if topic.Id == 0 || topic.VisibilityStatus != topics.VisibilityActive {
			// 目标已删：用举报时刻快照渲染，标注删除态，审核仍可继续（R6）。
			item.TargetDeleted = true
			item.Title = record.EvidenceSnapshot.Title
			item.Excerpt = record.EvidenceSnapshot.Excerpt
			item.TargetURL = record.EvidenceSnapshot.TargetURL
		} else {
			item.Title = topic.Title
			item.TargetURL = urlconfig.PostDetail(topic.Id)
		}
	case reports.TargetPost:
		post := batchMaps.PostMap[record.TargetId]
		topicID := record.TopicId
		if post == nil {
			// 目标回复已删：用快照渲染（R6）。
			item.TargetDeleted = true
			item.Title = record.EvidenceSnapshot.Title
			item.Excerpt = record.EvidenceSnapshot.Excerpt
			item.TargetURL = record.EvidenceSnapshot.TargetURL
		} else {
			if topicID == 0 {
				topicID = post.TopicId
			}
			topic := batchMaps.TopicMap[topicID]
			if topic.Id > 0 {
				item.Title = topic.Title
			}
			item.Excerpt = moderationExcerpt(post.Content)
			item.TargetURL = fmt.Sprintf("%s#post-%d", urlconfig.PostDetail(topicID), post.Id)
			if topic.Id == 0 || topic.VisibilityStatus != topics.VisibilityActive || post.VisibilityStatus != posts.VisibilityActive {
				item.TargetDeleted = true
			}
		}
	}
	if item.Title == "" {
		item.Title = fmt.Sprintf("#%d", record.TargetId)
	}
	return item, true
}

func appendUniqueUint64(items []uint64, item uint64) []uint64 {
	if item == 0 {
		return items
	}
	if slices.Contains(items, item) {
		return items
	}
	return append(items, item)
}

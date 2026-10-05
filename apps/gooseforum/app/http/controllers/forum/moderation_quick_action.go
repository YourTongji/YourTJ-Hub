package forum

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/urlconfig"
	"github.com/gin-gonic/gin"
)

// 举报快捷处置（issue #1049）：通知卡片的签名链接只把版主带到 Hub 确认页，真正的
// 处置由登录用户提交，并复用工作台的同一套封禁/隐藏/结案实现与权限模型。

// 快捷处置结果状态（与确认页契约一致）。
const (
	QuickActionReady     = "ready"
	QuickActionDone      = "done"
	QuickActionProcessed = "processed"
	QuickActionNotFound  = "notFound"
	QuickActionForbidden = "forbidden"
	QuickActionInvalid   = "invalid"
	QuickActionFailed    = "failed"
)

// 举报快捷动作。
const (
	ReportQuickActionBan     = "ban"     // topic/post：封禁目标并结案
	ReportQuickActionHide    = "hide"    // course_review：隐藏评价并结案
	ReportQuickActionDismiss = "dismiss" // 驳回举报
)

// ReportQuickActionPreview 确认页展示的举报摘要（已按权限与匿名规则裁剪）。
type ReportQuickActionPreview struct {
	Report    reports.Entity
	Title     string
	Excerpt   string
	Anonymous bool
	TargetURL string
}

// reportQuickActionAllowed 动作与举报目标的合法组合；私信举报不提供快捷动作
// （仅 Admin 在工作台查看证据后处理）。
func reportQuickActionAllowed(targetType string, action string) bool {
	switch targetType {
	case reports.TargetTopic, reports.TargetPost:
		return action == ReportQuickActionBan || action == ReportQuickActionDismiss
	case reports.TargetCourseReview:
		return action == ReportQuickActionHide || action == ReportQuickActionDismiss
	default:
		return false
	}
}

// InspectReportQuickAction 只读复核：举报存在、动作合法、当前用户对目标有处置权限、
// 举报仍待处理。无权限时不返回任何目标内容。
func InspectReportQuickAction(actorID uint64, reportID uint64, action string) (ReportQuickActionPreview, string) {
	report := reports.Get(reportID)
	if report.Id == 0 {
		return ReportQuickActionPreview{}, QuickActionNotFound
	}
	if !reportQuickActionAllowed(report.TargetType, action) {
		return ReportQuickActionPreview{}, QuickActionInvalid
	}
	if actorID == 0 || !canModerateReportTarget(actorID, report.TargetType, report.TargetId) {
		return ReportQuickActionPreview{}, QuickActionForbidden
	}
	preview := reportQuickActionPreview(report)
	if report.Status != reports.StatusOpen {
		return preview, QuickActionProcessed
	}
	return preview, QuickActionReady
}

// reportQuickActionPreview 以举报时刻快照为准（目标可能已被作者删除，R6），
// 匿名楼层快照本就不含作者；课评只展示正文摘要与匿名标记。
func reportQuickActionPreview(report reports.Entity) ReportQuickActionPreview {
	preview := ReportQuickActionPreview{
		Report:    report,
		Title:     report.EvidenceSnapshot.Title,
		Excerpt:   report.EvidenceSnapshot.Excerpt,
		TargetURL: report.EvidenceSnapshot.TargetURL,
	}
	switch report.TargetType {
	case reports.TargetTopic:
		// 快照缺失（早于证据快照的存量举报）时回退读取目标本身（含已删）。
		if preview.Title == "" {
			if topic := topics.UnscopedGet(report.TargetId); topic.Id > 0 {
				preview.Title = topic.Title
				preview.Excerpt = moderationExcerpt(topic.Excerpt)
				preview.TargetURL = urlconfig.PostDetail(topic.Id)
			}
		}
	case reports.TargetPost:
		if post := posts.UnscopedGet(report.TargetId); post.Id > 0 {
			preview.Anonymous = post.IsAnonymous
			if preview.Title == "" {
				preview.Title = topics.UnscopedGet(post.TopicId).Title
				preview.Excerpt = moderationExcerpt(post.Content)
				preview.TargetURL = fmt.Sprintf("%s#post-%d", urlconfig.PostDetail(post.TopicId), post.Id)
			}
		}
	case reports.TargetCourseReview:
		if review, err := course.GetReview(report.TargetId); err == nil && review.Id > 0 {
			preview.Excerpt = moderationExcerpt(review.Content)
			preview.Anonymous = review.IsAnonymous
			if offering, err := course.GetOffering(review.OfferingId); err == nil && offering.Id > 0 {
				if c := course.GetCourse(offering.CourseId); c.Id > 0 {
					preview.Title = c.Name
					preview.TargetURL = fmt.Sprintf("/courses/%d", c.Id)
				}
			}
		}
	}
	return preview
}

// ApplyReportQuickAction 举报处置的后端原子命令：先对目标执行与工作台相同的封禁/
// 隐藏（幂等），再以 CAS 结案；已被他人处理时返回 processed，不覆盖首个处理人。
// handler_id 与审核日志记录真实操作者。
func ApplyReportQuickAction(ctx context.Context, actorID uint64, reportID uint64, action string) string {
	_, state := InspectReportQuickAction(actorID, reportID, action)
	if state != QuickActionReady {
		return state
	}
	report := reports.Get(reportID)
	switch action {
	case ReportQuickActionDismiss:
		return closeReportIfOpen(actorID, report, reports.StatusRejected, reports.ResolutionIgnored)
	case ReportQuickActionBan:
		switch report.TargetType {
		case reports.TargetTopic:
			topic := topics.Get(report.TargetId)
			if topic.Id == 0 {
				return QuickActionNotFound
			}
			if err := applyTopicModerationStatus(ctx, actorID, topic, topics.ProcessStatusBlocked); err != nil {
				slog.Error("report quick action: ban topic failed", "reportId", report.Id, "err", err)
				return QuickActionFailed
			}
		case reports.TargetPost:
			post := posts.Get(report.TargetId)
			topic := topics.GetSimple(post.TopicId)
			if post.Id == 0 || topic.Id == 0 {
				return QuickActionNotFound
			}
			if err := applyPostModerationStatus(ctx, actorID, post, topic, posts.ProcessStatusBlocked); err != nil {
				slog.Error("report quick action: ban post failed", "reportId", report.Id, "err", err)
				return QuickActionFailed
			}
		}
		return closeReportIfOpen(actorID, report, reports.StatusResolved, reports.ResolutionBanned)
	case ReportQuickActionHide:
		if err := courseservice.SetReviewVisibility(report.TargetId, true); err != nil {
			if errors.Is(err, courseservice.ErrReviewNotFound) {
				return QuickActionNotFound
			}
			slog.Error("report quick action: hide course review failed", "reportId", report.Id, "err", err)
			return QuickActionFailed
		}
		moderationservice.ReviewStatusChanged(actorID, report.TargetId, true)
		return closeReportIfOpen(actorID, report, reports.StatusResolved, reports.ResolutionBanned)
	default:
		return QuickActionInvalid
	}
}

// closeReportIfOpen 以 CAS 结案并写审核日志（与工作台 UpdateModerationReportStatus 同口径）。
func closeReportIfOpen(actorID uint64, report reports.Entity, status string, resolution string) string {
	updated, err := reports.UpdateStatusIfOpen(report.Id, status, resolution, actorID)
	if err != nil {
		slog.Error("report quick action: close report failed", "reportId", report.Id, "err", err)
		return QuickActionFailed
	}
	if !updated {
		return QuickActionProcessed
	}
	if report.TargetType != reports.TargetChatMessage {
		moderationservice.ReportStatusChanged(actorID, buildReportLogSnapshot(report, resolution), status)
	} else {
		moderationservice.InvalidatePrivateReports()
	}
	moderationservice.InvalidateTopic(reportTopicID(report))
	return QuickActionDone
}

// ModerationAction 快捷审批确认页外壳（GET 无副作用）：服务端不读取 token，页面在
// POST body 中提交 token 预览与执行。禁止缓存与外发 Referer，页面 URL 不回显查询串，
// 避免链接中的 token 经缓存、Referer 或页面负载外泄。
func ModerationAction(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	lang := requestLang(c)
	payload := PagePayload{
		Component: PageComponentModerationAction,
		Props:     struct{}{},
		Meta: PageMeta{
			Title:       pageTitle(i18n.T(lang, "meta.moderationAction")),
			Description: i18n.T(lang, "meta.moderationActionDesc"),
		},
		Layout:  buildLayout(c, "moderation"),
		URL:     component.GetBaseUri(c) + c.Request.URL.Path,
		Version: payloadVersion,
	}
	renderAppShell(c, payload)
}

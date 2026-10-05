package api

import (
	"errors"
	"strconv"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/httpnotifyservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/tokenservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/urlconfig"
)

// 版主快捷审批确认页（issue #1049）：通知卡片按钮携带签名 token 打开 /moderation/action，
// 页面先调 preview（只读）展示目标与当前状态，用户显式确认后调 execute。
// token 只防篡改与限制有效期；两个端点都按当前登录会话重新判定权限，并复用
// 工作台同一套审核/举报命令，handler_id 与审核日志记录真实操作者。

// 确认页状态（preview/execute 共用）。
const (
	approvalActionExpired = "expired"
	approvalActionChanged = "changed"
)

const approvalActionExcerptRunes = 200

// ModerationApprovalActionReq 确认页请求：token 放在 POST body，不进入访问日志。
type ModerationApprovalActionReq struct {
	Token string `json:"token" validate:"required,max=2048"`
}

// ModerationApprovalActionView 确认页视图：无权限、token 无效或过期时不含目标内容。
type ModerationApprovalActionView struct {
	State        string `json:"state"`
	Subject      string `json:"subject,omitempty"`
	Action       string `json:"action,omitempty"`
	TargetType   string `json:"targetType,omitempty"`
	TargetID     uint64 `json:"targetId,omitempty"`
	ReportID     uint64 `json:"reportId,omitempty"`
	Title        string `json:"title,omitempty"`
	Excerpt      string `json:"excerpt,omitempty"`
	Anonymous    bool   `json:"anonymous,omitempty"`
	TargetURL    string `json:"targetUrl,omitempty"`
	WorkbenchURL string `json:"workbenchUrl"`
	ExpiresAt    string `json:"expiresAt,omitempty"`
}

// ModerationApprovalActionPreview 只读预览：校验签名与有效期、复核权限与当前状态。
func ModerationApprovalActionPreview(req component.BetterRequest[ModerationApprovalActionReq]) component.Response {
	view, _ := inspectApprovalAction(req.UserId, req.Params.Token)
	return component.SuccessResponse(view)
}

// ModerationApprovalActionExecute 执行快捷审批：仅当 preview 同口径判定为 ready 时执行；
// 其余状态（已处理、内容已更新、无权限、过期）原样返回且不产生任何变更。
func ModerationApprovalActionExecute(req component.BetterRequest[ModerationApprovalActionReq]) component.Response {
	view, claims := inspectApprovalAction(req.UserId, req.Params.Token)
	if view.State != forum.QuickActionReady {
		return component.SuccessResponse(view)
	}
	ctx := requestContext(req.GinContext)
	switch claims.Subject {
	case tokenservice.ModerationActionSubjectReviewTopic, tokenservice.ModerationActionSubjectReviewPost:
		revisionID, _ := strconv.ParseUint(claims.Version, 10, 64)
		response := reviewContent(ctx, ReviewActionReq{
			Kind:       reviewKindForSubject(claims.Subject),
			Id:         claims.ID,
			RevisionId: revisionID,
			Approve:    claims.Action == httpnotifyservice.ApprovalActionApprove,
		}, req.UserId, false)
		switch {
		case response.Data.Code == component.SUCCESS:
			view.State = forum.QuickActionDone
		case response.Data.MessageCode == component.MessageAdminReviewProcessed:
			// 发布层按版本加锁判定；落败时重新复核，区分“已处理”与“作者已改稿”。
			view.State = inspectReviewApprovalAction(req.UserId, claims, &view)
			if view.State == forum.QuickActionReady {
				view.State = forum.QuickActionProcessed
			}
		default:
			view.State = forum.QuickActionFailed
		}
	case tokenservice.ModerationActionSubjectReport:
		view.State = forum.ApplyReportQuickAction(ctx, req.UserId, claims.ID, claims.Action)
	default:
		view.State = forum.QuickActionInvalid
	}
	return component.SuccessResponse(view)
}

func inspectApprovalAction(userID uint64, token string) (ModerationApprovalActionView, tokenservice.ModerationActionClaims) {
	claims, err := tokenservice.ParseModerationAction(token, time.Now())
	if err != nil {
		if errors.Is(err, tokenservice.ErrModerationActionExpired) {
			return ModerationApprovalActionView{
				State: approvalActionExpired, Subject: claims.Subject, Action: claims.Action,
				WorkbenchURL: approvalWorkbenchURL(claims.Subject, ""),
			}, claims
		}
		return ModerationApprovalActionView{State: forum.QuickActionInvalid, WorkbenchURL: approvalWorkbenchURL("", "")}, tokenservice.ModerationActionClaims{}
	}
	view := ModerationApprovalActionView{
		Subject:      claims.Subject,
		Action:       claims.Action,
		WorkbenchURL: approvalWorkbenchURL(claims.Subject, ""),
		ExpiresAt:    time.Unix(claims.Expires, 0).UTC().Format(time.RFC3339),
	}
	switch claims.Subject {
	case tokenservice.ModerationActionSubjectReviewTopic, tokenservice.ModerationActionSubjectReviewPost:
		view.State = inspectReviewApprovalAction(userID, claims, &view)
	case tokenservice.ModerationActionSubjectReport:
		preview, state := forum.InspectReportQuickAction(userID, claims.ID, claims.Action)
		view.State = state
		if state == forum.QuickActionReady || state == forum.QuickActionProcessed {
			view.ReportID = preview.Report.Id
			view.TargetType = preview.Report.TargetType
			view.TargetID = preview.Report.TargetId
			view.Title = preview.Title
			view.Excerpt = httpnotifyservice.SafeText(preview.Excerpt, approvalActionExcerptRunes)
			view.Anonymous = preview.Anonymous
			view.TargetURL = preview.TargetURL
			view.WorkbenchURL = approvalWorkbenchURL(claims.Subject, preview.Report.TargetType)
		}
	default:
		view.State = forum.QuickActionInvalid
	}
	return view, claims
}

// inspectReviewApprovalAction 待审话题/回复：先判定审核范围（无权限不泄露是否存在），
// 再复核链接绑定的送审版本仍待审且仍是最新版本——作者改稿后旧卡片返回 changed。
// 预览展示送审版本本身（改稿待审时公开的仍是旧版）。
func inspectReviewApprovalAction(userID uint64, claims tokenservice.ModerationActionClaims, view *ModerationApprovalActionView) string {
	if claims.Action != httpnotifyservice.ApprovalActionApprove && claims.Action != httpnotifyservice.ApprovalActionReject {
		return forum.QuickActionInvalid
	}
	revisionID, err := strconv.ParseUint(claims.Version, 10, 64)
	if err != nil || revisionID == 0 {
		return forum.QuickActionInvalid
	}
	if _, _, ok := moderationReviewScope(userID); !ok {
		return forum.QuickActionForbidden
	}
	kind := reviewKindForSubject(claims.Subject)
	var topic topics.Entity
	var post posts.Entity
	if kind == "topic" {
		topic = topics.Get(claims.ID)
		post = posts.Get(topic.FirstPostId)
	} else {
		post = posts.Get(claims.ID)
		topic = topics.GetSimple(post.TopicId)
	}
	if topic.Id == 0 || post.Id == 0 {
		return forum.QuickActionNotFound
	}
	if !canReviewTarget(userID, kind, claims.ID, revisionID) {
		return forum.QuickActionForbidden
	}
	if topic.TopicType == topics.TopicTypeWiki && post.PostNo <= 1 {
		return forum.QuickActionInvalid
	}
	revision := postRevisions.Get(revisionID)
	if revision.Id == 0 || revision.PostId != post.Id {
		return forum.QuickActionInvalid
	}
	view.TargetID = claims.ID
	view.Excerpt = httpnotifyservice.SafeText(markdown2html.ExtractDescription(revision.Content, approvalActionExcerptRunes), approvalActionExcerptRunes)
	view.Anonymous = post.IsAnonymous
	if kind == "topic" {
		view.TargetType = reports.TargetTopic
		view.Title = revision.Title
		view.TargetURL = urlconfig.PostDetail(topic.Id)
	} else {
		view.TargetType = reports.TargetPost
		view.Title = topic.Title
		view.TargetURL = urlconfig.PostDetail(topic.Id) + "#post-" + strconv.FormatUint(post.Id, 10)
	}
	if revision.ProcessStatus != posts.ProcessStatusPending {
		return forum.QuickActionProcessed
	}
	if post.LatestRevisionId != revision.Id {
		return approvalActionChanged
	}
	return forum.QuickActionReady
}

func reviewKindForSubject(subject string) string {
	if subject == tokenservice.ModerationActionSubjectReviewPost {
		return "post"
	}
	return "topic"
}

// approvalWorkbenchURL 确认页“打开工作台”的落点：课评举报进入课评审核页，其余进入
// 版主工作台对应标签；token 无法解析时回到工作台首页。
func approvalWorkbenchURL(subject string, targetType string) string {
	switch {
	case subject == tokenservice.ModerationActionSubjectReport && targetType == reports.TargetCourseReview:
		return "/moderation/course-reviews"
	case subject == tokenservice.ModerationActionSubjectReport:
		return "/moderation?tab=reports"
	case subject == tokenservice.ModerationActionSubjectReviewTopic, subject == tokenservice.ModerationActionSubjectReviewPost:
		return "/moderation?tab=review"
	default:
		return "/moderation"
	}
}

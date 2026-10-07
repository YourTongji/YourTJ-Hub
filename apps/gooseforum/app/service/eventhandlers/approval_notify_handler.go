package eventhandlers

import (
	"context"
	"log/slog"
	"net/url"
	"strconv"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/httpnotifyservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/tokenservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/urlconfig"
)

// 版主审批通知（issue #1049）：把“需要人工介入”的语义事件转成统一审批摘要。
// 摘要在此完成匿名/隐私裁剪，通道渲染器不再回库拼接身份。

// 审批入口路径（站内）。
const (
	moderationReviewURL        = "/moderation?tab=review"
	moderationReportsURL       = "/moderation?tab=reports"
	moderationCourseReviewsURL = "/moderation/course-reviews"
	moderationActionPath       = "/moderation/action"
)

// 人工审核触发来源。
const (
	ReviewTriggerSensitiveWord = "sensitive_word"
	ReviewTriggerAI            = "ai"
)

// 审批摘要的外部展示长度上限。
const (
	approvalTitleRunes   = 100
	approvalExcerptRunes = 200
	approvalNoteRunes    = 300
)

// ModerationReviewRequestedEvent 待审版本转入人工审核队列（issue #1049）。由
// publicationservice 在“转人工”事务提交后经内容效果任务发布（至少一次，投递按审批
// 去重）；敏感词、AI 存疑与 AI 故障转人工都经这一处，自动检查中的待审不发布。
type ModerationReviewRequestedEvent struct {
	RevisionID uint64
}

func handleHttpNotifyReviewRequested(ctx context.Context, event *ModerationReviewRequestedEvent) error {
	if !httpnotifyservice.ShouldNotify(httpnotifyservice.EventReviewTopicRequested, httpnotifyservice.EventReviewPostRequested) {
		return nil
	}
	approval, eventName, ok := reviewApproval(event.RevisionID, time.Now())
	if !ok {
		return nil
	}
	httpnotifyservice.Publish(httpnotifyservice.Message{
		Alternatives: []httpnotifyservice.Alternative{{Event: eventName, Data: approval}},
		Approval:     &approval,
		DedupeKey:    approval.Approval.ID,
	})
	return nil
}

// ReviewSubject 待审版本的审核主体：首楼按话题审核（与审核队列一致），其余按回复。
func ReviewSubject(topic topics.Entity, post posts.Entity) (string, uint64) {
	if post.PostNo <= 1 {
		return "topic", topic.Id
	}
	return "post", post.Id
}

// reviewApproval 描述送审版本本身：改稿待审时公开的仍是旧版，摘要与快捷操作都
// 绑定该版本 ID。版本已不是最新、不再待审或尚未转人工时不通知。
func reviewApproval(revisionID uint64, now time.Time) (httpnotifyservice.ApprovalPayload, string, bool) {
	revision := postRevisions.Get(revisionID)
	if revision.Id == 0 || revision.ProcessStatus != posts.ProcessStatusPending || revision.ReviewedAt == nil {
		return httpnotifyservice.ApprovalPayload{}, "", false
	}
	post := posts.Get(revision.PostId)
	if post.Id == 0 || post.LatestRevisionId != revision.Id {
		return httpnotifyservice.ApprovalPayload{}, "", false
	}
	topic := topics.GetSimple(post.TopicId)
	if topic.Id == 0 || (topic.TopicType == topics.TopicTypeWiki && post.PostNo <= 1) {
		return httpnotifyservice.ApprovalPayload{}, "", false
	}
	reason := ReviewTriggerSensitiveWord
	if moderationDecision.ExistsForRevision(revision.Id) {
		reason = ReviewTriggerAI
	}
	kind, subjectID := ReviewSubject(topic, post)
	approval := httpnotifyservice.Approval{
		Kind:          httpnotifyservice.ApprovalKindReview,
		TargetID:      subjectID,
		TopicID:       topic.Id,
		Reason:        reason,
		Excerpt:       approvalExcerpt(revision.Content),
		Edited:        post.PublishedRevisionId != 0,
		Version:       strconv.FormatUint(revision.Id, 10),
		CreatedAt:     now.Format(time.RFC3339),
		ModerationURL: moderationReviewURL,
	}
	applyPostAuthor(&approval, post)
	eventName := httpnotifyservice.EventReviewPostRequested
	subject := tokenservice.ModerationActionSubjectReviewPost
	if kind == "topic" {
		eventName = httpnotifyservice.EventReviewTopicRequested
		subject = tokenservice.ModerationActionSubjectReviewTopic
		approval.TargetType = reports.TargetTopic
		approval.Title = revision.Title
		approval.Categories = approvalCategories(revision.CategoryIds)
		approval.TargetURL = urlconfig.PostDetail(topic.Id)
	} else {
		approval.TargetType = reports.TargetPost
		approval.Title = topic.Title
		approval.Categories = approvalCategories(topic.CategoryIds)
		approval.TargetURL = postURL(topic.Id, post.Id)
	}
	approval.Title = approvalText(approval.Title, approvalTitleRunes)
	approval.ID = "review:" + kind + ":" + strconv.FormatUint(subjectID, 10) + ":r" + approval.Version
	approval.Actions = approvalActions(subject, subjectID, approval.Version, now,
		httpnotifyservice.ApprovalActionApprove, httpnotifyservice.ApprovalActionReject)
	return httpnotifyservice.ApprovalPayload{BaseURI: baseURI(), Approval: approval}, eventName, true
}

// reportApprovalEvent 举报目标对应的具体审批事件名。
func reportApprovalEvent(targetType string) string {
	switch targetType {
	case reports.TargetTopic:
		return httpnotifyservice.EventReportTopicCreated
	case reports.TargetPost, "reply":
		return httpnotifyservice.EventReportPostCreated
	case reports.TargetChatMessage:
		return httpnotifyservice.EventReportChatMessageCreated
	case reports.TargetCourseReview:
		return httpnotifyservice.EventReportCourseReviewCreated
	default:
		return ""
	}
}

// reportApproval 举报审批摘要：私信举报只给最少元数据（不含正文、举报说明与当事人
// 身份），且不提供快捷动作（仅 Admin 可在工作台处理）；匿名楼层/匿名课评不输出作者。
func reportApproval(event *ReportCreatedEvent, now time.Time) (httpnotifyservice.ApprovalPayload, bool) {
	approval := httpnotifyservice.Approval{
		ID:            "report:" + strconv.FormatUint(event.ReportId, 10),
		Kind:          httpnotifyservice.ApprovalKindReport,
		TargetType:    event.TargetType,
		TargetID:      event.TargetId,
		ReportID:      event.ReportId,
		TopicID:       event.TopicId,
		Reason:        event.Reason,
		Note:          approvalText(event.Note, approvalNoteRunes),
		CreatedAt:     now.Format(time.RFC3339),
		ModerationURL: moderationReportsURL,
	}
	if event.ReportId == 0 {
		return httpnotifyservice.ApprovalPayload{}, false
	}
	var actions []string
	switch event.TargetType {
	case reports.TargetTopic:
		topic := topics.Get(event.TargetId)
		if topic.Id > 0 {
			approval.Title = topic.Title
			approval.Excerpt = approvalText(topic.Excerpt, approvalExcerptRunes)
			approval.Anonymous = topic.PersonaUID != ""
			if !approval.Anonymous {
				approval.Author = approvalAuthor(topic.UserId)
			}
			approval.Categories = approvalCategories(topic.CategoryIds)
			approval.TargetURL = urlconfig.PostDetail(topic.Id)
		}
		actions = []string{httpnotifyservice.ApprovalActionBan, httpnotifyservice.ApprovalActionDismiss}
	case reports.TargetPost, "reply":
		approval.TargetType = reports.TargetPost
		post := posts.Get(event.TargetId)
		if post.Id > 0 {
			topic := topics.GetSimple(post.TopicId)
			approval.TopicID = post.TopicId
			approval.Title = topic.Title
			approval.Excerpt = approvalExcerpt(post.Content)
			applyPostAuthor(&approval, post)
			approval.Categories = approvalCategories(topic.CategoryIds)
			approval.TargetURL = postURL(post.TopicId, post.Id)
		}
		actions = []string{httpnotifyservice.ApprovalActionBan, httpnotifyservice.ApprovalActionDismiss}
	case reports.TargetCourseReview:
		approval.ModerationURL = moderationCourseReviewsURL
		if review, err := course.GetReview(event.TargetId); err == nil && review.Id > 0 {
			approval.Excerpt = approvalExcerpt(review.Content)
			approval.Anonymous = review.IsAnonymous
			if !review.IsAnonymous && review.AuthorUserId != nil {
				approval.Author = approvalAuthor(*review.AuthorUserId)
			}
			if offering, err := course.GetOffering(review.OfferingId); err == nil && offering.Id > 0 {
				if c := course.GetCourse(offering.CourseId); c.Id > 0 {
					approval.Title = c.Name
					approval.TargetURL = "/courses/" + strconv.FormatUint(c.Id, 10)
				}
			}
		}
		actions = []string{httpnotifyservice.ApprovalActionHide, httpnotifyservice.ApprovalActionDismiss}
	case reports.TargetChatMessage:
		approval.Note = ""
	default:
		return httpnotifyservice.ApprovalPayload{}, false
	}
	approval.Title = approvalText(approval.Title, approvalTitleRunes)
	approval.Actions = approvalActions(tokenservice.ModerationActionSubjectReport, event.ReportId, "", now, actions...)
	return httpnotifyservice.ApprovalPayload{BaseURI: baseURI(), Approval: approval}, true
}

// applyPostAuthor 匿名楼层（issue #524）只标记匿名，不输出真实作者。
func applyPostAuthor(approval *httpnotifyservice.Approval, post posts.Entity) {
	if post.IsAnonymous {
		approval.Anonymous = true
		return
	}
	approval.Author = approvalAuthor(post.UserId)
}

func approvalAuthor(userID uint64) *httpnotifyservice.ApprovalAuthor {
	if userID == 0 {
		return nil
	}
	user := userNotifyPayload(userID)
	return &httpnotifyservice.ApprovalAuthor{ID: userID, DisplayName: user.DisplayName, URL: user.URL}
}

func approvalCategories(categoryIDs []uint64) []string {
	items := topicCategoryNotifyPayloads(categoryIDs)
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names
}

func approvalExcerpt(markdown string) string {
	return approvalText(markdown2html.ExtractDescription(markdown, approvalExcerptRunes), approvalExcerptRunes)
}

func approvalText(value string, maxRunes int) string {
	return httpnotifyservice.SafeText(value, maxRunes)
}

// approvalActions 为每个快捷动作签发确认页链接；签名密钥不可用时不生成按钮
// （卡片仍可打开工作台）。
func approvalActions(subject string, id uint64, version string, now time.Time, actions ...string) []httpnotifyservice.ApprovalAction {
	result := make([]httpnotifyservice.ApprovalAction, 0, len(actions))
	for _, action := range actions {
		token, err := tokenservice.IssueModerationAction(tokenservice.ModerationActionClaims{
			Subject: subject, ID: id, Action: action, Version: version,
		}, now)
		if err != nil {
			slog.Warn("moderation approval action link unavailable", "subject", subject, "err", err)
			return []httpnotifyservice.ApprovalAction{}
		}
		result = append(result, httpnotifyservice.ApprovalAction{
			Action: action,
			URL:    moderationActionPath + "?token=" + url.QueryEscape(token),
		})
	}
	return result
}

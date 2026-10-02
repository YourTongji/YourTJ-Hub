package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/gin-gonic/gin"
)

// applyAIModeration 在敏感词门禁之后、写库之前接入 AI 图文审查（issue #975）。
// 返回的 check 必须在写库后调用 Finish(subjectID) 落库决策（nil 安全）；
// pending=true 时调用方把内容置为 ProcessStatusPending；err 非空时拒绝写入，
// 编辑器保留草稿与图片（与敏感词 block 同语义）。shadow 模式不影响本次发布；
// 先发后审模式不等待模型，内容直接写为待审，Finish 后在后台判定。
func applyAIModeration(c *gin.Context, ctx context.Context, input moderationservice.AIContentInput) (*moderationservice.AIModeration, bool, error) {
	check := moderationservice.PrepareAIModeration(ctx, input)
	if check.Deferred() {
		if check.BlockExternalImagesNow() {
			check.Finish(input.SubjectID)
			return nil, false, component.NewMessageError(component.MessageContentAIExternalImageBlocked,
				"本站不显示站外图片。请把图片上传到本站后重新发布。", nil)
		}
		return check, true, nil
	}
	if !check.Enforcing() {
		return check, false, nil
	}
	extendWriteDeadline(c, check.RequestBudget())
	switch check.Enforce(ctx) {
	case moderationDecision.ActionBlock:
		check.Finish(input.SubjectID)
		if check.ExternalImageBlocked() {
			return nil, false, component.NewMessageError(component.MessageContentAIExternalImageBlocked,
				"本站不显示站外图片。请把图片上传到本站后重新发布。", nil)
		}
		return nil, false, component.NewMessageError(component.MessageContentAIBlocked,
			"这条内容不符合本站发布规则。请修改后重新发布。", nil)
	case moderationDecision.ActionReview:
		return check, true, nil
	default:
		return check, false, nil
	}
}

// extendWriteDeadline 同步审查可能超过 http.Server 的 10s WriteTimeout
// （console/serve.go）：仅对确实要调用模型的发布请求放宽本连接的写超时，
// 其余端点保持慢写保护不变（与 mcpservice 的做法一致）。
func extendWriteDeadline(c *gin.Context, budget time.Duration) {
	if c == nil || c.Writer == nil || budget <= 0 {
		return
	}
	_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(budget + 10*time.Second))
}

// publishSuccess 写入成功响应；待审时在成功信封上带 content.moderation.pendingReview，
// 让 Web/App 明确提示“已提交审核，通过后可见”（不暴露是敏感词还是 AI 触发）。
// 先发后审的自动检查（checking=true）改用 content.moderation.checking：作者被告知
// 通常很快公开，而不是等待人工。
func publishSuccess(data any, pendingReview bool, checking bool) component.Response {
	if pendingReview && checking {
		return component.SuccessResponseCode(data, component.MessageContentModerationChecking, nil)
	}
	if pendingReview {
		return component.SuccessResponseCode(data, component.MessageContentModerationPendingReview, nil)
	}
	return component.SuccessResponse(data)
}

func init() {
	moderationservice.SetDeferredOutcomeHandler(applyDeferredAIOutcome)
}

// applyDeferredAIOutcome 先发后审的后台结论：allow 自动公开（不发通知，作者发布时
// 已被告知通过后公开），block 自动拒绝并通知作者，review 留在审核队列等待人工。
// 只处理仍待审、且标题与正文仍是本次判定所评估版本的内容；版主已先处理或作者
// 已改稿时丢弃。
func applyDeferredAIOutcome(ctx context.Context, decision moderationDecision.Entity, input moderationservice.AIContentInput) {
	if decision.FinalAction == moderationDecision.ActionReview || decision.SubjectId == 0 {
		return
	}
	params := ReviewActionReq{Id: decision.SubjectId, Approve: decision.FinalAction == moderationDecision.ActionAllow}
	switch decision.SubjectType {
	case moderationDecision.SubjectTopic:
		topic := topics.Get(decision.SubjectId)
		if topic.Id == 0 || topic.ProcessStatus != topics.ProcessStatusPending || topic.Title != input.Title ||
			posts.Get(topic.FirstPostId).Content != input.Content {
			slog.Info("ai moderation deferred outcome skipped", "subjectType", decision.SubjectType, "subjectId", decision.SubjectId)
			return
		}
		params.Kind = "topic"
	case moderationDecision.SubjectPost:
		post := posts.Get(decision.SubjectId)
		if post.Id == 0 || post.ProcessStatus != posts.ProcessStatusPending || post.Content != input.Content {
			slog.Info("ai moderation deferred outcome skipped", "subjectType", decision.SubjectType, "subjectId", decision.SubjectId)
			return
		}
		params.Kind = "post"
	default:
		return
	}
	if response := reviewContent(ctx, params, 0, true); response.Data.Code != component.SUCCESS {
		slog.Warn("ai moderation deferred outcome failed", "subjectType", decision.SubjectType, "subjectId", decision.SubjectId,
			"messageCode", response.Data.MessageCode)
	}
}

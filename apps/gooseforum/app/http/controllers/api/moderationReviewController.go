package api

import (
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
)

// 前台版主工作台的待审队列（issue #975）：与管理后台审核队列共用实现，按版主
// 管辖范围收窄——全局版主与管理员看到全部，分类版主只看到管辖分类内的内容；
// 审核动作逐条复核目标所属分类，越权一律按“不存在”处理，不泄露内容状态。

// ModerationReviewQueue 列出当前版主可审核的待审话题或回复。
func ModerationReviewQueue(req component.BetterRequest[ReviewQueueReq]) component.Response {
	global, categoryIDs, ok := moderationReviewScope(req.UserId)
	if !ok {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	if global {
		categoryIDs = nil
	}
	return reviewQueue(req, categoryIDs)
}

// ModerationReviewAction 版主通过或拒绝一条待审内容（与后台 ReviewAction 同语义：
// 通过公开内容与图片、补发业务事件并通知作者；拒绝保持不公开并通知作者）。
func ModerationReviewAction(req component.BetterRequest[ReviewActionReq]) component.Response {
	if _, _, ok := moderationReviewScope(req.UserId); !ok {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}
	if !canReviewTarget(req.UserId, req.Params.Kind, req.Params.Id, req.Params.RevisionId) {
		return component.FailResponseCode(component.MessageAdminReviewNotFound, nil)
	}
	return ReviewAction(req)
}

func moderationReviewScope(userID uint64) (bool, []uint64, bool) {
	if !moderationservice.CanAccessModeration(userID) {
		return false, nil, false
	}
	global, categoryIDs := moderationservice.ScopeForUser(userID)
	return global, categoryIDs, global || len(categoryIDs) > 0
}

func canReviewTarget(userID uint64, kind string, id, revisionID uint64) bool {
	topicID := id
	var post posts.Entity
	if kind == "post" {
		post = posts.Get(id)
		if post.Id == 0 {
			return false
		}
		topicID = post.TopicId
	}
	topic := topics.GetSimple(topicID)
	if topic.Id == 0 || !moderationservice.CanModerateAnyCategory(userID, topic.CategoryIds) {
		return false
	}
	if kind == "topic" {
		post = posts.Get(topic.FirstPostId)
	}
	// A first post publishes the whole topic snapshot even when addressed as a post.
	// Check the exact requested revision; Review fences it against the latest revision.
	if post.PostNo == 1 && post.LatestRevisionId != 0 {
		revision := postRevisions.Get(revisionID)
		if revision.Id == 0 || revision.PostId != post.Id {
			return false
		}
		if !moderationservice.CanModerateAnyCategory(userID, revision.CategoryIds) {
			return false
		}
	}
	return true
}

// postContentWrittenAt 帖子当前正文的最后写入时间：编辑过取 last_edited_at，
// 否则取 created_at（AI 决策回写人工结论时据此排除旧版本决策）。
func postContentWrittenAt(post posts.Entity) time.Time {
	if post.LastEditedAt != nil {
		return *post.LastEditedAt
	}
	return post.CreatedAt
}

package moderationservice

import (
	"log/slog"

	"github.com/leancodebox/GooseForum/app/models/forum/moderationLog"
)

func TopicStatusChanged(actorUserId uint64, topicId uint64, title string, blocked bool) {
	action := moderationLog.ActionTopicUnblocked
	status := "unblocked"
	if blocked {
		action = moderationLog.ActionTopicBlocked
		status = "blocked"
	}
	create(moderationLog.Entity{
		ActorUserId: actorUserId,
		Action:      action,
		SubjectType: moderationLog.SubjectTopic,
		SubjectId:   topicId,
		Payload: moderationLog.Payload{
			MessageCode: "moderation.log.topic.statusChanged",
			Params: map[string]any{
				"topicId": topicId,
				"title":   title,
				"status":  status,
			},
		},
	})
}

type PostSnapshot struct {
	PostId       uint64
	TopicId      uint64
	TopicTitle   string
	PostNo       uint64
	PostAuthorId uint64
	PostAuthor   string
	Excerpt      string
}

type ReportSnapshot struct {
	ReportId   uint64
	TargetType string
	TargetId   uint64
	TargetURL  string
	TopicId    uint64
	TopicTitle string
	PostNo     uint64
	Reason     string
	Resolution string
	ReporterId uint64
	Reporter   string
	Excerpt    string
}

func PostStatusChanged(actorUserId uint64, snapshot PostSnapshot, blocked bool) {
	action := moderationLog.ActionPostUnblocked
	status := "unblocked"
	if blocked {
		action = moderationLog.ActionPostBlocked
		status = "blocked"
	}
	create(moderationLog.Entity{
		ActorUserId: actorUserId,
		Action:      action,
		SubjectType: moderationLog.SubjectPost,
		SubjectId:   snapshot.PostId,
		Payload: moderationLog.Payload{
			MessageCode: "moderation.log.post.statusChanged",
			Params: map[string]any{
				"topicId":      snapshot.TopicId,
				"postId":       snapshot.PostId,
				"title":        snapshot.TopicTitle,
				"postNo":       snapshot.PostNo,
				"postAuthorId": snapshot.PostAuthorId,
				"postAuthor":   snapshot.PostAuthor,
				"excerpt":      snapshot.Excerpt,
				"status":       status,
			},
		},
	})
}

func ReportStatusChanged(actorUserId uint64, snapshot ReportSnapshot, status string) {
	action := moderationLog.ActionReportResolved
	if status == "rejected" {
		action = moderationLog.ActionReportRejected
	}
	create(moderationLog.Entity{
		ActorUserId: actorUserId,
		Action:      action,
		SubjectType: moderationLog.SubjectReport,
		SubjectId:   snapshot.ReportId,
		Payload: moderationLog.Payload{
			MessageCode: "moderation.log.report.statusChanged",
			Params: map[string]any{
				"targetType": snapshot.TargetType,
				"targetId":   snapshot.TargetId,
				"targetUrl":  snapshot.TargetURL,
				"topicId":    snapshot.TopicId,
				"title":      snapshot.TopicTitle,
				"postNo":     snapshot.PostNo,
				"reason":     snapshot.Reason,
				"resolution": snapshot.Resolution,
				"reporterId": snapshot.ReporterId,
				"reporter":   snapshot.Reporter,
				"excerpt":    snapshot.Excerpt,
				"status":     status,
			},
		},
	})
}

func create(entity moderationLog.Entity) {
	if err := moderationLog.Create(&entity); err != nil {
		slog.Error("create moderation log failed", "action", entity.Action, "subjectType", entity.SubjectType, "subjectId", entity.SubjectId, "err", err)
	}
}

// SensitiveContentBlocked 记录内容因命中敏感词被拦截。
func SensitiveContentBlocked(actorUserId uint64, subjectType string, subjectId uint64, word string, excerpt string) {
	create(moderationLog.Entity{
		ActorUserId: actorUserId,
		Action:      moderationLog.ActionSensitiveBlocked,
		SubjectType: subjectType,
		SubjectId:   subjectId,
		Payload: moderationLog.Payload{
			MessageCode: "moderation.log.content.sensitiveBlocked",
			Params: map[string]any{
				"word":    word,
				"excerpt": excerpt,
			},
		},
	})
}

// SensitiveContentReview 记录内容因命中敏感词转入人工审核。
func SensitiveContentReview(actorUserId uint64, subjectType string, subjectId uint64, word string, excerpt string) {
	create(moderationLog.Entity{
		ActorUserId: actorUserId,
		Action:      moderationLog.ActionSensitiveReview,
		SubjectType: subjectType,
		SubjectId:   subjectId,
		Payload: moderationLog.Payload{
			MessageCode: "moderation.log.content.sensitiveReview",
			Params: map[string]any{
				"word":    word,
				"excerpt": excerpt,
			},
		},
	})
}

// ReviewStatusChanged 记录课评隐藏/恢复的审核操作（独立 course 审核日志）。
func ReviewStatusChanged(actorUserId, reviewId uint64, hidden bool) {
	action := moderationLog.ActionCourseReviewUnblocked
	status := "shown"
	if hidden {
		action = moderationLog.ActionCourseReviewBlocked
		status = "hidden"
	}
	create(moderationLog.Entity{
		ActorUserId: actorUserId,
		Action:      action,
		SubjectType: moderationLog.SubjectCourseReview,
		SubjectId:   reviewId,
		Payload: moderationLog.Payload{
			MessageCode: "moderation.log.courseReview.statusChanged",
			Params: map[string]any{
				"reviewId": reviewId,
				"status":   status,
			},
		},
	})
}

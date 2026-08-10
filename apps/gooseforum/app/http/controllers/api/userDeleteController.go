package api

import (
	"time"

	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/service/contentdeleteservice"
)

// DeleteTopicByUserReq 用户删除自己的话题请求。
type DeleteTopicByUserReq struct {
	TopicId uint64 `json:"topicId" validate:"required"`
}

// DeleteTopicByUser 用户删除自己的话题（R1）。
func DeleteTopicByUser(req component.BetterRequest[DeleteTopicByUserReq]) component.Response {
	if err := contentdeleteservice.DeleteTopicByUser(req.UserId, req.Params.TopicId); err != nil {
		return component.FailResponseError(err)
	}
	return component.SuccessResponse(true)
}

// DeletedContentListReq 最近删除列表请求。
type DeletedContentListReq struct {
	ContentType string `form:"contentType" validate:"required,oneof=topic post"`
	CursorID    uint64 `form:"cursorId"`
	Limit       int    `form:"limit"`
}

// DeletedContentItem 最近删除列表项。
type DeletedContentItem struct {
	ID           uint64 `json:"id"`
	ContentType  string `json:"contentType"`
	Title        string `json:"title"`
	Excerpt      string `json:"excerpt,omitempty"`
	TopicID      uint64 `json:"topicId,omitempty"`
	PostNo       uint64 `json:"postNo,omitempty"`
	Visibility   string `json:"visibility"`
	Retention    string `json:"retention"`
	DeletedAt    string `json:"deletedAt"`
	CanRestore   bool   `json:"canRestore"`
	CanPermanent bool   `json:"canPermanent"`
	HasReplies   bool   `json:"hasReplies,omitempty"`
}

// DeletedContentList 分页返回用户已删除的话题/回复（R3）。
func DeletedContentList(req component.BetterRequest[DeletedContentListReq]) component.Response {
	limit := req.Params.Limit
	if limit <= 0 || limit > 30 {
		limit = 20
	}

	switch contentdeleteservice.ContentType(req.Params.ContentType) {
	case contentdeleteservice.ContentTypeTopic:
		entities := topics.GetUserDeletedPage(req.UserId, req.Params.CursorID, limit)
		hasMore := len(entities) > limit
		if hasMore {
			entities = entities[:limit]
		}
		items := make([]DeletedContentItem, 0, len(entities))
		for _, topic := range entities {
			items = append(items, DeletedContentItem{
				ID:           topic.Id,
				ContentType:  "topic",
				Title:        topic.Title,
				Excerpt:      topic.Excerpt,
				Visibility:   topic.VisibilityStatus,
				Retention:    topic.RetentionStatus,
				DeletedAt:    formatDeletedAt(topic.DeletedAt.Time),
				CanRestore:   canRestore(topic.VisibilityStatus, topic.RetentionStatus),
				CanPermanent: canPermanentDelete(topic.VisibilityStatus, topic.RetentionStatus),
			})
		}
		return component.SuccessResponse(map[string]any{
			"items":        items,
			"hasMore":      hasMore,
			"nextCursorId": lastCursorID(items),
		})
	case contentdeleteservice.ContentTypePost:
		entities := posts.GetUserDeletedPage(req.UserId, req.Params.CursorID, limit)
		hasMore := len(entities) > limit
		if hasMore {
			entities = entities[:limit]
		}
		items := make([]DeletedContentItem, 0, len(entities))
		for _, post := range entities {
			deletedAt := post.DeletedAt.Time
			if !post.DeletedAt.Valid {
				deletedAt = post.UpdatedAt
			}
			items = append(items, DeletedContentItem{
				ID:           post.Id,
				ContentType:  "post",
				Excerpt:      excerptOf(post.Content),
				TopicID:      post.TopicId,
				PostNo:       post.PostNo,
				Visibility:   post.VisibilityStatus,
				Retention:    post.RetentionStatus,
				DeletedAt:    formatDeletedAt(deletedAt),
				CanRestore:   canRestore(post.VisibilityStatus, post.RetentionStatus),
				CanPermanent: canPermanentDelete(post.VisibilityStatus, post.RetentionStatus),
			})
		}
		return component.SuccessResponse(map[string]any{
			"items":        items,
			"hasMore":      hasMore,
			"nextCursorId": lastCursorID(items),
		})
	default:
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
}

// RestoreContentReq 恢复已删除内容请求。
type RestoreContentReq struct {
	ContentType string `json:"contentType" validate:"required,oneof=topic post"`
	ContentID   uint64 `json:"contentId" validate:"required"`
}

// RestoreContent 恢复 30 天窗口内的已删除内容（R3）。
func RestoreContent(req component.BetterRequest[RestoreContentReq]) component.Response {
	if err := contentdeleteservice.RestoreContent(req.UserId, contentdeleteservice.ContentType(req.Params.ContentType), req.Params.ContentID); err != nil {
		return component.FailResponseError(err)
	}
	return component.SuccessResponseCode("操作成功", component.MessageContentRestoreSuccess, nil)
}

// PurgeContentReq 永久删除请求。
type PurgeContentReq struct {
	ContentType string `json:"contentType" validate:"required,oneof=topic post"`
	ContentID   uint64 `json:"contentId" validate:"required"`
	Reason      string `json:"reason"`
}

// PurgeContent 永久删除（R4），跳过恢复窗口，保留治理证据与审计日志。
func PurgeContent(req component.BetterRequest[PurgeContentReq]) component.Response {
	if err := contentdeleteservice.PurgeContent(req.UserId, contentdeleteservice.ContentType(req.Params.ContentType), req.Params.ContentID, req.Params.Reason); err != nil {
		return component.FailResponseError(err)
	}
	return component.SuccessResponseCode("操作成功", component.MessageContentPurgeSuccess, nil)
}

// PrivacyEraseReq 隐私紧急删除请求（R8，跳过恢复窗口立即彻底删除）。
type PrivacyEraseReq struct {
	ContentType string `json:"contentType" validate:"required,oneof=topic post"`
	ContentID   uint64 `json:"contentId" validate:"required"`
}

// PrivacyErase 隐私紧急删除（R8）：与永久删除等价，但更强调全渠道立即清除。
func PrivacyErase(req component.BetterRequest[PrivacyEraseReq]) component.Response {
	if err := contentdeleteservice.PrivacyEraseContent(req.UserId, contentdeleteservice.ContentType(req.Params.ContentType), req.Params.ContentID); err != nil {
		return component.FailResponseError(err)
	}
	return component.SuccessResponseCode("操作成功", component.MessageContentPrivacyErased, nil)
}

func formatDeletedAt(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.DateTime)
}

func canRestore(visibility string, retention string) bool {
	return visibility == topics.VisibilityUserDeleted && retention == topics.RetentionRecoverable
}

func canPermanentDelete(visibility string, retention string) bool {
	return visibility == topics.VisibilityUserDeleted && retention == topics.RetentionRecoverable
}

func lastCursorID(items []DeletedContentItem) uint64 {
	if len(items) == 0 {
		return 0
	}
	return items[len(items)-1].ID
}

func excerptOf(content string) string {
	runes := []rune(content)
	if len(runes) > 100 {
		return string(runes[:100])
	}
	return content
}

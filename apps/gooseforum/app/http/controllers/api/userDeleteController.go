package api

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/contentDeleteEvent"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/contentdeleteservice"
	"github.com/leancodebox/GooseForum/app/service/userservice"
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

// ContentEventReq 前端删除生命周期埋点上报（PRD R14）。
// 仅接受前端点击/确认类事件；删除/恢复/永久删除/隐私删除等后端事件
// 由 contentdeleteservice 在状态变更成功后自行记录。
type ContentEventReq struct {
	EventType   string `json:"eventType" validate:"required"`
	ContentType string `json:"contentType" validate:"required,oneof=topic post"`
	ContentID   uint64 `json:"contentId" validate:"required"`
}

// ReportContentEvent 记录前端删除点击/确认埋点。
func ReportContentEvent(req component.BetterRequest[ContentEventReq]) component.Response {
	switch contentDeleteEvent.EventType(req.Params.EventType) {
	case contentDeleteEvent.EventDeleteClicked, contentDeleteEvent.EventDeleteConfirmed:
	default:
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	if err := contentDeleteEvent.Record(contentDeleteEvent.Entity{
		EventType:   req.Params.EventType,
		ContentType: req.Params.ContentType,
		ContentID:   req.Params.ContentID,
		ActorID:     req.UserId,
	}); err != nil {
		slog.Error("record content delete click event failed", "eventType", req.Params.EventType, "contentId", req.Params.ContentID, "err", err)
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(true)
}

// MyContentItem 本人仍公开的话题/回复条目（PRD R9 批量管理）。
type MyContentItem struct {
	ID          uint64 `json:"id"`
	ContentType string `json:"contentType"`
	Title       string `json:"title"`
	Excerpt     string `json:"excerpt,omitempty"`
	TopicID     uint64 `json:"topicId,omitempty"`
	PostNo      uint64 `json:"postNo,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

// MyContentListReq 我的内容列表请求。
type MyContentListReq struct {
	ContentType string `form:"contentType" validate:"required,oneof=topic post"`
	CursorID    uint64 `form:"cursorId"`
	Limit       int    `form:"limit"`
}

// MyContentList 分页返回本人公开（ACTIVE）的话题/回复，供批量删除（R9）。
func MyContentList(req component.BetterRequest[MyContentListReq]) component.Response {
	limit := req.Params.Limit
	if limit <= 0 || limit > 30 {
		limit = 20
	}
	switch contentdeleteservice.ContentType(req.Params.ContentType) {
	case contentdeleteservice.ContentTypeTopic:
		entities := topics.GetActiveByUserPage(req.UserId, req.Params.CursorID, limit)
		hasMore := len(entities) > limit
		if hasMore {
			entities = entities[:limit]
		}
		items := make([]MyContentItem, 0, len(entities))
		for _, topic := range entities {
			items = append(items, MyContentItem{
				ID:          topic.Id,
				ContentType: "topic",
				Title:       topic.Title,
				Excerpt:     topic.Excerpt,
				CreatedAt:   topic.CreatedAt.Format(time.DateTime),
			})
		}
		return component.SuccessResponse(map[string]any{
			"items":        items,
			"hasMore":      hasMore,
			"nextCursorId": myContentCursorID(items),
		})
	case contentdeleteservice.ContentTypePost:
		entities := posts.GetActiveByUserPage(req.UserId, req.Params.CursorID, limit)
		hasMore := len(entities) > limit
		if hasMore {
			entities = entities[:limit]
		}
		items := make([]MyContentItem, 0, len(entities))
		for _, post := range entities {
			items = append(items, MyContentItem{
				ID:          post.Id,
				ContentType: "post",
				Title:       fmt.Sprintf("回复 #%d", post.PostNo),
				Excerpt:     excerptOf(post.Content),
				TopicID:     post.TopicId,
				PostNo:      post.PostNo,
				CreatedAt:   post.CreatedAt.Format(time.DateTime),
			})
		}
		return component.SuccessResponse(map[string]any{
			"items":        items,
			"hasMore":      hasMore,
			"nextCursorId": myContentCursorID(items),
		})
	default:
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
}

// BatchDeleteContentReq 批量删除请求（PRD R9）。
type BatchDeleteContentReq struct {
	ContentType string   `json:"contentType" validate:"required,oneof=topic post"`
	ContentIDs  []uint64 `json:"contentIds" validate:"required,min=1,max=50"`
	Force       bool     `json:"force"`    // 频率超限时的二次确认标记
	Password    string   `json:"password"` // force=true 时必须提供密码二次认证
}

// BatchDeleteResultItem 批量删除逐条结果。
type BatchDeleteResultItem struct {
	ContentID uint64 `json:"contentId"`
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
}

// batchDeleteWindow 与 batchDeleteLimit 定义批量删除频率控制（PRD R9：20 条/10 分钟）。
const (
	batchDeleteWindow = 10 * time.Minute
	batchDeleteLimit  = int64(20)
)

// BatchDeleteContent 批量删除本人内容（R9）。
// 10 分钟内删除超过 20 条时要求二次确认：force=true 且校验当前用户密码
// （防止账号被盗后无脑清空）。单条删除端点与隐私擦除同样计入该窗口。
func BatchDeleteContent(req component.BetterRequest[BatchDeleteContentReq]) component.Response {
	if len(req.Params.ContentIDs) == 0 {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	// 频率窗口同时计入普通删除与隐私紧急删除（PRD R9），避免通过隐私删除绕过限速。
	recent, err := contentDeleteEvent.CountRecentByActorEvents(req.UserId, []string{
		string(contentDeleteEvent.EventDeleted),
		string(contentDeleteEvent.EventPrivacyDelete),
	}, time.Now().Add(-batchDeleteWindow))
	if err != nil {
		slog.Error("count recent content deletes failed", "userId", req.UserId, "err", err)
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	if recent+int64(len(req.Params.ContentIDs)) > batchDeleteLimit {
		if !req.Params.Force {
			return component.FailResponseCode(component.MessageContentBatchConfirmRequired,
				component.MessageParams{"count": recent + int64(len(req.Params.ContentIDs))})
		}
		// force 必须通过密码二次认证，防止攻击者仅凭被盗会话绕过限速清空内容。
		user, userErr := users.Get(req.UserId)
		if userErr != nil || user.Id == 0 {
			return component.FailResponseCode(component.MessageUserFetchFailed, nil)
		}
		if _, verifyErr := users.Verify(user.Username, req.Params.Password); verifyErr != nil {
			return component.FailResponseCode(component.MessageAuthInvalidCredentials, nil)
		}
	}

	results := make([]BatchDeleteResultItem, 0, len(req.Params.ContentIDs))
	for _, contentID := range req.Params.ContentIDs {
		var deleteErr error
		switch contentdeleteservice.ContentType(req.Params.ContentType) {
		case contentdeleteservice.ContentTypeTopic:
			deleteErr = contentdeleteservice.DeleteTopicByUser(req.UserId, contentID)
		case contentdeleteservice.ContentTypePost:
			_, deleteErr = contentdeleteservice.DeletePostByUser(req.UserId, contentID)
		default:
			deleteErr = component.NewMessageError(component.MessageRequestInvalidParams, "无效的内容类型", nil)
		}
		item := BatchDeleteResultItem{ContentID: contentID, Success: deleteErr == nil}
		if deleteErr != nil {
			item.Message = deleteErr.Error()
		}
		results = append(results, item)
	}
	succeeded := 0
	for _, result := range results {
		if result.Success {
			succeeded++
		}
	}
	return component.SuccessResponse(map[string]any{
		"succeeded": succeeded,
		"failed":    len(results) - succeeded,
		"results":   results,
	})
}

func myContentCursorID(items []MyContentItem) uint64 {
	if len(items) == 0 {
		return 0
	}
	return items[len(items)-1].ID
}

// AccountCloseReq 注销账号请求（PRD R10）。
// mode=anonymize 保留内容但匿名化；mode=delete 先删除全部内容再注销。
// 注销为不可逆操作，必须提供当前密码二次认证（与批量删除 force 保持一致，
// 防止账号被盗后仅凭已登录会话即可注销清空）。
type AccountCloseReq struct {
	Mode     string `json:"mode" validate:"required,oneof=anonymize delete"`
	Password string `json:"password" validate:"required"`
}

// AccountClose 注销当前账号（R10）：
//   - anonymize：软删账号，历史内容保留并以「已注销用户」展示；
//   - delete：先按删除流程处理该账号全部话题/回复（他人回复不随删），再软删账号。
//
// 注销后 token_version 自增吊销全部会话。
func AccountClose(req component.BetterRequest[AccountCloseReq]) component.Response {
	if req.UserId == 0 {
		return component.FailResponseCode(component.MessageAuthRequired, nil)
	}
	if users.IsAccountClosed(req.UserId) {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}

	// 密码二次认证：注销不可逆，校验当前密码防止账号被盗后被无脑清空。
	user, userErr := users.Get(req.UserId)
	if userErr != nil || user.Id == 0 {
		return component.FailResponseCode(component.MessageUserFetchFailed, nil)
	}
	if _, verifyErr := users.Verify(user.Username, req.Params.Password); verifyErr != nil {
		return component.FailResponseCode(component.MessageAuthInvalidCredentials, nil)
	}

	if req.Params.Mode == "delete" {
		if err := contentdeleteservice.DeleteAllUserContent(req.UserId); err != nil {
			slog.Error("delete all user content on account close failed", "userId", req.UserId, "err", err)
			return component.FailResponseCode(component.MessageOperationFailed, nil)
		}
	}

	if err := users.CloseAccount(req.UserId); err != nil {
		slog.Error("close account failed", "userId", req.UserId, "err", err)
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	if err := users.IncrementTokenVersionWithDB(dbconnect.Connect(), req.UserId); err != nil {
		slog.Error("increment token version on account close failed", "userId", req.UserId, "err", err)
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	// 失效 user-info 缓存：users.CloseAccount 只改 DB（软删），不清缓存的话
	// authsessionservice.ValidateToken 会在缓存 TTL（2 分钟）内继续读到旧用户，
	// 旧 token 仍被接受，注销即时性被破坏（review E）。
	userservice.InvalidateUserInfoCache(req.UserId)
	slog.Info("account closed", "userId", req.UserId, "mode", req.Params.Mode)
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
				CanRestore:   canRestoreInWindow(topic.VisibilityStatus, topic.RetentionStatus, topic.DeletedAt.Time),
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
				CanRestore:   canRestoreInWindow(post.VisibilityStatus, post.RetentionStatus, deletedAt),
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

// canRestoreInWindow 在 canRestore 基础上叠加恢复窗口校验：墓碑态行无
// deleted_at，以 updated_at 近似（与 checkRestorable 一致）。避免"窗口已过、
// 定时清理尚未执行"时前端仍展示可恢复按钮但后端拒绝的 UX 不一致（PRD R3）。
func canRestoreInWindow(visibility string, retention string, deletedAt time.Time) bool {
	if !canRestore(visibility, retention) || deletedAt.IsZero() {
		return false
	}
	return time.Since(deletedAt) <= contentdeleteservice.RecoveryWindow
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

package api

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/eventbus"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/http/controllers/markdown2html"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topicCategoryIndex"
	"github.com/leancodebox/GooseForum/app/models/forum/topicUserAction"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/userFollow"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"github.com/leancodebox/GooseForum/app/service/eventhandlers"
	"github.com/leancodebox/GooseForum/app/service/fileusageservice"
	"github.com/leancodebox/GooseForum/app/service/moderationservice"
	"github.com/leancodebox/GooseForum/app/service/postservice"
	"github.com/leancodebox/GooseForum/app/service/topicunseenservice"
	"github.com/leancodebox/GooseForum/app/service/userservice"
)

// checkContentPolicy 检查内容是否命中敏感词。
// 返回 (pendingReview, word, err)：
//   - pendingReview=true：命中且配置为转人工审核，调用方应将 ProcessStatus 置为待审（2）。
//   - err!=nil：命中且配置为直接拦截，调用方应拒绝写入并返回错误。
func checkContentPolicy(userId uint64, content string, subjectType string, subjectId uint64) (bool, string, error) {
	securityConfig := hotdataserve.GetSecuritySettingsConfigCache()
	hit, word := moderationservice.CheckContentAllowedWithConfig(content, securityConfig)
	if !hit {
		return false, "", nil
	}
	excerpt := content
	if len(excerpt) > 100 {
		excerpt = excerpt[:100]
	}
	if securityConfig.SensitiveAction == "review" {
		moderationservice.SensitiveContentReview(userId, subjectType, subjectId, word, excerpt)
		return true, word, nil
	}
	moderationservice.SensitiveContentBlocked(userId, subjectType, subjectId, word, excerpt)
	return false, word, component.NewMessageError(
		component.MessageContentSensitiveBlocked,
		"内容包含敏感词，已被拦截",
		component.MessageParams{"word": word},
	)
}

// truncateExcerpt 截断内容为审核日志摘要。
func truncateExcerpt(content string) string {
	if len(content) <= 100 {
		return content
	}
	return content[:100]
}

func GetSiteStatistics() component.Response {
	return component.SuccessResponse(hotdataserve.GetSiteStatisticsData())
}

type WriteTopicReq struct {
	TopicId     uint64   `json:"topicId"`
	Content     string   `json:"content" validate:"required"`
	Title       string   `json:"title" validate:"required"`
	CategoryId  []uint64 `json:"categoryId" validate:"min=1,max=3"`
	TopicStatus int8     `json:"topicStatus" validate:"oneof=0 1"`
	Website     string   `json:"website,omitempty"` // 蜜罐字段，正常用户不可见
	CaptchaId   string   `json:"captchaId,omitempty"`
	CaptchaCode string   `json:"captchaCode,omitempty"`
}

// WriteTopic creates or updates a topic and its first post.
func WriteTopic(req component.BetterRequest[WriteTopicReq]) component.Response {
	// 获取发布设置
	postingConfig := hotdataserve.GetPostingSettingsConfigCache()

	userEntity, err := req.GetUser()
	if err != nil || userEntity.Id == 0 {
		return component.FailResponseCode(component.MessageUserFetchFailed, nil)
	}

	// 蜜罐字段：填了即机器，静默拒绝。
	if strings.TrimSpace(req.Params.Website) != "" {
		slog.Warn("honeypot_hit", "action", "topic.write", "ip", clientIPOf(req.GinContext), "userId", req.UserId)
		return component.SuccessResponse(true)
	}

	// 新用户高频发帖触发验证码
	rateLimitConfig := hotdataserve.GetRateLimitConfigCache()
	if newUserCaptchaRequired(userEntity.CreatedAt, req.UserId, "topic.write", rateLimitConfig.NewUserCaptchaAfterPosts, rateLimitConfig.NewUserCaptchaDays) {
		if ok, needCaptcha := checkCaptchaForRequest(req.GinContext, req.Params.CaptchaId, req.Params.CaptchaCode, true, rateLimitConfig.MinSubmitSeconds, "topic.write"); !ok {
			if needCaptcha {
				return component.FailResponseCode(component.MessageCaptchaRequired, component.MessageParams{"action": "topic.write"})
			}
			return component.FailResponseCode(component.MessageAuthCaptchaInvalid, nil)
		}
	}

	// 统一权限检查
	if _, err := component.CheckUserPermission(&userEntity, component.PermissionActionPost); err != nil {
		return component.FailResponseError(err)
	}

	if len(req.Params.Title) < postingConfig.TextControl.MinTitleLength {
		minLength := postingConfig.TextControl.MinTitleLength
		return component.FailResponseCode(
			component.MessageTopicTitleTooShort,

			component.MessageParams{"minLength": minLength})

	}

	if len(req.Params.Title) > postingConfig.TextControl.MaxTitleLength {
		maxLength := postingConfig.TextControl.MaxTitleLength
		return component.FailResponseCode(
			component.MessageTopicTitleTooLong,

			component.MessageParams{"maxLength": maxLength})

	}

	if len(req.Params.Content) < postingConfig.TextControl.MinPostLength {
		minLength := postingConfig.TextControl.MinPostLength
		return component.FailResponseCode(
			component.MessageTopicContentTooShort,

			component.MessageParams{"minLength": minLength})

	}

	if len(req.Params.Content) > postingConfig.TextControl.MaxPostLength {
		maxLength := postingConfig.TextControl.MaxPostLength
		return component.FailResponseCode(
			component.MessageTopicContentTooLong,

			component.MessageParams{"maxLength": maxLength})

	}

	// 检查新用户冷却时间
	if postingConfig.TextControl.NewUserPostCooldownMinutes > 0 {
		cooldownTime := userEntity.CreatedAt.Add(time.Duration(postingConfig.TextControl.NewUserPostCooldownMinutes) * time.Minute)
		if time.Now().Before(cooldownTime) {
			minutes := postingConfig.TextControl.NewUserPostCooldownMinutes
			availableAt := cooldownTime.Format("2006-01-02 15:04:05")
			return component.FailResponseCode(
				component.MessageTopicPostCooldown,

				component.MessageParams{"minutes": minutes, "availableAt": availableAt})

		}
	}

	if topics.CantWriteNew(req.UserId, 10) {
		return component.FailResponseCode(component.MessageTopicDailyLimit, nil)
	}
	// 敏感词检查（标题+内容）
	pendingReview, _, policyErr := checkContentPolicy(req.UserId, req.Params.Title+"\n"+req.Params.Content, "topic", req.Params.TopicId)
	if policyErr != nil {
		return component.FailResponseError(policyErr)
	}
	var topic topics.Entity
	var firstPost posts.Entity
	if req.Params.TopicId != 0 {
		topic = topics.Get(req.Params.TopicId)
		if topic.UserId != req.UserId {
			return component.FailResponseCode(component.MessageTopicOwnerMismatch, nil)
		}
		firstPost = posts.Get(topic.FirstPostId)
		if firstPost.Id == 0 {
			firstPost, _ = posts.GetByTopicPostNoAtOrAfter(topic.Id, 1)
		}
	} else {
		topic.UserId = req.UserId
	}
	topic.CategoryIds = req.Params.CategoryId
	topic.Status = req.Params.TopicStatus
	topic.Title = req.Params.Title
	topic.Excerpt = markdown2html.ExtractDescription(req.Params.Content, 200)
	topic.FirstImageURL = markdown2html.ExtractFirstImageURL(req.Params.Content)
	if pendingReview {
		topic.ProcessStatus = topics.ProcessStatusPending
	}
	if topic.Id > 0 {
		if firstPost.Id == 0 {
			return component.FailResponseCode(component.MessageTopicNotFound, nil)
		}
		firstPost.Content = req.Params.Content
		firstPost.RenderedHTML = ""
		firstPost.RenderedVersion = markdown2html.GetPostVersion()
		if pendingReview {
			firstPost.ProcessStatus = posts.ProcessStatusPending
		}
		if err := topics.Save(&topic); err != nil {
			return component.FailResponseCode(component.MessageOperationFailed, nil)
		}
		if err := posts.Save(&firstPost); err != nil {
			return component.FailResponseCode(component.MessageOperationFailed, nil)
		}
		fileusageservice.ReplaceTopic(topic.Id, req.UserId, firstPost.Content)
		hotdataserve.ClearTopicListCache()
		// 待审（pendingReview）内容未上线，跳过事件发布（通知/webhook/统计/积分），
		// 由审核批准路径补发对应事件，避免敏感内容在审核前外泄。
		if topic.Status == 1 && !pendingReview {
			eventbus.Publish(context.Background(), &eventhandlers.TopicUpdatedEvent{Topic: &topic, FirstPost: &firstPost})
		}
		if err := topicCategoryIndex.ReplaceTopicCategories(topic.Id, req.Params.CategoryId); err != nil {
			return component.FailResponseCode(component.MessageOperationFailed, nil)
		}
	} else {
		topic.PostCount = 1
		topic.PostSeq = 1
		topic.Posters = []topics.Poster{{UserID: req.UserId}}
		if err := topics.Create(&topic); err != nil {
			return component.FailResponseCode(component.MessageOperationFailed, nil)
		}
		firstPost = posts.Entity{
			TopicId:         topic.Id,
			PostNo:          1,
			UserId:          req.UserId,
			Content:         req.Params.Content,
			RenderedHTML:    "",
			RenderedVersion: markdown2html.GetPostVersion(),
			ProcessStatus:   posts.ProcessStatusNormal,
		}
		if pendingReview {
			firstPost.ProcessStatus = posts.ProcessStatusPending
		}
		if err := posts.Create(&firstPost); err != nil {
			return component.FailResponseCode(component.MessageOperationFailed, nil)
		}
		topic.FirstPostId = firstPost.Id
		topic.LastPostId = firstPost.Id
		now := time.Now()
		topic.LastPostedAt = &now
		if err := topics.Save(&topic); err != nil {
			return component.FailResponseCode(component.MessageOperationFailed, nil)
		}
		fileusageservice.ReplaceTopic(topic.Id, req.UserId, firstPost.Content)
		if topic.Status == 1 && !pendingReview {
			userStatistics.WriteTopic(req.UserId)
		}
		userservice.InvalidateUserPublicProfileCache(req.UserId)
		if err := topicCategoryIndex.ReplaceTopicCategories(topic.Id, req.Params.CategoryId); err != nil {
			return component.FailResponseCode(component.MessageOperationFailed, nil)
		}
		hotdataserve.ClearTopicListCache()
		if topic.Status == 1 && !pendingReview {
			eventbus.Publish(context.Background(), &eventhandlers.TopicPublishedEvent{Topic: &topic, FirstPost: &firstPost})
		}
		if err := topicunseenservice.MarkVisited(req.UserId, topic.Id, firstPost.Id, time.Now()); err != nil {
			slog.Warn("mark created topic visited failed", "userId", req.UserId, "topicId", topic.Id, "error", err)
		}
	}
	recordSuccessfulWrite(req.UserId, "topic.write")
	return component.SuccessResponse(topic.Id)
}

type TopicStatusReq struct {
	TopicId     uint64 `json:"topicId" validate:"required"`
	TopicStatus int8   `json:"topicStatus" validate:"oneof=0 1"`
}

func UpdateTopicStatus(req component.BetterRequest[TopicStatusReq]) component.Response {
	topic := topics.Get(req.Params.TopicId)
	if topic.Id == 0 {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}
	if topic.UserId != req.UserId {
		return component.FailResponseCode(component.MessageTopicOperationDenied, nil)
	}
	nextStatus := req.Params.TopicStatus
	if topic.Status == nextStatus {
		return component.SuccessResponse(true)
	}
	topic.Status = nextStatus
	if err := topics.Save(&topic); err != nil {
		return component.FailResponseCode(component.MessageTopicSaveFailed, nil)
	}
	firstPost := posts.Get(topic.FirstPostId)
	hotdataserve.ClearTopicListCache()
	if topic.Status == 1 {
		eventbus.Publish(context.Background(), &eventhandlers.TopicPublishedEvent{Topic: &topic, FirstPost: &firstPost})
	}
	return component.SuccessResponse(true)
}

type CreatePostReq struct {
	TopicId       uint64 `json:"topicId"`
	Content       string `json:"content"`
	ReplyToPostId uint64 `json:"replyToPostId"`
	Website       string `json:"website,omitempty"` // 蜜罐字段，正常用户不可见
	CaptchaId     string `json:"captchaId,omitempty"`
	CaptchaCode   string `json:"captchaCode,omitempty"`
}

func CreatePost(req component.BetterRequest[CreatePostReq]) component.Response {
	// 获取发布设置
	postingConfig := hotdataserve.GetPostingSettingsConfigCache()

	userEntity, err := req.GetUser()
	if err != nil || userEntity.Id == 0 {
		return component.FailResponseCode(component.MessageUserFetchFailed, nil)
	}

	// 蜜罐字段：填了即机器，静默拒绝。
	if strings.TrimSpace(req.Params.Website) != "" {
		slog.Warn("honeypot_hit", "action", "post.create", "ip", clientIPOf(req.GinContext), "userId", req.UserId)
		return component.SuccessResponse(true)
	}

	// 新用户高频发帖触发验证码
	rateLimitConfig := hotdataserve.GetRateLimitConfigCache()
	if newUserCaptchaRequired(userEntity.CreatedAt, req.UserId, "post.create", rateLimitConfig.NewUserCaptchaAfterPosts, rateLimitConfig.NewUserCaptchaDays) {
		if ok, needCaptcha := checkCaptchaForRequest(req.GinContext, req.Params.CaptchaId, req.Params.CaptchaCode, true, rateLimitConfig.MinSubmitSeconds, "post.create"); !ok {
			if needCaptcha {
				return component.FailResponseCode(component.MessageCaptchaRequired, component.MessageParams{"action": "post.create"})
			}
			return component.FailResponseCode(component.MessageAuthCaptchaInvalid, nil)
		}
	}

	// 统一权限检查
	if _, err := component.CheckUserPermission(&userEntity, component.PermissionActionComment); err != nil {
		return component.FailResponseError(err)
	}

	content := strings.TrimSpace(req.Params.Content)
	if len(content) < postingConfig.TextControl.MinPostLength {
		minLength := postingConfig.TextControl.MinPostLength
		return component.FailResponseCode(
			component.MessageCommentContentTooShort,

			component.MessageParams{"minLength": minLength})

	}

	if len(content) > postingConfig.TextControl.MaxPostLength {
		maxLength := postingConfig.TextControl.MaxPostLength
		return component.FailResponseCode(
			component.MessageCommentContentTooLong,

			component.MessageParams{"maxLength": maxLength})

	}

	// 评论也受发帖冷却限制
	if postingConfig.TextControl.NewUserPostCooldownMinutes > 0 {
		cooldownTime := userEntity.CreatedAt.Add(time.Duration(postingConfig.TextControl.NewUserPostCooldownMinutes) * time.Minute)
		if time.Now().Before(cooldownTime) {
			minutes := postingConfig.TextControl.NewUserPostCooldownMinutes
			availableAt := cooldownTime.Format("2006-01-02 15:04:05")
			return component.FailResponseCode(
				component.MessageCommentPostCooldown,

				component.MessageParams{"minutes": minutes, "availableAt": availableAt})

		}
	}

	topicEntity := topics.GetSimple(req.Params.TopicId)
	if topicEntity.Id == 0 {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}

	var parentPost posts.Entity
	if req.Params.ReplyToPostId > 0 {
		parentPost = posts.Get(req.Params.ReplyToPostId)
		if parentPost.Id == 0 || parentPost.TopicId != req.Params.TopicId {
			return component.FailResponseCode(component.MessageCommentParentPostMissing, nil)
		}
	}

	postEntity := &posts.Entity{
		TopicId:         req.Params.TopicId,
		Content:         content,
		RenderedHTML:    markdown2html.PostMarkdownToHTML(content),
		RenderedVersion: markdown2html.GetPostVersion(),
		UserId:          req.UserId,
		ReplyToPostId:   req.Params.ReplyToPostId,
	}

	// 敏感词检查
	pendingReview, _, policyErr := checkContentPolicy(req.UserId, content, "post", 0)
	if policyErr != nil {
		return component.FailResponseError(policyErr)
	}
	if pendingReview {
		postEntity.ProcessStatus = posts.ProcessStatusPending
	}

	err = postservice.CreateTopicPost(postEntity, topicEntity)
	if err != nil {
		return component.FailResponseCode(
			component.MessageCommentCreateFailed,

			component.MessageParams{"error": err.Error()})

	}
	if err := topicunseenservice.MarkVisited(req.UserId, topicEntity.Id, postEntity.Id, time.Now()); err != nil {
		slog.Warn("mark created post visited failed", "userId", req.UserId, "topicId", topicEntity.Id, "postId", postEntity.Id, "error", err)
	}
	fileusageservice.ReplacePost(postEntity.Id, req.UserId, postEntity.Content)
	if !pendingReview {
		userStatistics.WriteComment(req.UserId)
	}
	userservice.InvalidateUserPublicProfileCache(req.UserId)
	hotdataserve.ClearTopicListCache()

	// 获取父 post 作者 ID
	var parentPostAuthorID uint64
	if req.Params.ReplyToPostId > 0 {
		parentPostAuthorID = parentPost.UserId
	}

	// 待审（pendingReview）内容未上线，跳过事件发布（通知/webhook/统计/积分），
	// 批准后由 ReviewAction 补发对应事件，避免敏感内容在审核前外泄。
	if !pendingReview {
		eventbus.Publish(context.Background(), &eventhandlers.CommentCreatedEvent{
			TopicId:             topicEntity.Id,
			PostId:              postEntity.Id,
			UserId:              req.UserId,
			Content:             req.Params.Content,
			TopicAuthorId:       topicEntity.UserId,
			ReplyToPostId:       req.Params.ReplyToPostId,
			ReplyToPostAuthorId: parentPostAuthorID,
		})
	}
	// 发帖计数无条件累加（与 WriteTopic 的 topic.write 一致），
	// 保证待审内容也计入"新用户连续发帖"验证码门槛，避免滥用防护被绕过。
	recordSuccessfulWrite(req.UserId, "post.create")

	return component.SuccessResponse(map[string]any{
		"id":              postEntity.Id,
		"postNo":          postEntity.PostNo,
		"renderedContent": postEntity.RenderedHTML,
	})
}

type DeletePostReq struct {
	PostId uint64 `json:"postId"`
}

type UpdatePostReq struct {
	PostId  uint64 `json:"postId"`
	Content string `json:"content"`
}

func UpdatePost(req component.BetterRequest[UpdatePostReq]) component.Response {
	postingConfig := hotdataserve.GetPostingSettingsConfigCache()
	postEntity := posts.Get(req.Params.PostId)
	if postEntity.Id == 0 || postEntity.PostNo <= 1 {
		return component.FailResponseCode(component.MessagePostNotFound, nil)
	}
	if postEntity.UserId != req.UserId {
		return component.FailResponseCode(component.MessageTopicOperationDenied, nil)
	}

	content := strings.TrimSpace(req.Params.Content)
	if len(content) < postingConfig.TextControl.MinPostLength {
		minLength := postingConfig.TextControl.MinPostLength
		return component.FailResponseCode(
			component.MessageCommentContentTooShort,

			component.MessageParams{"minLength": minLength})

	}

	if len(content) > postingConfig.TextControl.MaxPostLength {
		maxLength := postingConfig.TextControl.MaxPostLength
		return component.FailResponseCode(
			component.MessageCommentContentTooLong,

			component.MessageParams{"maxLength": maxLength})

	}

	// 敏感词检查：block 直接拦截；review 转待审（编辑后的内容进入审核队列）
	pendingReview, _, policyErr := checkContentPolicy(req.UserId, content, "post", postEntity.Id)
	if policyErr != nil {
		return component.FailResponseError(policyErr)
	}
	if pendingReview {
		postEntity.ProcessStatus = posts.ProcessStatusPending
	}

	postEntity.Content = content
	postEntity.RenderedHTML = markdown2html.PostMarkdownToHTML(content)
	postEntity.RenderedVersion = markdown2html.GetPostVersion()
	if err := posts.Save(&postEntity); err != nil {
		return component.FailResponseCode(
			component.MessagePostUpdateFailed,

			component.MessageParams{"error": err.Error()})

	}
	fileusageservice.ReplacePost(postEntity.Id, req.UserId, postEntity.Content)

	return component.SuccessResponse(map[string]any{
		"id":              postEntity.Id,
		"postNo":          postEntity.PostNo,
		"content":         postEntity.Content,
		"renderedContent": postEntity.RenderedHTML,
		"updatedAt":       postEntity.UpdatedAt.Format(time.DateTime),
	})
}

func DeletePost(req component.BetterRequest[DeletePostReq]) component.Response {
	postEntity := posts.Get(req.Params.PostId)
	if postEntity.Id == 0 || postEntity.PostNo <= 1 {
		return component.FailResponseCode(component.MessagePostNotFound, nil)
	}
	if postEntity.UserId != req.UserId {
		return component.FailResponseCode(component.MessageTopicOperationDenied, nil)
	}
	posts.DeleteEntity(&postEntity)
	topicEntity := topics.GetSimple(postEntity.TopicId)
	if topicEntity.Id > 0 {
		postservice.SyncTopicPostStats(topicEntity, postEntity, true)
		hotdataserve.ClearTopicListCache()
	}
	return component.SuccessResponse(true)
}

type LikeTopicReq struct {
	TopicId uint64 `json:"topicId"`
	Action  int    `json:"action" validate:"min=1,max=2"` // 1 点赞，2 取消
}

func LikeTopic(req component.BetterRequest[LikeTopicReq]) component.Response {
	topicEntity := topics.Get(req.Params.TopicId)
	if topicEntity.Id == 0 {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}
	state := topicUserAction.GetByTopicId(req.UserId, topicEntity.Id)
	targetLiked := req.Params.Action == 1
	if state.Id == 0 && !targetLiked {
		return component.SuccessResponse(true)
	}
	if state.Id != 0 && (state.LikedAt != nil) == targetLiked {
		return component.SuccessResponse(true)
	}
	if topicUserAction.SetLiked(req.UserId, topicEntity.Id, targetLiked) {
		if req.Params.Action == 1 {
			topics.IncrementLike(topicEntity)
			userStatistics.LikeTopic(topicEntity.UserId)
			userStatistics.GivenLike(req.UserId)
			userservice.InvalidateUserPublicProfileCache(topicEntity.UserId)
			userservice.InvalidateUserPublicProfileCache(req.UserId)
			hotdataserve.ClearTopicListCache()

			// 发送点赞事件
			eventbus.Publish(context.Background(), &eventhandlers.TopicLikedEvent{
				UserId:  topicEntity.UserId,
				TopicId: topicEntity.Id,
				Title:   topicEntity.Title,
				LikerId: req.UserId,
			})
		} else {
			topics.DecrementLike(topicEntity)
			userStatistics.CancelLikeTopic(topicEntity.UserId)
			userStatistics.CancelGivenLike(req.UserId)
			userservice.InvalidateUserPublicProfileCache(topicEntity.UserId)
			userservice.InvalidateUserPublicProfileCache(req.UserId)
			hotdataserve.ClearTopicListCache()
		}
	}
	return component.SuccessResponse(true)
}

type BookmarkTopicReq struct {
	TopicId uint64 `json:"topicId"`
	Action  int    `json:"action" validate:"min=1,max=2"` // 1 收藏，2 取消
}

func BookmarkTopic(req component.BetterRequest[BookmarkTopicReq]) component.Response {
	topicEntity := topics.Get(req.Params.TopicId)
	if topicEntity.Id == 0 {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}

	state := topicUserAction.GetByTopicId(req.UserId, topicEntity.Id)
	targetBookmarked := req.Params.Action == 1
	if state.Id == 0 && !targetBookmarked {
		return component.SuccessResponse(true)
	}
	if state.Id != 0 && (state.BookmarkedAt != nil) == targetBookmarked {
		return component.SuccessResponse(true)
	}

	if topicUserAction.SetBookmarked(req.UserId, topicEntity.Id, targetBookmarked) {
		updateBookmarkStats(req.UserId, targetBookmarked)
		userservice.InvalidateUserPublicProfileCache(req.UserId)
	}
	return component.SuccessResponse(true)
}

func updateBookmarkStats(userID uint64, bookmarked bool) {
	if bookmarked {
		userStatistics.Collection(userID)
		return
	}
	userStatistics.CancelCollection(userID)
}

type WatchTopicReq struct {
	TopicId uint64 `json:"topicId"`
	Action  int    `json:"action" validate:"min=1,max=2"` // 1 关注，2 取消
}

func WatchTopic(req component.BetterRequest[WatchTopicReq]) component.Response {
	topicEntity := topics.Get(req.Params.TopicId)
	if topicEntity.Id == 0 {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}

	state := topicUserAction.GetByTopicId(req.UserId, topicEntity.Id)
	targetWatched := req.Params.Action == 1
	if state.Id == 0 && !targetWatched {
		return component.SuccessResponse(true)
	}
	if state.Id != 0 && (state.WatchedAt != nil) == targetWatched {
		return component.SuccessResponse(true)
	}

	topicUserAction.SetWatched(req.UserId, topicEntity.Id, targetWatched)
	return component.SuccessResponse(true)
}

type FollowUserReq struct {
	Id     uint64 `json:"id"`
	Action int    `json:"action" validate:"min=1,max=2"` // 1 关注，2 取消
}

func FollowUser(req component.BetterRequest[FollowUserReq]) component.Response {
	userEntity, _ := users.Get(req.Params.Id)
	if userEntity.Id == 0 {
		return component.FailResponseCode(component.MessageUserNotFound, nil)
	}
	userFollowEntity := userFollow.GetByUserId(req.UserId, req.Params.Id)
	if userFollowEntity.Id == 0 {
		userFollowEntity.UserId = req.UserId
		userFollowEntity.FollowUserId = req.Params.Id
	}
	var targetStatus int
	if req.Params.Action == 1 {
		targetStatus = 1
	} else {
		targetStatus = 0
	}

	if userFollowEntity.Status == targetStatus {
		return component.SuccessResponse(true)
	}
	userFollowEntity.Status = targetStatus
	if userFollow.SaveOrCreateById(&userFollowEntity) > 0 {
		if req.Params.Action == 1 {
			userStatistics.Following(req.UserId)
			userStatistics.Follower(req.Params.Id)
			userservice.InvalidateUserPublicProfileCache(req.UserId)
			userservice.InvalidateUserPublicProfileCache(req.Params.Id)

			// 发送关注通知
			followerUser, _ := req.GetUser()
			eventbus.Publish(context.Background(), &eventhandlers.UserFollowedEvent{
				UserId:       req.Params.Id,
				FollowerId:   req.UserId,
				FollowerName: followerUser.Username,
			})
		} else {
			userStatistics.CancelFollowing(req.UserId)
			userStatistics.CancelFollower(req.Params.Id)
			userservice.InvalidateUserPublicProfileCache(req.UserId)
			userservice.InvalidateUserPublicProfileCache(req.Params.Id)
		}
	}
	return component.SuccessResponse(true)
}

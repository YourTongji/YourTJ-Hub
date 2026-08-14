package api

import (
	"context"
	"log/slog"
	"strings"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/contentdeleteservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/llmsservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/postservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/topicunseenservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"gorm.io/gorm"
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
	return writeTopic(req, false)
}

// writeTopic is the shared topic write core. The agent flag skips
// browser-only guards (honeypot, captcha, new-user cooldown); every other
// rule and side effect behaves identically for human and Agent writers.
func writeTopic(req component.BetterRequest[WriteTopicReq], agent bool) component.Response {
	// 获取发布设置
	postingConfig := hotdataserve.GetPostingSettingsConfigCache()

	userEntity, err := req.GetUser()
	if err != nil || userEntity.Id == 0 {
		return component.FailResponseCode(component.MessageUserFetchFailed, nil)
	}
	// 蜜罐字段：填了即机器，静默拒绝。Agent 请求不携带该字段。
	if !agent && strings.TrimSpace(req.Params.Website) != "" {
		slog.Warn("honeypot_hit", "action", "topic.write", "ip", clientIPOf(req.GinContext), "userId", req.UserId)
		return component.SuccessResponse(true)
	}

	// 新用户高频发帖触发验证码（浏览器专用，Agent 跳过）
	if !agent {
		rateLimitConfig := hotdataserve.GetRateLimitConfigCache()
		if newUserCaptchaRequired(userEntity.CreatedAt, req.UserId, "topic.write", rateLimitConfig.NewUserCaptchaAfterPosts, rateLimitConfig.NewUserCaptchaDays) {
			if ok, needCaptcha := checkCaptchaForRequest(req.GinContext, req.Params.CaptchaId, req.Params.CaptchaCode, true, rateLimitConfig.MinSubmitSeconds, "topic.write"); !ok {
				if needCaptcha {
					return component.FailResponseCode(component.MessageCaptchaRequired, component.MessageParams{"action": "topic.write"})
				}
				return component.FailResponseCode(component.MessageAuthCaptchaInvalid, nil)
			}
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

	// 检查新用户冷却时间（浏览器专用，Agent 跳过）
	if !agent && postingConfig.TextControl.NewUserPostCooldownMinutes > 0 {
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
		// wiki 分站页面由 wiki 修订审核流程管理，禁止经论坛编辑端点直接改写
		// topic 行/首楼，避免绕过 wiki_page_revisions 版本流（review N1）。
		if topic.TopicType == topics.TopicTypeWiki {
			return component.FailResponseCode(component.MessageTopicOperationDenied, nil)
		}
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
	topic.ImageUrls = markdown2html.ExtractImageURLs(req.Params.Content)
	if pendingReview {
		topic.ProcessStatus = topics.ProcessStatusPending
	}
	// 覆写首帖正文前保存旧内容/旧状态：存量帖子（无版本快照）首次编辑时
	// 惰性播种 v1 = 旧正文；v1 的状态必须取旧状态，而非本次待审覆写后的
	// Pending（否则此前公开的旧正文会对非版主永久隐藏，review 发现）。
	oldContent := firstPost.Content
	oldProcessStatus := firstPost.ProcessStatus
	if topic.Id > 0 {
		if firstPost.Id == 0 {
			return component.FailResponseCode(component.MessageTopicNotFound, nil)
		}
		firstPost.Content = req.Params.Content
		firstPost.RenderedHTML = markdown2html.PostMarkdownToHTML(req.Params.Content)
		firstPost.RenderedVersion = markdown2html.GetPostVersion()
		if pendingReview {
			firstPost.ProcessStatus = posts.ProcessStatusPending
		}
	}
	// 记录是否为编辑：事务内 topics.CreateTx 会回填 topic.Id，因此提交后分支判断
	// 必须使用此快照（isEdit），不能复用已被回填的 topic.Id。
	isEdit := topic.Id > 0
	// 单事务原子提交：话题 + 首帖 + 指针（首/末帖 ID、最后回复时间）+ 分类索引。
	// 任一步失败整体回滚，不留孤立话题/缺首帖/缺分类索引；事件与缓存失效仅在提交后执行。
	err = db.Connect().Transaction(func(tx *gorm.DB) error {
		if isEdit {
			// 锁序与 UpdatePost 首楼分支保持一致（posts → topics）：先写
			// firstPost 行、再写 topic 派生字段。若这里先锁 topics 再锁
			// posts，与 UpdatePost 的 posts→topics 形成锁环，同一话题双
			// 路径并发编辑时可能死锁（数据库回滚其中一个事务）。
			if err := posts.SaveTx(tx, &firstPost); err != nil {
				return err
			}
			// 首楼编辑追加版本历史 + 最后编辑者/时间（与 UpdatePost 同语义）。
			// 存量帖子无版本快照时用旧正文惰性播种 v1（状态取编辑前 oldProcessStatus）。
			if err := postservice.AppendPostRevisionWithOld(tx, &firstPost, req.UserId, firstPost.ProcessStatus, oldContent, oldProcessStatus); err != nil {
				return err
			}
			// 只更新话题编辑者可写的字段（标题/分类/上下架/摘要/首图等），
			// 不整行保存事务外读取的 topic——整行 Save 会把并发新建回复
			// 刚写入的 post_count/post_seq/posters/last_post_id/
			// last_posted_at 回写为旧值，导致统计倒退或 post_seq 复写后
			// 新回复撞 post_no 唯一约束（与 UpdatePost 首楼分支同源问题）。
			if err := topics.UpdateTopicEditableTx(tx, &topic); err != nil {
				return err
			}
		} else {
			topic.PostCount = 1
			topic.PostSeq = 1
			topic.Posters = []topics.Poster{{UserID: req.UserId}}
			if err := topics.CreateTx(tx, &topic); err != nil {
				return err
			}
			firstPost = posts.Entity{
				TopicId:         topic.Id,
				PostNo:          1,
				UserId:          req.UserId,
				Content:         req.Params.Content,
				RenderedHTML:    markdown2html.PostMarkdownToHTML(req.Params.Content),
				RenderedVersion: markdown2html.GetPostVersion(),
				ProcessStatus:   posts.ProcessStatusNormal,
			}
			if pendingReview {
				firstPost.ProcessStatus = posts.ProcessStatusPending
			}
			if err := posts.CreateTx(tx, &firstPost); err != nil {
				return err
			}
			// 新话题播种版本 v1（editor = 作者）。
			if err := postservice.SeedPostRevision(tx, &firstPost); err != nil {
				return err
			}
			topic.FirstPostId = firstPost.Id
			topic.LastPostId = firstPost.Id
			now := time.Now()
			topic.LastPostedAt = &now
			if err := topics.SaveTx(tx, &topic); err != nil {
				return err
			}
		}
		return topicCategoryIndex.ReplaceTopicCategoriesTx(tx, topic.Id, req.Params.CategoryId)
	})
	if err != nil {
		slog.Error("topic write transaction failed", "topicId", topic.Id, "isEdit", isEdit, "err", err)
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}

	// ---- 事务已提交：此后才允许缓存失效、统计与事件发布 ----
	fileusageservice.ReplaceTopic(topic.Id, req.UserId, firstPost.Content)
	hotdataserve.ClearTopicListCache()
	if isEdit {
		// 编辑分支：无条件重建搜索索引——下架（TopicStatus=0）或转入待审
		// （ProcessStatus=Pending）时 BuildSingleTopicSearchDocument 会把文档
		// 从索引删除，避免非公开内容残留在公共搜索（issue #132）。
		// LLMS 投影缓存同步失效，避免下架内容在 10s 窗口内继续导出。
		llmsservice.ClearCache()
		if _, err := searchservice.BuildSingleTopicSearchDocument(&topic, &firstPost); err != nil {
			slog.Error("failed to rebuild topic search document", "topicId", topic.Id, "err", err)
		}
		// 待审（pendingReview）内容未上线，跳过业务事件发布（通知/webhook/统计/积分），
		// 由审核批准路径补发对应事件，避免敏感内容在审核前外泄。
		if topic.Status == 1 && !pendingReview {
			eventbus.Publish(context.Background(), &eventhandlers.TopicUpdatedEvent{Topic: &topic, FirstPost: &firstPost})
		}
	} else {
		if topic.Status == 1 && !pendingReview {
			userStatistics.WriteTopic(req.UserId)
		}
		userservice.InvalidateUserPublicProfileCache(req.UserId)
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
	// wiki 分站页面上下架/发布由 wiki 修订流程管理，禁止经论坛端点操作。
	if topic.TopicType == topics.TopicTypeWiki {
		return component.FailResponseCode(component.MessageTopicOperationDenied, nil)
	}
	if topic.UserId != req.UserId {
		return component.FailResponseCode(component.MessageTopicOperationDenied, nil)
	}
	nextStatus := req.Params.TopicStatus
	if nextStatus == 1 && topic.ProcessStatus != topics.ProcessStatusNormal {
		return component.FailResponseCode(component.MessageTopicOperationDenied, nil)
	}
	if topic.Status == nextStatus {
		return component.SuccessResponse(true)
	}
	topic.Status = nextStatus
	if err := topics.Save(&topic); err != nil {
		return component.FailResponseCode(component.MessageTopicSaveFailed, nil)
	}
	firstPost := posts.Get(topic.FirstPostId)
	hotdataserve.ClearTopicListCache()
	// 无条件重建搜索索引（issue #132）：1→0（取消发布）时
	// BuildSingleTopicSearchDocument 会把文档从索引删除，避免已下架话题
	// 残留在公共搜索；0→1（重新发布）时 upsert 恢复，幂等无害。
	// 不发布 TopicUpdatedEvent：下架属用户隐私操作，不触发 webhook 通知。
	llmsservice.ClearCache()
	if _, err := searchservice.BuildSingleTopicSearchDocument(&topic, &firstPost); err != nil {
		slog.Error("failed to rebuild topic search document", "topicId", topic.Id, "err", err)
	}
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
	return createPost(req, false)
}

// createPost is the shared post write core. The agent flag skips browser-only
// guards (honeypot, captcha, new-user cooldown); every other rule and side
// effect behaves identically for human and Agent writers.
func createPost(req component.BetterRequest[CreatePostReq], agent bool) component.Response {
	postingConfig := hotdataserve.GetPostingSettingsConfigCache()

	userEntity, err := req.GetUser()
	if err != nil || userEntity.Id == 0 {
		return component.FailResponseCode(component.MessageUserFetchFailed, nil)
	}
	// 蜜罐字段：填了即机器，静默拒绝。Agent 请求不携带该字段。
	if !agent && strings.TrimSpace(req.Params.Website) != "" {
		slog.Warn("honeypot_hit", "action", "post.create", "ip", clientIPOf(req.GinContext), "userId", req.UserId)
		return component.SuccessResponse(true)
	}

	// 新用户高频发帖触发验证码（浏览器专用，Agent 跳过）
	if !agent {
		rateLimitConfig := hotdataserve.GetRateLimitConfigCache()
		if newUserCaptchaRequired(userEntity.CreatedAt, req.UserId, "post.create", rateLimitConfig.NewUserCaptchaAfterPosts, rateLimitConfig.NewUserCaptchaDays) {
			if ok, needCaptcha := checkCaptchaForRequest(req.GinContext, req.Params.CaptchaId, req.Params.CaptchaCode, true, rateLimitConfig.MinSubmitSeconds, "post.create"); !ok {
				if needCaptcha {
					return component.FailResponseCode(component.MessageCaptchaRequired, component.MessageParams{"action": "post.create"})
				}
				return component.FailResponseCode(component.MessageAuthCaptchaInvalid, nil)
			}
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

	// 评论也受发帖冷却限制（浏览器专用，Agent 跳过）
	if !agent && postingConfig.TextControl.NewUserPostCooldownMinutes > 0 {
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
	if topicEntity.Id == 0 || !forum.CanViewTopicSimple(&topicEntity, req.UserId) {
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
	PostId   uint64 `json:"postId"`
	Force    bool   `json:"force"`
	Password string `json:"password"`
}

type UpdatePostReq struct {
	PostId  uint64 `json:"postId"`
	Content string `json:"content"`
}

// UpdatePost 编辑帖子内容。首楼（PostNo=1）与回复同权限：作者本人。
// 首楼编辑联动话题摘要/首图（列表卡片与搜索文档派生自首楼）并重建
// 搜索索引；所有内容编辑在同一事务内追加版本历史（post_revisions，
// 用户只读查看）并更新最后编辑者/时间。
func UpdatePost(req component.BetterRequest[UpdatePostReq]) component.Response {
	postingConfig := hotdataserve.GetPostingSettingsConfigCache()
	postEntity := posts.Get(req.Params.PostId)
	if postEntity.Id == 0 {
		return component.FailResponseCode(component.MessagePostNotFound, nil)
	}
	if postEntity.UserId != req.UserId {
		return component.FailResponseCode(component.MessageTopicOperationDenied, nil)
	}
	// 话题可见性守卫：与 LikePost/BookmarkPost 一致，禁止在读路径不可见（隐藏/封禁）的话题中编辑回复
	topicEntity := topics.GetSimple(postEntity.TopicId)
	if topicEntity.Id == 0 || !forum.CanViewTopicSimple(&topicEntity, req.UserId) {
		return component.FailResponseCode(component.MessagePostNotFound, nil)
	}
	// wiki 分站首楼由 wiki_page_revisions 版本流独占（review High：此前
	// UpdatePost 可直改 wiki 首楼，绕过版本流 + 写时敏感词拦截，导致 posts
	// 行与 published_revision_no 指向的修订脱同步，下次 wiki Edit 静默覆盖）。
	// 回复流（post_no>1）不受影响。
	if topicEntity.TopicType == topics.TopicTypeWiki {
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
	// 覆写正文前捕获旧状态：存量帖子首次编辑惰性播种 v1 时，v1 的状态
	// 必须取编辑前的旧状态，而非被本次待审覆写后的 Pending（否则此前
	// 公开的旧正文会对非版主永久隐藏，post_revisions review 发现）。
	oldProcessStatus := postEntity.ProcessStatus
	if pendingReview {
		postEntity.ProcessStatus = posts.ProcessStatusPending
	}
	// 覆写正文前保存旧内容：存量帖子（无版本快照）首次编辑时惰性播种
	// v1 = 旧正文，避免原始正文永久丢失（已有 v1 的帖子走正常追加）。
	oldContent := postEntity.Content
	postEntity.Content = content
	postEntity.RenderedHTML = markdown2html.PostMarkdownToHTML(content)
	postEntity.RenderedVersion = markdown2html.GetPostVersion()

	isFirstPost := postEntity.PostNo == 1
	if isFirstPost {
		// 首楼是话题正文：摘要/首图与待审状态随正文联动（与 writeTopic
		// 编辑分支同语义），保证列表卡片、搜索文档与正文一致。
		topicEntity.Excerpt = markdown2html.ExtractDescription(content, 200)
		topicEntity.FirstImageURL = markdown2html.ExtractFirstImageURL(content)
		topicEntity.ImageUrls = markdown2html.ExtractImageURLs(content)
		if pendingReview {
			topicEntity.ProcessStatus = topics.ProcessStatusPending
		}
	}

	now := time.Now()
	if err := db.Connect().Transaction(func(tx *gorm.DB) error {
		if err := posts.SaveTx(tx, &postEntity); err != nil {
			return err
		}
		// 版本历史与帖子更新同事务：追加失败则整体回滚，不留无版本的编辑。
		if err := postservice.AppendPostRevisionWithOld(tx, &postEntity, req.UserId, postEntity.ProcessStatus, oldContent, oldProcessStatus); err != nil {
			return err
		}
		if isFirstPost {
			// 首楼编辑只更新由正文派生的字段，绝不整行保存事务外读取的
			// topicEntity——整行 Save 会把并发回复刚写入的 post_count/
			// post_seq/posters/last_post_id/last_posted_at 回写为旧值，
			// 导致统计倒退或 post_seq 复写后新回复撞 post_no 唯一约束。
			if err := topics.UpdateFirstPostDerivedTx(tx, &topicEntity); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return component.FailResponseCode(
			component.MessagePostUpdateFailed,

			component.MessageParams{"error": err.Error()})
	}
	postEntity.LastEditorId = req.UserId
	postEntity.LastEditedAt = &now

	fileusageservice.ReplacePost(postEntity.Id, req.UserId, postEntity.Content)
	if isFirstPost {
		// 首楼编辑联动：附件重映射、列表缓存、搜索索引与业务事件
		// （TopicUpdatedEvent 驱动通知/webhook/搜索），与 writeTopic 编辑分支一致。
		fileusageservice.ReplaceTopic(topicEntity.Id, req.UserId, postEntity.Content)
		hotdataserve.ClearTopicListCache()
		llmsservice.ClearCache()
		if _, err := searchservice.BuildSingleTopicSearchDocument(&topicEntity, &postEntity); err != nil {
			slog.Error("failed to rebuild topic search document", "topicId", topicEntity.Id, "err", err)
		}
		if topicEntity.Status == 1 && !pendingReview {
			eventbus.Publish(context.Background(), &eventhandlers.TopicUpdatedEvent{Topic: &topicEntity, FirstPost: &postEntity})
		}
	} else {
		// 回复编辑不发布事件，同步清理 LLMS 投影缓存。
		llmsservice.ClearCache()
	}

	return component.SuccessResponse(map[string]any{
		"id":              postEntity.Id,
		"postNo":          postEntity.PostNo,
		"content":         postEntity.Content,
		"renderedContent": postEntity.RenderedHTML,
		"updatedAt":       postEntity.UpdatedAt.Format(time.DateTime),
		"lastEditorId":    postEntity.LastEditorId,
		"lastEditedAt":    postEntity.LastEditedAt.Format(time.DateTime),
		"revisionCount":   postRevisions.CountByPostIds([]uint64{postEntity.Id})[postEntity.Id],
	})
}

func DeletePost(req component.BetterRequest[DeletePostReq]) component.Response {
	if err := contentdeleteservice.CheckDeleteRate(req.UserId, 1, req.Params.Force, req.Params.Password); err != nil {
		return component.FailResponseError(err)
	}
	postEntity := posts.Get(req.Params.PostId)
	if postEntity.Id == 0 || postEntity.PostNo <= 1 {
		return component.FailResponseCode(component.MessagePostNotFound, nil)
	}
	if postEntity.UserId != req.UserId {
		return component.FailResponseCode(component.MessageTopicOperationDenied, nil)
	}
	// 回复删除沿用读路径可见性守卫，避免隐藏或封禁话题中的回复继续被写操作探测。
	topicEntity := topics.GetSimple(postEntity.TopicId)
	if topicEntity.Id == 0 || !forum.CanViewTopicSimple(&topicEntity, req.UserId) {
		return component.FailResponseCode(component.MessagePostNotFound, nil)
	}

	// PR #99 删除生命周期：软删 + 墓碑态（保留讨论树），替代 dev 的物理删除实现。
	result, err := contentdeleteservice.DeletePostByUser(req.UserId, req.Params.PostId)
	if err != nil {
		return component.FailResponseError(err)
	}
	return component.SuccessResponse(result)
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
	// 仅"新增互动"要求话题可见；已持状态者可取消（Action=2）清理对已隐藏/封禁话题的
	// 既有点赞，避免 like_count 与 user_action 行被永久卡住（无状态者仍按不可见拒绝）。
	if !forum.CanViewTopicSimple(&topicEntity, req.UserId) && !(req.Params.Action == 2 && state.Id != 0) {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}
	targetLiked := req.Params.Action == 1
	if state.Id == 0 && !targetLiked {
		return component.SuccessResponse(true)
	}
	if state.Id != 0 && (state.LikedAt != nil) == targetLiked {
		return component.SuccessResponse(true)
	}
	if topicUserAction.SetLiked(req.UserId, topicEntity.Id, targetLiked) {
		// 仅状态迁移时执行统计与事件副作用（并发重复请求不会重复计数）
		if targetLiked {
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
	// 仅"新增互动"要求话题可见；已持状态者可取消（Action=2）清理对已隐藏/封禁话题的既有收藏。
	if !forum.CanViewTopicSimple(&topicEntity, req.UserId) && !(req.Params.Action == 2 && state.Id != 0) {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}

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
	// 仅"新增互动"要求话题可见；已持状态者可取消（Action=2）退订对已隐藏/封禁话题的关注。
	if !forum.CanViewTopicSimple(&topicEntity, req.UserId) && !(req.Params.Action == 2 && state.Id != 0) {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}

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

type LikePostReq struct {
	PostId uint64 `json:"postId" validate:"required"`
	Action int    `json:"action" validate:"min=1,max=2"` // 1 点赞，2 取消
}

// LikePost 楼层点赞/取消点赞，计数以 post_user_action 行数聚合
func LikePost(req component.BetterRequest[LikePostReq]) component.Response {
	postEntity := posts.Get(req.Params.PostId)
	if postEntity.Id == 0 {
		return component.FailResponseCode(component.MessagePostNotFound, nil)
	}
	topicEntity := topics.GetSimple(postEntity.TopicId)
	if topicEntity.Id == 0 {
		return component.FailResponseCode(component.MessagePostNotFound, nil)
	}
	state := postUserAction.GetByPostId(req.UserId, postEntity.Id)
	// 仅"新增互动"要求话题可见；已持状态者可取消（Action=2）清理对已隐藏/封禁话题的既有点赞。
	if !forum.CanViewTopicSimple(&topicEntity, req.UserId) && !(req.Params.Action == 2 && state.Id != 0) {
		return component.FailResponseCode(component.MessagePostNotFound, nil)
	}

	targetLiked := req.Params.Action == 1
	if state.Id == 0 && !targetLiked {
		return component.SuccessResponse(true)
	}
	if state.Id != 0 && (state.LikedAt != nil) == targetLiked {
		return component.SuccessResponse(true)
	}

	if postUserAction.SetLiked(req.UserId, postEntity.Id, targetLiked) {
		// 仅状态迁移时执行统计与事件副作用（并发重复请求不会重复计数）
		if targetLiked {
			userStatistics.GivenLike(req.UserId)
			// 楼层点赞计入作者"获赞"统计，并发布点赞事件（动态/徽章/通知）
			userStatistics.LikeTopic(postEntity.UserId)
			eventbus.Publish(context.Background(), &eventhandlers.PostLikedEvent{
				UserId:     postEntity.UserId,
				PostId:     postEntity.Id,
				TopicId:    postEntity.TopicId,
				TopicTitle: topicEntity.Title,
				LikerId:    req.UserId,
			})
		} else {
			userStatistics.CancelGivenLike(req.UserId)
			userStatistics.CancelLikeTopic(postEntity.UserId)
		}
		userservice.InvalidateUserPublicProfileCache(postEntity.UserId)
		userservice.InvalidateUserPublicProfileCache(req.UserId)
	}
	return component.SuccessResponse(true)
}

type BookmarkPostReq struct {
	PostId uint64 `json:"postId" validate:"required"`
	Action int    `json:"action" validate:"min=1,max=2"` // 1 收藏，2 取消
}

// BookmarkPost 楼层收藏/取消收藏
func BookmarkPost(req component.BetterRequest[BookmarkPostReq]) component.Response {
	postEntity := posts.Get(req.Params.PostId)
	if postEntity.Id == 0 {
		return component.FailResponseCode(component.MessagePostNotFound, nil)
	}
	topicEntity := topics.GetSimple(postEntity.TopicId)
	if topicEntity.Id == 0 {
		return component.FailResponseCode(component.MessagePostNotFound, nil)
	}
	state := postUserAction.GetByPostId(req.UserId, postEntity.Id)
	// 仅"新增互动"要求话题可见；已持状态者可取消（Action=2）清理对已隐藏/封禁话题的既有收藏。
	if !forum.CanViewTopicSimple(&topicEntity, req.UserId) && !(req.Params.Action == 2 && state.Id != 0) {
		return component.FailResponseCode(component.MessagePostNotFound, nil)
	}

	targetBookmarked := req.Params.Action == 1
	if state.Id == 0 && !targetBookmarked {
		return component.SuccessResponse(true)
	}
	if state.Id != 0 && (state.BookmarkedAt != nil) == targetBookmarked {
		return component.SuccessResponse(true)
	}

	if postUserAction.SetBookmarked(req.UserId, postEntity.Id, targetBookmarked) {
		updateBookmarkStats(req.UserId, targetBookmarked)
		userservice.InvalidateUserPublicProfileCache(req.UserId)
	}
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

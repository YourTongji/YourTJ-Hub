package forum

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/pageutil"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/badgeservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/postservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/topicunseenservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/topicviewservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

const postWindowLimit = 20

func TopicDetail(c *gin.Context) {
	id := cast.ToUint64(c.Param("id"))
	if id == 0 {
		renderNotFound(c)
		return
	}

	topic := topics.Get(id)
	if topic.Id == 0 {
		topic = topics.UnscopedGet(id)
	}
	if topic.Id == 0 {
		renderNotFound(c)
		return
	}
	// 彻底删除（PURGED）的内容对 SEO 返回 410 Gone（PRD R12）。
	if topic.RetentionStatus == topics.RetentionPurged {
		renderGone(c)
		return
	}
	loginUser := component.GetLoginUser(c)
	if !canViewTopic(&topic, loginUser.UserId) {
		renderNotFound(c)
		return
	}
	postNo := cast.ToUint64(c.Param("postNo"))
	if postNo > topic.PostSeq {
		postNo = topic.PostSeq
	}

	firstPost := posts.Get(topic.FirstPostId)
	if firstPost.Id == 0 && topic.VisibilityStatus != topics.VisibilityActive {
		firstPost = posts.UnscopedGet(topic.FirstPostId)
	}
	if firstPost.Id == 0 {
		firstPost, _ = posts.GetByTopicPostNoAtOrAfter(topic.Id, 1)
	}
	postservice.EnsureRenderedHTML(&firstPost)
	if loginUser.UserId > 0 {
		if err := topicunseenservice.MarkVisited(loginUser.UserId, topic.Id, topic.LastPostId, time.Now()); err != nil {
			slog.Warn("mark topic visited failed", "userId", loginUser.UserId, "topicId", topic.Id, "error", err)
		}
	}
	props := buildTopicDetailProps(c, &topic, &firstPost, postNo)
	payload := PagePayload{
		Component: PageComponentTopic,
		Props:     props,
		Meta:      buildTopicMeta(c, props.Topic, props.PostStream.Posts),
		Layout:    buildLayout(c, activeKeyForTopic(props.Topic)),
		URL:       buildPageURL(c),
		Version:   payloadVersion,
	}
	renderPage(c, "topic.gohtml", payload)
	if shouldCountTopicView(&topic) {
		topicviewservice.RecordView(topic.Id)
	}
}

type PostWindowReq struct {
	TopicID      uint64 `form:"topicId"`
	AnchorPostID uint64 `form:"anchorPostId"`
	AnchorPostNo uint64 `form:"anchorPostNo"`
	BeforePostNo uint64 `form:"beforePostNo"`
	AfterPostNo  uint64 `form:"afterPostNo"`
	Limit        int    `form:"limit"`
}

func PostWindow(req component.BetterRequest[PostWindowReq]) component.Response {
	topicID := req.Params.TopicID
	if topicID == 0 {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}

	topicEntity := topics.GetSimple(topicID)
	if topicEntity.Id == 0 {
		topicEntity = topics.UnscopedGet(topicID)
	}
	if topicEntity.Id == 0 {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}
	if !CanViewTopicSimple(&topicEntity, req.UserId) {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}

	limit := req.Params.Limit
	if limit <= 0 || limit > 50 {
		limit = postWindowLimit
	}

	var postEntities []*posts.Entity
	hasBefore := false
	var hasAfter bool

	switch {
	case req.Params.AnchorPostNo > 0:
		anchor, ok := posts.GetByTopicPostNoAtOrAfter(topicID, req.Params.AnchorPostNo)
		if !ok {
			anchor, ok = posts.GetByTopicPostNoAtOrBefore(topicID, req.Params.AnchorPostNo)
		}
		if !ok || anchor.Id == 0 || anchor.TopicId != topicID || anchor.PostNo < 1 {
			return component.FailResponseCode(component.MessagePostNotFound, nil)
		}
		beforeLimit := min(5, limit/2)
		afterLimit := limit - beforeLimit - 1
		beforePosts := posts.GetByTopicPostNoBefore(topicID, anchor.PostNo, beforeLimit+1)
		afterPosts := posts.GetByTopicPostNoAfter(topicID, anchor.PostNo, afterLimit+1)
		hasBefore = len(beforePosts) > beforeLimit
		hasAfter = len(afterPosts) > afterLimit
		if hasBefore {
			beforePosts = beforePosts[1:]
		}
		if hasAfter {
			afterPosts = afterPosts[:afterLimit]
		}
		postEntities = append(postEntities, beforePosts...)
		postEntities = append(postEntities, &anchor)
		postEntities = append(postEntities, afterPosts...)
	case req.Params.AnchorPostID > 0:
		anchor := posts.Get(req.Params.AnchorPostID)
		if anchor.Id == 0 {
			anchor = posts.UnscopedGet(req.Params.AnchorPostID)
		}
		if anchor.Id == 0 || anchor.TopicId != topicID || anchor.PostNo < 1 {
			return component.FailResponseCode(component.MessagePostNotFound, nil)
		}
		beforeLimit := min(5, limit/2)
		afterLimit := limit - beforeLimit - 1
		beforePosts := posts.GetByTopicIdBefore(topicID, anchor.Id, beforeLimit+1)
		afterPosts := posts.GetByTopicIdAfter(topicID, anchor.Id, afterLimit+1)
		hasBefore = len(beforePosts) > beforeLimit
		hasAfter = len(afterPosts) > afterLimit
		if hasBefore {
			beforePosts = beforePosts[1:]
		}
		if hasAfter {
			afterPosts = afterPosts[:afterLimit]
		}
		postEntities = append(postEntities, beforePosts...)
		postEntities = append(postEntities, &anchor)
		postEntities = append(postEntities, afterPosts...)
	case req.Params.BeforePostNo > 0:
		postEntities = posts.GetByTopicPostNoBefore(topicID, req.Params.BeforePostNo, limit+1)
		hasBefore = len(postEntities) > limit
		if hasBefore {
			postEntities = postEntities[1:]
		}
		hasAfter = true
	case req.Params.AfterPostNo > 0:
		postEntities = posts.GetByTopicPostNoAfter(topicID, req.Params.AfterPostNo, limit+1)
		hasAfter = len(postEntities) > limit
		if hasAfter {
			postEntities = postEntities[:limit]
		}
		hasBefore = true
	default:
		postEntities = posts.GetByTopicPostNoAfter(topicID, 0, limit+1)
		hasAfter = len(postEntities) > limit
		if hasAfter {
			postEntities = postEntities[:limit]
		}
	}

	userIDs := make([]uint64, 0, len(postEntities))
	seenUserIDs := make(map[uint64]struct{}, len(postEntities))
	for _, item := range postEntities {
		if item == nil {
			continue
		}
		if _, seen := seenUserIDs[item.UserId]; seen {
			continue
		}
		seenUserIDs[item.UserId] = struct{}{}
		userIDs = append(userIDs, item.UserId)
	}
	userMap := users.GetMapByIds(userIDs)
	canModeratePosts := moderationservice.CanModerateAnyCategory(req.UserId, topicEntity.CategoryIds)
	maxPostNo := uint64(0)
	if topicEntity.PostSeq > 0 {
		maxPostNo = topicEntity.PostSeq
	}
	if maxPostNo == 0 && topicEntity.ReplyCount > 0 {
		maxPostSeq := posts.GetMaxPostNoByTopicId(topicID)
		if maxPostSeq > 0 {
			maxPostNo = maxPostSeq
		}
	}

	return component.SuccessResponse(buildPostWindowPayloadFromEntities(
		postEntities,
		userMap,
		req.UserId,
		canModeratePosts,
		hasBefore,
		hasAfter,
		int64(maxPostNo),
		maxPostNo,
		req.Params.AnchorPostID,
	))
}

// canViewTopic 为历史别名，委托共享可见性谓词 CanViewTopicSimple，避免两处
// 安全边界实现漂移。调用方：TopicDetail（读路径）。
func canViewTopic(entity *topics.Entity, userID uint64) bool {
	return CanViewTopicSimple(entity, userID)
}

// CanViewTopicSimple is the shared read-path visibility predicate for topics
// loaded via the simple projection (topics.GetSimple). It is used by read paths
// (e.g. PostWindow, reportTargetInfo) and by topic write actions
// (posts/create, topics/like, etc.) so that hidden (Status != 1) and moderated
// (ProcessStatus != 0) topics are rejected with the same shape callers see on
// the read path, see issue #112 (CWE-862).
func CanViewTopicSimple(entity *topics.Entity, userID uint64) bool {
	if entity.VisibilityStatus != topics.VisibilityActive {
		return canViewDeletedTopic(entity, userID)
	}
	if entity.Status != 1 {
		return userID != 0 && userID == entity.UserId
	}
	if entity.ProcessStatus != 0 && !currentUserCanViewProcessedTopic(userID) && !moderationservice.CanModerateAnyCategory(userID, entity.CategoryIds) {
		return false
	}
	return true
}

func canViewDeletedTopic(entity *topics.Entity, userID uint64) bool {
	if entity.RetentionStatus == topics.RetentionPurged {
		return false
	}
	// 隐私擦除的内容对普通用户一律 404，但保留版主在作用域内的只读通道，
	// 供举报取证/审计查阅；版主仍不能恢复或对外暴露该内容。
	if entity.VisibilityStatus == topics.VisibilityAccountAnonymized {
		return moderationservice.CanModerateAnyCategory(userID, entity.CategoryIds)
	}
	if entity.VisibilityStatus == topics.VisibilityModeratorRemoved {
		return moderationservice.CanModerateAnyCategory(userID, entity.CategoryIds)
	}
	if userID > 0 && userID == entity.UserId {
		return true
	}
	if moderationservice.CanModerateAnyCategory(userID, entity.CategoryIds) {
		return true
	}
	// 话题作者删除首帖后，仍有正常回复时保留讨论上下文；没有回复的内容只在作者的最近删除中可见。
	return len(posts.GetByTopicPostNoAfter(entity.Id, 1, 1)) > 0
}

func currentUserCanViewProcessedTopic(userID uint64) bool {
	if userID == 0 {
		return false
	}
	roleID, ok := userservice.GetUserRoleId(userID)
	return ok && permission.CheckRole(roleID, permission.TopicsManager)
}

func shouldCountTopicView(entity *topics.Entity) bool {
	return entity.Status == 1 && entity.ProcessStatus == 0 && entity.VisibilityStatus == topics.VisibilityActive
}

func renderNotFound(c *gin.Context) {
	renderNotFoundWithMessage(c, component.MessagePageNotFound)
}

// renderGone 内容已被永久删除（PURGED），对 SEO 返回 410 Gone（PRD R12）。
func renderGone(c *gin.Context) {
	payload := PagePayload{
		Component: PageComponentError,
		Props: ErrorPageProps{
			Code:  "410",
			Title: i18n.T(requestLang(c), "meta.contentGone"),
		},
		Meta: PageMeta{
			Title: pageTitle(i18n.T(requestLang(c), "meta.contentGone")),
		},
		Layout:  buildLayout(c, "topics"),
		URL:     buildPageURL(c),
		Version: payloadVersion,
	}
	renderPageWithStatus(c, http.StatusGone, "error.gohtml", payload)
}

func renderNotFoundWithMessage(c *gin.Context, messageCode component.MessageCode) {
	payload := PagePayload{
		Component: PageComponentError,
		Props: ErrorPageProps{
			Code:        "404",
			Title:       i18n.T(requestLang(c), "meta.notFound"),
			MessageCode: messageCode,
		},
		Meta: PageMeta{
			Title: pageTitle(i18n.T(requestLang(c), "meta.notFound")),
		},
		Layout:  buildLayout(c, "topics"),
		URL:     buildPageURL(c),
		Version: payloadVersion,
	}

	renderPageWithStatus(c, http.StatusNotFound, "error.gohtml", payload)
}

func activeKeyForTopic(topic TopicDetailPayload) string {
	if len(topic.Categories) > 0 {
		return "category_" + cast.ToString(topic.Categories[0].ID)
	}
	return "topics"
}

type PostRevisionsReq struct {
	PostID        uint64 `form:"postId"`
	BeforeVersion uint64 `form:"beforeVersion"`
	Limit         int    `form:"limit"`
}

// PostRevisions 返回某帖的版本历史（用户只读查看，无回滚/写入接口）。
// 可见性与楼层窗口一致：话题可见即可读历史；已删除/匿名化帖的全部版本
// 正文清空，待审（pending）版本与封禁帖正文仅版主可见，普通用户看到
// 空内容 + 状态标记，避免敏感内容经历史泄露。
// 版本按版本号游标分页（beforeVersion=0 取最新一页，返回升序 + hasMore）。
func PostRevisions(req component.BetterRequest[PostRevisionsReq]) component.Response {
	postEntity := posts.Get(req.Params.PostID)
	if postEntity.Id == 0 {
		postEntity = posts.UnscopedGet(req.Params.PostID)
	}
	if postEntity.Id == 0 {
		return component.FailResponseCode(component.MessagePostNotFound, nil)
	}
	topicEntity := topics.GetSimple(postEntity.TopicId)
	if topicEntity.Id == 0 {
		topicEntity = topics.UnscopedGet(postEntity.TopicId)
	}
	if topicEntity.Id == 0 || !CanViewTopicSimple(&topicEntity, req.UserId) {
		return component.FailResponseCode(component.MessagePostNotFound, nil)
	}

	limit := pageutil.BoundPageSize(req.Params.Limit)
	versions, hasMore := postRevisions.PageByPostId(postEntity.Id, req.Params.BeforeVersion, limit)
	canModerate := moderationservice.CanModerateAnyCategory(req.UserId, topicEntity.CategoryIds)
	// 帖子级可见性（与 buildPostPayloads 的楼层正文过滤同语义）：
	// 删除/匿名化帖无条件清空全部版本正文；封禁帖正文仅版主可见。
	postDeleted := isAuthorDeletedVisibility(postEntity.VisibilityStatus) || isModeratorRemovedVisibility(postEntity.VisibilityStatus)
	postBlocked := postEntity.ProcessStatus == posts.ProcessStatusBlocked

	editorIDs := make([]uint64, 0, len(versions))
	seen := make(map[uint64]struct{}, len(versions))
	for _, v := range versions {
		if v == nil {
			continue
		}
		if _, ok := seen[v.EditorId]; !ok {
			seen[v.EditorId] = struct{}{}
			editorIDs = append(editorIDs, v.EditorId)
		}
	}
	userMap := users.GetMapByIds(editorIDs)
	wornBadges := badgeservice.GetWornBadges(selectedWornBadges(userMap))

	type revisionPayload struct {
		Version       uint64             `json:"version"`
		Editor        TopicAuthorPayload `json:"editor"`
		Content       string             `json:"content"`
		RenderedHTML  string             `json:"renderedHTML"`
		ProcessStatus int8               `json:"processStatus"`
		CreatedAt     string             `json:"createdAt"`
	}
	list := make([]revisionPayload, 0, len(versions))
	for _, v := range versions {
		if v == nil {
			continue
		}
		content := v.Content
		rendered := v.RenderedHTML
		if postDeleted {
			// 删除/匿名化帖的版本快照不得绕过删除留存原文
			content = ""
			rendered = ""
		} else if (v.ProcessStatus == posts.ProcessStatusPending || postBlocked) && !canModerate {
			// 待审版本与封禁帖正文对非版主屏蔽，与楼层窗口过滤同语义
			content = ""
			rendered = ""
		}
		list = append(list, revisionPayload{
			Version:       v.Version,
			Editor:        userPayloadWithWornBadge(v.EditorId, userMap, wornBadges[v.EditorId]),
			Content:       content,
			RenderedHTML:  rendered,
			ProcessStatus: v.ProcessStatus,
			CreatedAt:     v.CreatedAt.Format(time.DateTime),
		})
	}
	nextCursor := uint64(0)
	if len(list) > 0 {
		nextCursor = list[0].Version - 1
	}
	return component.SuccessResponse(map[string]any{
		"postId":        postEntity.Id,
		"versions":      list,
		"hasMore":       hasMore,
		"beforeVersion": nextCursor,
	})
}

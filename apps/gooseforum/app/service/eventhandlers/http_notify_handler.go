package eventhandlers

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/anonymousidentityservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/httpnotifyservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/urlconfig"
)

// ReportCreatedEvent 新举报已创建（仅新建，重复举报不发布）。四类举报目标
// （topic/post/chat_message/course_review）均发布本事件（issue #1049）。
type ReportCreatedEvent struct {
	ReportId   uint64
	TargetType string
	TargetId   uint64
	TopicId    uint64
	ReporterId uint64
	Reason     string
	Note       string
}

func handleHttpNotifyTopicPublished(ctx context.Context, event *TopicPublishedEvent) error {
	if !httpnotifyservice.ShouldNotify(httpnotifyservice.EventTopicPublished) {
		return nil
	}
	httpnotifyservice.Notify(httpnotifyservice.EventTopicPublished, topicEventNotifyPayload(event))
	return nil
}

func handleHttpNotifyTopicUpdated(ctx context.Context, event *TopicUpdatedEvent) error {
	if !httpnotifyservice.ShouldNotify(httpnotifyservice.EventTopicUpdated) {
		return nil
	}
	httpnotifyservice.Notify(httpnotifyservice.EventTopicUpdated, topicUpdatedEventNotifyPayload(event))
	return nil
}

func handleHttpNotifyCommentCreated(ctx context.Context, event *CommentCreatedEvent) error {
	if !httpnotifyservice.ShouldNotify(httpnotifyservice.EventCommentCreated) {
		return nil
	}
	// 匿名楼层不推 webhook（issue #524）：webhook 负载含完整评论者用户信息，
	// 会向订阅方泄露匿名身份。与 in-app 通知同口径。
	if event.IsAnonymous && event.PersonaUID == "" {
		return nil
	}
	topic := topics.GetSimple(event.TopicId)
	topicPayload := topicNotifyPayloadFromSmall(topic)
	commenter := publicNotifyAuthor(event.UserId, event.PersonaUID)
	post := posts.Get(event.PostId)
	postNo := uint64(0)
	if post.Id > 0 {
		postNo = post.PostNo
	}

	postPayload := notifyPost{
		ID:                  event.PostId,
		PostNo:              postNo,
		UserID:              commenter.ID,
		User:                commenter,
		ReplyToPostID:       event.ReplyToPostId,
		ReplyToPostAuthorID: event.ReplyToPostAuthorId,
		URL:                 postURL(event.TopicId, event.PostId),
	}
	payload := notifyEventData{
		BaseURI:        baseURI(),
		ContentPreview: TakeUpTo64Chars(event.Content),
		Topic:          &topicPayload,
		User:           &commenter,
		Post:           &postPayload,
	}
	if event.ReplyToPostAuthorId > 0 {
		parent := posts.Get(event.ReplyToPostId)
		parentAuthor := publicNotifyAuthor(event.ReplyToPostAuthorId, parent.PersonaUID)
		if parent.IsAnonymous && parent.PersonaUID == "" {
			parentAuthor = notifyUser{}
		}
		payload.Post.ReplyToPostAuthorID = parentAuthor.ID
		payload.Post.ReplyToPostAuthor = &parentAuthor
	}
	httpnotifyservice.Notify(httpnotifyservice.EventCommentCreated, payload)
	return nil
}

func handleHttpNotifyUserSignUp(ctx context.Context, event *UserSignUpEvent) error {
	if !httpnotifyservice.ShouldNotify(httpnotifyservice.EventUserSignup) {
		return nil
	}
	user := userNotifyPayload(event.UserId)
	httpnotifyservice.Notify(httpnotifyservice.EventUserSignup, notifyEventData{
		BaseURI: baseURI(),
		User:    &user,
	})
	return nil
}

// handleHttpNotifyReportCreated 新举报进入审批通知层（issue #1049）：具体举报类型
// 事件携带统一审批摘要，“全部举报”聚合事件保持既有负载形状；同一 Endpoint 同时
// 订阅二者时只收到具体事件一份。
func handleHttpNotifyReportCreated(ctx context.Context, event *ReportCreatedEvent) error {
	specific := reportApprovalEvent(event.TargetType)
	if !httpnotifyservice.ShouldNotify(specific, httpnotifyservice.EventReportCreated) {
		return nil
	}
	approval, ok := reportApproval(event, time.Now())
	if !ok {
		// 没有审批摘要的举报类型（如今后新增的目标）仍投递“全部举报”聚合事件，
		// 不因审批层不认识而静默丢弃既有订阅。
		httpnotifyservice.Notify(httpnotifyservice.EventReportCreated, legacyReportNotifyPayload(event))
		return nil
	}
	alternatives := make([]httpnotifyservice.Alternative, 0, 2)
	if specific != "" {
		alternatives = append(alternatives, httpnotifyservice.Alternative{Event: specific, Data: approval})
	}
	alternatives = append(alternatives, httpnotifyservice.Alternative{Event: httpnotifyservice.EventReportCreated, Data: legacyReportNotifyPayload(event)})
	httpnotifyservice.Publish(httpnotifyservice.Message{
		Alternatives: alternatives,
		Approval:     &approval,
		DedupeKey:    approval.Approval.ID,
	})
	return nil
}

// legacyReportNotifyPayload moderation.report.created 的既有负载形状。匿名楼层不携带
// 真实作者（issue #524，与举报证据快照同口径）；私信举报仅 Admin 可处理，不外带
// 举报人身份（与审核日志不复制举报人同口径）。
func legacyReportNotifyPayload(event *ReportCreatedEvent) notifyEventData {
	payload := notifyEventData{
		BaseURI:       baseURI(),
		ReportID:      new(event.ReportId),
		TargetType:    event.TargetType,
		TargetID:      new(event.TargetId),
		Reason:        new(event.Reason),
		ModerationURL: moderationTargetURL(event),
	}
	if event.TargetType != reports.TargetChatMessage {
		payload.ReporterID = new(event.ReporterId)
		payload.Reporter = new(userNotifyPayload(event.ReporterId))
	}
	if event.TopicId > 0 {
		payload.Topic = new(topicNotifyPayloadFromSmall(topics.GetSimple(event.TopicId)))
	}
	if event.TargetType == "reply" || event.TargetType == reports.TargetPost {
		post := posts.Get(event.TargetId)
		postPayload := notifyPost{
			ID:     post.Id,
			PostNo: post.PostNo,
			URL:    postURL(post.TopicId, post.Id),
		}
		if !post.IsAnonymous {
			postPayload.UserID = post.UserId
			postPayload.User = userNotifyPayload(post.UserId)
		}
		payload.Post = &postPayload
	}
	return payload
}

type notifyEventData struct {
	BaseURI        string       `json:"baseUri"`
	ContentPreview string       `json:"contentPreview,omitempty"`
	Topic          *notifyTopic `json:"topic,omitempty"`
	User           *notifyUser  `json:"user,omitempty"`
	Post           *notifyPost  `json:"post,omitempty"`
	ReportID       *uint64      `json:"reportId,omitempty"`
	TargetType     string       `json:"targetType,omitempty"`
	TargetID       *uint64      `json:"targetId,omitempty"`
	ReporterID     *uint64      `json:"reporterId,omitempty"`
	Reason         *string      `json:"reason,omitempty"`
	Reporter       *notifyUser  `json:"reporter,omitempty"`
	ModerationURL  string       `json:"moderationUrl,omitempty"`
}

type notifyTopic struct {
	ID            uint64           `json:"id"`
	Title         string           `json:"title"`
	URL           string           `json:"url"`
	Description   string           `json:"description"`
	FirstImageURL string           `json:"firstImageUrl"`
	UserID        uint64           `json:"userId"`
	User          notifyUser       `json:"user"`
	CategoryIDs   []uint64         `json:"categoryIds"`
	Categories    []notifyCategory `json:"categories"`
}

type notifyPost struct {
	ID                  uint64      `json:"id"`
	PostNo              uint64      `json:"postNo"`
	UserID              uint64      `json:"userId"`
	User                notifyUser  `json:"user"`
	ReplyToPostID       uint64      `json:"replyToPostId,omitempty"`
	ReplyToPostAuthorID uint64      `json:"replyToPostAuthorId,omitempty"`
	ReplyToPostAuthor   *notifyUser `json:"replyToPostAuthor,omitempty"`
	URL                 string      `json:"url"`
}

type notifyCategory struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type notifyUser struct {
	Kind        string `json:"kind,omitempty"`
	PublicUID   string `json:"publicUid,omitempty"`
	ID          uint64 `json:"id"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl"`
	URL         string `json:"url"`
}

func topicEventNotifyPayload(event *TopicPublishedEvent) notifyEventData {
	if event != nil && event.Topic != nil {
		return topicNotifyPayload(event.Topic)
	}
	return notifyEventData{BaseURI: baseURI()}
}

func topicUpdatedEventNotifyPayload(event *TopicUpdatedEvent) notifyEventData {
	if event != nil && event.Topic != nil {
		return topicNotifyPayload(event.Topic)
	}
	return notifyEventData{BaseURI: baseURI()}
}

func topicNotifyPayload(topic *topics.Entity) notifyEventData {
	if topic == nil {
		return notifyEventData{BaseURI: baseURI()}
	}
	summary := buildNotifyTopic(topic.Id, topic.Title, topic.Excerpt, topic.FirstImageURL, topic.UserId, topic.CategoryIds, topic.PersonaUID)
	user := summary.User
	return notifyEventData{
		BaseURI: baseURI(),
		Topic:   &summary,
		User:    &user,
	}
}

func topicNotifyPayloadFromSmall(topic topics.Entity) notifyTopic {
	if topic.Id == 0 {
		return notifyTopic{}
	}
	return buildNotifyTopic(topic.Id, topic.Title, topic.Excerpt, topic.FirstImageURL, topic.UserId, topic.CategoryIds, topic.PersonaUID)
}

func buildNotifyTopic(id uint64, title string, description string, firstImageURL string, userID uint64, categoryIDs []uint64, personaUID string) notifyTopic {
	author := publicNotifyAuthor(userID, personaUID)
	return notifyTopic{
		ID:            id,
		Title:         title,
		URL:           urlconfig.PostDetail(id),
		Description:   description,
		FirstImageURL: firstImageURL,
		UserID:        author.ID,
		User:          author,
		CategoryIDs:   categoryIDs,
		Categories:    topicCategoryNotifyPayloads(categoryIDs),
	}
}

func topicCategoryNotifyPayloads(categoryIDs []uint64) []notifyCategory {
	if len(categoryIDs) == 0 {
		return []notifyCategory{}
	}
	categories := category.All()
	categoryByID := make(map[uint64]*category.Entity, len(categories))
	for _, item := range categories {
		categoryByID[item.Id] = item
	}
	payloads := make([]notifyCategory, 0, len(categoryIDs))
	for _, categoryID := range categoryIDs {
		item, ok := categoryByID[categoryID]
		if !ok {
			continue
		}
		payloads = append(payloads, notifyCategory{
			ID:   item.Id,
			Name: item.Name,
			Slug: item.Slug,
		})
	}
	return payloads
}

func userNotifyPayload(userID uint64) notifyUser {
	if userID == 0 {
		return notifyUser{}
	}
	user, err := users.Get(userID)
	if err != nil || user.Id == 0 {
		return notifyUser{ID: userID, URL: urlconfig.User(userID)}
	}
	displayName := user.Nickname
	if displayName == "" {
		displayName = user.Username
	}
	return notifyUser{
		ID:          user.Id,
		Username:    user.Username,
		Nickname:    user.Nickname,
		DisplayName: displayName,
		AvatarURL:   user.GetWebAvatarUrl(),
		URL:         urlconfig.User(user.Id),
	}
}

func postURL(topicID uint64, postID uint64) string {
	if topicID == 0 {
		return ""
	}
	if postID == 0 {
		return urlconfig.PostDetail(topicID)
	}
	return urlconfig.PostDetail(topicID) + "#post-" + uintToString(postID)
}

func moderationTargetURL(event *ReportCreatedEvent) string {
	switch event.TargetType {
	case reports.TargetChatMessage:
		return moderationReportsURL
	case reports.TargetCourseReview:
		return moderationCourseReviewsURL
	}
	if event.TopicId == 0 {
		return ""
	}
	if (event.TargetType == "reply" || event.TargetType == "post") && event.TargetId > 0 {
		return postURL(event.TopicId, event.TargetId)
	}
	return urlconfig.PostDetail(event.TopicId)
}

func uintToString(value uint64) string {
	return strconv.FormatUint(value, 10)
}

func baseURI() string {
	return strings.TrimRight(hotdataserve.GetSiteSettingsConfigCache().SiteUrl, "/")
}

func publicNotifyAuthor(owner uint64, uid string) notifyUser {
	if uid == "" {
		return userNotifyPayload(owner)
	}
	p := anonymousidentityservice.Lookup([]string{uid})[uid]
	return notifyUser{Kind: "persona", PublicUID: uid, Username: p.Name, DisplayName: p.Name, AvatarURL: baseURI() + p.AvatarURL, URL: baseURI() + p.ProfileURL}
}

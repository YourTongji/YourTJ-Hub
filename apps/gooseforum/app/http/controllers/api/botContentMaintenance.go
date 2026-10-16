package api

import (
	"context"
	"errors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/botcontentservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
	"net/http"
)

type botContentMaintenance = botcontentservice.Maintenance
type BotContentSourceReq = botcontentservice.SourceRequest
type BotContentSource struct {
	TopicId    uint64   `json:"topicId"`
	PostId     uint64   `json:"postId"`
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	CategoryId []uint64 `json:"categoryId"`
	RevisionNo uint64   `json:"revisionNo"`
	BotUserId  uint64   `json:"botUserId"`
	BotName    string   `json:"botName"`
}
type BotEditTopicReq struct {
	TopicId            uint64   `json:"topicId" validate:"required"`
	Title              string   `json:"title"`
	Content            string   `json:"content" validate:"required"`
	CategoryId         []uint64 `json:"categoryId" validate:"min=1,max=3"`
	ExpectedRevisionNo *uint64  `json:"expectedRevisionNo" validate:"required"`
}
type BotEditPostReq struct {
	PostId             uint64  `json:"postId" validate:"required"`
	Content            string  `json:"content" validate:"required"`
	ExpectedRevisionNo *uint64 `json:"expectedRevisionNo" validate:"required"`
}

func maintenanceEditor(authorID uint64, m *botContentMaintenance) uint64 {
	if m != nil {
		return m.ActorID
	}
	return authorID
}
func submitMaintainedContent(ctx context.Context, topic *topics.Entity, post *posts.Entity, m *botContentMaintenance) error {
	if m == nil {
		return publicationservice.Submit(ctx, topic, post)
	}
	return publicationservice.SubmitWithEditorHooks(ctx, topic, post, m.ActorID, m.CheckTx, m.AuditTx)
}
func botContentFailure(err error) component.Response {
	if errors.Is(err, botcontentservice.ErrConflict) {
		return component.BuildResponse(http.StatusConflict, component.FailDataCode("admin.agent.contentConflict", nil))
	}
	if errors.Is(err, botcontentservice.ErrUnavailable) {
		return component.BuildResponse(http.StatusForbidden, component.FailDataCode("admin.agent.contentUnavailable", nil))
	}
	return component.FailResponseCode(component.MessageOperationFailed, nil)
}
func AgentContentSource(req component.BetterRequest[BotContentSourceReq]) component.Response {
	m, err := botcontentservice.Load(req.UserId, req.Params)
	if err != nil {
		return botContentFailure(err)
	}
	bot, _ := users.Get(m.Post.UserId)
	content, title, categories := m.Post.Content, m.Topic.Title, m.Topic.CategoryIds
	if m.SourceRevision != nil {
		content = m.SourceRevision.Content
		if m.Post.PostNo == 1 {
			title, categories = m.SourceRevision.Title, m.SourceRevision.CategoryIds
		}
	}
	if categories == nil {
		categories = []uint64{}
	}
	name := bot.Nickname
	if name == "" {
		name = bot.Username
	}
	return component.SuccessResponse(BotContentSource{TopicId: m.Topic.Id, PostId: m.Post.Id, Title: title, Content: content, CategoryId: categories, RevisionNo: m.ExpectedRevisionNo, BotUserId: m.Post.UserId, BotName: name})
}
func AgentEditTopic(req component.BetterRequest[BotEditTopicReq]) component.Response {
	if req.Params.ExpectedRevisionNo == nil || len(req.Params.CategoryId) < 1 || len(req.Params.CategoryId) > 3 {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	m, err := botcontentservice.Load(req.UserId, BotContentSourceReq{TopicId: req.Params.TopicId})
	if err != nil {
		return botContentFailure(err)
	}
	m.ExpectedRevisionNo = *req.Params.ExpectedRevisionNo
	m.CategoryIDs = req.Params.CategoryId
	return writeTopicWithMaintenance(component.BetterRequest[WriteTopicReq]{UserId: m.Post.UserId, GinContext: req.GinContext, Context: req.Context, Params: WriteTopicReq{TopicId: m.Topic.Id, Title: req.Params.Title, Content: req.Params.Content, CategoryId: req.Params.CategoryId, TopicStatus: m.Topic.Status, ContentType: m.Post.ContentType}}, false, m)
}
func AgentEditPost(req component.BetterRequest[BotEditPostReq]) component.Response {
	if req.Params.ExpectedRevisionNo == nil {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	m, err := botcontentservice.Load(req.UserId, BotContentSourceReq{PostId: req.Params.PostId})
	if err != nil {
		return botContentFailure(err)
	}
	m.ExpectedRevisionNo = *req.Params.ExpectedRevisionNo
	return updatePostWithMaintenance(component.BetterRequest[UpdatePostReq]{UserId: m.Post.UserId, GinContext: req.GinContext, Context: req.Context, Params: UpdatePostReq{PostId: m.Post.Id, Content: req.Params.Content}}, m)
}

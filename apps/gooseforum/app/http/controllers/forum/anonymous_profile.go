package forum

import (
	"net/http"
	"strconv"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/anonymousnames"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/transform"
	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/anonymousidentityservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/urlconfig"
	"github.com/gin-gonic/gin"
)

type AnonymousProfileProps struct {
	Persona    anonymousidentityservice.PublicPersona `json:"persona"`
	Topics     []TopicPayload                         `json:"topics"`
	Replies    []AnonymousProfileReply                `json:"replies"`
	TopicCount int64                                  `json:"topicCount"`
	ReplyCount int64                                  `json:"replyCount"`
	Page       int                                    `json:"page"`
	HasNext    bool                                   `json:"hasNext"`
}
type AnonymousProfileReply struct {
	ID      uint64 `json:"id"`
	URL     string `json:"url"`
	Excerpt string `json:"excerpt"`
}

func AnonymousAvatar(c *gin.Context) {
	var p identity.Persona
	if err := db.ConnectContext(c.Request.Context()).First(&p, "uid = ?", c.Param("publicUid")).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "image/svg+xml", []byte(anonymousnames.Avatar(p.AvatarSeed)))
}
func AnonymousProfile(c *gin.Context) {
	var p identity.Persona
	if err := db.ConnectContext(c.Request.Context()).First(&p, "uid = ?", c.Param("publicUid")).Error; err != nil {
		RenderNotFoundPage(c, component.MessagePageNotFound)
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	page = max(1, min(page, 10000))
	rows, totalTopics, totalReplies, err := posts.PublicPersonaPosts(db.ConnectContext(c.Request.Context()), p.UID, page)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	topicPage := topics.Page(topics.PageQuery{Page: page, PageSize: 20, PersonaUID: p.UID, FilterStatus: true, TopicType: topics.TopicTypePtr(topics.TopicTypeForum)})
	topicPtrs := make([]*topics.Entity, 0, len(topicPage.Data))
	for i := range topicPage.Data {
		topicPtrs = append(topicPtrs, &topicPage.Data[i])
	}
	replies := make([]AnonymousProfileReply, 0, len(rows))
	for _, row := range rows {
		replies = append(replies, AnonymousProfileReply{row.Id, urlconfig.PostDetail(row.TopicId) + "/" + strconv.FormatUint(row.PostNo, 10), markdown2html.ExtractPreview(row.Content, 200)})
	}
	props := AnonymousProfileProps{anonymousidentityservice.Public(p), buildTopicPayloads(transform.Topics2Vo(topicPtrs, hotdataserve.CategoryMap())), replies, totalTopics, totalReplies, page, topicPage.HasNext || int64(page*20) < totalReplies}
	payload := PagePayload{Component: PageComponentAnonymous, Props: props, Meta: PageMeta{Title: p.Name, Canonical: component.GetBaseUri(c) + "/a/" + p.UID}, Layout: buildLayout(c, "user"), URL: buildPageURL(c), Version: payloadVersion}
	renderPage(c, "anonymous.gohtml", payload)
}

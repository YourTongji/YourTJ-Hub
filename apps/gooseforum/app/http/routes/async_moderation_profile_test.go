package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
)

func TestAsyncProfileAndEditorUseAuthorsLatestVersion(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) { o.TextModeration = true })
	router.GET("/u/:userId/:section/:subsection", middleware.JWTAuth, forum.UserProfile)
	router.GET("/publish", middleware.JWTAuthCheck, forum.Publish)
	author := createHTTPContractUser(t, conn, contractTestID())
	stranger := createHTTPContractUser(t, conn, author.Id+1)
	token := contractSessionToken(t, author)
	submit := func(t *testing.T, id uint64, title string) uint64 {
		t.Helper()
		ratelimit.Default().ResetAll()
		e := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", fmt.Sprintf(`{"topicId":%d,"title":%q,"content":"这是一条完整的瞬间正文","categoryId":[1],"topicStatus":1,"contentType":2}`, id, title), token))
		if e.Code != 0 {
			t.Fatalf("submit: %+v", e)
		}
		var result uint64
		if err := json.Unmarshal(e.Result, &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	read := func(t *testing.T, path, session string, target any) {
		t.Helper()
		r := httptest.NewRequest(http.MethodGet, path, nil)
		r.Header.Set("X-Goose-Page", "true")
		if session != "" {
			r.Header.Set("Authorization", "Bearer "+session)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		if err := json.Unmarshal(w.Body.Bytes(), target); err != nil {
			t.Fatal(err)
		}
	}
	publicID := submit(t, 0, "已经公开的瞬间")
	runSubmission(t, conn, posts.Get(topics.Get(publicID).FirstPostId).LatestRevisionId)
	submit(t, publicID, "修改后的待审瞬间")
	pendingID := submit(t, 0, "新待审瞬间")
	blockedID := submit(t, 0, "拒绝后只在管理页")
	if err := publicationservice.Review(context.Background(), posts.Get(topics.Get(blockedID).FirstPostId).LatestRevisionId, moderationDecision.ActionBlock, "请修改", 0); err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/u/%d/activity/topics", author.Id)
	t.Run("profile private overlay and public isolation", func(t *testing.T) {
		var own struct {
			Props forum.UserProfileProps `json:"props"`
		}
		read(t, path, token, &own)
		if len(own.Props.Topics) != 2 {
			t.Fatalf("owner topics = %+v", own.Props.Topics)
		}
		for _, item := range own.Props.Topics {
			if item.ProcessStatus != 2 || item.ContentType != 2 {
				t.Fatalf("owner candidate = %+v", item)
			}
		}
		read(t, path+fmt.Sprintf("?cursor=%d", pendingID), token, &own)
		if len(own.Props.Topics) != 1 || own.Props.Topics[0].Title != "修改后的待审瞬间" {
			t.Fatalf("owner cursor = %+v", own.Props.Topics)
		}
		for _, session := range []string{"", contractSessionToken(t, stranger)} {
			var other struct {
				Props forum.UserProfileProps `json:"props"`
			}
			read(t, path, session, &other)
			if len(other.Props.Topics) != 1 || other.Props.Topics[0].Title != "已经公开的瞬间" || other.Props.Topics[0].ProcessStatus != 0 {
				t.Fatalf("public profile = %+v", other.Props.Topics)
			}
		}
	})
	t.Run("pending and blocked thought editor keeps type and resubmits", func(t *testing.T) {
		for _, id := range []uint64{pendingID, blockedID, publicID} {
			var page struct {
				Props forum.PublishPageProps `json:"props"`
			}
			read(t, fmt.Sprintf("/publish?id=%d", id), token, &page)
			if !page.Props.IsEditing || page.Props.Topic.ContentType != 2 || page.Props.Topic.Content != "这是一条完整的瞬间正文" {
				t.Fatalf("editor = %+v", page.Props)
			}
			postID := topics.Get(id).FirstPostId
			before := posts.Get(postID).LatestRevisionId
			submit(t, id, "再次提交瞬间")
			after := posts.Get(postID)
			if after.LatestRevisionId == before || postRevisions.Get(after.LatestRevisionId).ProcessStatus != posts.ProcessStatusPending {
				t.Fatal("edit did not create a new pending revision")
			}
		}
	})
}

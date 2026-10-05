package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
)

func TestDraftImagesAreNotModeratorReviewAttachments(t *testing.T) {
	conn, router, stub := setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) { o.TextModeration = true })
	author := createHTTPContractUser(t, conn, contractTestID())
	authorToken := contractSessionToken(t, author)
	const categoryID = uint64(97561)
	moderatorTokens := make(map[string]string)
	for _, scope := range []string{moderators.ScopeCategory, moderators.ScopeGlobal} {
		moderator := createHTTPContractUser(t, conn, contractTestID())
		if err := conn.Create(&moderators.Entity{UserId: moderator.Id, ScopeType: scope, ScopeId: categoryID, Status: moderators.StatusEnabled}).Error; err != nil {
			t.Fatal(err)
		}
		moderatorTokens[scope] = contractSessionToken(t, moderator)
		t.Cleanup(func() {
			conn.Where("user_id = ?", moderator.Id).Delete(&moderators.Entity{})
			moderationservice.Invalidate()
		})
	}
	moderationservice.Invalidate()
	unusedImage, _ := saveContractImage(t, author.Id)
	submittedImage, _ := saveContractImage(t, author.Id)
	write := func(id uint64, status int, images []string) uint64 {
		body := mustReviewJSON(t, map[string]any{"topicId": id, "title": "草稿图片隐私回归", "content": "草稿只有点击发布之后才会送审", "categoryId": []uint64{categoryID}, "topicStatus": status, "contentType": 2, "images": images})
		envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", string(body), authorToken))
		if envelope.Code != 0 {
			t.Fatalf("write = %+v", envelope)
		}
		if err := json.Unmarshal(envelope.Result, &id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	assertImage := func(label, image, token string, want int) {
		t.Helper()
		response := getImage(router, image, token)
		if response.Code != want {
			t.Errorf("%s: image status=%d, want %d", label, response.Code, want)
		}
		if want == http.StatusOK && token != "" && response.Header().Get("Cache-Control") != "private, no-store" {
			t.Errorf("%s: preview entered shared cache", label)
		}
	}
	topicID := write(0, 0, []string{unusedImage, submittedImage})
	if stub.jevCalls.Load() != 0 || posts.Get(topics.Get(topicID).FirstPostId).LatestRevisionId != 0 {
		t.Fatal("draft entered review")
	}
	assertImage("draft owner", submittedImage, authorToken, http.StatusOK)
	assertImage("anonymous draft", submittedImage, "", http.StatusNotFound)
	for scope, token := range moderatorTokens {
		assertImage(scope+" draft", submittedImage, token, http.StatusNotFound)
	}

	// Older pending content has topic/post references rather than revision refs.
	// Those remain reviewable once submitted, but never while the parent is a draft.
	legacyTopicID, legacyPostID := contractTestID(), contractTestID()
	createContractPublishedTopic(t, conn, legacyTopicID, legacyPostID, author.Id)
	if err := conn.Model(&topics.Entity{}).Where("id = ?", legacyTopicID).Updates(map[string]any{"category_id": fmt.Sprintf("[%d]", categoryID), "process_status": topics.ProcessStatusPending}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&posts.Entity{}).Where("id = ?", legacyPostID).Update("process_status", posts.ProcessStatusPending).Error; err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{fileUsage.TargetTopic, fileUsage.TargetPost} {
		image, name := saveContractImage(t, author.Id)
		id := legacyTopicID
		if target == fileUsage.TargetPost {
			id = legacyPostID
		}
		if err := conn.Create(&fileUsage.Entity{FileName: name, TargetType: target, TargetId: id, UserId: author.Id, UsageType: fileUsage.UsageInlineImage, Status: fileUsage.UsageStatusPending}).Error; err != nil {
			t.Fatal(err)
		}
		for scope, token := range moderatorTokens {
			assertImage(scope+" legacy "+target, image, token, http.StatusOK)
		}
		if err := conn.Model(&topics.Entity{}).Where("id = ?", legacyTopicID).Update("status", 0).Error; err != nil {
			t.Fatal(err)
		}
		for scope, token := range moderatorTokens {
			assertImage(scope+" legacy draft "+target, image, token, http.StatusNotFound)
		}
		if err := conn.Model(&topics.Entity{}).Where("id = ?", legacyTopicID).Update("status", 1).Error; err != nil {
			t.Fatal(err)
		}
	}

	// The discarded draft attachment never enters the submitted revision. A
	// PENDING legacy topic reference must not grant access once the topic is sent.
	write(topicID, 1, []string{submittedImage})
	for scope, token := range moderatorTokens {
		assertImage(scope+" submitted", submittedImage, token, http.StatusOK)
		assertImage(scope+" discarded draft", unusedImage, token, http.StatusNotFound)
	}
	assertImage("anonymous pending", submittedImage, "", http.StatusNotFound)

	// Withdrawing a candidate into drafts revokes moderator preview even though
	// its immutable revision and private file reference remain for the author.
	response := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/status", fmt.Sprintf(`{"topicId":%d,"topicStatus":0}`, topicID), authorToken))
	if response.Code != 0 {
		t.Fatalf("withdraw = %+v", response)
	}
	for scope, token := range moderatorTokens {
		assertImage(scope+" withdrawn", submittedImage, token, http.StatusNotFound)
	}
	assertImage("withdrawn owner", submittedImage, authorToken, http.StatusOK)
}

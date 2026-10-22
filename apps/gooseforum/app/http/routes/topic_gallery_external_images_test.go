package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupGalleryImageContractTest 在通用 HTTP 契约环境上准备 file_data 表，
// 供图集外链回归用例写入本站上传图。AI 审核保持默认（未配置即关闭）。
func setupGalleryImageContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := db4fileconnect.Connect().AutoMigrate(&filedata.Entity{}); err != nil {
		t.Fatalf("migrate file_data: %v", err)
	}
	return conn, router
}

// externalGalleryImage 造一条既不在 /file/img 下、也不带本站上传名的外链图片。
func externalGalleryImage(i int) string {
	return fmt.Sprintf("https://i.imgur.com/issue-1315-%d.png", i)
}

func externalGalleryImages(count int) []string {
	images := make([]string, 0, count)
	for i := range count {
		images = append(images, externalGalleryImage(i))
	}
	return images
}

// galleryMarkdown 把图片按 markdown 内嵌进正文，外链图同样会出现在正文里。
func galleryMarkdown(images []string) string {
	content := "外链图集回归正文，文字部分满足最小正文长度。"
	for _, image := range images {
		content += "\n\n![](" + image + ")"
	}
	return content
}

// writeGalleryTopic 发布一篇正文与图集都携带 images 的话题，返回响应与话题 id。
func writeGalleryTopic(t *testing.T, router http.Handler, token, title string, images []string) (contractEnvelope, uint64) {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"title": title, "content": galleryMarkdown(images),
		"categoryId": []uint64{1}, "topicStatus": 1, "contentType": 2, "images": images,
	})
	if err != nil {
		t.Fatal(err)
	}
	envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", string(body), token))
	var topicID uint64
	if envelope.Code == 0 {
		if err := json.Unmarshal(envelope.Result, &topicID); err != nil {
			t.Fatalf("decode topic id %q: %v", envelope.Result, err)
		}
	}
	return envelope, topicID
}

// TestTopicWriteAcceptsManyExternalGalleryImages 是 issue #1315 的回归：
// Web 发布页把正文 markdown 里的外链图也并入图集，外链图解析不出站内文件，
// 因此不能计入图集上限，否则正文插 25 张外链图即发布失败。它们也不会成为
// image_urls 条目；但 FirstImageURL 仍可能取正文首图，作为 PublicGallery 的
// 回退封面，所以这里只断言图集为空。
func TestTopicWriteAcceptsManyExternalGalleryImages(t *testing.T) {
	conn, router := setupGalleryImageContractTest(t)
	author := createHTTPContractUser(t, conn, contractTestID())

	envelope, topicID := writeGalleryTopic(t, router, contractSessionToken(t, author), "外链图集发布回归", externalGalleryImages(25))
	if envelope.Code != 0 {
		t.Fatalf("write = %+v, want success", envelope)
	}
	if topicID == 0 {
		t.Fatal("write returned no topic id")
	}
	if got := topics.Get(topicID).ImageUrls; len(got) != 0 {
		t.Fatalf("external gallery images persisted as topic gallery: %v", got)
	}
}

// TestTopicWriteRejectsTooManyInSiteGalleryCandidates 保证上限对站内候选仍生效：
// 即便这些文件都未上传，图集候选数超过上限仍按参数错误拒绝。
func TestTopicWriteRejectsTooManyInSiteGalleryCandidates(t *testing.T) {
	conn, router := setupGalleryImageContractTest(t)
	author := createHTTPContractUser(t, conn, contractTestID())

	images := make([]string, 0, 10)
	for i := range 10 {
		images = append(images, fmt.Sprintf("/file/img/2026/10/01/gallery-missing-%d.png", i))
	}
	envelope, _ := writeGalleryTopic(t, router, contractSessionToken(t, author), "站内图集上限回归", images)
	if envelope.Code == 0 || envelope.MessageCode != "common.request.invalidParams" {
		t.Fatalf("write = %+v, want common.request.invalidParams", envelope)
	}
}

// TestTopicWriteAcceptsGalleryAtCandidateCap 钉住上限的 ">" 边界：恰好上限数量的
// 站内候选（都不曾上传，归属过滤后一条不剩）加上外链仍发布成功，且图集为空——
// 成功只可能来自上限判断，不是别的校验放行。
func TestTopicWriteAcceptsGalleryAtCandidateCap(t *testing.T) {
	conn, router := setupGalleryImageContractTest(t)
	author := createHTTPContractUser(t, conn, contractTestID())

	images := make([]string, 0, fileusageservice.MaxGalleryImagesPerTopicWrite+3)
	for i := range fileusageservice.MaxGalleryImagesPerTopicWrite {
		images = append(images, fmt.Sprintf("/file/img/2026/10/01/gallery-capped-%d.png", i))
	}
	images = append(images, externalGalleryImages(3)...)

	envelope, topicID := writeGalleryTopic(t, router, contractSessionToken(t, author), "图集上限边界回归", images)
	if envelope.Code != 0 {
		t.Fatalf("write = %+v, want success", envelope)
	}
	if topicID == 0 {
		t.Fatal("write returned no topic id")
	}
	if got := topics.Get(topicID).ImageUrls; len(got) != 0 {
		t.Fatalf("unowned in-site candidates persisted as topic gallery: %v", got)
	}
}

// TestTopicWritePersistsOnlyOwnedInSiteGalleryImages 覆盖混合图集：本站已上传图
// 落库成话题图集，其余外链条目继续被丢弃，且不影响发布成功。
func TestTopicWritePersistsOnlyOwnedInSiteGalleryImages(t *testing.T) {
	conn, router := setupGalleryImageContractTest(t)
	author := createHTTPContractUser(t, conn, contractTestID())
	first, _ := saveContractImage(t, author.Id)
	second, _ := saveContractImage(t, author.Id)
	images := append([]string{first, second}, externalGalleryImages(25)...)

	envelope, topicID := writeGalleryTopic(t, router, contractSessionToken(t, author), "混合图集落库回归", images)
	if envelope.Code != 0 {
		t.Fatalf("write = %+v, want success", envelope)
	}
	got := topics.Get(topicID).ImageUrls
	if len(got) != 2 || got[0] != first || got[1] != second {
		t.Fatalf("topic gallery = %v, want [%s %s]", got, first, second)
	}
}

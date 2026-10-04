package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// aiContractStub Jev 替身：probabilities 决定本次判定；视觉替身恒返回合法证据。
// blockMarker 非空时，请求正文含该标记的判定返回 adult=0.99（按内容区分新旧版本）；
// holds[i] 非空时第 i 次 Jev 调用等待该通道关闭（控制先发后审后台判定的完成顺序）。
// 两者须在发起请求前设置。
type aiContractStub struct {
	probabilities map[string]float64
	severity      float64
	blockMarker   string
	holds         []chan struct{}
	jevCalls      atomic.Int32
	released      sync.Map
}

// release 放开一个被拦住的 Jev 调用（可重复调用）。
func (s *aiContractStub) release(hold chan struct{}) {
	if _, loaded := s.released.LoadOrStore(hold, true); !loaded {
		close(hold)
	}
}

func setupAIModerationContractTest(t *testing.T, mutate func(*pageConfig.AiModerationOptions)) (*gorm.DB, *gin.Engine, *aiContractStub) {
	t.Helper()
	conn, router := setupForumInteractionContractTest(t)
	if err := conn.AutoMigrate(&moderationDecision.Entity{}, &rolePermissionRs.Entity{}, &eventNotification.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := db4fileconnect.Connect().AutoMigrate(&filedata.Entity{}); err != nil {
		t.Fatal(err)
	}
	router.GET("/file/img/*filename", api.GetFileByFileName)
	adminAPI := router.Group("/api/admin", middleware.JWTAuthCheck, middleware.CheckWritableAccount, middleware.CheckPermission(permission.SiteManager))
	adminAPI.POST("/review-queue", UpButterReq(api.ReviewQueue))
	adminAPI.POST("/review-action", UpButterReq(api.ReviewAction))
	adminAPI.GET("/ai-moderation-settings", UpButterReq(api.GetAiModerationSettings))
	adminAPI.POST("/save-ai-moderation-settings", UpButterReq(api.SaveAiModerationSettings))
	adminAPI.POST("/ai-moderation/decisions", UpButterReq(api.ListAiModerationDecisions))
	adminAPI.POST("/ai-moderation/decisions/label", UpButterReq(api.LabelAiModerationDecision))
	adminAPI.POST("/ai-moderation/replay", UpButterReq(api.ReplayAiModerationDecisions))
	adminAPI.POST("/ai-moderation/test", UpButterReq(api.TestAiModerationConnection))

	stub := &aiContractStub{}
	vision := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		content := `{"ocr_text":"","scene":"a photo","visible_symbols":[],"adult_evidence":[],"violence_evidence":[],"other_risk_evidence":[],"uncertain":false,"refused":false}`
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": content}}}})
	}))
	jev := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := int(stub.jevCalls.Add(1)) - 1
		if call < len(stub.holds) && stub.holds[call] != nil {
			<-stub.holds[call]
		}
		raw, _ := io.ReadAll(r.Body)
		answers := map[string]any{
			"severity":      map[string]any{"type": "score", "score": stub.severity},
			"review_needed": map[string]any{"type": "noul", "noul": 0.05},
		}
		for _, key := range pageConfig.AiModerationPolicyKeys {
			answers[key] = map[string]any{"type": "noul", "noul": stub.probabilities[key]}
		}
		if stub.blockMarker != "" && bytes.Contains(raw, []byte(stub.blockMarker)) {
			answers[pageConfig.AiPolicyAdult] = map[string]any{"type": "noul", "noul": 0.99}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"model": "jev-test", "answers": answers, "usage": map[string]any{"input_tokens": 1, "output_tokens": 1}})
	}))
	opts := pageConfig.AiModerationOptions{
		Enabled: true, Mode: pageConfig.AiModerationModeEnforce,
		JevEndpoint: jev.URL + "/v1/systemone", JevModel: "jev-latest",
		VisionBaseURL: vision.URL, VisionModel: "vl-test",
		Policies: pageConfig.DefaultAiModerationPolicies(),
	}
	opts.Policies[0].Action = pageConfig.AiModerationActionBlock // adult → block
	if mutate != nil {
		mutate(&opts)
	}
	persistHTTPContractConfig(t, conn, pageConfig.AiModerationPage, pageConfig.AiModerationSettingsStorage{AiModerationOptions: opts})
	hotdataserve.ClearAiModerationConfigCache()
	t.Cleanup(func() {
		// 测试中途失败时放开所有仍被拦住的 Jev 调用，避免 Close 永久等待。
		for _, hold := range stub.holds {
			stub.release(hold)
		}
		vision.Close()
		jev.Close()
		conn.Where("page_type = ?", pageConfig.AiModerationPage).Delete(&pageConfig.Entity{})
		hotdataserve.ClearAiModerationConfigCache()
	})
	return conn, router, stub
}

// saveContractImage 写入一张就绪 PNG（每次内容不同），返回 /file/img 地址与对象名。
func saveContractImage(t *testing.T, ownerID uint64) (string, string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 3, 3))
	shade := uint8(time.Now().UnixNano())
	img.Set(1, 1, color.RGBA{R: shade, G: shade / 2, B: 99, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("2026/10/01/ai-contract-%d.png", time.Now().UnixNano())
	if _, err := filedata.SaveFile(ownerID, name, "image/png", buf.Bytes()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = filedata.DeleteByName(name) })
	return "/file/img/" + name, name
}

func writeTopicBody(title, content string) string {
	body, err := json.Marshal(map[string]any{"title": title, "content": content, "categoryId": []uint64{1}, "topicStatus": 1, "contentType": 3})
	if err != nil {
		panic(err)
	}
	return string(body)
}

func getImage(router http.Handler, url, token string) *httptest.ResponseRecorder {
	return serveAuthSecurityJSON(router, http.MethodGet, url, "", token)
}

func usageStatuses(t *testing.T, conn *gorm.DB, fileName string) []string {
	t.Helper()
	var rows []fileUsage.Entity
	conn.Where("file_name = ? AND usage_type <> ?", fileName, fileUsage.UsageUploadOwner).Find(&rows)
	statuses := make([]string, 0, len(rows))
	for _, row := range rows {
		statuses = append(statuses, row.Status)
	}
	return statuses
}

func latestNotification(t *testing.T, conn *gorm.DB, userID uint64) eventNotification.Entity {
	t.Helper()
	var notice eventNotification.Entity
	if err := conn.Where("user_id = ?", userID).Order("id DESC").First(&notice).Error; err != nil {
		t.Fatalf("load notification for user %d: %v", userID, err)
	}
	return notice
}

func TestAdminAiModerationSettingsHTTPContract(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, nil)
	conn.Where("page_type = ?", pageConfig.AiModerationPage).Delete(&pageConfig.Entity{})
	hotdataserve.ClearAiModerationConfigCache()
	serveAdminSiteOK(t, conn, router, http.MethodGet, "/api/admin/ai-moderation-settings", "", "admin-ai-moderation-settings-success.json")

	save := `{"settings":{"enabled":true,"mode":"enforce","jevEndpoint":"https://openrouter.ai/api/alpha/decisions","jevModel":"typesafe/jev-1.13","jevApiKey":"sk-jev-secret-975","visionBaseUrl":"https://openrouter.ai/api/v1","visionModel":"inclusionai/ling-3.0-flash-vl","visionApiKey":"sk-vision-secret-975"}}`
	serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/save-ai-moderation-settings", save)
	stored := hotdataserve.GetAiModerationSettingsStorage()
	if plain, err := securestore.DecryptPurpose(stored.JevAPIKeyEncrypted, securestore.ModerationJevAPIKeyPurpose); err != nil || plain != "sk-jev-secret-975" {
		t.Fatalf("jev key decrypt = %q err=%v", plain, err)
	}
	if strings.Contains(storedAsJSON(t, conn, pageConfig.AiModerationPage), "secret-975") {
		t.Fatal("plaintext key persisted")
	}
	view := serveAdminSiteRaw(t, conn, router, http.MethodGet, "/api/admin/ai-moderation-settings", "")
	if strings.Contains(view.Body.String(), "secret-975") || strings.Contains(view.Body.String(), stored.JevAPIKeyEncrypted) {
		t.Fatal("settings view leaked key material")
	}
	var result pageConfig.AiModerationSettingsView
	_ = json.Unmarshal(decodeContractEnvelope(t, view).Result, &result)
	if !result.JevAPIKeyConfigured || !result.VisionAPIKeyConfigured || result.Mode != pageConfig.AiModerationModeEnforce {
		t.Fatalf("view = %+v", result)
	}

	// 空 key 保留旧密文；clearVisionApiKey 显式清除。
	serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/save-ai-moderation-settings",
		`{"settings":{"enabled":true,"mode":"shadow","clearVisionApiKey":true}}`)
	after := hotdataserve.GetAiModerationSettingsStorage()
	if after.JevAPIKeyEncrypted != stored.JevAPIKeyEncrypted || after.VisionAPIKeyEncrypted != "" {
		t.Fatalf("keep/clear semantics broken: jevKept=%v visionCleared=%v", after.JevAPIKeyEncrypted == stored.JevAPIKeyEncrypted, after.VisionAPIKeyEncrypted == "")
	}

	invalid := serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/save-ai-moderation-settings", `{"settings":{"jevEndpoint":"javascript:alert(1)"}}`)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, invalid), contractFixture(t, "admin-ai-moderation-save-invalid-url.json"))
}

func TestAdminAiModerationDecisionsLabelAndReplayHTTPContract(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, nil)
	severity, reviewNeeded := 0.4, 0.1
	decision := moderationDecision.Entity{SubjectType: moderationDecision.SubjectPost, SubjectId: contractTestID(), Mode: pageConfig.AiModerationModeShadow,
		EvidenceStatus: moderationDecision.EvidenceComplete, FinalAction: moderationDecision.ActionAllow, AppliedAction: moderationDecision.ActionAllow,
		Signals: moderationDecision.Signals{RuleProbabilities: map[string]float64{pageConfig.AiPolicyViolence: 0.45}, Severity: &severity, ReviewNeeded: &reviewNeeded}}
	if err := moderationDecision.Create(&decision); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Delete(&moderationDecision.Entity{}, decision.Id) })

	list := serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/ai-moderation/decisions", `{"humanAction":"none","mode":"shadow","pageSize":50}`)
	if !strings.Contains(list.Body.String(), fmt.Sprintf(`"id":%d`, decision.Id)) {
		t.Fatalf("decision list missing id %d: %s", decision.Id, list.Body.String())
	}
	missing := serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/ai-moderation/decisions/label", `{"id":987654321,"label":"rejected"}`)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, missing), contractFixture(t, "admin-ai-moderation-label-not-found.json"))
	labeled := serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/ai-moderation/decisions/label", fmt.Sprintf(`{"id":%d,"label":"rejected"}`, decision.Id))
	if decodeContractEnvelope(t, labeled).Code != 0 || moderationDecision.Get(decision.Id).HumanAction != moderationDecision.HumanRejected {
		t.Fatalf("label failed: %s", labeled.Body.String())
	}

	replay := serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/ai-moderation/replay", `{"options":{"defaultReviewThreshold":0.4}}`)
	var report struct {
		Samples         int `json:"samples"`
		MissedViolation int `json:"missedViolation"`
		Changed         int `json:"changed"`
	}
	if err := json.Unmarshal(decodeContractEnvelope(t, replay).Result, &report); err != nil || report.Samples == 0 || report.Changed == 0 {
		t.Fatalf("replay report = %+v err=%v body=%s", report, err, replay.Body.String())
	}
}

func TestAdminAiModerationConnectionTestHTTPContract(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, nil)
	serveAdminSiteOK(t, conn, router, http.MethodPost, "/api/admin/ai-moderation/test",
		`{"target":"vision","settings":{"visionBaseUrl":"","visionModel":""}}`, "admin-ai-moderation-test-not-configured.json")
	// 表单值未保存也可测试：使用替身端点，key 留空回落已存 key。
	stored := hotdataserve.GetAiModerationConfigCache()
	body := fmt.Sprintf(`{"target":"vision","settings":{"visionBaseUrl":%q,"visionModel":"vl-test"}}`, stored.VisionBaseURL)
	recorder := serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/ai-moderation/test", body)
	var result struct {
		OK   bool   `json:"ok"`
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(decodeContractEnvelope(t, recorder).Result, &result); err != nil || !result.OK {
		t.Fatalf("vision test = %+v err=%v body=%s", result, err, recorder.Body.String())
	}
	invalid := serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/ai-moderation/test", `{"target":"jev","settings":{"jevEndpoint":"ftp://x"}}`)
	if envelope := decodeContractEnvelope(t, invalid); envelope.Code != 1 || envelope.MessageCode != "admin.aiModeration.saveFailed" {
		t.Fatalf("invalid endpoint envelope = %+v", envelope)
	}
}

// 前台版主工作台审核（issue #975）：分类版主只看到并只能审核管辖分类内的待审
// 内容；普通用户无权访问；通过后作者收到通知、图片公开。
func TestModerationWorkbenchReviewQueueIsScopedToModeratorCategories(t *testing.T) {
	conn, router, stub := setupAIModerationContractTest(t, nil)
	router.POST("/api/forum/moderation/review-queue", middleware.JWTAuthCheck, UpButterReq(api.ModerationReviewQueue))
	router.POST("/api/forum/moderation/review-action", middleware.JWTAuthCheck, middleware.CheckWritableAccount, UpButterReq(api.ModerationReviewAction))
	stub.probabilities = map[string]float64{pageConfig.AiPolicyViolence: 0.8}
	stub.severity = 2
	const ownCategory, otherCategory = uint64(9751), uint64(9752)
	author := createHTTPContractUser(t, conn, contractTestID())
	authorToken := contractSessionToken(t, author)
	write := func(title string, category uint64) (uint64, string) {
		url, _ := saveContractImage(t, author.Id)
		body := mustReviewJSON(t, map[string]any{"title": title, "content": "这是一段带配图的正文 ![](" + url + ")", "categoryId": []uint64{category}, "topicStatus": 1, "contentType": 3})
		envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", string(body), authorToken))
		var id uint64
		if err := json.Unmarshal(envelope.Result, &id); err != nil || envelope.MessageCode != "content.moderation.checking" {
			t.Fatalf("pending write = %+v", envelope)
		}
		return id, url
	}
	ownTopic, _ := write("版主可审核的话题", ownCategory)
	otherTopic, otherImage := write("其他分类的话题", otherCategory)

	moderator := createHTTPContractUser(t, conn, contractTestID())
	if err := conn.Create(&moderators.Entity{UserId: moderator.Id, ScopeType: moderators.ScopeCategory, ScopeId: ownCategory, Status: moderators.StatusEnabled, CreatedBy: moderator.Id}).Error; err != nil {
		t.Fatal(err)
	}
	moderationservice.Invalidate()
	t.Cleanup(func() {
		conn.Where("user_id = ?", moderator.Id).Delete(&moderators.Entity{})
		moderationservice.Invalidate()
	})
	token := contractSessionToken(t, moderator)

	queue := decodeContractEnvelope(t, serveJSON(router, "/api/forum/moderation/review-queue", `{"kind":"topic","pageSize":50}`, token))
	var result struct {
		Items []api.ReviewQueueItem `json:"items"`
		Total int64                 `json:"total"`
	}
	if err := json.Unmarshal(queue.Result, &result); err != nil {
		t.Fatalf("queue = %+v", queue)
	}
	seen := map[uint64]bool{}
	for _, item := range result.Items {
		seen[item.Id] = true
	}
	if !seen[ownTopic] || seen[otherTopic] || result.Total != int64(len(result.Items)) {
		t.Fatalf("scoped queue ids=%v total=%d, want own %d only", seen, result.Total, ownTopic)
	}
	// 待审图片：版主可预览。
	if item := result.Items[0]; len(item.Images) == 0 || getImage(router, item.Images[0], token).Code != http.StatusOK {
		t.Fatalf("moderator cannot preview pending image: %+v", item)
	}
	// 其他分类的待审图片：分类版主无权预览；全局版主可以。
	if got := getImage(router, otherImage, token); got.Code != http.StatusNotFound {
		t.Fatalf("out-of-scope pending image status = %d, want 404", got.Code)
	}
	globalModerator := createHTTPContractUser(t, conn, contractTestID())
	if err := conn.Create(&moderators.Entity{UserId: globalModerator.Id, ScopeType: moderators.ScopeGlobal, Status: moderators.StatusEnabled, CreatedBy: globalModerator.Id}).Error; err != nil {
		t.Fatal(err)
	}
	moderationservice.Invalidate()
	t.Cleanup(func() { conn.Where("user_id = ?", globalModerator.Id).Delete(&moderators.Entity{}) })
	if got := getImage(router, otherImage, contractSessionToken(t, globalModerator)); got.Code != http.StatusOK {
		t.Fatalf("global moderator pending image status = %d, want 200", got.Code)
	}

	other := decodeContractEnvelope(t, serveJSON(router, "/api/forum/moderation/review-action", fmt.Sprintf(`{"kind":"topic","id":%d,"approve":true}`, otherTopic), token))
	if other.Code != 1 || other.MessageCode != "admin.review.notFound" || topics.Get(otherTopic).ProcessStatus != topics.ProcessStatusPending {
		t.Fatalf("out-of-scope review = %+v", other)
	}
	own := decodeContractEnvelope(t, serveJSON(router, "/api/forum/moderation/review-action", fmt.Sprintf(`{"kind":"topic","id":%d,"approve":true,"revisionId":%d}`, ownTopic, posts.Get(topics.Get(ownTopic).FirstPostId).LatestRevisionId), token))
	if own.Code != 0 || topics.Get(ownTopic).ProcessStatus != topics.ProcessStatusNormal {
		t.Fatalf("in-scope review = %+v", own)
	}
	if notificationCount(conn, author.Id, eventNotification.EventTypeReviewApproved) != 0 {
		t.Fatal("approvals should be quiet")
	}

	stranger := createHTTPContractUser(t, conn, contractTestID())
	denied := serveJSON(router, "/api/forum/moderation/review-queue", `{"kind":"topic"}`, contractSessionToken(t, stranger))
	assertFixtureEnvelope(t, decodeContractEnvelope(t, denied), contractFixture(t, "permission-denied.json"))
}

// waitFor 轮询等待先发后审的后台判定生效（最长 5 秒）。
func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func notificationCount(conn *gorm.DB, userID uint64, eventType string) int64 {
	var count int64
	conn.Model(&eventNotification.Entity{}).Where("user_id = ? AND event_type = ?", userID, eventType).Count(&count)
	return count
}

func TestPendingEditPreservesPublishedTopic(t *testing.T) {
	conn, router, stub := setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) {
		o.Mode = pageConfig.AiModerationModeDeferred
		o.TextModeration = true
	})
	hold := make(chan struct{})
	stub.holds = []chan struct{}{hold}
	author := createHTTPContractUser(t, conn, contractTestID())
	topicID, postID := contractTestID(), contractTestID()
	createContractPublishedTopic(t, conn, topicID, postID, author.Id)
	oldTopic, oldPost := topics.Get(topicID), posts.Get(postID)
	body := mustReviewJSON(t, map[string]any{"topicId": topicID, "title": "修改后的新标题", "content": "修改后的正文等待审核", "categoryId": []uint64{1}, "topicStatus": 1, "contentType": 3})
	envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", string(body), contractSessionToken(t, author)))
	if envelope.Code != 0 {
		t.Fatalf("save failed: %+v", envelope)
	}
	current, post := topics.Get(topicID), posts.Get(postID)
	if current.ProcessStatus != topics.ProcessStatusNormal || current.Title != oldTopic.Title || post.Content != oldPost.Content {
		t.Fatalf("pending edit replaced live content: topic=%+v post=%+v", current, post)
	}
}

// 先发后审的后台判定可能早于请求剩余步骤完成（issue #975 review）：钩子让请求在
// 启动判定后等到自动通过生效再继续。图片引用必须在启动判定前登记，否则会把已公开
// 内容的图片写回 PENDING。覆盖发主题、发回复与编辑回复三条路径。
func getTopicPage(router http.Handler, topicID uint64, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/p/post/%d", topicID), nil)
	request.Header.Set("X-Goose-Page", "true")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

// 回归：作者发布后跳转到自己的话题，待审期间（发布后检查或人工审核）作者能看到
// 正文且不能回复；他人仍是 404。作者在公开话题里的待审回复同样只对作者可见。
func TestAIModerationAuthorCanReadOwnPendingContent(t *testing.T) {
	conn, router, stub := setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) {
		o.Mode = pageConfig.AiModerationModeDeferred
		o.TextModeration = true
	})
	stub.holds = []chan struct{}{make(chan struct{}), make(chan struct{})}
	router.GET("/p/post/:id", middleware.JWTAuth, forum.TopicDetail)
	author := createHTTPContractUser(t, conn, contractTestID())
	stranger := createHTTPContractUser(t, conn, contractTestID())
	authorToken, strangerToken := contractSessionToken(t, author), contractSessionToken(t, stranger)
	url, _ := saveContractImage(t, author.Id)

	envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", writeTopicBody("作者能看到待审话题", "检查期间作者自己能看到这段正文 ![]("+url+")"), authorToken))
	var topicID uint64
	if err := json.Unmarshal(envelope.Result, &topicID); err != nil || envelope.MessageCode != "content.moderation.checking" {
		t.Fatalf("publish envelope = %+v", envelope)
	}

	page := getTopicPage(router, topicID, authorToken)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "检查期间作者自己能看到这段正文") {
		t.Fatalf("author topic page = %d %.300s", page.Code, page.Body.String())
	}
	var payload struct {
		Props struct {
			Topic struct {
				ProcessStatus int8 `json:"processStatus"`
			} `json:"topic"`
			Permissions struct {
				CanPost bool `json:"canPost"`
			} `json:"permissions"`
		} `json:"props"`
	}
	if err := json.Unmarshal(page.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Props.Topic.ProcessStatus != topics.ProcessStatusPending || payload.Props.Permissions.CanPost {
		t.Fatalf("author pending topic payload: processStatus=%d canPost=%v", payload.Props.Topic.ProcessStatus, payload.Props.Permissions.CanPost)
	}
	if got := getTopicPage(router, topicID, strangerToken); got.Code != http.StatusNotFound {
		t.Fatalf("stranger pending topic page = %d, want 404", got.Code)
	}
	if got := getTopicPage(router, topicID, ""); got.Code != http.StatusNotFound {
		t.Fatalf("anonymous pending topic page = %d, want 404", got.Code)
	}
	window := serveAuthSecurityJSON(router, http.MethodGet, fmt.Sprintf("/api/forum/posts/window?topicId=%d", topicID), "", authorToken)
	if decodeContractEnvelope(t, window).Code != 0 || !strings.Contains(window.Body.String(), "检查期间作者自己能看到这段正文") {
		t.Fatalf("author post window = %.300s", window.Body.String())
	}
	if got := decodeContractEnvelope(t, serveAuthSecurityJSON(router, http.MethodGet, fmt.Sprintf("/api/forum/posts/window?topicId=%d", topicID), "", strangerToken)); got.Code == 0 {
		t.Fatal("stranger must not read the pending topic window")
	}

	// 公开话题里的待审回复：作者在楼层窗口里能看到正文，他人看不到这一楼。
	publicTopicID, firstPostID := contractTestID(), contractTestID()
	createContractPublishedTopic(t, conn, publicTopicID, firstPostID, stranger.Id)
	body := mustReviewJSON(t, map[string]any{"topicId": publicTopicID, "content": "作者自己审核中的这条回复"})
	if reply := decodeContractEnvelope(t, serveJSON(router, "/api/forum/posts/create", string(body), authorToken)); reply.MessageCode != "content.moderation.checking" {
		t.Fatalf("reply envelope = %+v", reply)
	}
	path := fmt.Sprintf("/api/forum/posts/window?topicId=%d", publicTopicID)
	if got := serveAuthSecurityJSON(router, http.MethodGet, path, "", authorToken); !strings.Contains(got.Body.String(), "作者自己审核中的这条回复") {
		t.Fatalf("author must see own pending reply: %.300s", got.Body.String())
	}
	if got := serveAuthSecurityJSON(router, http.MethodGet, path, "", strangerToken); strings.Contains(got.Body.String(), "作者自己审核中的这条回复") {
		t.Fatal("other readers must not see a pending reply")
	}
}

func runSubmission(t *testing.T, conn *gorm.DB, revisionID uint64) {
	t.Helper()
	var jobs []taskQueue.Entity
	conn.Where("type = ?", publicationservice.TaskType).Find(&jobs)
	for _, job := range jobs {
		var payload publicationservice.Task
		if err := json.Unmarshal([]byte(job.TaskJson), &payload); err != nil {
			t.Fatal(err)
		}
		if payload.RevisionId == revisionID {
			if err := publicationservice.RunReviewTask(context.Background(), &job); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatalf("no durable job for revision %d", revisionID)
}

func submitContractTopic(t *testing.T, router *gin.Engine, token, title, content string) uint64 {
	t.Helper()
	envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", writeTopicBody(title, content), token))
	var id uint64
	if err := json.Unmarshal(envelope.Result, &id); err != nil || id == 0 || envelope.Code != 0 {
		t.Fatalf("submit: %+v", envelope)
	}
	return id
}

func TestAsyncModerationDurableSaveApproveRejectAndResubmit(t *testing.T) {
	for _, block := range []bool{false, true} {
		t.Run(fmt.Sprint(block), func(t *testing.T) {
			conn, router, stub := setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) { o.TextModeration = true })
			if block {
				stub.probabilities = map[string]float64{pageConfig.AiPolicyAdult: 0.99}
				stub.severity = 3
			}
			author := createHTTPContractUser(t, conn, contractTestID())
			token := contractSessionToken(t, author)
			url, _ := saveContractImage(t, author.Id)
			id := submitContractTopic(t, router, token, "异步审核发布测试", "发送成功后后台开始审核 ![]("+url+")")
			post := posts.Get(topics.Get(id).FirstPostId)
			if stub.jevCalls.Load() != 0 || post.ProcessStatus != posts.ProcessStatusPending || post.LatestRevisionId == 0 {
				t.Fatalf("publish waited for a model or lost revision: %+v", post)
			}
			if getImage(router, url, "").Code != 404 || getImage(router, url, token).Code != 200 {
				t.Fatal("private image permissions")
			}
			runSubmission(t, conn, post.LatestRevisionId)
			runSubmission(t, conn, post.LatestRevisionId)
			if block {
				if topics.Get(id).ProcessStatus != topics.ProcessStatusBlocked || postRevisions.Get(post.LatestRevisionId).ProcessStatus != posts.ProcessStatusBlocked {
					t.Fatal("rejected state missing")
				}
				if notificationCount(conn, author.Id, eventNotification.EventTypeReviewRejected) != 1 {
					t.Fatal("rejection must notify exactly once")
				}
				if getImage(router, url, "").Code != 404 || getImage(router, url, token).Code != 200 {
					t.Fatal("rejected image must remain private and editable")
				}
				stub.probabilities = map[string]float64{}
				stub.severity = 0
				body := mustReviewJSON(t, map[string]any{"topicId": id, "title": "已修改重新提交", "content": "修改后的内容重新进入审核", "categoryId": []uint64{1}, "topicStatus": 1, "contentType": 3})
				if result := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", string(body), token)); result.Code != 0 {
					t.Fatalf("blocked resubmit: %+v", result)
				}
				next := posts.Get(post.Id)
				if next.LatestRevisionId == post.LatestRevisionId || next.ProcessStatus != posts.ProcessStatusPending {
					t.Fatal("blocked resubmission did not create a pending revision")
				}
				runSubmission(t, conn, next.LatestRevisionId)
			}
			if topics.Get(id).ProcessStatus != topics.ProcessStatusNormal {
				t.Fatal("allow did not publish")
			}
			if !block && getImage(router, url, "").Code != 200 {
				t.Fatal("approved image not public")
			}
			if notificationCount(conn, author.Id, eventNotification.EventTypeReviewApproved) != 0 {
				t.Fatal("automatic approval should be quiet")
			}
		})
	}
}

func TestAsyncEditRejectPreservesLiveAndFencesOldDecisions(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) { o.TextModeration = true })
	author := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, author)
	topicID, postID := contractTestID(), contractTestID()
	createContractPublishedTopic(t, conn, topicID, postID, author.Id)
	old := posts.Get(postID)
	submit := func(content string) uint64 {
		body := mustReviewJSON(t, map[string]any{"postId": postID, "content": content})
		if e := decodeContractEnvelope(t, serveJSON(router, "/api/forum/posts/update", string(body), token)); e.Code != 0 {
			t.Fatalf("edit: %+v", e)
		}
		return posts.Get(postID).LatestRevisionId
	}
	first := submit("第一次修改仍在审核中的内容")
	second := submit("第二次修改仍在审核中的内容")
	if err := publicationservice.Review(context.Background(), first, moderationDecision.ActionAllow, "", 0); !errors.Is(err, publicationservice.ErrUnavailable) {
		t.Fatalf("stale review = %v", err)
	}
	if err := publicationservice.Review(context.Background(), second, moderationDecision.ActionBlock, "测试拒绝原因", 0); err != nil {
		t.Fatal(err)
	}
	if got := posts.Get(postID); got.Content != old.Content || got.ProcessStatus != posts.ProcessStatusNormal {
		t.Fatal("rejection withdrew public original")
	}
	notice := latestNotification(t, conn, author.Id)
	if !strings.Contains(notice.Payload.Content, "原有公开版本") || notice.Payload.PostId != postID {
		t.Fatalf("notice: %+v", notice)
	}
	// Deletion after a submission makes its delayed result inapplicable.
	third := submit("删除发生前提交的最新内容")
	conn.Model(&posts.Entity{}).Where("id = ?", postID).Update("visibility_status", posts.VisibilityUserDeleted)
	if err := publicationservice.Review(context.Background(), third, moderationDecision.ActionAllow, "", 0); !errors.Is(err, publicationservice.ErrUnavailable) {
		t.Fatalf("deleted review = %v", err)
	}
}

func TestAsyncReplyCountersAndManualRevisionFence(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) { o.TextModeration = true })
	author := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, author)
	topicID, postID := contractTestID(), contractTestID()
	createContractPublishedTopic(t, conn, topicID, postID, author.Id)
	before := topics.Get(topicID)
	body := mustReviewJSON(t, map[string]any{"topicId": topicID, "content": "尚未通过审核的新增回复内容"})
	e := decodeContractEnvelope(t, serveJSON(router, "/api/forum/posts/create", string(body), token))
	var result struct {
		Id uint64 `json:"id"`
	}
	if err := json.Unmarshal(e.Result, &result); err != nil || result.Id == 0 {
		t.Fatalf("reply: %+v", e)
	}
	if after := topics.Get(topicID); after.ReplyCount != before.ReplyCount || after.LastPostId != before.LastPostId {
		t.Fatal("pending reply changed public counters")
	}
	post := posts.Get(result.Id)
	manager := createContractSiteManager(t, conn)
	for _, revision := range []uint64{0, post.LatestRevisionId + 1} {
		b := fmt.Sprintf(`{"kind":"post","id":%d,"approve":true,"revisionId":%d}`, post.Id, revision)
		if got := decodeContractEnvelope(t, serveJSON(router, "/api/admin/review-action", b, contractSessionToken(t, manager))); got.Code == 0 {
			t.Fatal("manual action without matching revision was accepted")
		}
	}
	runSubmission(t, conn, post.LatestRevisionId)
	if after := topics.Get(topicID); after.ReplyCount != before.ReplyCount+1 {
		t.Fatalf("approved reply count %d", after.ReplyCount)
	}
}

func TestAsyncCandidatePrivacyAndCompletePublication(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) { o.TextModeration = true })
	router.GET("/", middleware.JWTAuth, forum.Home)
	router.GET("/p/post/:id", middleware.JWTAuth, forum.TopicDetail)
	router.GET("/api/forum/user/my-content", middleware.JWTAuthCheck, UpQueryReq(api.MyContentList))
	author := createHTTPContractUser(t, conn, contractTestID())
	stranger := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, author)
	topicID, postID := contractTestID(), contractTestID()
	createContractPublishedTopic(t, conn, topicID, postID, author.Id)
	imageURL, _ := saveContractImage(t, author.Id)
	candidate := "private-candidate-body-that-must-not-leak"
	body := mustReviewJSON(t, map[string]any{"topicId": topicID, "title": "private-candidate-title", "content": candidate, "images": []string{imageURL}, "categoryId": []uint64{1}, "topicStatus": 1, "contentType": 2})
	if e := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", string(body), token)); e.Code != 0 {
		t.Fatalf("submit: %+v", e)
	}
	// A missing render cache must rebuild only the owner projection.
	conn.Model(&postRevisions.Entity{}).Where("id = ?", posts.Get(postID).LatestRevisionId).Update("rendered_html", "")
	page := func(path, session string) string {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		r.Header.Set("X-Goose-Page", "1")
		if session != "" {
			r.Header.Set("Authorization", "Bearer "+session)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("%s: %d %.300s", path, w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	for _, path := range []string{"/", fmt.Sprintf("/p/post/%d", topicID), fmt.Sprintf("/api/forum/posts/window?topicId=%d", topicID)} {
		own := page(path, token)
		if !strings.Contains(own, candidate) && !strings.Contains(own, "private-candidate-title") {
			t.Fatalf("owner missing candidate: %s", path)
		}
		if live := posts.Get(postID); live.Content != "Topic first post content" || live.ProcessStatus != posts.ProcessStatusNormal {
			t.Fatalf("owner render modified public projection via %s: %+v", path, live)
		}
		// Read after the author to catch accidental mutation of shared cached entries.
		for _, session := range []string{"", contractSessionToken(t, stranger)} {
			public := page(path, session)
			if strings.Contains(public, candidate) || strings.Contains(public, "private-candidate-title") || strings.Contains(public, imageURL) {
				t.Fatalf("candidate leaked via %s", path)
			}
		}
	}
	history := page(fmt.Sprintf("/api/forum/posts/revisions?postId=%d", postID), contractSessionToken(t, stranger))
	if strings.Contains(history, candidate) {
		t.Fatal("candidate leaked via revision history")
	}
	rev := posts.Get(postID).LatestRevisionId
	runSubmission(t, conn, rev)
	published, first := topics.Get(topicID), posts.Get(postID)
	if published.Title != "private-candidate-title" || len(published.CategoryIds) != 1 || published.CategoryIds[0] != 1 || len(published.ImageUrls) != 1 || published.ImageUrls[0] != imageURL || first.Content != candidate || first.ContentType != 2 {
		t.Fatalf("partial publication: %+v / %+v", published, first)
	}
	if getImage(router, imageURL, "").Code != 200 {
		t.Fatal("published gallery remains private")
	}
	// Reject a later edit and verify the management route owns the rejected body.
	body = mustReviewJSON(t, map[string]any{"postId": postID, "content": "rejected-body-retained-for-author"})
	if e := decodeContractEnvelope(t, serveJSON(router, "/api/forum/posts/update", string(body), token)); e.Code != 0 {
		t.Fatalf("edit: %+v", e)
	}
	if err := publicationservice.Review(context.Background(), posts.Get(postID).LatestRevisionId, moderationDecision.ActionBlock, "请修改正文", 0); err != nil {
		t.Fatal(err)
	}
	manager := page("/api/forum/user/my-content?contentType=topic", token)
	if !strings.Contains(manager, "rejected-body-retained-for-author") || !strings.Contains(manager, "请修改正文") || !strings.Contains(manager, `"hasPublishedVersion":true`) {
		t.Fatalf("missing recovery content: %.800s", manager)
	}
	if strings.Contains(page("/", token), "rejected-body-retained-for-author") {
		t.Fatal("rejected candidate remained in feed")
	}
}

func TestAsyncSaveIsAtomicAndDraftPublishCannotBypassReview(t *testing.T) {
	conn, router, stub := setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) { o.TextModeration = true })
	author := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, author)
	imageURL, _ := saveContractImage(t, author.Id)
	var before int64
	conn.Model(&posts.Entity{}).Count(&before)
	if err := conn.Callback().Create().Before("gorm:create").Register("test:fail-review-task", func(tx *gorm.DB) {
		if tx.Statement.Table == "task_queue" {
			_ = tx.AddError(errors.New("injected task failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	e := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", writeTopicBody("事务失败测试", "保存失败必须回滚全部内容 ![]("+imageURL+")"), token))
	if err := conn.Callback().Create().Remove("test:fail-review-task"); err != nil {
		t.Fatal(err)
	}
	var after int64
	conn.Model(&posts.Entity{}).Count(&after)
	if e.Code == 0 || before != after {
		t.Fatalf("task failure left content: %+v, count %d -> %d", e, before, after)
	}
	var refs int64
	conn.Model(&fileUsage.Entity{}).Where("user_id = ?", author.Id).Count(&refs)
	if refs != 0 {
		t.Fatal("failed submission left private references")
	}
	body := mustReviewJSON(t, map[string]any{"title": "保存草稿测试", "content": "这份草稿在点击发布时必须审核", "categoryId": []uint64{1}, "topicStatus": 0, "contentType": 3})
	e = decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", string(body), token))
	var topicID uint64
	if err := json.Unmarshal(e.Result, &topicID); err != nil || e.Code != 0 {
		t.Fatalf("draft: %+v", e)
	}
	if posts.Get(topics.Get(topicID).FirstPostId).LatestRevisionId != 0 || stub.jevCalls.Load() != 0 {
		t.Fatal("saving a draft started review")
	}
	e = decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/status", fmt.Sprintf(`{"topicId":%d,"topicStatus":1}`, topicID), token))
	if e.Code != 0 || topics.Get(topicID).ProcessStatus != topics.ProcessStatusPending {
		t.Fatalf("draft publication bypassed review: %+v", e)
	}
}

func TestAsyncLegacyPendingAdoptionIsIdempotent(t *testing.T) {
	conn, _, _ := setupAIModerationContractTest(t, nil)
	author := createHTTPContractUser(t, conn, contractTestID())
	topicID, postID := contractTestID(), contractTestID()
	createContractPublishedTopic(t, conn, topicID, postID, author.Id)
	conn.Model(&posts.Entity{}).Where("id = ?", postID).Update("process_status", posts.ProcessStatusPending)
	conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("process_status", topics.ProcessStatusPending)
	for i := 0; i < 2; i++ {
		if err := publicationservice.AdoptPending(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	post := posts.Get(postID)
	var count int64
	raw := mustReviewJSON(t, publicationservice.Task{RevisionId: post.LatestRevisionId})
	conn.Model(&taskQueue.Entity{}).Where("type = ? AND task_json = ?", publicationservice.TaskType, string(raw)).Count(&count)
	if post.LatestRevisionId == 0 || count != 1 || post.PublishedRevisionId != 0 || post.Content != "Topic first post content" {
		t.Fatalf("legacy adoption lost content or duplicated job: %+v, jobs %d", post, count)
	}
}

func mustReviewJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestModerationCandidateCategoryScopeCannotBeBypassedWithFirstPost(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) { o.TextModeration = true })
	router.POST("/api/forum/moderation/review-queue", middleware.JWTAuthCheck, UpButterReq(api.ModerationReviewQueue))
	router.POST("/api/forum/moderation/review-action", middleware.JWTAuthCheck, middleware.CheckWritableAccount, UpButterReq(api.ModerationReviewAction))
	author := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, author)
	write := func(id, category uint64) uint64 {
		body := mustReviewJSON(t, map[string]any{"topicId": id, "title": "分类权限回归", "content": "迁往其他分类的候选正文", "categoryId": []uint64{category}, "topicStatus": 1, "contentType": 3})
		e := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", string(body), token))
		var result uint64
		if err := json.Unmarshal(e.Result, &result); err != nil || e.Code != 0 {
			t.Fatalf("write: %+v", e)
		}
		return result
	}
	ownCategory := contractTestID()
	otherCategory := ownCategory + 1
	topicID := write(0, ownCategory)
	postID := topics.Get(topicID).FirstPostId
	runSubmission(t, conn, posts.Get(postID).LatestRevisionId)
	write(topicID, otherCategory)
	revisionID := posts.Get(postID).LatestRevisionId
	moderator := createHTTPContractUser(t, conn, contractTestID())
	grant := func(category uint64) {
		if err := conn.Create(&moderators.Entity{UserId: moderator.Id, ScopeType: moderators.ScopeCategory, ScopeId: category, Status: moderators.StatusEnabled}).Error; err != nil {
			t.Fatal(err)
		}
		moderationservice.Invalidate()
	}
	grant(ownCategory)
	t.Cleanup(func() {
		conn.Where("user_id = ?", moderator.Id).Delete(&moderators.Entity{})
		moderationservice.Invalidate()
	})
	moderatorToken := contractSessionToken(t, moderator)
	queue := func() struct {
		Items []api.ReviewQueueItem
		Total int64
	} {
		e := decodeContractEnvelope(t, serveJSON(router, "/api/forum/moderation/review-queue", `{"kind":"topic","pageSize":1}`, moderatorToken))
		var result struct {
			Items []api.ReviewQueueItem
			Total int64
		}
		if err := json.Unmarshal(e.Result, &result); err != nil || e.Code != 0 {
			t.Fatalf("queue: %+v", e)
		}
		return result
	}
	if q := queue(); q.Total != 0 || len(q.Items) != 0 {
		t.Errorf("candidate outside moderator categories leaked: %+v", q)
	}
	for _, target := range []struct {
		kind string
		id   uint64
	}{{"topic", topicID}, {"post", postID}} {
		body := fmt.Sprintf(`{"kind":%q,"id":%d,"approve":true,"revisionId":%d}`, target.kind, target.id, revisionID)
		e := decodeContractEnvelope(t, serveJSON(router, "/api/forum/moderation/review-action", body, moderatorToken))
		if e.Code != 1 || e.MessageCode != "admin.review.notFound" {
			t.Errorf("out-of-scope %s review accepted: %+v", target.kind, e)
		}
	}
	if postRevisions.Get(revisionID).ProcessStatus != posts.ProcessStatusPending {
		t.Fatal("out-of-scope action published candidate")
	}
	history := decodeContractEnvelope(t, serveAuthSecurityJSON(router, http.MethodGet, fmt.Sprintf("/api/forum/posts/revisions?postId=%d", postID), "", moderatorToken))
	var versions struct {
		Versions []struct {
			Content       string
			ProcessStatus int8
		}
	}
	if err := json.Unmarshal(history.Result, &versions); err != nil || history.Code != 0 {
		t.Fatalf("history: %+v", history)
	}
	for _, version := range versions.Versions {
		if version.ProcessStatus == posts.ProcessStatusPending && version.Content != "" {
			t.Fatal("out-of-scope revision history exposed candidate")
		}
	}
	grant(otherCategory)
	if q := queue(); q.Total != 1 || len(q.Items) != 1 || q.Items[0].RevisionId != revisionID {
		t.Fatalf("both-category moderator queue: %+v", q)
	}
	body := fmt.Sprintf(`{"kind":"post","id":%d,"approve":true,"revisionId":%d}`, postID, revisionID)
	if e := decodeContractEnvelope(t, serveJSON(router, "/api/forum/moderation/review-action", body, moderatorToken)); e.Code != 0 {
		t.Fatalf("authorized first-post review: %+v", e)
	}
}

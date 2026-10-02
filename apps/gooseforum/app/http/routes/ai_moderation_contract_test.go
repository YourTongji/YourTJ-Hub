package routes

import (
	"bytes"
	"encoding/json"
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
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
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
	body, _ := json.Marshal(map[string]any{"title": title, "content": content, "categoryId": []uint64{1}, "topicStatus": 1, "contentType": 3})
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

func TestAIModerationTopicWriteBlocksWithoutCreatingContent(t *testing.T) {
	conn, router, stub := setupAIModerationContractTest(t, nil)
	stub.probabilities = map[string]float64{pageConfig.AiPolicyAdult: 0.98}
	stub.severity = 2.9
	author := createHTTPContractUser(t, conn, contractTestID())
	url, name := saveContractImage(t, author.Id)

	recorder := serveJSON(router, "/api/forum/topics/write", writeTopicBody("AI block contract", "正常文字配图 ![]("+url+")"), contractSessionToken(t, author))
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "topic-write-ai-blocked.json"))
	var count int64
	conn.Model(&topics.Entity{}).Where("user_id = ?", author.Id).Count(&count)
	if count != 0 {
		t.Fatalf("blocked publish created %d topics", count)
	}
	if statuses := usageStatuses(t, conn, name); len(statuses) != 0 {
		t.Fatalf("blocked publish registered usages %v", statuses)
	}
	if decision := moderationDecision.LatestForSubjects(moderationDecision.SubjectTopic, []uint64{0})[0]; decision.FinalAction != moderationDecision.ActionBlock {
		t.Fatalf("block decision not recorded: %+v", decision)
	}
}

func TestAIModerationExternalImageBlockedHTTPContract(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) {
		o.ExternalImageAction = pageConfig.AiModerationActionBlock
	})
	author := createHTTPContractUser(t, conn, contractTestID())
	recorder := serveJSON(router, "/api/forum/topics/write", writeTopicBody("AI external contract", "这里引用了一张外链图片 ![](https://img.example.com/x.png)"), contractSessionToken(t, author))
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "topic-write-ai-external-image-blocked.json"))
}

// 待审内容（AI review）的图片对匿名/他人不可读，作者与站点管理员可授权预览；
// 人工批准后图片随内容公开，并回写人工结论。
func TestAIModerationPendingTopicImagesStayPrivateUntilApproved(t *testing.T) {
	conn, router, stub := setupAIModerationContractTest(t, nil)
	stub.probabilities = map[string]float64{pageConfig.AiPolicyViolence: 0.8}
	stub.severity = 2
	author := createHTTPContractUser(t, conn, contractTestID())
	stranger := createHTTPContractUser(t, conn, contractTestID())
	manager := createContractSiteManager(t, conn)
	url, name := saveContractImage(t, author.Id)

	recorder := serveJSON(router, "/api/forum/topics/write", writeTopicBody("AI review contract", "这是一段带配图的正文 ![]("+url+")"), contractSessionToken(t, author))
	envelope := decodeContractEnvelope(t, recorder)
	assertFixtureEnvelope(t, envelope, contractFixture(t, "topic-write-pending-review.json"))
	var topicID uint64
	if err := json.Unmarshal(envelope.Result, &topicID); err != nil || topicID == 0 {
		t.Fatalf("topic id = %s", envelope.Result)
	}
	topic := topics.Get(topicID)
	if topic.ProcessStatus != topics.ProcessStatusPending {
		t.Fatalf("topic process status = %d, want pending", topic.ProcessStatus)
	}
	if statuses := usageStatuses(t, conn, name); len(statuses) != 1 || statuses[0] != fileUsage.UsageStatusPending {
		t.Fatalf("usage statuses = %v, want [PENDING]", statuses)
	}

	if got := getImage(router, url, ""); got.Code != http.StatusNotFound {
		t.Fatalf("anonymous pending image status = %d, want 404", got.Code)
	}
	if got := getImage(router, url, contractSessionToken(t, stranger)); got.Code != http.StatusNotFound {
		t.Fatalf("stranger pending image status = %d, want 404", got.Code)
	}
	ownerView := getImage(router, url, contractSessionToken(t, author))
	if ownerView.Code != http.StatusOK || ownerView.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("owner preview status=%d cache=%q", ownerView.Code, ownerView.Header().Get("Cache-Control"))
	}
	managerToken := contractSessionToken(t, manager)
	if got := getImage(router, url, managerToken); got.Code != http.StatusOK {
		t.Fatalf("site manager preview status = %d, want 200", got.Code)
	}

	queue := serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/review-queue", `{"kind":"topic","pageSize":50}`, managerToken)
	var queueResult struct {
		Items []api.ReviewQueueItem `json:"items"`
	}
	if err := json.Unmarshal(decodeContractEnvelope(t, queue).Result, &queueResult); err != nil {
		t.Fatal(err)
	}
	var item *api.ReviewQueueItem
	for i := range queueResult.Items {
		if queueResult.Items[i].Id == topicID {
			item = &queueResult.Items[i]
		}
	}
	if item == nil || item.AiReview == nil || len(item.Images) != 1 || item.AiReview.TriggeredPolicies[0] != pageConfig.AiPolicyViolence {
		t.Fatalf("review queue item = %+v", item)
	}
	if item.AiReview.Images[0].Evidence == "" {
		t.Fatal("review queue must expose the truncated evidence summary to moderators")
	}

	action := serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/review-action", fmt.Sprintf(`{"kind":"topic","id":%d,"approve":true}`, topicID), managerToken)
	if envelope := decodeContractEnvelope(t, action); envelope.Code != 0 {
		t.Fatalf("approve failed: %s", action.Body.String())
	}
	if statuses := usageStatuses(t, conn, name); len(statuses) != 1 || statuses[0] != fileUsage.UsageStatusActive {
		t.Fatalf("usage statuses after approve = %v, want [ACTIVE]", statuses)
	}
	if got := getImage(router, url, ""); got.Code != http.StatusOK || !strings.Contains(got.Header().Get("Cache-Control"), "public") {
		t.Fatalf("approved image status=%d cache=%q", got.Code, got.Header().Get("Cache-Control"))
	}
	if decision := moderationDecision.LatestForSubjects(moderationDecision.SubjectTopic, []uint64{topicID})[topicID]; decision.HumanAction != moderationDecision.HumanApproved {
		t.Fatalf("human outcome = %q, want approved", decision.HumanAction)
	}
	// 作者收到“已通过”通知，链接到话题。
	notice := latestNotification(t, conn, author.Id)
	if notice.EventType != eventNotification.EventTypeReviewApproved || notice.Payload.TopicId != topicID ||
		notice.Payload.TemplateKey != eventNotification.TemplateReviewApproved || notice.Payload.TopicTitle != "AI review contract" {
		t.Fatalf("approval notification = %+v", notice)
	}
}

func latestNotification(t *testing.T, conn *gorm.DB, userID uint64) eventNotification.Entity {
	t.Helper()
	var notice eventNotification.Entity
	if err := conn.Where("user_id = ?", userID).Order("id DESC").First(&notice).Error; err != nil {
		t.Fatalf("load notification for user %d: %v", userID, err)
	}
	return notice
}

func TestAIModerationRejectedReplyImagesNeverBecomePublic(t *testing.T) {
	conn, router, stub := setupAIModerationContractTest(t, nil)
	stub.probabilities = map[string]float64{pageConfig.AiPolicyIllegalOrDangerous: 0.9}
	author := createHTTPContractUser(t, conn, contractTestID())
	manager := createContractSiteManager(t, conn)
	topicID, firstPostID := contractTestID(), contractTestID()
	createContractPublishedTopic(t, conn, topicID, firstPostID, author.Id)
	url, name := saveContractImage(t, author.Id)

	body, _ := json.Marshal(map[string]any{"topicId": topicID, "content": "这是一条带配图的回复 ![](" + url + ")"})
	envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/posts/create", string(body), contractSessionToken(t, author)))
	if envelope.Code != 0 || envelope.MessageCode != "content.moderation.pendingReview" {
		t.Fatalf("reply envelope = %+v", envelope)
	}
	var created struct {
		Id uint64 `json:"id"`
	}
	_ = json.Unmarshal(envelope.Result, &created)
	if post := posts.Get(created.Id); post.ProcessStatus != posts.ProcessStatusPending {
		t.Fatalf("reply process status = %d", post.ProcessStatus)
	}
	reject := serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/review-action", fmt.Sprintf(`{"kind":"post","id":%d,"approve":false}`, created.Id), contractSessionToken(t, manager))
	if decodeContractEnvelope(t, reject).Code != 0 {
		t.Fatalf("reject failed: %s", reject.Body.String())
	}
	if statuses := usageStatuses(t, conn, name); len(statuses) != 1 || statuses[0] != fileUsage.UsageStatusPending {
		t.Fatalf("rejected reply usages = %v, want [PENDING]", statuses)
	}
	if got := getImage(router, url, ""); got.Code != http.StatusNotFound {
		t.Fatalf("rejected image status = %d, want 404", got.Code)
	}
	if decision := moderationDecision.LatestForSubjects(moderationDecision.SubjectPost, []uint64{created.Id})[created.Id]; decision.HumanAction != moderationDecision.HumanRejected {
		t.Fatalf("human outcome = %q, want rejected", decision.HumanAction)
	}
	// 作者收到“未通过”通知：只带标题快照，不带可跳转的话题/楼层。
	notice := latestNotification(t, conn, author.Id)
	if notice.EventType != eventNotification.EventTypeReviewRejected || notice.Payload.TopicId != 0 || notice.Payload.PostNo != 0 ||
		notice.TopicID != topicID || notice.Payload.TopicTitle == "" {
		t.Fatalf("rejection notification = %+v", notice)
	}
}

func TestAIModerationAllowAndEditPaths(t *testing.T) {
	conn, router, stub := setupAIModerationContractTest(t, nil)
	author := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, author)
	url, name := saveContractImage(t, author.Id)
	envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", writeTopicBody("AI allow contract", "今天拍到的校园风景 ![]("+url+")"), token))
	if envelope.Code != 0 || envelope.MessageCode != "" {
		t.Fatalf("allowed publish envelope = %+v", envelope)
	}
	if statuses := usageStatuses(t, conn, name); len(statuses) != 1 || statuses[0] != fileUsage.UsageStatusActive {
		t.Fatalf("allowed usages = %v", statuses)
	}
	var topicID uint64
	_ = json.Unmarshal(envelope.Result, &topicID)
	topic := topics.Get(topicID)

	// 编辑首楼后命中 AI block：拒绝编辑，原内容与公开状态保持不变。
	stub.probabilities = map[string]float64{pageConfig.AiPolicyAdult: 0.99}
	editURL, _ := saveContractImage(t, author.Id)
	body, _ := json.Marshal(map[string]any{"postId": topic.FirstPostId, "content": "编辑后替换了一张配图 ![](" + editURL + ")"})
	edit := decodeContractEnvelope(t, serveJSON(router, "/api/forum/posts/update", string(body), token))
	if edit.Code != 1 || edit.MessageCode != "content.aiModeration.blocked" {
		t.Fatalf("blocked edit envelope = %+v", edit)
	}
	if post := posts.Get(topic.FirstPostId); strings.Contains(post.Content, editURL) || topics.Get(topicID).ProcessStatus != topics.ProcessStatusNormal {
		t.Fatal("blocked edit must not change the stored content or status")
	}
}

// 人工结论只回写到评估过当前正文的 AI 决策：AI 放行后作者再编辑、编辑因
// 敏感词转审（跳过 AI），版主拒绝的是新版本，不能记到旧版本的放行决策上。
func TestAIModerationHumanOutcomeSkipsStaleDecision(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, nil)
	security := hotdataserve.GetSecuritySettingsConfigCache()
	security.SensitiveWords = []string{"违禁词九七五"}
	security.SensitiveAction = "review"
	persistHTTPContractConfig(t, conn, pageConfig.SecuritySettings, security)
	hotdataserve.ClearSecuritySettingsConfigCache()
	author := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, author)
	url, _ := saveContractImage(t, author.Id)
	envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", writeTopicBody("AI stale label", "今天拍到的校园一角 ![]("+url+")"), token))
	var topicID uint64
	if err := json.Unmarshal(envelope.Result, &topicID); err != nil || envelope.Code != 0 || envelope.MessageCode != "" {
		t.Fatalf("allowed publish envelope = %+v", envelope)
	}
	before := moderationDecision.LatestForSubjects(moderationDecision.SubjectTopic, []uint64{topicID})[topicID]
	if before.Id == 0 || before.FinalAction != moderationDecision.ActionAllow {
		t.Fatalf("initial decision = %+v", before)
	}

	topic := topics.Get(topicID)
	body, _ := json.Marshal(map[string]any{"postId": topic.FirstPostId, "content": "编辑后加入了违禁词九七五 ![](" + url + ")"})
	if edit := decodeContractEnvelope(t, serveJSON(router, "/api/forum/posts/update", string(body), token)); edit.Code != 0 || edit.MessageCode != "content.moderation.pendingReview" {
		t.Fatalf("sensitive edit envelope = %+v", edit)
	}
	if topics.Get(topicID).ProcessStatus != topics.ProcessStatusPending {
		t.Fatal("sensitive edit must send the topic to review")
	}

	manager := createContractSiteManager(t, conn)
	action := serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/review-action", fmt.Sprintf(`{"kind":"topic","id":%d,"approve":false}`, topicID), contractSessionToken(t, manager))
	if decodeContractEnvelope(t, action).Code != 0 {
		t.Fatalf("reject failed: %s", action.Body.String())
	}
	after := moderationDecision.LatestForSubjects(moderationDecision.SubjectTopic, []uint64{topicID})[topicID]
	if after.Id != before.Id || after.HumanAction != "" {
		t.Fatalf("stale decision labeled: %+v", after)
	}
}

// 回归：现有敏感词转审路径同样不得让待审内容的图片提前公开（issue #975 收口）。
func TestSensitiveReviewRegistersPendingImages(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) { o.Enabled = false })
	security := hotdataserve.GetSecuritySettingsConfigCache()
	security.SensitiveWords = []string{"违禁词九七五"}
	security.SensitiveAction = "review"
	persistHTTPContractConfig(t, conn, pageConfig.SecuritySettings, security)
	hotdataserve.ClearSecuritySettingsConfigCache()
	author := createHTTPContractUser(t, conn, contractTestID())
	url, name := saveContractImage(t, author.Id)
	envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", writeTopicBody("敏感词转审", "违禁词九七五 ![]("+url+")"), contractSessionToken(t, author)))
	if envelope.Code != 0 || envelope.MessageCode != "content.moderation.pendingReview" {
		t.Fatalf("sensitive review envelope = %+v", envelope)
	}
	if statuses := usageStatuses(t, conn, name); len(statuses) != 1 || statuses[0] != fileUsage.UsageStatusPending {
		t.Fatalf("sensitive review usages = %v, want [PENDING]", statuses)
	}
	if got := getImage(router, url, ""); got.Code != http.StatusNotFound {
		t.Fatalf("anonymous image status = %d, want 404", got.Code)
	}
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
	write := func(title string, category uint64) uint64 {
		url, _ := saveContractImage(t, author.Id)
		body, _ := json.Marshal(map[string]any{"title": title, "content": "这是一段带配图的正文 ![](" + url + ")", "categoryId": []uint64{category}, "topicStatus": 1, "contentType": 3})
		envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", string(body), authorToken))
		var id uint64
		if err := json.Unmarshal(envelope.Result, &id); err != nil || envelope.MessageCode != "content.moderation.pendingReview" {
			t.Fatalf("pending write = %+v", envelope)
		}
		return id
	}
	ownTopic, otherTopic := write("版主可审核的话题", ownCategory), write("其他分类的话题", otherCategory)

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

	other := decodeContractEnvelope(t, serveJSON(router, "/api/forum/moderation/review-action", fmt.Sprintf(`{"kind":"topic","id":%d,"approve":true}`, otherTopic), token))
	if other.Code != 1 || other.MessageCode != "admin.review.notFound" || topics.Get(otherTopic).ProcessStatus != topics.ProcessStatusPending {
		t.Fatalf("out-of-scope review = %+v", other)
	}
	own := decodeContractEnvelope(t, serveJSON(router, "/api/forum/moderation/review-action", fmt.Sprintf(`{"kind":"topic","id":%d,"approve":true}`, ownTopic), token))
	if own.Code != 0 || topics.Get(ownTopic).ProcessStatus != topics.ProcessStatusNormal {
		t.Fatalf("in-scope review = %+v", own)
	}
	if notice := latestNotification(t, conn, author.Id); notice.EventType != eventNotification.EventTypeReviewApproved || notice.Payload.TopicId != ownTopic {
		t.Fatalf("author notification = %+v", notice)
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

func deferredMode(o *pageConfig.AiModerationOptions) { o.Mode = pageConfig.AiModerationModeDeferred }

// 先发后审：发布立即返回且对他人不可见；后台判定 allow 后自动公开、图片转 ACTIVE，
// 不打扰作者（不发通知），也不记人工标签。
func TestAIModerationDeferredAllowPublishesSilently(t *testing.T) {
	conn, router, stub := setupAIModerationContractTest(t, deferredMode)
	hold := make(chan struct{})
	stub.holds = []chan struct{}{hold}
	author := createHTTPContractUser(t, conn, contractTestID())
	manager := createContractSiteManager(t, conn)
	url, name := saveContractImage(t, author.Id)

	envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", writeTopicBody("先发后审的话题", "今天拍到的校园风景 ![]("+url+")"), contractSessionToken(t, author)))
	assertFixtureEnvelope(t, envelope, contractFixture(t, "topic-write-checking.json"))
	var topicID uint64
	if err := json.Unmarshal(envelope.Result, &topicID); err != nil || topicID == 0 {
		t.Fatalf("topic id = %s", envelope.Result)
	}
	if topics.Get(topicID).ProcessStatus != topics.ProcessStatusPending {
		t.Fatal("deferred publish must store the topic as pending until the check finishes")
	}
	if got := getImage(router, url, ""); got.Code != http.StatusNotFound {
		t.Fatalf("anonymous image during check = %d, want 404", got.Code)
	}

	// 检查进行中：审核队列标记“自动检查中”。
	queue := serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/review-queue", `{"kind":"topic","pageSize":50}`, contractSessionToken(t, manager))
	var queueResult struct {
		Items []api.ReviewQueueItem `json:"items"`
	}
	if err := json.Unmarshal(decodeContractEnvelope(t, queue).Result, &queueResult); err != nil {
		t.Fatal(err)
	}
	checking := false
	for _, item := range queueResult.Items {
		if item.Id == topicID {
			checking = item.AiChecking && item.AiReview == nil
		}
	}
	if !checking {
		t.Fatalf("queue must flag the topic as being checked: %+v", queueResult.Items)
	}

	stub.release(hold)
	waitFor(t, "auto approval", func() bool { return topics.Get(topicID).ProcessStatus == topics.ProcessStatusNormal })
	if statuses := usageStatuses(t, conn, name); len(statuses) != 1 || statuses[0] != fileUsage.UsageStatusActive {
		t.Fatalf("usage statuses after auto approval = %v, want [ACTIVE]", statuses)
	}
	if got := getImage(router, url, ""); got.Code != http.StatusOK {
		t.Fatalf("anonymous image after approval = %d, want 200", got.Code)
	}
	decision := moderationDecision.LatestForSubjects(moderationDecision.SubjectTopic, []uint64{topicID})[topicID]
	if decision.Mode != pageConfig.AiModerationModeDeferred || decision.AppliedAction != moderationDecision.ActionAllow || decision.HumanAction != "" {
		t.Fatalf("deferred decision = %+v", decision)
	}
	if n := notificationCount(conn, author.Id, eventNotification.EventTypeReviewApproved); n != 0 {
		t.Fatalf("auto approval sent %d notifications, want none", n)
	}
	if moderationservice.DeferredChecking(moderationDecision.SubjectTopic, []uint64{topicID})[topicID] {
		t.Fatal("finished check must leave the in-flight set")
	}
}

// 先发后审：后台判定 block 时自动拒绝，图片保持不可读，作者收到“未通过”通知。
func TestAIModerationDeferredBlockRejectsAndNotifies(t *testing.T) {
	conn, router, stub := setupAIModerationContractTest(t, deferredMode)
	stub.probabilities = map[string]float64{pageConfig.AiPolicyAdult: 0.99}
	author := createHTTPContractUser(t, conn, contractTestID())
	topicID, firstPostID := contractTestID(), contractTestID()
	createContractPublishedTopic(t, conn, topicID, firstPostID, author.Id)
	url, name := saveContractImage(t, author.Id)

	body, _ := json.Marshal(map[string]any{"topicId": topicID, "content": "这是一条带配图的回复 ![](" + url + ")"})
	envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/posts/create", string(body), contractSessionToken(t, author)))
	if envelope.Code != 0 || envelope.MessageCode != "content.moderation.checking" {
		t.Fatalf("reply envelope = %+v", envelope)
	}
	var created struct {
		Id uint64 `json:"id"`
	}
	if err := json.Unmarshal(envelope.Result, &created); err != nil || created.Id == 0 {
		t.Fatalf("reply result = %s", envelope.Result)
	}
	waitFor(t, "auto rejection", func() bool { return posts.Get(created.Id).ProcessStatus == posts.ProcessStatusBlocked })
	if statuses := usageStatuses(t, conn, name); len(statuses) != 1 || statuses[0] != fileUsage.UsageStatusPending {
		t.Fatalf("usage statuses after auto rejection = %v, want [PENDING]", statuses)
	}
	if notice := latestNotification(t, conn, author.Id); notice.EventType != eventNotification.EventTypeReviewRejected {
		t.Fatalf("author notification = %q, want review_rejected", notice.EventType)
	}
	if decision := moderationDecision.LatestForSubjects(moderationDecision.SubjectPost, []uint64{created.Id})[created.Id]; decision.AppliedAction != moderationDecision.ActionBlock || decision.HumanAction != "" {
		t.Fatalf("deferred decision = %+v", decision)
	}
}

// 先发后审：介于两线（review）时留在审核队列并附带 AI 原因，不自动处理。
func TestAIModerationDeferredReviewStaysQueued(t *testing.T) {
	conn, router, stub := setupAIModerationContractTest(t, deferredMode)
	stub.probabilities = map[string]float64{pageConfig.AiPolicyViolence: 0.8}
	author := createHTTPContractUser(t, conn, contractTestID())
	manager := createContractSiteManager(t, conn)
	url, _ := saveContractImage(t, author.Id)
	envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", writeTopicBody("先发后审转人工", "这是一段带配图的正文 ![]("+url+")"), contractSessionToken(t, author)))
	var topicID uint64
	_ = json.Unmarshal(envelope.Result, &topicID)
	waitFor(t, "deferred review decision", func() bool {
		return moderationDecision.LatestForSubjects(moderationDecision.SubjectTopic, []uint64{topicID})[topicID].Id != 0 &&
			!moderationservice.DeferredChecking(moderationDecision.SubjectTopic, []uint64{topicID})[topicID]
	})
	if topics.Get(topicID).ProcessStatus != topics.ProcessStatusPending {
		t.Fatal("review result must keep the topic pending for a moderator")
	}
	queue := serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/review-queue", `{"kind":"topic","pageSize":50}`, contractSessionToken(t, manager))
	var queueResult struct {
		Items []api.ReviewQueueItem `json:"items"`
	}
	if err := json.Unmarshal(decodeContractEnvelope(t, queue).Result, &queueResult); err != nil {
		t.Fatal(err)
	}
	for _, item := range queueResult.Items {
		if item.Id == topicID {
			if item.AiChecking || item.AiReview == nil || item.AiReview.AppliedAction != moderationDecision.ActionReview {
				t.Fatalf("queued item = %+v", item)
			}
			return
		}
	}
	t.Fatal("topic missing from the review queue")
}

// 先发后审：站点拦截外链图片时无需模型即可判定，仍在编辑器里同步提示，不写入内容。
func TestAIModerationDeferredBlocksExternalImagesUpfront(t *testing.T) {
	conn, router, stub := setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) {
		o.Mode = pageConfig.AiModerationModeDeferred
		o.ExternalImageAction = pageConfig.AiModerationActionBlock
	})
	author := createHTTPContractUser(t, conn, contractTestID())
	countTopics := func() (n int64) { conn.Model(&topics.Entity{}).Count(&n); return }
	before := countTopics()
	envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", writeTopicBody("引用站外图片的话题", "引用一张站外图片 ![](https://example.com/a.png)"), contractSessionToken(t, author)))
	if envelope.Code != 1 || envelope.MessageCode != "content.aiModeration.externalImageBlocked" {
		t.Fatalf("external image envelope = %+v", envelope)
	}
	if countTopics() != before || stub.jevCalls.Load() != 0 {
		t.Fatal("upfront external-image block must not write content or call models")
	}
}

// 先发后审：检查期间作者再次编辑，旧版本的 allow 结论作废，只有新版本的结论生效。
func TestAIModerationDeferredIgnoresSupersededOutcome(t *testing.T) {
	conn, router, stub := setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) {
		o.Mode = pageConfig.AiModerationModeDeferred
		o.TextModeration = true
	})
	first, second := make(chan struct{}), make(chan struct{})
	stub.holds = []chan struct{}{first, second}
	stub.blockMarker = "编辑后的违规版本"
	author := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, author)
	envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", writeTopicBody("检查期间改稿", "最初版本的正文内容足够长"), token))
	var topicID uint64
	if err := json.Unmarshal(envelope.Result, &topicID); err != nil || envelope.MessageCode != "content.moderation.checking" {
		t.Fatalf("publish envelope = %+v", envelope)
	}
	waitFor(t, "first check to start", func() bool { return stub.jevCalls.Load() == 1 })
	body, _ := json.Marshal(map[string]any{"topicId": topicID, "title": "检查期间改稿", "content": "编辑后的违规版本正文内容",
		"categoryId": []uint64{1}, "topicStatus": 1, "contentType": 3})
	if edit := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", string(body), token)); edit.Code != 0 || edit.MessageCode != "content.moderation.checking" {
		t.Fatalf("edit envelope = %+v", edit)
	}
	waitFor(t, "second check to start", func() bool { return stub.jevCalls.Load() == 2 })

	// 旧版本先完成且判定 allow：不得公开改稿后的内容。
	stub.release(first)
	waitFor(t, "first decision", func() bool {
		var count int64
		conn.Model(&moderationDecision.Entity{}).Where("subject_type = ? AND subject_id = ?", moderationDecision.SubjectTopic, topicID).Count(&count)
		return count == 1
	})
	time.Sleep(100 * time.Millisecond)
	if status := topics.Get(topicID).ProcessStatus; status != topics.ProcessStatusPending {
		t.Fatalf("superseded allow changed the topic status to %d", status)
	}
	stub.release(second)
	waitFor(t, "rejection of the edited version", func() bool { return topics.Get(topicID).ProcessStatus == topics.ProcessStatusBlocked })
}

// getTopicPage 以 SPA 页面请求（X-Goose-Page）读取话题详情 JSON。
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
	body, _ := json.Marshal(map[string]any{"topicId": publicTopicID, "content": "作者自己审核中的这条回复"})
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

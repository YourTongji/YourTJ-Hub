package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/tokenservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// contractApprovalIssuedAt 固定签发时刻（远未来），使 fixture 中的 expiresAt 确定且链接有效。
var contractApprovalIssuedAt = time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)

const (
	approvalPreviewPath = "/api/forum/moderation/approval-action/preview"
	approvalExecutePath = "/api/forum/moderation/approval-action/execute"
)

// setupModerationApprovalActionContractTest 注册快捷审批确认页的两条路由，中间件链与
// route4api.go 生产注册一致（权限判定在控制器内按当前会话复核）。
func setupModerationApprovalActionContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupForumModerationContractTest(t)
	registerModerationApprovalActionRoutes(t, router)
	return conn, router
}

func registerModerationApprovalActionRoutes(t *testing.T, router *gin.Engine) {
	t.Helper()
	old := preferences.GetString("app.signingKey", "")
	preferences.Set("app.signingKey", "approval-action-contract-key-0123456789")
	t.Cleanup(func() { preferences.Set("app.signingKey", old) })
	loginAPI := router.Group("/api/forum").Use(middleware.JWTAuthCheck)
	loginAPI.POST("/moderation/approval-action/preview", middleware.NoUpdateUserActivity, UpButterReq(api.ModerationApprovalActionPreview))
	loginAPI.POST("/moderation/approval-action/execute", middleware.CheckWritableAccount, UpButterReq(api.ModerationApprovalActionExecute))
}

func contractApprovalToken(t *testing.T, subject string, id uint64, action string, version string, issuedAt time.Time) string {
	t.Helper()
	token, err := tokenservice.IssueModerationAction(tokenservice.ModerationActionClaims{
		Subject: subject, ID: id, Action: action, Version: version,
	}, issuedAt)
	if err != nil {
		t.Fatalf("issue approval token: %v", err)
	}
	return token
}

func approvalBody(token string) string {
	return `{"token":"` + token + `"}`
}

func serveApprovalFixture(t *testing.T, router *gin.Engine, path string, body string, user *users.EntityComplete, fixture string) {
	t.Helper()
	recorder := serveJSON(router, path, body, contractSessionToken(t, user))
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s status = %d, want 200: %s", path, recorder.Code, recorder.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixture))
}

func approvalState(t *testing.T, router *gin.Engine, path string, body string, user *users.EntityComplete) map[string]any {
	t.Helper()
	recorder := serveJSON(router, path, body, contractSessionToken(t, user))
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s status = %d: %s", path, recorder.Code, recorder.Body.String())
	}
	envelope := decodeContractEnvelope(t, recorder)
	var result map[string]any
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		t.Fatalf("result = %s: %v", envelope.Result, err)
	}
	return result
}

func TestModerationApprovalActionReportHTTPContract(t *testing.T) {
	t.Run("preview is read-only and execute dismisses once with the real handler", func(t *testing.T) {
		conn, router := setupModerationApprovalActionContractTest(t)
		prepareContractModerationTopic(t, conn)
		createContractModerationReport(t, conn)
		moderator := createContractCategoryModerator(t, conn)
		body := approvalBody(contractApprovalToken(t, tokenservice.ModerationActionSubjectReport, contractModerationReportID, "dismiss", "", contractApprovalIssuedAt))

		serveApprovalFixture(t, router, approvalPreviewPath, body, moderator, "moderation-approval-action-preview-ready.json")
		if got := reports.Get(contractModerationReportID); got.Status != reports.StatusOpen {
			t.Fatalf("preview mutated the report: %+v", got)
		}

		serveApprovalFixture(t, router, approvalExecutePath, body, moderator, "moderation-approval-action-execute-done.json")
		got := reports.Get(contractModerationReportID)
		if got.Status != reports.StatusRejected || got.Resolution != reports.ResolutionIgnored || got.HandlerId != moderator.Id {
			t.Fatalf("report after dismiss = %+v, want rejected by %d", got, moderator.Id)
		}
		var logs int64
		conn.Model(&moderationLog.Entity{}).Where("subject_type = ? AND subject_id = ? AND actor_user_id = ?",
			moderationLog.SubjectReport, contractModerationReportID, moderator.Id).Count(&logs)
		if logs != 1 {
			t.Fatalf("moderation logs for real actor = %d, want 1", logs)
		}

		// 重复点击：返回已处理，不覆盖首个处理人。
		serveApprovalFixture(t, router, approvalExecutePath, body, moderator, "moderation-approval-action-execute-processed.json")
	})

	t.Run("ban blocks the target and resolves the report", func(t *testing.T) {
		conn, router := setupModerationApprovalActionContractTest(t)
		prepareContractModerationTopic(t, conn)
		createContractModerationReport(t, conn)
		moderator := createContractCategoryModerator(t, conn)
		body := approvalBody(contractApprovalToken(t, tokenservice.ModerationActionSubjectReport, contractModerationReportID, "ban", "", contractApprovalIssuedAt))
		if state := approvalState(t, router, approvalExecutePath, body, moderator)["state"]; state != "done" {
			t.Fatalf("ban state = %v", state)
		}
		if topic := topics.Get(contractModerationTopicID); topic.ProcessStatus != topics.ProcessStatusBlocked {
			t.Fatalf("topic process status = %d, want blocked", topic.ProcessStatus)
		}
		got := reports.Get(contractModerationReportID)
		if got.Status != reports.StatusResolved || got.Resolution != reports.ResolutionBanned || got.HandlerId != moderator.Id {
			t.Fatalf("report after ban = %+v", got)
		}
	})

	t.Run("a valid link does not grant authority", func(t *testing.T) {
		conn, router := setupModerationApprovalActionContractTest(t)
		prepareContractModerationTopic(t, conn)
		createContractModerationReport(t, conn)
		outsider := createHTTPContractUser(t, conn, contractTestID())
		body := approvalBody(contractApprovalToken(t, tokenservice.ModerationActionSubjectReport, contractModerationReportID, "dismiss", "", contractApprovalIssuedAt))
		serveApprovalFixture(t, router, approvalPreviewPath, body, outsider, "moderation-approval-action-forbidden.json")
		serveApprovalFixture(t, router, approvalExecutePath, body, outsider, "moderation-approval-action-forbidden.json")
		if got := reports.Get(contractModerationReportID); got.Status != reports.StatusOpen {
			t.Fatalf("forbidden execute mutated the report: %+v", got)
		}
	})

	t.Run("tampered, foreign and expired links are rejected", func(t *testing.T) {
		conn, router := setupModerationApprovalActionContractTest(t)
		prepareContractModerationTopic(t, conn)
		createContractModerationReport(t, conn)
		moderator := createContractCategoryModerator(t, conn)
		serveApprovalFixture(t, router, approvalExecutePath, approvalBody("v1.forged.token"), moderator, "moderation-approval-action-invalid.json")
		// 私信举报不提供快捷动作；即便签名有效，动作与目标组合不合法也拒绝。
		chatToken := contractApprovalToken(t, tokenservice.ModerationActionSubjectReport, contractModerationReportID, "hide", "", contractApprovalIssuedAt)
		if state := approvalState(t, router, approvalExecutePath, approvalBody(chatToken), moderator)["state"]; state != "invalid" {
			t.Fatalf("hide on a topic report state = %v, want invalid", state)
		}
		expired := contractApprovalToken(t, tokenservice.ModerationActionSubjectReport, contractModerationReportID, "dismiss", "", time.Now().Add(-48*time.Hour))
		result := approvalState(t, router, approvalExecutePath, approvalBody(expired), moderator)
		if result["state"] != "expired" || result["title"] != nil {
			t.Fatalf("expired result = %v", result)
		}
		if got := reports.Get(contractModerationReportID); got.Status != reports.StatusOpen {
			t.Fatalf("rejected links mutated the report: %+v", got)
		}
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupModerationApprovalActionContractTest(t)
		assertInteractionUnauthenticated(t, router, approvalExecutePath, `{}`, "auth-required.json")
	})

	t.Run("frozen account cannot execute", func(t *testing.T) {
		conn, router := setupModerationApprovalActionContractTest(t)
		assertInteractionForbidden(t, conn, router, approvalExecutePath, `{}`, "account-frozen.json")
	})

	t.Run("missing token is a validation failure", func(t *testing.T) {
		conn, router := setupModerationApprovalActionContractTest(t)
		moderator := createContractCategoryModerator(t, conn)
		serveApprovalFixture(t, router, approvalPreviewPath, `{}`, moderator, "invalid-params.json")
	})
}

var (
	reviewRequestedCaptureOnce sync.Once
	reviewRequestedCaptured    = make(chan uint64, 16)
)

// captureReviewRequested 启动事件总线并订阅转人工事件；用哨兵事件回环确认已订阅
// （发布早于订阅的事件会被丢弃）。
func captureReviewRequested(t *testing.T) <-chan uint64 {
	t.Helper()
	reviewRequestedCaptureOnce.Do(func() {
		eventbus.Start(cqrs.NewEventHandler("ReviewRequestedContractCapture", func(_ context.Context, event *eventhandlers.ModerationReviewRequestedEvent) error {
			reviewRequestedCaptured <- event.RevisionID
			return nil
		}))
	})
	const sentinel = uint64(0xDEADBEEF)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		eventbus.Publish(context.Background(), &eventhandlers.ModerationReviewRequestedEvent{RevisionID: sentinel})
		select {
		case id := <-reviewRequestedCaptured:
			if id != sentinel {
				t.Fatalf("unexpected event during readiness probe: %d", id)
			}
			return reviewRequestedCaptured
		case <-time.After(100 * time.Millisecond):
		}
	}
	t.Fatal("event bus never delivered the readiness probe")
	return nil
}

// runNewEffectTasks 执行尚未执行过的内容效果任务。
func runNewEffectTasks(t *testing.T, conn *gorm.DB, done map[uint64]bool) {
	t.Helper()
	var tasks []taskQueue.Entity
	if err := conn.Where("type = ?", publicationservice.EffectTaskType).Find(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		if done[task.Id] {
			continue
		}
		done[task.Id] = true
		if err := publicationservice.RunEffectsTask(context.Background(), &task); err != nil {
			t.Fatal(err)
		}
	}
}

// 转人工后经内容效果任务发布一次审批事件；确认页按送审版本 ID 执行，改稿后旧链接失效。
func TestModerationApprovalActionReviewHTTPContract(t *testing.T) {
	received := captureReviewRequested(t)
	conn, router, _ := setupAIModerationContractTest(t, func(o *pageConfig.AiModerationOptions) { o.TextModeration = true })
	registerModerationApprovalActionRoutes(t, router)
	author := createHTTPContractUser(t, conn, contractTestID())
	body := `{"title":"快捷审批确认页测试","content":"等待版主确认的完整内容","categoryId":[1],"topicStatus":1,"contentType":2}`
	envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", body, contractSessionToken(t, author)))
	var topicID uint64
	if err := json.Unmarshal(envelope.Result, &topicID); err != nil || topicID == 0 {
		t.Fatalf("submission: %+v", envelope)
	}
	post := posts.Get(topics.Get(topicID).FirstPostId)
	moderator := createHTTPContractUser(t, conn, contractTestID())
	if err := conn.Create(&moderators.Entity{UserId: moderator.Id, ScopeType: moderators.ScopeCategory, ScopeId: 1, Status: moderators.StatusEnabled}).Error; err != nil {
		t.Fatal(err)
	}
	moderationservice.Invalidate()
	t.Cleanup(func() {
		conn.Where("user_id = ?", moderator.Id).Delete(&moderators.Entity{})
		moderationservice.Invalidate()
	})

	ran := map[uint64]bool{}
	for range 2 {
		if err := publicationservice.Review(context.Background(), post.LatestRevisionId, moderationDecision.ActionReview, "等待人工审核", 0); err != nil {
			t.Fatal(err)
		}
	}
	runNewEffectTasks(t, conn, ran)
	select {
	case id := <-received:
		if id != post.LatestRevisionId {
			t.Fatalf("review requested revision = %d, want %d", id, post.LatestRevisionId)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("handoff did not publish ModerationReviewRequestedEvent")
	}
	select {
	case id := <-received:
		t.Fatalf("repeated handoff published again: %d", id)
	case <-time.After(200 * time.Millisecond):
	}

	token := func(revisionID uint64) string {
		return approvalBody(contractApprovalToken(t, tokenservice.ModerationActionSubjectReviewTopic, topicID, "approve", strconv.FormatUint(revisionID, 10), contractApprovalIssuedAt))
	}
	stale := token(post.LatestRevisionId)
	if state := approvalState(t, router, approvalPreviewPath, stale, moderator)["state"]; state != "ready" {
		t.Fatalf("pending topic preview state = %v", state)
	}
	if state := approvalState(t, router, approvalPreviewPath, stale, author)["state"]; state != "forbidden" {
		t.Fatalf("author preview state = %v, want forbidden", state)
	}

	// 作者改稿形成新版本后，旧卡片返回 changed，且不执行。
	next := postRevisions.Entity{PostId: post.Id, Title: "快捷审批确认页测试（修改）", Content: "改过的完整内容", CategoryIds: []uint64{1}, ProcessStatus: posts.ProcessStatusPending, EditorId: author.Id}
	if err := conn.Create(&next).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&posts.Entity{}).Where("id = ?", post.Id).Update("latest_revision_id", next.Id).Error; err != nil {
		t.Fatal(err)
	}
	if state := approvalState(t, router, approvalExecutePath, stale, moderator)["state"]; state != "changed" {
		t.Fatalf("stale approval state = %v, want changed", state)
	}
	if postRevisions.Get(next.Id).ProcessStatus != posts.ProcessStatusPending || topics.Get(topicID).ProcessStatus != topics.ProcessStatusPending {
		t.Fatal("stale approval changed the pending content")
	}

	fresh := token(next.Id)
	if state := approvalState(t, router, approvalExecutePath, fresh, moderator)["state"]; state != "done" {
		t.Fatalf("fresh approval state = %v, want done", state)
	}
	if got := posts.Get(post.Id); got.ProcessStatus != posts.ProcessStatusNormal || got.PublishedRevisionId != next.Id {
		t.Fatalf("approved post = %+v", got)
	}
	if state := approvalState(t, router, approvalExecutePath, fresh, moderator)["state"]; state != "processed" {
		t.Fatalf("repeated approval state = %v, want processed", state)
	}
	// 审结后的效果任务（含旧的转人工任务）不再发布审批事件。
	runNewEffectTasks(t, conn, ran)
	select {
	case id := <-received:
		t.Fatalf("approved content published a review request: %d", id)
	case <-time.After(200 * time.Millisecond):
	}
}

// 快捷审批确认页的查询串携带签名 token，不得进入请求日志（含未登录续跳地址）。
func TestModerationActionQueryIsRedactedFromLogs(t *testing.T) {
	page, _ := url.Parse("/moderation/action?token=v1.secret")
	if !middleware.ShouldRedactQuery(page) {
		t.Fatal("moderation action token query must be redacted")
	}
	login, _ := url.Parse("/login?redirect=" + url.QueryEscape("/moderation/action?token=v1.secret"))
	if !middleware.ShouldRedactQuery(login) {
		t.Fatal("login continuation to the moderation action page must be redacted")
	}
}

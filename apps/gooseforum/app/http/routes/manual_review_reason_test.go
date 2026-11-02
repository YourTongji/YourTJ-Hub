package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationAppeals"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/tokenservice"
)

func TestManualReviewReasonRequiredBeforeDecision(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, nil)
	author := createHTTPContractUser(t, conn, contractTestID())
	manager := createContractSiteManager(t, conn)
	for _, reason := range []string{"", " \n\u2003 "} {
		t.Run(fmt.Sprintf("blank_%d", len(reason)), func(t *testing.T) {
			topic := topics.Entity{UserId: author.Id, Title: "Pending reason", Status: 1, ProcessStatus: topics.ProcessStatusPending}
			post := posts.Entity{UserId: author.Id, PostNo: 1, Content: "Private candidate", ProcessStatus: posts.ProcessStatusPending}
			if err := publicationservice.Submit(t.Context(), &topic, &post); err != nil {
				t.Fatal(err)
			}
			body, err := json.Marshal(map[string]any{"kind": "topic", "id": topic.Id, "revisionId": post.LatestRevisionId, "approve": false, "reason": reason})
			if err != nil {
				t.Fatal(err)
			}
			result := decodeContractEnvelope(t, serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/review-action", string(body), contractSessionToken(t, manager)))
			if result.MessageCode != "common.request.invalidParams" {
				t.Fatalf("blank rejection applied: code=%d message=%s", result.Code, result.MessageCode)
			}
			revision := postRevisions.Get(post.LatestRevisionId)
			if revision.ProcessStatus != posts.ProcessStatusPending || revision.ReviewedAt != nil {
				t.Fatalf("blank reason changed decision: %+v", revision)
			}
			var notices int64
			if err := conn.Model(&eventNotification.Entity{}).Where("user_id = ? AND event_type = ?", author.Id, eventNotification.EventTypeReviewRejected).Count(&notices).Error; err != nil {
				t.Fatal(err)
			}
			if notices != 0 {
				t.Fatalf("blank reason emitted %d rejection notices", notices)
			}
		})
	}
}

func TestLegacyReviewReasonIsRetainedForAuthorOnly(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, nil)
	author := createHTTPContractUser(t, conn, contractTestID())
	topic := seedContractReviewTopic(t, conn, topics.Entity{Id: contractTestID(), UserId: author.Id, Title: "Legacy pending", Status: 1, ProcessStatus: topics.ProcessStatusPending, TopicType: topics.TopicTypeForum})
	post := seedContractReviewPost(t, conn, posts.Entity{Id: contractTestID(), TopicId: topic.Id, UserId: author.Id, PostNo: 1, Content: "Original private candidate", ProcessStatus: posts.ProcessStatusPending, VisibilityStatus: posts.VisibilityActive})
	if err := conn.Model(&topic).Update("first_post_id", post.Id).Error; err != nil {
		t.Fatal(err)
	}
	reason := "Remove another member's private details. <script>author-visible plain text</script>"
	body, err := json.Marshal(map[string]any{"kind": "topic", "id": topic.Id, "approve": false, "reason": "  " + reason + "  "})
	if err != nil {
		t.Fatal(err)
	}
	result := decodeContractEnvelope(t, serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/review-action", string(body)))
	if result.Code != 0 {
		t.Fatalf("reject failed: %+v", result)
	}
	post = posts.Get(post.Id)
	topic = topics.Get(topic.Id)
	if post.LatestRevisionId == 0 {
		t.Fatal("legacy rejection discarded its author-visible reason")
	}
	revision := postRevisions.Get(post.LatestRevisionId)
	if revision.ReviewReason != reason || revision.ReviewActorId == 0 || revision.ReviewedAt == nil {
		t.Fatalf("reason/decision audit lost: %+v", revision)
	}
	ownerTopic, ownerPost := topic, post
	if got := publicationservice.OwnerSnapshot(&ownerTopic, &ownerPost, author.Id, true); got.ReviewReason != reason {
		t.Fatalf("owner lost reason: %+v", got)
	}
	otherTopic, otherPost := topic, post
	if got := publicationservice.OwnerSnapshot(&otherTopic, &otherPost, author.Id+1, true); got.Id != 0 {
		t.Fatal("reason exposed to nonowner")
	}
	if err := conn.AutoMigrate(&moderationAppeals.Entity{}); err != nil {
		t.Fatal(err)
	}
	appeal, err := publicationservice.SubmitAppeal(t.Context(), author.Id, "topic", topic.Id, revision.Id, "Please reconsider after checking the context.")
	if err != nil {
		t.Fatal(err)
	}
	var audit moderationAppeals.Entity
	if err := conn.First(&audit, appeal.Id).Error; err != nil {
		t.Fatal(err)
	}
	if appeal.OriginalReviewReason != reason || audit.OriginalReviewActorId != revision.ReviewActorId {
		t.Fatal("appeal lost the original decision explanation")
	}
	var notices []eventNotification.Entity
	if err := conn.Where("user_id = ? AND event_type = ?", author.Id, eventNotification.EventTypeReviewRejected).Find(&notices).Error; err != nil {
		t.Fatal(err)
	}
	if len(notices) != 1 {
		t.Fatalf("rejection notices = %d", len(notices))
	}
	raw, err := json.Marshal(notices)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), reason) || strings.Contains(string(raw), "author-visible plain text") {
		t.Fatal("private rejection reason leaked into a notification")
	}
}

func TestReviewQuickActionRequiresAndRetainsReason(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, nil)
	registerModerationApprovalActionRoutes(t, router)
	author := createHTTPContractUser(t, conn, contractTestID())
	manager := createHTTPContractUser(t, conn, contractTestID())
	grant := moderators.Entity{UserId: manager.Id, ScopeType: moderators.ScopeCategory, ScopeId: 1, Status: moderators.StatusEnabled}
	if err := conn.Create(&grant).Error; err != nil {
		t.Fatal(err)
	}
	moderationservice.Invalidate()
	t.Cleanup(func() { conn.Delete(&grant); moderationservice.Invalidate() })
	topic := topics.Entity{UserId: author.Id, Title: "Quick review", Status: 1, ProcessStatus: topics.ProcessStatusPending, CategoryIds: []uint64{1}}
	post := posts.Entity{UserId: author.Id, PostNo: 1, Content: "Private quick candidate", ProcessStatus: posts.ProcessStatusPending}
	if err := publicationservice.Submit(t.Context(), &topic, &post); err != nil {
		t.Fatal(err)
	}
	token := contractApprovalToken(t, tokenservice.ModerationActionSubjectReviewTopic, topic.Id, "reject", strconv.FormatUint(post.LatestRevisionId, 10), time.Now())
	for _, reason := range []string{"", " \n\u2003 ", strings.Repeat("界", 513)} {
		body, err := json.Marshal(map[string]string{"token": token, "reason": reason})
		if err != nil {
			t.Fatal(err)
		}
		result := decodeContractEnvelope(t, serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/approval-action/execute", string(body), contractSessionToken(t, manager)))
		if result.MessageCode != "common.request.invalidParams" {
			t.Fatalf("quick rejection accepted invalid reason: %+v", result)
		}
		if got := postRevisions.Get(post.LatestRevisionId); got.ProcessStatus != posts.ProcessStatusPending {
			t.Fatal("invalid quick reason committed")
		}
	}
	reason := strings.Repeat("😀", 512)
	body, err := json.Marshal(map[string]string{"token": token, "reason": "  " + reason + "  "})
	if err != nil {
		t.Fatal(err)
	}
	result := approvalState(t, router, "/api/forum/moderation/approval-action/execute", string(body), manager)
	if result["state"] != "done" {
		t.Fatalf("quick rejection failed: %+v", result)
	}
	if got := postRevisions.Get(post.LatestRevisionId); got.ReviewReason != reason || got.ReviewActorId != manager.Id {
		t.Fatal("quick rejection discarded explanation or actor")
	}
}

func TestLegacyReviewReasonRollbackIncludesSnapshot(t *testing.T) {
	conn, router, _ := setupAIModerationContractTest(t, nil)
	author := createHTTPContractUser(t, conn, contractTestID())
	topic := seedContractReviewTopic(t, conn, topics.Entity{Id: contractTestID(), UserId: author.Id, Title: "Legacy parent", Status: 1, ProcessStatus: topics.ProcessStatusNormal, TopicType: topics.TopicTypeForum})
	post := seedContractReviewPost(t, conn, posts.Entity{Id: contractTestID(), TopicId: topic.Id, UserId: author.Id, PostNo: 2, Content: "Private legacy reply", ProcessStatus: posts.ProcessStatusPending, VisibilityStatus: posts.VisibilityActive})
	trigger := fmt.Sprintf("reason_rollback_%d", post.Id)
	sql := fmt.Sprintf("CREATE TRIGGER %s BEFORE UPDATE OF process_status ON posts WHEN OLD.id = %d BEGIN SELECT RAISE(ABORT, 'synthetic status failure'); END", trigger, post.Id)
	if err := conn.Exec(sql).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Exec("DROP TRIGGER IF EXISTS " + trigger) })
	body, err := json.Marshal(map[string]any{"kind": "post", "id": post.Id, "approve": false, "reason": "Please remove the private address."})
	if err != nil {
		t.Fatal(err)
	}
	result := decodeContractEnvelope(t, serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/review-action", string(body)))
	if result.Code == 0 {
		t.Fatal("synthetic status failure succeeded")
	}
	if got := posts.Get(post.Id); got.LatestRevisionId != 0 || got.ProcessStatus != posts.ProcessStatusPending {
		t.Fatal("failed rejection left a committed snapshot or status")
	}
	var count int64
	if err := conn.Model(&postRevisions.Entity{}).Where("post_id = ?", post.Id).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed rejection left a decision record")
	}
	if err := conn.Model(&eventNotification.Entity{}).Where("user_id = ?", author.Id).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed rejection emitted a notification")
	}
}

package eventhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/httpnotifyservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/tokenservice"
)

type approvalEnvelope struct {
	Event string                            `json:"event"`
	Data  httpnotifyservice.ApprovalPayload `json:"data"`
}

func withApprovalSigningKey(t *testing.T) {
	t.Helper()
	old := preferences.GetString("app.signingKey", "")
	preferences.Set("app.signingKey", "approval-notify-test-key-0123456789")
	t.Cleanup(func() { preferences.Set("app.signingKey", old) })
}

func decodeApproval(t *testing.T, body []byte) approvalEnvelope {
	t.Helper()
	var envelope approvalEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return envelope
}

func expectNoMoreHttpNotify(t *testing.T, received <-chan []byte) {
	t.Helper()
	select {
	case body := <-received:
		t.Fatalf("unexpected extra delivery: %s", body)
	case <-time.After(200 * time.Millisecond):
	}
}

// 同时订阅“全部举报”与具体举报类型的 Endpoint 只收到一份具体审批事件。
func TestReportApprovalDeliversSpecificEventOncePerEndpoint(t *testing.T) {
	withApprovalSigningKey(t)
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	topic := topics.Entity{UserId: 9999, Status: 1, Title: "report dedupe", Excerpt: "topic excerpt"}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	received := captureHttpNotify(t, httpnotifyservice.EventReportCreated, httpnotifyservice.EventReportTopicCreated)
	if err := handleHttpNotifyReportCreated(context.Background(), &ReportCreatedEvent{
		ReportId: 101, TargetType: "topic", TargetId: topic.Id, TopicId: topic.Id, ReporterId: 9998, Reason: "spam", Note: "看这里",
	}); err != nil {
		t.Fatal(err)
	}
	envelope := decodeApproval(t, waitHttpNotify(t, received))
	expectNoMoreHttpNotify(t, received)
	a := envelope.Data.Approval
	if envelope.Event != httpnotifyservice.EventReportTopicCreated || a.ID != "report:101" || a.Kind != "report" || a.Note != "看这里" {
		t.Fatalf("approval = %+v (event %s)", a, envelope.Event)
	}
	if len(a.Actions) != 2 || a.Actions[0].Action != "ban" || a.Actions[1].Action != "dismiss" {
		t.Fatalf("actions = %+v", a.Actions)
	}
	link, err := url.Parse(a.Actions[0].URL)
	if err != nil || link.Path != "/moderation/action" {
		t.Fatalf("action url = %q", a.Actions[0].URL)
	}
	claims, err := tokenservice.ParseModerationAction(link.Query().Get("token"), time.Now())
	if err != nil || claims.Subject != tokenservice.ModerationActionSubjectReport || claims.ID != 101 || claims.Action != "ban" {
		t.Fatalf("token claims = %+v err=%v", claims, err)
	}
}

// 私信举报只给最少元数据：不含正文、举报说明、当事人身份，也没有快捷动作。
func TestChatReportApprovalIsMinimal(t *testing.T) {
	withApprovalSigningKey(t)
	received := captureHttpNotify(t, httpnotifyservice.EventReportChatMessageCreated)
	if err := handleHttpNotifyReportCreated(context.Background(), &ReportCreatedEvent{
		ReportId: 202, TargetType: "chat_message", TargetId: 77, ReporterId: 9998, Reason: "abuse", Note: "私信里说了很过分的话",
	}); err != nil {
		t.Fatal(err)
	}
	body := waitHttpNotify(t, received)
	if strings.Contains(string(body), "过分") || strings.Contains(string(body), "9998") {
		t.Fatalf("chat report leaked note or reporter: %s", body)
	}
	a := decodeApproval(t, body).Data.Approval
	if a.Author != nil || a.Excerpt != "" || len(a.Actions) != 0 || a.ModerationURL != "/moderation?tab=reports" {
		t.Fatalf("chat approval = %+v", a)
	}
}

// 匿名课评被举报：通知只标记匿名，不含作者身份。
func TestCourseReviewReportApprovalHidesAnonymousAuthor(t *testing.T) {
	withApprovalSigningKey(t)
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &course.Entity{}, &course.OfferingEntity{}, &course.ReviewEntity{}); err != nil {
		t.Fatal(err)
	}
	author := users.MakeUser("anon_course_author", "pass1234", "anon_course_author@example.com")
	if err := users.Create(author); err != nil {
		t.Fatal(err)
	}
	c := course.Entity{Name: "高等数学", PrimaryCode: "MATH-1049"}
	if err := conn.Create(&c).Error; err != nil {
		t.Fatal(err)
	}
	offering := course.OfferingEntity{CourseId: c.Id, ClassCode: "MATH-1049-01"}
	if err := conn.Create(&offering).Error; err != nil {
		t.Fatal(err)
	}
	authorID := author.Id
	review := course.ReviewEntity{OfferingId: offering.Id, AuthorUserId: &authorID, Content: "老师讲得很清楚", IsAnonymous: true}
	if err := conn.Create(&review).Error; err != nil {
		t.Fatal(err)
	}
	received := captureHttpNotify(t, httpnotifyservice.EventReportCourseReviewCreated)
	if err := handleHttpNotifyReportCreated(context.Background(), &ReportCreatedEvent{
		ReportId: 303, TargetType: "course_review", TargetId: review.Id, ReporterId: 9998, Reason: "other",
	}); err != nil {
		t.Fatal(err)
	}
	body := waitHttpNotify(t, received)
	if strings.Contains(string(body), "anon_course_author") {
		t.Fatalf("anonymous course review author leaked: %s", body)
	}
	a := decodeApproval(t, body).Data.Approval
	if !a.Anonymous || a.Author != nil || a.Title != "高等数学" || a.ModerationURL != "/moderation/course-reviews" {
		t.Fatalf("course review approval = %+v", a)
	}
	if len(a.Actions) != 2 || a.Actions[0].Action != "hide" {
		t.Fatalf("actions = %+v", a.Actions)
	}
}

// 人工审核通知绑定送审版本：只在该版本仍是最新、待审且已转人工时发送；同一版本重复
// 发布只投递一次；改稿形成新版本与新审批。
func TestReviewRequestedApprovalTracksPendingRevision(t *testing.T) {
	withApprovalSigningKey(t)
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &topics.Entity{}, &posts.Entity{}, &postRevisions.Entity{}, &moderationDecision.Entity{}); err != nil {
		t.Fatal(err)
	}
	topic := topics.Entity{UserId: 9999, Status: 1, Title: "visible topic"}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	post := posts.Entity{TopicId: topic.Id, PostNo: 2, UserId: 9999, Content: "published reply", ProcessStatus: posts.ProcessStatusPending, IsAnonymous: true}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	reviewedAt := time.Now()
	submit := func(content string, reviewed *time.Time) postRevisions.Entity {
		t.Helper()
		revision := postRevisions.Entity{PostId: post.Id, Content: content, ProcessStatus: posts.ProcessStatusPending, ReviewedAt: reviewed}
		if err := conn.Create(&revision).Error; err != nil {
			t.Fatal(err)
		}
		if err := conn.Model(&posts.Entity{}).Where("id = ?", post.Id).Update("latest_revision_id", revision.Id).Error; err != nil {
			t.Fatal(err)
		}
		return revision
	}
	received := captureHttpNotify(t, httpnotifyservice.EventReviewPostRequested)
	notify := func(revisionID uint64) {
		t.Helper()
		if err := handleHttpNotifyReviewRequested(context.Background(), &ModerationReviewRequestedEvent{RevisionID: revisionID}); err != nil {
			t.Fatal(err)
		}
	}

	// 自动检查中的版本（尚未转人工）不通知。
	checking := submit("still checking", nil)
	notify(checking.Id)
	expectNoMoreHttpNotify(t, received)

	first := submit("pending reply", &reviewedAt)
	notify(first.Id)
	notify(first.Id)
	a := decodeApproval(t, waitHttpNotify(t, received)).Data.Approval
	expectNoMoreHttpNotify(t, received)
	if a.Version != strconv.FormatUint(first.Id, 10) || a.ID != fmt.Sprintf("review:post:%d:r%d", post.Id, first.Id) {
		t.Fatalf("review approval identity = %+v", a)
	}
	if !a.Anonymous || a.Author != nil || a.Reason != ReviewTriggerSensitiveWord || a.Edited || a.Excerpt != "pending reply" {
		t.Fatalf("review approval = %+v", a)
	}
	if len(a.Actions) != 2 || a.Actions[0].Action != "approve" || a.Actions[1].Action != "reject" {
		t.Fatalf("actions = %+v", a.Actions)
	}

	// 已有公开版本时改稿送审：摘要取送审版本，标记为修改，AI 判定转人工的原因是 ai。
	if err := conn.Model(&posts.Entity{}).Where("id = ?", post.Id).Update("published_revision_id", first.Id).Error; err != nil {
		t.Fatal(err)
	}
	second := submit("edited pending reply", &reviewedAt)
	if err := conn.Create(&moderationDecision.Entity{RevisionId: second.Id, SubjectType: moderationDecision.SubjectPost, FinalAction: moderationDecision.ActionReview}).Error; err != nil {
		t.Fatal(err)
	}
	notify(first.Id) // 旧版本已不是最新
	notify(second.Id)
	b := decodeApproval(t, waitHttpNotify(t, received)).Data.Approval
	expectNoMoreHttpNotify(t, received)
	if b.ID == a.ID || !b.Edited || b.Reason != ReviewTriggerAI || b.Excerpt != "edited pending reply" {
		t.Fatalf("edited revision approval = %+v", b)
	}

	// 版本审结后补发的事件不再通知。
	if err := conn.Model(&postRevisions.Entity{}).Where("id = ?", second.Id).Update("process_status", posts.ProcessStatusNormal).Error; err != nil {
		t.Fatal(err)
	}
	notify(second.Id)
	expectNoMoreHttpNotify(t, received)
}

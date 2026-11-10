package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/migration"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationAppeals"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
)

func TestModerationAppealsHTTPContract(t *testing.T) {
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(migration.SchemaModels()...); err != nil {
		t.Fatal(err)
	}
	group := router.Group("/api/forum", middleware.CSRFProtection, middleware.JWTAuthCheck)
	group.POST("/content/appeals", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpLimitedJsonReq(8192, api.SubmitContentAppeal))
	group.GET("/content/appeals", middleware.NoUpdateUserActivity, UpQueryReq(api.ContentAppeals))
	group.POST("/moderation/appeals", middleware.NoUpdateUserActivity, UpLimitedJsonReq(8192, api.ModerationAppeals))
	group.POST("/moderation/appeal-action", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpLimitedJsonReq(8192, api.ModerationAppealAction))
	owner := createHTTPContractUser(t, conn, contractTestID())
	moderator := createHTTPContractUser(t, conn, contractTestID())
	stranger := createHTTPContractUser(t, conn, contractTestID())
	grant := moderators.Entity{UserId: moderator.Id, ScopeType: moderators.ScopeGlobal, Status: 1}
	if err := conn.Create(&grant).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Delete(&grant) })
	topicID, postID := contractTestID(), contractTestID()
	createContractPublishedTopic(t, conn, topicID, postID, owner.Id)
	conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("process_status", topics.ProcessStatusBlocked)
	conn.Model(&posts.Entity{}).Where("id = ?", postID).Update("process_status", posts.ProcessStatusBlocked)
	rev := postRevisions.Entity{PostId: postID, Version: 1, Title: "Rejected snapshot", Content: "Rejected immutable content", ProcessStatus: posts.ProcessStatusBlocked}
	if err := conn.Create(&rev).Error; err != nil {
		t.Fatal(err)
	}
	conn.Model(&posts.Entity{}).Where("id = ?", postID).Update("latest_revision_id", rev.Id)
	body := fmt.Sprintf(`{"kind":"topic","id":%d,"revisionId":%d,"reason":"Please review"}`, topicID, rev.Id)
	ownerToken, modToken, strangerToken := contractSessionToken(t, owner), contractSessionToken(t, moderator), contractSessionToken(t, stranger)
	reply := posts.Entity{Id: contractTestID(), TopicId: topicID, UserId: owner.Id, PostNo: 2, Content: "Private appeal target", ProcessStatus: posts.ProcessStatusBlocked}
	if err := conn.Create(&reply).Error; err != nil {
		t.Fatal(err)
	}
	replyRevision := postRevisions.Entity{PostId: reply.Id, Version: 1, Content: reply.Content, ProcessStatus: posts.ProcessStatusBlocked}
	if err := conn.Create(&replyRevision).Error; err != nil {
		t.Fatal(err)
	}
	for _, target := range []struct {
		kind           string
		id, revisionID uint64
	}{{"topic", topicID, rev.Id}, {"post", reply.Id, replyRevision.Id}} {
		missingID := contractTestID()
		requestBody := func(id, revisionID uint64) string {
			return fmt.Sprintf(`{"kind":%q,"id":%d,"revisionId":%d,"reason":"Please review"}`, target.kind, id, revisionID)
		}
		for _, scenario := range []struct {
			name, method, path, body, token string
		}{
			{"nonowner history", http.MethodGet, fmt.Sprintf("/api/forum/content/appeals?kind=%s&id=%d", target.kind, target.id), "", strangerToken},
			{"missing history", http.MethodGet, fmt.Sprintf("/api/forum/content/appeals?kind=%s&id=%d", target.kind, missingID), "", ownerToken},
			{"nonowner submit", http.MethodPost, "/api/forum/content/appeals", requestBody(target.id, target.revisionID), strangerToken},
			{"missing target", http.MethodPost, "/api/forum/content/appeals", requestBody(missingID, target.revisionID), ownerToken},
			{"missing revision", http.MethodPost, "/api/forum/content/appeals", requestBody(target.id, missingID), ownerToken},
		} {
			t.Run(target.kind+"/"+scenario.name, func(t *testing.T) {
				rec := serveAuthSecurityJSON(router, scenario.method, scenario.path, scenario.body, scenario.token)
				if rec.Code != http.StatusOK {
					t.Fatalf("status=%d, want 200: %s", rec.Code, rec.Body.String())
				}
				assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "content-appeal-not-found.json"))
			})
		}
	}
	if w := serveJSON(router, "/api/forum/content/appeals", body, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status %d", w.Code)
	}
	cross := httptest.NewRequest(http.MethodPost, "/api/forum/content/appeals", strings.NewReader(body))
	cross.Header.Set("Content-Type", "application/json")
	cross.Header.Set("Origin", "https://attacker.example")
	cross.AddCookie(&http.Cookie{Name: "access_token", Value: ownerToken})
	csrf := httptest.NewRecorder()
	router.ServeHTTP(csrf, cross)
	if csrf.Code != http.StatusForbidden || decodeContractEnvelope(t, csrf).MessageCode != "auth.csrf.rejected" {
		t.Fatalf("CSRF response %s", csrf.Body.String())
	}
	if e := decodeContractEnvelope(t, serveJSON(router, "/api/forum/content/appeals", body, strangerToken)); e.Code == 0 || e.MessageCode != "content.appeal.notFound" {
		t.Fatalf("nonowner submit: code=%d messageCode=%s", e.Code, e.MessageCode)
	}
	conn.Model(&users.EntityComplete{}).Where("id = ?", owner.Id).Update("is_frozen", users.StatusFrozen)
	userservice.InvalidateUserInfoCache(owner.Id)
	if e := decodeContractEnvelope(t, serveJSON(router, "/api/forum/content/appeals", body, ownerToken)); e.Code == 0 {
		t.Fatal("frozen author submitted")
	}
	conn.Model(&users.EntityComplete{}).Where("id = ?", owner.Id).Update("is_frozen", users.StatusNormal)
	userservice.InvalidateUserInfoCache(owner.Id)
	submitted := serveJSON(router, "/api/forum/content/appeals", body, ownerToken)
	if submitted.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("private response cacheable")
	}
	e := decodeContractEnvelope(t, submitted)
	if e.Code != 0 {
		t.Fatalf("submission %s", submitted.Body.String())
	}
	var item publicationservice.AppealItem
	if err := json.Unmarshal(e.Result, &item); err != nil {
		t.Fatal(err)
	}
	if item.Id == 0 || item.RevisionId != rev.Id || item.Content != rev.Content || item.Status != "pending" || item.Images == nil {
		t.Fatalf("item %+v", item)
	}
	for _, key := range []string{`"userId"`, `"authorId"`, `"reviewActorId"`, `"originalReviewActorId"`} {
		if strings.Contains(string(e.Result), key) {
			t.Fatalf("identity leak %s", key)
		}
	}
	if e := decodeContractEnvelope(t, serveJSON(router, "/api/forum/content/appeals", body, ownerToken)); e.Code == 0 || e.MessageCode != "content.appeal.duplicate" {
		t.Fatalf("duplicate appeal: code=%d messageCode=%s", e.Code, e.MessageCode)
	}
	historyPath := fmt.Sprintf("/api/forum/content/appeals?kind=topic&id=%d", topicID)
	history := serveAuthSecurityJSON(router, http.MethodGet, historyPath, "", ownerToken)
	if e := decodeContractEnvelope(t, history); e.Code != 0 || !strings.Contains(string(e.Result), "Rejected immutable content") {
		t.Fatalf("history %s", history.Body.String())
	}
	deniedHistory := serveAuthSecurityJSON(router, http.MethodGet, historyPath, "", strangerToken)
	if deniedHistory.Code != http.StatusOK {
		t.Fatalf("nonowner history status %d", deniedHistory.Code)
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, deniedHistory), contractFixture(t, "content-appeal-not-found.json"))
	deniedQueue := serveJSON(router, "/api/forum/moderation/appeals", `{}`, strangerToken)
	if deniedQueue.Code != http.StatusOK {
		t.Fatalf("nonmoderator queue status %d", deniedQueue.Code)
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, deniedQueue), contractFixture(t, "permission-denied.json"))
	queue := serveJSON(router, "/api/forum/moderation/appeals", `{"page":1,"pageSize":20,"status":"pending"}`, modToken)
	if e := decodeContractEnvelope(t, queue); e.Code != 0 || !strings.Contains(string(e.Result), "Rejected immutable content") {
		t.Fatalf("queue %s", queue.Body.String())
	}
	action := fmt.Sprintf(`{"id":%d,"approve":true}`, item.Id)
	deniedAction := serveJSON(router, "/api/forum/moderation/appeal-action", action, strangerToken)
	if deniedAction.Code != http.StatusOK {
		t.Fatalf("nonmoderator action status %d", deniedAction.Code)
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, deniedAction), contractFixture(t, "permission-denied.json"))
	decision := serveJSON(router, "/api/forum/moderation/appeal-action", action, modToken)
	if e := decodeContractEnvelope(t, decision); e.Code != 0 || string(e.Result) != "true" {
		t.Fatalf("decision %s", decision.Body.String())
	}
	if got := posts.Get(postID); got.PublishedRevisionId != rev.Id || got.Content != rev.Content {
		t.Fatal("exact rejected candidate not published")
	}
	if e := decodeContractEnvelope(t, serveJSON(router, "/api/forum/moderation/appeal-action", action, modToken)); e.Code == 0 || e.MessageCode != "moderation.appeal.processed" {
		t.Fatalf("decision replay: code=%d messageCode=%s", e.Code, e.MessageCode)
	}
	var audit moderationAppeals.Entity
	conn.First(&audit, item.Id)
	if audit.Status != moderationAppeals.Approved || audit.ReviewActorId != moderator.Id {
		t.Fatalf("audit %+v", audit)
	}
}

package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func TestAgentContentMaintenanceHTTPContract(t *testing.T) {
	conn, router := setupHTTPContractTest(t)
	t.Setenv("YOURTJ_AGENT_INSTANCE_ID", "")
	t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "")
	if err := conn.AutoMigrate(&agents.Entity{}, &optRecord.Entity{}); err != nil {
		t.Fatal(err)
	}
	group := router.Group("/api/admin/agents", middleware.CSRFProtection, middleware.JWTAuthCheck, middleware.CheckWritableAccount)
	group.POST("/content-source", UpButterReq(api.AgentContentSource))
	group.POST("/edit-topic", middleware.RateLimit(middleware.RateLimitTopicWrite), UpLimitedButterReq(maxContentWriteBodyBytes, api.AgentEditTopic))
	group.POST("/edit-post", middleware.RateLimit(middleware.RateLimitPostUpdate), UpLimitedButterReq(maxContentWriteBodyBytes, api.AgentEditPost))
	owner := createHTTPContractUser(t, conn, contractTestID())
	stranger := createHTTPContractUser(t, conn, contractTestID())
	bot := createHTTPContractUser(t, conn, 42)
	bot.ActorType = users.ActorTypeBot
	if err := conn.Model(bot).Update("actor_type", users.ActorTypeBot).Error; err != nil {
		t.Fatal(err)
	}
	for _, row := range []any{&agents.Entity{UserId: 42, CreatedBy: owner.Id, TokenPrefix: "contract-content", Enabled: 1}, &category.Entity{Id: 1, Name: "General"}} {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	createContractPublishedTopic(t, conn, 1001, 2001, 42)
	if err := conn.Model(&topics.Entity{}).Where("id = ?", 1001).Updates(map[string]any{"title": "Agent topic", "category_id": "[1]"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec("UPDATE posts SET content = ?, content_type = 3 WHERE id = 2001", "Editable Markdown").Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(bot).Update("nickname", "Agent").Error; err != nil {
		t.Fatal(err)
	}
	for v := uint64(1); v <= 2; v++ {
		if err := conn.Create(&postRevisions.Entity{PostId: 2001, Version: v, EditorId: 42, Content: "Editable Markdown"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	body := `{"topicId":1001}`
	ownerToken := contractSessionToken(t, owner)
	rec := serveJSON(router, "/api/admin/agents/content-source", body, ownerToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("source HTTP=%d: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "admin-agent-content-source-success.json"))
	var source api.BotContentSource
	if err := json.Unmarshal(decodeContractEnvelope(t, rec).Result, &source); err != nil {
		t.Fatal(err)
	}
	if source.BotUserId != 42 || source.RevisionNo != 2 {
		t.Fatalf("source=%+v", source)
	}
	rec = serveJSON(router, "/api/admin/agents/content-source", body, contractSessionToken(t, stranger))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("stranger HTTP=%d: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "admin-agent-content-unavailable.json"))
	rec = serveJSON(router, "/api/admin/agents/edit-topic", `{"topicId":1001,"title":"Corrected title","content":"Corrected bot body content","categoryId":[1],"expectedRevisionNo":1}`, ownerToken)
	if rec.Code != http.StatusConflict {
		t.Fatalf("stale HTTP=%d: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "admin-agent-content-conflict.json"))
	rec = serveJSON(router, "/api/admin/agents/edit-topic", `{"topicId":1001,"title":"Corrected title","content":"Corrected bot body content","categoryId":[1],"expectedRevisionNo":2}`, ownerToken)
	if rec.Code != http.StatusOK || decodeContractEnvelope(t, rec).Code != 0 {
		t.Fatalf("edit: %s", rec.Body.String())
	}
	rec = serveJSON(router, "/api/admin/agents/edit-post", `{"postId":2001,"content":"Updated first post through post editor","expectedRevisionNo":3}`, ownerToken)
	if rec.Code != http.StatusOK || decodeContractEnvelope(t, rec).Code != 0 {
		t.Fatalf("post edit: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), fmt.Sprintf(`"lastEditorId":%d`, owner.Id)) {
		t.Fatalf("operator missing: %s", rec.Body.String())
	}
	for _, path := range []string{"content-source", "edit-topic", "edit-post"} {
		assertInteractionUnauthenticated(t, router, "/api/admin/agents/"+path, `{}`, "auth-required.json")
	}
	rec = serveJSON(router, "/api/admin/agents/edit-post", `{"postId":2001,"content":"No expected version"}`, ownerToken)
	if rec.Code != http.StatusOK || decodeContractEnvelope(t, rec).MessageCode != "common.request.invalidParams" {
		t.Fatalf("missing concurrency guard HTTP=%d: %s", rec.Code, rec.Body.String())
	}
}

package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/anonymousidentityservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/gin-gonic/gin"
)

func TestAnonymousStateHTTPRateLimit(t *testing.T) {
	conn, _ := setupAccountContractTest(t)
	if err := conn.AutoMigrate(&identity.Persona{}, &identity.Binding{}, &identity.Quota{}, &identity.Batch{}); err != nil {
		t.Fatal(err)
	}
	restrictContractRateLimit(t, conn, middleware.RateLimitInteract)
	router := gin.New()
	RegisterByGin(router)
	owner := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, owner)
	for range 5 {
		rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/anonymous/state", "", token)
		if rec.Code != http.StatusOK || decodeContractEnvelope(t, rec).Code != 0 {
			t.Fatalf("state before rate limit: %d %s", rec.Code, rec.Body.String())
		}
	}
	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/anonymous/state", "", token)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" || !strings.Contains(rec.Body.String(), `"action":"interact"`) {
		t.Fatalf("state rate limit: %d %s", rec.Code, rec.Body.String())
	}
}

func TestAnonymousIdentityHTTPContract(t *testing.T) {
	conn, router := setupAccountContractTest(t)
	if err := conn.AutoMigrate(&identity.Persona{}, &identity.Binding{}, &identity.Quota{}, &identity.Batch{}, &identity.RevealAudit{}, &rolePermissionRs.Entity{}); err != nil {
		t.Fatal(err)
	}
	g := router.Group("/api/forum/anonymous").Use(middleware.JWTAuthCheck, middleware.NoUpdateUserActivity)
	g.GET("state", middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.AnonymousState))
	g.POST("batches", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.AnonymousGenerate))
	g.POST("confirm", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.AnonymousConfirm))
	g.POST("disable", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.AnonymousDisable))
	g.POST("reveal", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.AnonymousReveal))
	g.POST("govern", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpButterReq(api.AnonymousGovern))
	router.POST("/api/forum/posts/create", middleware.JWTAuthCheck, middleware.CheckWritableAccount, UpLimitedButterReq(maxContentWriteBodyBytes, api.CreatePost))
	owner := createHTTPContractUser(t, conn, contractTestID())
	auditor := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, owner)
	auditor.RoleId = contractTestID()
	conn.Model(auditor).Update("role_id", auditor.RoleId)
	auditToken := contractSessionToken(t, auditor)
	for _, op := range []string{"state", "batches", "confirm", "disable", "reveal", "govern"} {
		method := http.MethodPost
		if op == "state" {
			method = http.MethodGet
		}
		rec := serveAuthSecurityJSON(router, method, "/api/forum/anonymous/"+op, `{}`, "")
		if rec.Code != 401 {
			t.Fatalf("%s unauthenticated: %d %s", op, rec.Code, rec.Body)
		}
	}
	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/anonymous/state", "", token)
	state := decodeContractEnvelope(t, rec)
	if state.Code != 0 || rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal(rec.Body.String())
	}
	var initial anonymousidentityservice.State
	if err := json.Unmarshal(state.Result, &initial); err != nil || initial.Remaining != 10 || initial.Persona != nil {
		t.Fatal(initial, err)
	}
	body := fmt.Sprintf(`{"day":%q,"requestKey":"contract-anonymous-key"}`, initial.Day)
	batchResponse := decodeContractEnvelope(t, serveJSON(router, "/api/forum/anonymous/batches", body, token))
	var batch identity.Batch
	if err := json.Unmarshal(batchResponse.Result, &batch); err != nil || len(batch.Words) != 10 {
		t.Fatal(batchResponse, err)
	}
	repeated := decodeContractEnvelope(t, serveJSON(router, "/api/forum/anonymous/batches", body, token))
	if string(repeated.Result) != string(batchResponse.Result) {
		t.Fatal("idempotent draw changed")
	}
	confirmed := decodeContractEnvelope(t, serveJSON(router, "/api/forum/anonymous/confirm", fmt.Sprintf(`{"batchId":%q,"index":0}`, batch.ID), token))
	var public anonymousidentityservice.PublicPersona
	if err := json.Unmarshal(confirmed.Result, &public); err != nil || public.Name != batch.Words[0] || len(public.PublicUID) != 32 {
		t.Fatal(confirmed, err)
	}
	if strings.Contains(string(confirmed.Result), "owner") || strings.Contains(string(confirmed.Result), "seed") {
		t.Fatal("private projection")
	}
	for _, disabled := range []bool{true, false} {
		r := serveJSON(router, "/api/forum/anonymous/disable", fmt.Sprintf(`{"disabled":%t}`, disabled), token)
		assertFixtureEnvelope(t, decodeContractEnvelope(t, r), contractFixture(t, "anonymous-disable-success.json"))
	}
	revealBody := fmt.Sprintf(`{"publicUid":%q,"reason":"abuse investigation"}`, public.PublicUID)
	conn.Create(&rolePermissionRs.Entity{RoleId: auditor.RoleId, PermissionId: permission.Admin.Id(), Effective: 1})
	denied := decodeContractEnvelope(t, serveJSON(router, "/api/forum/anonymous/reveal", revealBody, auditToken))
	if denied.Code == 0 {
		t.Fatal("admin implied reveal")
	}
	conn.Create(&rolePermissionRs.Entity{RoleId: auditor.RoleId, PermissionId: permission.RevealAnonymousIdentity.Id(), Effective: 1})
	rec = serveJSON(router, "/api/forum/anonymous/reveal", revealBody, auditToken)
	var revealed anonymousidentityservice.RevealedOwner
	if err := json.Unmarshal(decodeContractEnvelope(t, rec).Result, &revealed); err != nil || revealed.UserID != owner.Id || rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal(revealed, err, rec.Body.String())
	}
	topic := topics.Entity{Id: contractTestID(), UserId: owner.Id, PersonaUID: public.PublicUID, Status: 1, CategoryIds: []uint64{999991}}
	post := posts.Entity{Id: contractTestID(), TopicId: topic.Id, UserId: owner.Id, PersonaUID: public.PublicUID, IsAnonymous: true, PostNo: 1}
	conn.Create(&topic)
	conn.Create(&post)
	t.Cleanup(func() { conn.Unscoped().Delete(&post); conn.Unscoped().Delete(&topic); moderationservice.Invalidate() })
	governBody := fmt.Sprintf(`{"postId":%d,"disabled":true,"reason":"scoped abuse"}`, post.Id)
	ordinary := createHTTPContractUser(t, conn, contractTestID())
	modToken := contractSessionToken(t, ordinary)
	grant := moderators.Entity{UserId: ordinary.Id, ScopeType: moderators.ScopeCategory, ScopeId: 999990, Status: moderators.StatusEnabled}
	conn.Create(&grant)
	moderationservice.Invalidate()
	t.Cleanup(func() { conn.Delete(&grant) })
	denied = decodeContractEnvelope(t, serveJSON(router, "/api/forum/anonymous/govern", governBody, modToken))
	if denied.Code == 0 {
		t.Fatal("category scope bypass")
	}
	conn.Model(&grant).Update("scope_id", 999991)
	moderationservice.Invalidate()
	governed := serveJSON(router, "/api/forum/anonymous/govern", governBody, modToken)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, governed), contractFixture(t, "anonymous-govern-success.json"))
	if strings.Contains(governed.Body.String(), owner.Username) {
		t.Fatal("governance revealed owner")
	}
	rec = serveJSON(router, "/api/forum/anonymous/disable", `{"disabled":false}`, token)
	if rec.Code != 403 {
		t.Fatal("restricted owner bypass", rec.Code, rec.Body.String())
	}
	// Governance is account-wide, so neither public identity can bypass it.
	for _, path := range []string{"/api/forum/topics/write", "/api/forum/posts/create"} {
		for _, identity := range []string{"member", "persona"} {
			rec = serveJSON(router, path, fmt.Sprintf(`{"identity":%q}`, identity), token)
			if rec.Code != 403 || decodeContractEnvelope(t, rec).MessageCode != "permission.userFrozen" {
				t.Fatalf("governed %s bypassed writing guard on %s: %d %s", identity, path, rec.Code, rec.Body.String())
			}
		}
	}
	if err := sessionservice.RevokeAllAndInvalidate(owner.Id); err != nil {
		t.Fatal(err)
	}
	rec = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/anonymous/state", "", token)
	if rec.Code != 401 {
		t.Fatal("revoked session read", rec.Code, rec.Body.String())
	}
}

func TestPersonaPublishingAndAuthorImmutabilityHTTP(t *testing.T) {
	conn, router := setupForumInteractionContractTest(t)
	if err := conn.AutoMigrate(&identity.Persona{}, &identity.Binding{}); err != nil {
		t.Fatal(err)
	}
	owner := createHTTPContractUser(t, conn, contractTestID())
	uid := strings.Repeat("e", 32)
	persona := identity.Persona{UID: uid, Name: "C++", AvatarSeed: strings.Repeat("f", 32)}
	binding := identity.Binding{OwnerID: owner.Id, PersonaUID: uid}
	for _, row := range []any{&persona, &binding} {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Delete(row) })
	}
	categoryID := contractTestID()
	cat := category.Entity{Id: categoryID, Name: "Anonymous", Slug: fmt.Sprintf("anonymous-%d", categoryID)}
	if err := conn.Create(&cat).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Delete(&cat) })
	token := contractSessionToken(t, owner)
	body := fmt.Sprintf(`{"title":"Anonymous topic contract","content":"Anonymous first post body long enough to publish safely.","categoryId":[%d],"topicStatus":1,"identity":"persona"}`, categoryID)
	response := decodeContractEnvelope(t, serveJSON(router, "/api/forum/topics/write", body, token))
	if response.Code != 0 {
		t.Fatal(response)
	}
	var topicID uint64
	if err := json.Unmarshal(response.Result, &topicID); err != nil {
		t.Fatal(err)
	}
	var topic topics.Entity
	if err := conn.First(&topic, topicID).Error; err != nil {
		t.Fatal(err)
	}
	if topic.PersonaUID != uid || topic.UserId != owner.Id || len(topic.Posters) != 0 {
		t.Fatal("topic authorship not separated", topic)
	}
	createReply := func(identity string) posts.Entity {
		t.Helper()
		body := fmt.Sprintf(`{"topicId":%d,"content":"Reply body long enough for publishing rules.","identity":%q}`, topicID, identity)
		response := decodeContractEnvelope(t, serveJSON(router, "/api/forum/posts/create", body, token))
		if response.Code != 0 {
			t.Fatal(response)
		}
		var result struct {
			ID uint64 `json:"id"`
		}
		if err := json.Unmarshal(response.Result, &result); err != nil {
			t.Fatal(err)
		}
		var post posts.Entity
		if err := conn.First(&post, result.ID).Error; err != nil {
			t.Fatal(err)
		}
		return post
	}
	anonymous := createReply("persona")
	member := createReply("member")
	if anonymous.PersonaUID != uid || !anonymous.IsAnonymous || member.PersonaUID != "" || member.IsAnonymous {
		t.Fatal("cross-identity reply choice lost")
	}
	changed := decodeContractEnvelope(t, serveJSON(router, "/api/forum/posts/update", fmt.Sprintf(`{"postId":%d,"content":"Edited body retains the original anonymous author.","identity":"member"}`, anonymous.Id), token))
	if changed.Code != 0 {
		t.Fatal(changed)
	}
	var saved posts.Entity
	conn.First(&saved, anonymous.Id)
	if saved.PersonaUID != uid || saved.UserId != owner.Id {
		t.Fatal("body edit changed author")
	}
	selfLike := decodeContractEnvelope(t, serveJSON(router, "/api/forum/posts/like", fmt.Sprintf(`{"postId":%d,"action":1}`, anonymous.Id), token))
	if selfLike.Code == 0 {
		t.Fatal("persona owner liked their own post")
	}
	window := serveAuthSecurityJSON(router, http.MethodGet, fmt.Sprintf("/api/forum/posts/window?topicId=%d", topicID), "", "")
	var publicWindow forum.PostWindowPayload
	if err := json.Unmarshal(decodeContractEnvelope(t, window).Result, &publicWindow); err != nil {
		t.Fatal(err)
	}
	for _, post := range publicWindow.Posts {
		if post.Author.PublicUID == uid && (post.Author.ID != 0 || post.Author.Username == owner.Username) {
			t.Fatal("public window exposes anonymous owner", post)
		}
	}
	if err := conn.Model(&persona).Update("disabled", true).Error; err != nil {
		t.Fatal(err)
	}
	denied := decodeContractEnvelope(t, serveJSON(router, "/api/forum/posts/create", fmt.Sprintf(`{"topicId":%d,"content":"Disabled persona must not silently become member.","identity":"persona"}`, topicID), token))
	if denied.Code == 0 {
		t.Fatal("disabled persona published")
	}
}

package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/http/controllers/api"
	"github.com/leancodebox/GooseForum/app/models/forum/agents"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
)

func setupAgentAdminTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &agents.Entity{}, &userStatistics.Entity{}); err != nil {
		t.Fatalf("migrate agent tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&agents.Entity{})
	conn.Where("1 = 1").Delete(&userStatistics.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
}

// agentAdminRouter registers the agent routes with an authenticated admin
// user injected via middleware, matching the production POST contract.
func agentAdminRouter(userID uint64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", userID)
		c.Next()
	})
	router.POST("api/admin/agent-list", UpButterReq(api.AgentList))
	router.POST("api/admin/agent-create", UpButterReq(api.AgentCreate))
	router.POST("api/admin/agent-update", UpButterReq(api.AgentUpdate))
	router.POST("api/admin/agent-rotate-token", UpButterReq(api.AgentRotateToken))
	router.POST("api/admin/agent-disable", UpButterReq(api.AgentDisable))
	return router
}

func postAgent(t *testing.T, router http.Handler, path, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return rec.Code, resp
}

func decodeAgentResult(t *testing.T, resp map[string]any) map[string]any {
	t.Helper()
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("result missing or wrong type in %#v", resp)
	}
	return result
}

func TestAgentCreateReturnsTokenOnceAndPersists(t *testing.T) {
	setupAgentAdminTestDB(t)
	router := agentAdminRouter(1)

	code, resp := postAgent(t, router, "/api/admin/agent-create", `{"username":"admin-agent-1","nickname":"Admin Agent","webhookEndpoint":"https://example.com/hook"}`)
	if code != http.StatusOK || resp["code"].(float64) != 0 {
		t.Fatalf("create failed: status=%d body=%s", code, mustJSON(resp))
	}
	result := decodeAgentResult(t, resp)
	token, _ := result["token"].(string)
	if !strings.HasPrefix(token, "agt_") {
		t.Fatalf("token = %q, want agt_ prefix", token)
	}
	agent := result["agent"].(map[string]any)
	if agent["username"] != "admin-agent-1" {
		t.Fatalf("agent username = %v", agent["username"])
	}
	if agent["enabled"].(float64) != 1 {
		t.Fatalf("agent enabled = %v, want 1", agent["enabled"])
	}
	if agent["webhookEndpoint"] != "https://example.com/hook" {
		t.Fatalf("webhook = %v", agent["webhookEndpoint"])
	}
	if _, hasHash := agent["tokenHash"]; hasHash {
		t.Fatal("tokenHash must not be exposed")
	}
	agentID := uint64(agent["agentId"].(float64))
	stored := agents.GetByUserID(agentID)
	if stored == nil || stored.TokenHash == "" || stored.TokenHash == token {
		t.Fatal("stored token must be a hash, not plaintext")
	}
	if stored.TokenPrefix != token[:len("agt_")+8] {
		t.Fatalf("stored prefix = %q, want %q", stored.TokenPrefix, token[:len("agt_")+8])
	}
}

func TestAgentCreateRejectsInvalidUsername(t *testing.T) {
	setupAgentAdminTestDB(t)
	router := agentAdminRouter(1)

	_, resp := postAgent(t, router, "/api/admin/agent-create", `{"username":"bad"}`)
	if resp["code"].(float64) == 0 {
		t.Fatalf("expected failure: %s", mustJSON(resp))
	}
	if resp["messageCode"] != "admin.agent.usernameInvalid" {
		t.Fatalf("messageCode = %v", resp["messageCode"])
	}
}

func TestAgentCreateRejectsDuplicateUsername(t *testing.T) {
	setupAgentAdminTestDB(t)
	router := agentAdminRouter(1)
	if _, resp := postAgent(t, router, "/api/admin/agent-create", `{"username":"dup-admin"}`); resp["code"].(float64) != 0 {
		t.Fatalf("first create failed: %s", mustJSON(resp))
	}
	_, resp := postAgent(t, router, "/api/admin/agent-create", `{"username":"dup-admin"}`)
	if resp["code"].(float64) == 0 {
		t.Fatal("duplicate create should fail")
	}
	if resp["messageCode"] != "admin.agent.usernameExists" {
		t.Fatalf("messageCode = %v", resp["messageCode"])
	}
}

func TestAgentCreateRejectsInvalidWebhook(t *testing.T) {
	setupAgentAdminTestDB(t)
	router := agentAdminRouter(1)
	_, resp := postAgent(t, router, "/api/admin/agent-create", `{"username":"bad-hook-admin","webhookEndpoint":"ftp://x"}`)
	if resp["code"].(float64) == 0 {
		t.Fatal("invalid webhook should fail")
	}
	if resp["messageCode"] != "admin.agent.webhookInvalid" {
		t.Fatalf("messageCode = %v", resp["messageCode"])
	}
}

func TestAgentListReturnsAgents(t *testing.T) {
	setupAgentAdminTestDB(t)
	router := agentAdminRouter(1)
	for _, username := range []string{"list-admin-1", "list-admin-2"} {
		if _, resp := postAgent(t, router, "/api/admin/agent-create", `{"username":"`+username+`"}`); resp["code"].(float64) != 0 {
			t.Fatalf("create %s failed: %s", username, mustJSON(resp))
		}
	}

	code, resp := postAgent(t, router, "/api/admin/agent-list", `{}`)
	if code != http.StatusOK || resp["code"].(float64) != 0 {
		t.Fatalf("list failed: %s", mustJSON(resp))
	}
	list, ok := resp["result"].([]any)
	if !ok || len(list) != 2 {
		t.Fatalf("list result = %#v, want 2 items", resp["result"])
	}
}

func TestAgentUpdateProfileWebhookAndEnable(t *testing.T) {
	setupAgentAdminTestDB(t)
	router := agentAdminRouter(1)
	_, resp := postAgent(t, router, "/api/admin/agent-create", `{"username":"upd-admin","nickname":"Before","webhookEndpoint":"https://a.example.com"}`)
	if resp["code"].(float64) != 0 {
		t.Fatalf("create failed: %s", mustJSON(resp))
	}
	agent := decodeAgentResult(t, resp)["agent"].(map[string]any)
	agentID := int(agent["agentId"].(float64))

	code, resp := postAgent(t, router, "/api/admin/agent-update", `{"agentId":`+strconv.Itoa(agentID)+`,"nickname":"After","webhookEndpoint":"https://b.example.com"}`)
	if code != http.StatusOK || resp["code"].(float64) != 0 {
		t.Fatalf("update failed: %s", mustJSON(resp))
	}
	updated := decodeAgentResult(t, resp)
	if updated["nickname"] != "After" {
		t.Fatalf("nickname = %v, want After", updated["nickname"])
	}
	if updated["webhookEndpoint"] != "https://b.example.com" {
		t.Fatalf("webhook = %v", updated["webhookEndpoint"])
	}
	_, resp = postAgent(t, router, "/api/admin/agent-update", `{"agentId":`+strconv.Itoa(agentID)+`,"enabled":0}`)
	if resp["code"].(float64) != 0 {
		t.Fatalf("disable via update failed: %s", mustJSON(resp))
	}
	if decodeAgentResult(t, resp)["enabled"].(float64) != 0 {
		t.Fatal("enabled should be 0 after disable")
	}
}

func TestAgentRotateTokenReturnsNewTokenAndInvalidatesOld(t *testing.T) {
	setupAgentAdminTestDB(t)
	router := agentAdminRouter(1)
	_, resp := postAgent(t, router, "/api/admin/agent-create", `{"username":"rotate-admin"}`)
	if resp["code"].(float64) != 0 {
		t.Fatalf("create failed: %s", mustJSON(resp))
	}
	result := decodeAgentResult(t, resp)
	oldToken := result["token"].(string)
	agent := result["agent"].(map[string]any)
	agentID := int(agent["agentId"].(float64))

	code, resp := postAgent(t, router, "/api/admin/agent-rotate-token", `{"agentId":`+strconv.Itoa(agentID)+`}`)
	if code != http.StatusOK || resp["code"].(float64) != 0 {
		t.Fatalf("rotate failed: %s", mustJSON(resp))
	}
	newToken := decodeAgentResult(t, resp)["token"].(string)
	if newToken == oldToken || !strings.HasPrefix(newToken, "agt_") {
		t.Fatalf("new token = %q", newToken)
	}
	stored := agents.GetByUserID(uint64(agentID))
	if stored == nil || stored.TokenHash == "" {
		t.Fatal("rotated token hash missing")
	}
	if stored.TokenHash == sha256Hex(oldToken) {
		t.Fatal("old token hash still stored after rotate")
	}
}

func TestAgentDisableEndpoint(t *testing.T) {
	setupAgentAdminTestDB(t)
	router := agentAdminRouter(1)
	_, resp := postAgent(t, router, "/api/admin/agent-create", `{"username":"disable-admin"}`)
	if resp["code"].(float64) != 0 {
		t.Fatalf("create failed: %s", mustJSON(resp))
	}
	agent := decodeAgentResult(t, resp)["agent"].(map[string]any)
	agentID := int(agent["agentId"].(float64))

	code, resp := postAgent(t, router, "/api/admin/agent-disable", `{"agentId":`+strconv.Itoa(agentID)+`}`)
	if code != http.StatusOK || resp["code"].(float64) != 0 {
		t.Fatalf("disable failed: %s", mustJSON(resp))
	}
	stored := agents.GetByUserID(uint64(agentID))
	if stored == nil || stored.Enabled != agents.StatusDisabled {
		t.Fatalf("agent enabled = %v, want disabled", stored.Enabled)
	}
}

func TestAgentUpdateNotFound(t *testing.T) {
	setupAgentAdminTestDB(t)
	router := agentAdminRouter(1)
	_, resp := postAgent(t, router, "/api/admin/agent-update", `{"agentId":999999,"nickname":"X"}`)
	if resp["code"].(float64) == 0 {
		t.Fatal("update of missing agent should fail")
	}
	if resp["messageCode"] != "admin.agent.notFound" {
		t.Fatalf("messageCode = %v", resp["messageCode"])
	}
}

func sha256Hex(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func mustJSON(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}

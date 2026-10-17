package routes

import (
	"net/http"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
)

// The static Authorization header is write-only: responses expose only whether it is set.
func TestAgentWebhookAuthorizationIsWriteOnly(t *testing.T) {
	preferences.Set("app.signingKey", "J2h4lsy5qr57jgvPUSvwUxHrxrOXTr92eCaHGAMtfTuK3vJA")
	setupAgentAdminTestDB(t)
	router := agentAdminRouter(t)
	router.POST("api/admin/agent-webhook-config", UpButterReq(api.AgentWebhookConfigure))

	_, resp := postAgent(t, router, "/api/admin/agent-create", `{"username":"auth-header-agent"}`)
	if resp["code"].(float64) != 0 {
		t.Fatalf("create failed: %s", mustJSON(resp))
	}
	agent := decodeAgentResult(t, resp)["agent"].(map[string]any)
	if agent["webhookAuthorizationConfigured"] != false {
		t.Fatalf("new agent webhookAuthorizationConfigured = %v", agent["webhookAuthorizationConfigured"])
	}
	agentID := uint64(agent["agentId"].(float64))
	idJSON := mustJSON(agent["agentId"])

	const secret = "Bearer route-level-secret-token"
	version := mustJSON(agent["configVersion"])
	code, resp := postAgent(t, router, "/api/admin/agent-webhook-config", `{"agentId":`+idJSON+`,"configVersion":`+version+`,"webhookAuthorization":"`+secret+`"}`)
	if code != http.StatusOK || resp["code"].(float64) != 0 {
		t.Fatalf("configure failed: %s", mustJSON(resp))
	}
	body := mustJSON(resp)
	if strings.Contains(body, "route-level-secret-token") {
		t.Fatalf("configure response leaks authorization: %s", body)
	}
	item := decodeAgentResult(t, resp)
	if item["webhookAuthorizationConfigured"] != true {
		t.Fatalf("webhookAuthorizationConfigured = %v", item["webhookAuthorizationConfigured"])
	}
	stored := agents.GetByUserID(agentID)
	if stored.WebhookAuthorizationCiphertext == "" || strings.Contains(stored.WebhookAuthorizationCiphertext, "route-level-secret-token") {
		t.Fatalf("authorization not encrypted at rest: %q", stored.WebhookAuthorizationCiphertext)
	}
	if strings.Contains(body, stored.WebhookAuthorizationCiphertext) {
		t.Fatal("configure response leaks ciphertext")
	}

	_, list := postAgent(t, router, "/api/admin/agent-list", `{}`)
	if listBody := mustJSON(list); strings.Contains(listBody, "route-level-secret-token") || strings.Contains(listBody, stored.WebhookAuthorizationCiphertext) {
		t.Fatalf("list leaks authorization: %s", listBody)
	}

	// Saving without the field keeps it; an empty string clears it.
	version = mustJSON(item["configVersion"])
	_, resp = postAgent(t, router, "/api/admin/agent-webhook-config", `{"agentId":`+idJSON+`,"configVersion":`+version+`}`)
	item = decodeAgentResult(t, resp)
	if item["webhookAuthorizationConfigured"] != true {
		t.Fatal("omitted webhookAuthorization cleared the stored value")
	}
	version = mustJSON(item["configVersion"])
	_, resp = postAgent(t, router, "/api/admin/agent-webhook-config", `{"agentId":`+idJSON+`,"configVersion":`+version+`,"webhookAuthorization":""}`)
	if decodeAgentResult(t, resp)["webhookAuthorizationConfigured"] != false || agents.GetByUserID(agentID).WebhookAuthorizationCiphertext != "" {
		t.Fatalf("empty webhookAuthorization did not clear: %s", mustJSON(resp))
	}
}

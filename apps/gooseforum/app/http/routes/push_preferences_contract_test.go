package routes

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestPushPreferencesHTTPContract(t *testing.T) {
	conn, router := setupNotificationChatContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	other := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	response := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/push/preferences", "", token)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "push-preferences-defaults.json"))
	response = serveAuthSecurityJSON(router, http.MethodPut, "/api/forum/push/preferences", `{"chat":false}`, token)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	var envelope struct {
		Code   int `json:"code"`
		Result struct {
			Chat        bool `json:"chat"`
			System      bool `json:"system"`
			DefaultOnly bool `json:"systemDefaultOnly"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Code != 0 || envelope.Result.Chat || !envelope.Result.System || !envelope.Result.DefaultOnly {
		t.Fatalf("partial override changed selective defaults: %+v", envelope)
	}
	response = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/push/preferences", "", contractSessionToken(t, other))
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "push-preferences-defaults.json"))
	response = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/push/preferences", "", "")
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status %d", response.Code)
	}
}

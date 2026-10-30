package httpnotifyservice

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

func TestFeishuSignupSharesNotificationEndpointWithoutApprovalActions(t *testing.T) {
	endpoint := pageConfig.HttpNotifyEndpoint{Enabled: true, URL: "https://example.test/hook", ChannelType: pageConfig.HttpNotifyChannelFeishu, Events: []string{EventReportCreated, EventUserSignup}}
	message := Message{Alternatives: []Alternative{{Event: EventUserSignup, Data: map[string]any{"user": map[string]any{"username": "new-student", "displayName": "<at id=all>student</at>", "email": "private@example.com"}}}}}
	alt, ok := selectAlternative(endpoint, message)
	if !ok {
		t.Fatal("signup event was excluded from the existing endpoint")
	}
	body, err := channelFor(endpoint).encode(endpoint, alt, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "new-student") || !strings.Contains(string(body), "plain_text") {
		t.Fatalf("missing safe signup summary: %s", body)
	}
	for _, forbidden := range []string{"private@example.com", `"tag":"button"`, `"tag":"markdown"`} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("signup contains private data or approval action: %s", body)
		}
	}
	if _, ok := selectAlternative(endpoint, Message{Alternatives: []Alternative{{Event: EventReportCreated}}}); ok {
		t.Fatal("report without approval payload must still be rejected")
	}
}

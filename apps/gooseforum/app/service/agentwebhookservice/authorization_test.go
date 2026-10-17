package agentwebhookservice

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentWebhook"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"gorm.io/gorm"
)

func strPtr(s string) *string { return &s }

func configureAuthorization(t *testing.T, agentID uint64, value *string) *agents.Entity {
	t.Helper()
	current := agents.GetByUserID(agentID)
	row, err := Configure(agentID, current.ConfigVersion, ConfigParams{EventsEnabled: true, EventTypes: []string{"agent.mentioned"}, WebhookEnabled: true, WebhookEndpoint: "https://example.com/hook", WebhookAuthorization: value})
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func authorizePermit(t *testing.T, conn *gorm.DB, agentID uint64) permit {
	t.Helper()
	row, err := Test(agentID, 2)
	if err != nil {
		t.Fatal(err)
	}
	task := claimTask(t, row.TaskID)
	var p permit
	if err := conn.Transaction(func(tx *gorm.DB) error { var err error; p, err = authorizeTx(tx, &task, row.ID); return err }); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestWebhookAuthorizationHeaderSentOnlyWhenConfigured(t *testing.T) {
	conn, row, task := configuredDelivery(t)
	var p permit
	if err := conn.Transaction(func(tx *gorm.DB) error { var err error; p, err = authorizeTx(tx, &task, row.ID); return err }); err != nil {
		t.Fatal(err)
	}
	if _, ok := deliveryHeaders(p, 1, "v1,sig")["Authorization"]; ok {
		t.Fatal("Authorization header sent without configuration")
	}

	configureAuthorization(t, row.AgentID, strPtr("  Bearer receiver-token-123  "))
	p = authorizePermit(t, conn, row.AgentID)
	headers := deliveryHeaders(p, 1, "v1,sig")
	if got := headers.Get("Authorization"); got != "Bearer receiver-token-123" {
		t.Fatalf("Authorization = %q", got)
	}
	if headers.Get("Webhook-Signature") != "v1,sig" || headers.Get("Webhook-Id") == "" {
		t.Fatalf("standard webhook headers changed: %v", headers)
	}
}

func TestWebhookAuthorizationEncryptedAtRestKeepAndClear(t *testing.T) {
	conn, row, _ := configuredDelivery(t)
	const plain = "Bearer at-rest-secret-value"
	configureAuthorization(t, row.AgentID, strPtr(plain))

	stored := agents.GetByUserID(row.AgentID)
	if stored.WebhookAuthorizationCiphertext == "" || strings.Contains(stored.WebhookAuthorizationCiphertext, "at-rest-secret-value") {
		t.Fatalf("authorization not encrypted at rest: %q", stored.WebhookAuthorizationCiphertext)
	}
	if got, err := securestore.DecryptPurpose(stored.WebhookAuthorizationCiphertext, securestore.AgentWebhookAuthorizationPurpose); err != nil || got != plain {
		t.Fatalf("decrypt = %q, %v", got, err)
	}
	if _, err := securestore.DecryptPurpose(stored.WebhookAuthorizationCiphertext, securestore.AgentWebhookSecretPurpose); err == nil {
		t.Fatal("authorization ciphertext must not decrypt under the HMAC secret purpose")
	}
	encoded, _ := json.Marshal(stored)
	if strings.Contains(string(encoded), "at-rest-secret-value") || strings.Contains(string(encoded), stored.WebhookAuthorizationCiphertext) {
		t.Fatalf("entity JSON leaks authorization: %s", encoded)
	}
	if summary, err := Summary(row.AgentID); err != nil || !summary.AuthorizationConfigured {
		t.Fatalf("summary = %#v, %v", summary, err)
	}

	// Omitted field keeps the stored value.
	configureAuthorization(t, row.AgentID, nil)
	if agents.GetByUserID(row.AgentID).WebhookAuthorizationCiphertext != stored.WebhookAuthorizationCiphertext {
		t.Fatal("omitted webhookAuthorization changed the stored value")
	}
	// Empty string clears it, and deliveries stop sending the header.
	configureAuthorization(t, row.AgentID, strPtr(""))
	if agents.GetByUserID(row.AgentID).WebhookAuthorizationCiphertext != "" {
		t.Fatal("empty webhookAuthorization did not clear the stored value")
	}
	_ = conn
}

func TestWebhookAuthorizationRejectsHeaderInjection(t *testing.T) {
	_, row, _ := configuredDelivery(t)
	for _, value := range []string{"Bearer a\r\nX-Evil: 1", "Bearer a\nb", "Bearer \x00", "Bearer é", strings.Repeat("a", MaxWebhookAuthorizationLength+1)} {
		current := agents.GetByUserID(row.AgentID)
		_, err := Configure(row.AgentID, current.ConfigVersion, ConfigParams{EventsEnabled: true, EventTypes: []string{"agent.mentioned"}, WebhookEnabled: true, WebhookEndpoint: "https://example.com/hook", WebhookAuthorization: strPtr(value)})
		if !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("value %q: err = %v, want ErrInvalidConfig", value, err)
		}
	}
	if agents.GetByUserID(row.AgentID).WebhookAuthorizationCiphertext != "" {
		t.Fatal("rejected value was persisted")
	}
}

func TestWebhookAuthorizationDamagedCiphertextPausesDelivery(t *testing.T) {
	conn, row, task := configuredDelivery(t)
	if err := agents.UpdateColumns(conn, row.AgentID, map[string]any{"webhook_authorization_ciphertext": "not-a-valid-ciphertext"}); err != nil {
		t.Fatal(err)
	}
	var p permit
	if err := conn.Transaction(func(tx *gorm.DB) error { var err error; p, err = authorizeTx(tx, &task, row.ID); return err }); err != nil {
		t.Fatal(err)
	}
	if p.attemptID != "" {
		t.Fatal("damaged authorization must not authorize a send")
	}
	got, _ := agentWebhook.GetTx(conn, "inst_test", row.ID, false)
	if got.Status != agentWebhook.Dead || got.Reason != "authorization_invalid" {
		t.Fatalf("delivery = %#v", got)
	}
	if agents.GetByUserID(row.AgentID).WebhookPausedReason != "authorization_invalid" {
		t.Fatal("agent not paused for damaged authorization")
	}
}

func claimTask(t *testing.T, id uint64) taskQueue.Entity {
	t.Helper()
	task, claimed, err := taskQueue.ClaimTask(id)
	if err != nil || !claimed {
		t.Fatalf("claim=%t err=%v", claimed, err)
	}
	return task
}

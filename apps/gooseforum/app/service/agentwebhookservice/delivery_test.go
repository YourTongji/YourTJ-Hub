package agentwebhookservice

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/agentinstance"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/safefetch"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentWebhook"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"gorm.io/gorm"
)

func setup(t *testing.T) *gorm.DB {
	t.Helper()
	t.Setenv("YOURTJ_AGENT_INSTANCE_ID", "inst_test")
	t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "epoch_test")
	t.Setenv("YOURTJ_AGENT_WEBHOOK_ENABLED", "true")
	preferences.Set("app.signingKey", "J2h4lsy5qr57jgvPUSvwUxHrxrOXTr92eCaHGAMtfTuK3vJA")
	previousSender := sender
	sender = safefetch.New(safefetch.Config{Resolver: webhookTestResolver{}})
	t.Cleanup(func() { sender = previousSender })
	conn := db.Connect()
	if err := conn.AutoMigrate(&agents.Entity{}, &agentEvents.Entity{}, &agentWebhook.Delivery{}, &agentWebhook.Attempt{}, &taskQueue.Entity{}); err != nil {
		t.Fatal(err)
	}
	for _, row := range []any{&agents.Entity{}, &agentEvents.Entity{}, &agentWebhook.Delivery{}, &agentWebhook.Attempt{}, &taskQueue.Entity{}} {
		if err := conn.Where("1 = 1").Delete(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	return conn
}
func TestStandardWebhookSignatureRawBytes(t *testing.T) {
	secret := "whsec_" + base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	body := []byte("{\"id\":\"evt_fixed\", \"instanceId\":\"inst_a\"}\n")
	got, err := Sign(secret, "evt_fixed", 1700000000, body)
	if err != nil {
		t.Fatal(err)
	}
	// Fixed protocol vector catches accidental JSON normalization or prefix signing.
	want := "v1,q+aN/U3Dm33/41erTjlCe7v71qllg4fDizScpGv/hdI="
	mac := hmac.New(sha256.New, []byte("01234567890123456789012345678901"))
	_, _ = mac.Write([]byte("evt_fixed.1700000000."))
	_, _ = mac.Write(body)
	if got != "v1,"+base64.StdEncoding.EncodeToString(mac.Sum(nil)) {
		t.Fatalf("signature=%s", got)
	}
	if got != want {
		t.Fatalf("fixed vector: %s, want %s", got, want)
	}
	changed, err := Sign(secret, "evt_fixed", 1700000000, []byte(strings.TrimSpace(string(body))))
	if err != nil {
		t.Fatal(err)
	}
	if changed == got {
		t.Fatal("raw whitespace must participate in signature")
	}
}
func TestConfigCASGenerationsAndSecretRotation(t *testing.T) {
	conn := setup(t)
	a := agents.Entity{UserId: 12, TokenPrefix: "agt_cfg", Enabled: 1}
	if err := conn.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	first, err := RotateSecret(a.UserId, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if first.SecretVersion != 1 || first.ConfigVersion != 1 || !strings.HasPrefix(first.Secret, "whsec_") {
		t.Fatalf("result=%#v", first)
	}
	if _, err := RotateSecret(a.UserId, 0, false); !errors.Is(err, ErrConfigConflict) {
		t.Fatalf("stale CAS=%v", err)
	}
	row, err := Configure(a.UserId, 1, ConfigParams{EventsEnabled: true, EventTypes: []string{"agent.mentioned"}, WebhookEnabled: true, WebhookEndpoint: "https://example.com/hook"})
	if err != nil {
		t.Fatal(err)
	}
	if row.EndpointGeneration != 1 || row.SubscriptionGeneration != 1 {
		t.Fatalf("generations=%#v", row)
	}
	second, err := RotateSecret(a.UserId, row.ConfigVersion, false)
	if err != nil {
		t.Fatal(err)
	}
	row = agents.GetByUserID(a.UserId)
	if row.PreviousSecretCiphertext == "" || row.PreviousSecretExpiresAt == nil {
		t.Fatal("missing persisted overlap")
	}
	if _, err := RotateSecret(a.UserId, second.ConfigVersion, true); err != nil {
		t.Fatal(err)
	}
	row = agents.GetByUserID(a.UserId)
	if row.PreviousSecretCiphertext != "" || row.PreviousSecretExpiresAt != nil {
		t.Fatal("emergency rotation must revoke old secret")
	}
	if _, err := Configure(a.UserId, row.ConfigVersion, ConfigParams{WebhookEndpoint: "http://example.com/hook"}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("unsafe config=%v", err)
	}
}
func TestConfigureAcceptsForumBroadcastTypes(t *testing.T) {
	conn := setup(t)
	a := agents.Entity{UserId: 13, TokenPrefix: "agt_broadcast", Enabled: 1}
	if err := conn.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	row, err := Configure(a.UserId, 0, ConfigParams{EventsEnabled: true, EventTypes: []string{"forum.topic_created", "forum.post_created", "agent.mentioned"}})
	if err != nil {
		t.Fatal(err)
	}
	if row.EventTypes != `["agent.mentioned","forum.post_created","forum.topic_created"]` {
		t.Fatalf("eventTypes=%s", row.EventTypes)
	}
	if _, err := Configure(a.UserId, row.ConfigVersion, ConfigParams{EventsEnabled: true, EventTypes: []string{"forum.unknown"}}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("unknown type accepted: %v", err)
	}
}
func TestPersistentRetryBudgetAndFencing(t *testing.T) {
	conn := setup(t)
	a := agents.Entity{UserId: 14, TokenPrefix: "agt_budget", Enabled: 1}
	if err := conn.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	secret, err := RotateSecret(a.UserId, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Configure(a.UserId, secret.ConfigVersion, ConfigParams{EventsEnabled: true, EventTypes: []string{"agent.mentioned"}, WebhookEnabled: true, WebhookEndpoint: "https://example.com/hook"})
	if err != nil {
		t.Fatal(err)
	}
	row, err := Test(a.UserId, 2)
	if err != nil {
		t.Fatal(err)
	}
	task, claimed, err := taskQueue.ClaimTask(row.TaskID)
	if err != nil || !claimed {
		t.Fatalf("claim=%v,%v", claimed, err)
	}
	var p permit
	if err := conn.Transaction(func(tx *gorm.DB) error { var err error; p, err = authorizeTx(tx, &task, row.ID); return err }); err != nil {
		t.Fatal(err)
	}
	if err := complete(&task, p, safefetch.Result{StatusCode: 429, RetryAfter: "120"}, nil); err != nil {
		t.Fatal(err)
	}
	updated, err := taskQueue.GetByID(row.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != taskQueue.StatusRetrying || updated.NextRunAt == nil || !updated.NextRunAt.After(time.Now()) {
		t.Fatalf("retry=%#v", updated)
	}
	if _, claimed, err := taskQueue.ClaimTask(row.TaskID); err != nil || claimed {
		t.Fatalf("future task claimed=%v err=%v", claimed, err)
	}
	// Old worker cannot turn the newly delayed task into accepted.
	if err := complete(&task, p, safefetch.Result{StatusCode: 200}, nil); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("stale result=%v", err)
	}
	delivery, err := agentWebhook.GetTx(conn, agentinstance.Current().ID, row.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.Status != agentWebhook.RetryWait || delivery.AcceptedAt != nil || delivery.AttemptCount != 1 {
		t.Fatalf("delivery=%#v", delivery)
	}
	// Crash/restart state retains the finite attempt count and absolute deadline.
	if err := conn.Model(&agentWebhook.Delivery{}).Where("id = ?", row.ID).Update("attempt_count", 9).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&taskQueue.Entity{}).Where("id = ?", row.TaskID).Update("next_run_at", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	task, claimed, err = taskQueue.ClaimTask(row.TaskID)
	if err != nil || !claimed {
		t.Fatal(err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { _, err := authorizeTx(tx, &task, row.ID); return err }); err != nil {
		t.Fatal(err)
	}
	delivery, err = agentWebhook.GetTx(conn, agentinstance.Current().ID, row.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.Reason != "retry_budget_exhausted" {
		t.Fatalf("reason=%q", delivery.Reason)
	}
}

func TestRedeliveryRoundAndOldWorkerCannotPauseReplacement(t *testing.T) {
	conn := setup(t)
	a := agents.Entity{UserId: 19, TokenPrefix: "agt_round", Enabled: 1}
	if err := conn.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	rotated, err := RotateSecret(a.UserId, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	configured, err := Configure(a.UserId, rotated.ConfigVersion, ConfigParams{EventsEnabled: true, EventTypes: []string{"agent.mentioned"}, WebhookEnabled: true, WebhookEndpoint: "https://example.com/old"})
	if err != nil {
		t.Fatal(err)
	}
	row, err := Test(a.UserId, 20)
	if err != nil {
		t.Fatal(err)
	}
	task, claimed, err := taskQueue.ClaimTask(row.TaskID)
	if err != nil || !claimed {
		t.Fatal(err)
	}
	var p permit
	if err := conn.Transaction(func(tx *gorm.DB) error { var err error; p, err = authorizeTx(tx, &task, row.ID); return err }); err != nil {
		t.Fatal(err)
	}
	if err := complete(&task, p, safefetch.Result{StatusCode: 404}, nil); err != nil {
		t.Fatal(err)
	}
	if err := Redeliver(a.UserId, row.ID, 20); err != nil {
		t.Fatal(err)
	}
	delivery, err := agentWebhook.GetTx(conn, "inst_test", row.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.Round != 2 || delivery.AttemptCount != 0 || delivery.TotalAttempts != 1 || delivery.Body != row.Body || delivery.EventID != row.EventID {
		t.Fatalf("replay=%#v", delivery)
	}
	task, claimed, err = taskQueue.ClaimTask(row.TaskID)
	if err != nil || !claimed {
		t.Fatal(err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { var err error; p, err = authorizeTx(tx, &task, row.ID); return err }); err != nil {
		t.Fatal(err)
	}
	replacement, err := Configure(a.UserId, configured.ConfigVersion, ConfigParams{EventsEnabled: true, EventTypes: []string{"agent.mentioned"}, WebhookEnabled: true, WebhookEndpoint: "https://example.com/new"})
	if err != nil {
		t.Fatal(err)
	}
	if err := complete(&task, p, safefetch.Result{StatusCode: 410}, nil); err != nil {
		t.Fatal(err)
	}
	current := agents.GetByUserID(a.UserId)
	if current.WebhookPausedReason != "" || current.EndpointGeneration != replacement.EndpointGeneration || current.WebhookEndpoint != "https://example.com/new" {
		t.Fatalf("old worker corrupted config=%#v", current)
	}
	if err := Redeliver(a.UserId, row.ID, 20); !errors.Is(err, ErrNotReplayable) {
		t.Fatalf("changed endpoint accepted historical delivery: %v", err)
	}
}
func TestWithdrawalClearsRetainedPayloadButKeepsInFlightAttemptDiagnosis(t *testing.T) {
	conn := setup(t)
	a := agents.Entity{UserId: 21, TokenPrefix: "agt_withdraw", Enabled: 1}
	if err := conn.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	rotated, err := RotateSecret(a.UserId, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Configure(a.UserId, rotated.ConfigVersion, ConfigParams{EventsEnabled: true, EventTypes: []string{"agent.mentioned"}, WebhookEnabled: true, WebhookEndpoint: "https://example.com/hook"})
	if err != nil {
		t.Fatal(err)
	}
	row, err := Test(a.UserId, 2)
	if err != nil {
		t.Fatal(err)
	}
	task, claimed, err := taskQueue.ClaimTask(row.TaskID)
	if err != nil || !claimed {
		t.Fatal(err)
	}
	var p permit
	if err := conn.Transaction(func(tx *gorm.DB) error { var err error; p, err = authorizeTx(tx, &task, row.ID); return err }); err != nil {
		t.Fatal(err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error {
		return agentWebhook.RedactBySourceEventsTx(tx, "inst_test", []string{row.EventID})
	}); err != nil {
		t.Fatal(err)
	}
	if err := complete(&task, p, safefetch.Result{StatusCode: 200}, nil); err != nil {
		t.Fatal(err)
	}
	rowPtr, err := agentWebhook.GetTx(conn, "inst_test", row.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if rowPtr.Body != "" || rowPtr.Status != agentWebhook.Cancelled || rowPtr.AcceptedAt == nil {
		t.Fatalf("withdrawn=%#v", rowPtr)
	}
	attempts, err := GetAttempts(a.UserId, row.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 1 || attempts[0].HTTPStatus != 200 || attempts[0].CompletedAt == nil {
		t.Fatalf("lost in-flight diagnosis=%#v", attempts)
	}
}

func TestGlobalOutboundPauseRetainsFrozenDelivery(t *testing.T) {
	conn := setup(t)
	a := agents.Entity{UserId: 19, TokenPrefix: "paused_outbound", Enabled: 1}
	if err := conn.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	secret, err := RotateSecret(a.UserId, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	configured, err := Configure(a.UserId, secret.ConfigVersion, ConfigParams{EventsEnabled: true, EventTypes: []string{"agent.mentioned"}, WebhookEnabled: true, WebhookEndpoint: "https://example.com/hook"})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("YOURTJ_AGENT_WEBHOOK_ENABLED", "false")
	now := time.Now().UTC()
	e := agentEvents.Entity{ID: "evt_paused", InstanceID: "inst_test", AgentID: a.UserId, Seq: 1, SourceIntentID: "source_paused", Type: "agent.mentioned", Reasons: []string{"mention"}, OccurredAt: now, ExpiresAt: now.Add(time.Hour)}
	if err := conn.Transaction(func(tx *gorm.DB) error { return EnqueueEventTx(tx, e, configured.EndpointGeneration) }); err != nil {
		t.Fatal(err)
	}
	var delivery agentWebhook.Delivery
	if err := conn.Where("event_id = ?", e.ID).Take(&delivery).Error; err != nil {
		t.Fatalf("paused outbound switch dropped frozen delivery: %v", err)
	}
	if delivery.TaskID == 0 || delivery.Status != agentWebhook.Pending {
		t.Fatalf("missing recoverable delivery %#v", delivery)
	}
	task, claimed, err := taskQueue.ClaimTask(delivery.TaskID)
	if err != nil || !claimed {
		t.Fatalf("claim %t %v", claimed, err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { _, err := authorizeTx(tx, &task, delivery.ID); return err }); err != nil {
		t.Fatal(err)
	}
	delayed, err := taskQueue.GetByID(task.Id)
	if err != nil || delayed.Status != taskQueue.StatusRetrying || delayed.NextRunAt == nil {
		t.Fatalf("paused sender didn't delay recoverably %#v %v", delayed, err)
	}
}

type webhookTestResolver struct{ private bool }

func (r webhookTestResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	addresses := []netip.Addr{netip.MustParseAddr("8.8.8.8")}
	if r.private {
		addresses = append(addresses, netip.MustParseAddr("127.0.0.1"))
	}
	return addresses, nil
}
func TestConfigureRejectsPrivateDNSBeforePersistingDestination(t *testing.T) {
	conn := setup(t)
	previous := sender
	sender = safefetch.New(safefetch.Config{Resolver: webhookTestResolver{private: true}})
	t.Cleanup(func() { sender = previous })
	a := agents.Entity{UserId: 20, TokenPrefix: "private_dns", Enabled: 1}
	if err := conn.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	_, err := Configure(a.UserId, 0, ConfigParams{WebhookEndpoint: "https://mixed.example/hook"})
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("private DNS destination accepted at save: %v", err)
	}
	row := agents.GetByUserID(a.UserId)
	if row.WebhookEndpoint != "" || row.ConfigVersion != 0 {
		t.Fatalf("failed target validation persisted config %#v", row)
	}
}

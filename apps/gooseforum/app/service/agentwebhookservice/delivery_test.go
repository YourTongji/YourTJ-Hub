package agentwebhookservice

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/agentinstance"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/safefetch"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
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

func configuredDelivery(t *testing.T) (*gorm.DB, agentWebhook.Delivery, taskQueue.Entity) {
	t.Helper()
	conn := setup(t)
	a := agents.Entity{UserId: 40, TokenPrefix: "delivery_failures", Enabled: 1}
	if err := conn.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	secret, err := RotateSecret(a.UserId, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Configure(a.UserId, secret.ConfigVersion, ConfigParams{EventsEnabled: true, EventTypes: []string{"agent.mentioned"}, WebhookEnabled: true, WebhookEndpoint: "https://example.com/hook"}); err != nil {
		t.Fatal(err)
	}
	row, err := Test(a.UserId, 2)
	if err != nil {
		t.Fatal(err)
	}
	task, claimed, err := taskQueue.ClaimTask(row.TaskID)
	if err != nil || !claimed {
		t.Fatalf("claim=%t err=%v", claimed, err)
	}
	return conn, row, task
}

func TestResponseBodyFailureUsesKnownHTTPOutcome(t *testing.T) {
	for _, tc := range []struct {
		name           string
		code           int
		status, reason string
	}{
		{"accepted", http.StatusOK, agentWebhook.Accepted, ""},
		{"gone", http.StatusGone, agentWebhook.Dead, "receiver_gone"},
		{"redirect", http.StatusTemporaryRedirect, agentWebhook.Dead, "receiver_rejected"},
		{"rejected", http.StatusBadRequest, agentWebhook.Dead, "receiver_rejected"},
		{"rate_limited", http.StatusTooManyRequests, agentWebhook.RetryWait, "receiver_retryable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn, row, task := configuredDelivery(t)
			var p permit
			if err := conn.Transaction(func(tx *gorm.DB) error { var err error; p, err = authorizeTx(tx, &task, row.ID); return err }); err != nil {
				t.Fatal(err)
			}
			before := time.Now()
			if err := complete(&task, p, safefetch.Result{StatusCode: tc.code, RetryAfter: "3600"}, &safefetch.FetchError{Class: safefetch.ErrorTooLarge}); err != nil {
				t.Fatal(err)
			}
			got, err := agentWebhook.GetTx(conn, "inst_test", row.ID, false)
			if err != nil || got.Status != tc.status || got.Reason != tc.reason {
				t.Fatalf("delivery=%#v err=%v", got, err)
			}
			if tc.code == http.StatusTooManyRequests && (got.NextRunAt == nil || got.NextRunAt.Before(before.Add(time.Hour))) {
				t.Fatalf("Retry-After discarded: %#v", got.NextRunAt)
			}
		})
	}
}

func TestRedeliveryClearsPreviousRoundAcceptance(t *testing.T) {
	conn, row, task := configuredDelivery(t)
	var p permit
	if err := conn.Transaction(func(tx *gorm.DB) error { var err error; p, err = authorizeTx(tx, &task, row.ID); return err }); err != nil {
		t.Fatal(err)
	}
	if err := complete(&task, p, safefetch.Result{StatusCode: http.StatusOK}, nil); err != nil {
		t.Fatal(err)
	}
	if err := Redeliver(row.AgentID, row.ID, 2); err != nil {
		t.Fatal(err)
	}
	got, err := agentWebhook.GetTx(conn, "inst_test", row.ID, false)
	if err != nil || got.Status != agentWebhook.Pending || got.AcceptedAt != nil {
		t.Fatalf("new round retains acceptance: %#v err=%v", got, err)
	}
}

func failNextDeliveryQuery(t *testing.T, conn *gorm.DB) {
	t.Helper()
	failed := false
	name := "test:delivery_query_failure"
	if err := conn.Callback().Query().Before("gorm:query").Register(name, func(tx *gorm.DB) {
		if !failed && tx.Statement.Table == "agent_webhook_deliveries" {
			failed = true
			_ = tx.AddError(errors.New("injected delivery read failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Query().Remove(name) })
}

func TestInfrastructureExhaustionMakesDeliveryReplayable(t *testing.T) {
	conn, row, task := configuredDelivery(t)
	task.RetryCount = 8
	if err := conn.Model(&taskQueue.Entity{}).Where("id = ?", task.Id).Update("retry_count", task.RetryCount).Error; err != nil {
		t.Fatal(err)
	}
	failNextDeliveryQuery(t, conn)
	if err := HandleTask(t.Context(), &task); err != nil {
		// The managed scheduler's fallback must not leave a nonterminal delivery.
		if err := taskQueue.RetryOwned(task.Id, task.LeaseToken, time.Now().Add(time.Minute), "agent task infrastructure failure", 9); err != nil {
			t.Fatal(err)
		}
	}
	got, err := agentWebhook.GetTx(conn, "inst_test", row.ID, false)
	if err != nil || got.Status != agentWebhook.Dead || got.Reason != "infrastructure_error" {
		t.Fatalf("unrecoverable infrastructure failure: %#v err=%v", got, err)
	}
	queued, err := taskQueue.GetByID(task.Id)
	if err != nil || queued.Status != taskQueue.StatusFailed || queued.RetryCount != 9 {
		t.Fatalf("task=%#v err=%v", queued, err)
	}
	if err := Redeliver(row.AgentID, row.ID, 2); err != nil {
		t.Fatalf("cannot recover exhausted delivery: %v", err)
	}
}

func TestInfrastructureFailureCannotOverwriteNewTaskLease(t *testing.T) {
	conn, row, stale := configuredDelivery(t)
	stale.RetryCount = 8
	if err := conn.Model(&taskQueue.Entity{}).Where("id = ?", stale.Id).Update("status", taskQueue.StatusPending).Error; err != nil {
		t.Fatal(err)
	}
	current, claimed, err := taskQueue.ClaimTask(stale.Id)
	if err != nil || !claimed {
		t.Fatalf("claim=%t err=%v", claimed, err)
	}
	failNextDeliveryQuery(t, conn)
	_ = HandleTask(t.Context(), &stale)
	got, err := agentWebhook.GetTx(conn, "inst_test", row.ID, false)
	if err != nil || got.Status != agentWebhook.Pending || got.Reason != "" {
		t.Fatalf("stale failure overwrote delivery: %#v err=%v", got, err)
	}
	queued, err := taskQueue.GetByID(stale.Id)
	if err != nil || queued.Status != taskQueue.StatusRunning || queued.LeaseToken != current.LeaseToken || queued.RetryCount != 0 {
		t.Fatalf("stale failure overwrote task: %#v err=%v", queued, err)
	}
}

func TestInfrastructureDiagnosticFailureRollsBackTaskTransition(t *testing.T) {
	conn, row, task := configuredDelivery(t)
	task.RetryCount = 8
	if err := conn.Model(&taskQueue.Entity{}).Where("id = ?", task.Id).Update("retry_count", task.RetryCount).Error; err != nil {
		t.Fatal(err)
	}
	failNextDeliveryQuery(t, conn)
	name := "test:delivery_diagnostic_failure"
	if err := conn.Callback().Update().Before("gorm:update").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table == "agent_webhook_deliveries" {
			_ = tx.AddError(errors.New("injected diagnostic write failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Update().Remove(name) })
	if err := HandleTask(t.Context(), &task); err == nil {
		t.Fatal("diagnostic persistence failure must remain observable")
	}
	got, err := agentWebhook.GetTx(conn, "inst_test", row.ID, false)
	if err != nil || got.Status != agentWebhook.Pending {
		t.Fatalf("partial delivery transition: %#v err=%v", got, err)
	}
	queued, err := taskQueue.GetByID(task.Id)
	if err != nil || queued.Status != taskQueue.StatusRunning || queued.RetryCount != 8 || queued.LeaseToken != task.LeaseToken {
		t.Fatalf("partial task transition: %#v err=%v", queued, err)
	}
}

func TestRotateSecretRecoversInvalidSecretWithoutDamagedOverlap(t *testing.T) {
	for _, emergency := range []bool{false, true} {
		t.Run(fmt.Sprintf("emergency=%t", emergency), func(t *testing.T) {
			conn := setup(t)
			row := agents.Entity{UserId: 32, TokenPrefix: "invalid-secret", Enabled: 1, EventsEnabled: true, WebhookEnabled: true, WebhookEndpoint: "https://example.com/hook", SecretCiphertext: "corrupt-ciphertext", WebhookPausedReason: "secret_invalid"}
			if err := conn.Create(&row).Error; err != nil {
				t.Fatal(err)
			}
			rotated, err := RotateSecret(row.UserId, 0, emergency)
			if err != nil {
				t.Fatal(err)
			}
			current := agents.GetByUserID(row.UserId)
			if current.WebhookPausedReason != "" || current.PreviousSecretCiphertext != "" || current.PreviousSecretExpiresAt != nil {
				t.Fatalf("invalid-secret recovery left unusable state: %+v", current)
			}
			decoded, err := securestore.DecryptPurpose(current.SecretCiphertext, securestore.AgentWebhookSecretPurpose)
			if err != nil || decoded != rotated.Secret {
				t.Fatalf("new secret unavailable: %v", err)
			}
			delivery, err := Test(row.UserId, 0)
			if err != nil {
				t.Fatal(err)
			}
			task, claimed, err := taskQueue.ClaimTask(delivery.TaskID)
			if err != nil || !claimed {
				t.Fatalf("claim=%t %v", claimed, err)
			}
			var permit permit
			if err := conn.Transaction(func(tx *gorm.DB) error { var err error; permit, err = authorizeTx(tx, &task, delivery.ID); return err }); err != nil {
				t.Fatal(err)
			}
			if permit.attemptID == "" || permit.secret != rotated.Secret {
				t.Fatal("fresh secret did not recover send authorization")
			}
		})
	}
}

func TestConfigureCanDisableUnchangedEndpointDuringDNSFailure(t *testing.T) {
	for _, keepInbox := range []bool{false, true} {
		t.Run(fmt.Sprintf("keepInbox=%t", keepInbox), func(t *testing.T) {
			conn := setup(t)
			row := agents.Entity{UserId: 33, TokenPrefix: "disable-dns", Enabled: 1, ConfigVersion: 1, EventsEnabled: true, EventTypes: `["agent.mentioned"]`, WebhookEnabled: true, WebhookEndpoint: "https://offline.example/hook"}
			if err := conn.Create(&row).Error; err != nil {
				t.Fatal(err)
			}
			sender = safefetch.New(safefetch.Config{Resolver: webhookFailedResolver{}})
			disabled, err := Configure(row.UserId, 1, ConfigParams{EventsEnabled: keepInbox, EventTypes: []string{"agent.mentioned"}, WebhookEnabled: false, WebhookEndpoint: row.WebhookEndpoint})
			if err != nil {
				t.Fatalf("disable depends on receiver DNS: %v", err)
			}
			if disabled.EventsEnabled != keepInbox || disabled.WebhookEnabled || disabled.ConfigVersion != 2 {
				t.Fatalf("disable not persisted: %+v", disabled)
			}
			if _, err := Configure(row.UserId, 2, ConfigParams{WebhookEndpoint: "https://changed.example/hook"}); !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("new destination skipped DNS: %v", err)
			}
			if _, err := Configure(row.UserId, 2, ConfigParams{WebhookEndpoint: row.WebhookEndpoint, EventsEnabled: true, WebhookEnabled: true, EventTypes: []string{"agent.mentioned"}}); !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("outbound reactivation skipped DNS: %v", err)
			}
		})
	}
}

func TestRotateSecretPreservesUnrelatedPause(t *testing.T) {
	conn := setup(t)
	row := agents.Entity{UserId: 34, TokenPrefix: "paused-target", Enabled: 1, WebhookPausedReason: "target_blocked"}
	if err := conn.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := RotateSecret(row.UserId, 0, true); err != nil {
		t.Fatal(err)
	}
	if current := agents.GetByUserID(row.UserId); current.WebhookPausedReason != "target_blocked" {
		t.Fatal("secret rotation cleared an unrelated safety pause")
	}
}

type webhookFailedResolver struct{}

func (webhookFailedResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return nil, errors.New("receiver DNS unavailable")
}

package agentwebhookservice

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/safefetch"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentWebhook"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agenteventservice"
	"gorm.io/gorm"
)

func TestEventExpiryRedactsInFlightDeliveryAndKeepsExpiryDiagnosis(t *testing.T) {
	conn := setup(t)
	t.Setenv("YOURTJ_AGENT_API_ENABLED", "true")
	Register()
	t.Cleanup(func() {
		agenteventservice.RegisterWithdrawalHook(nil)
		agenteventservice.RegisterExpiryHook(nil)
	})
	agent := agents.Entity{UserId: 27, TokenPrefix: "agt_expiry", Enabled: 1}
	if err := conn.Create(&agent).Error; err != nil {
		t.Fatal(err)
	}
	secret, err := RotateSecret(agent.UserId, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Configure(agent.UserId, secret.ConfigVersion, ConfigParams{EventsEnabled: true, EventTypes: []string{"agent.mentioned"}, WebhookEnabled: true, WebhookEndpoint: "https://example.com/hook"}); err != nil {
		t.Fatal(err)
	}
	delivery, err := Test(agent.UserId, 2)
	if err != nil {
		t.Fatal(err)
	}
	task, claimed, err := taskQueue.ClaimTask(delivery.TaskID)
	if err != nil || !claimed {
		t.Fatalf("claim %t %v", claimed, err)
	}
	var p permit
	if err := conn.Transaction(func(tx *gorm.DB) error {
		var err error
		p, err = authorizeTx(tx, &task, delivery.ID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	event := agentEvents.Entity{ID: delivery.EventID, InstanceID: "inst_test", AgentID: agent.UserId, Seq: 1, SourceIntentID: "expired-test-source", Type: "agent.mentioned", ActorID: 900, TopicID: 901, PostID: 902, Reasons: []string{"mention"}, OccurredAt: time.Now().UTC().Add(-2 * time.Hour), ExpiresAt: time.Now().UTC().Add(-time.Hour)}
	if err := conn.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	got, err := agenteventservice.Get(agent.UserId, event.ID)
	if err != nil || got.State != "expired" || got.Data != nil {
		t.Fatalf("expiry read %#v, %v", got, err)
	}
	// Completion of an already authorized HTTP attempt retains its response
	// diagnosis, but cannot restore the retained copy or replace expiry with withdrawal.
	if err := complete(&task, p, safefetch.Result{StatusCode: 200}, nil); err != nil {
		t.Fatal(err)
	}
	stored, err := agentWebhook.GetTx(conn, event.InstanceID, delivery.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Body != "" || stored.Status != agentWebhook.Cancelled || stored.Reason != "expired" || stored.AcceptedAt == nil {
		t.Fatalf("expiry completion restored or misclassified the copy: %#v", stored)
	}
	attempts, err := GetAttempts(agent.UserId, delivery.ID)
	if err != nil || len(attempts) != 1 || attempts[0].HTTPStatus != 200 || attempts[0].CompletedAt == nil {
		t.Fatalf("expiry lost in-flight HTTP diagnosis: %#v, %v", attempts, err)
	}
}

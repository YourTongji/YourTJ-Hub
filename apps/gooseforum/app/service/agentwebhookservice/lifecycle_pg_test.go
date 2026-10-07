package agentwebhookservice

import (
	"context"
	"sync"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentWebhook"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agenteventservice"
	"gorm.io/gorm"
)

func TestPostgreSQLStaleExpirySelectionPreservesPermanentWithdrawal(t *testing.T) {
	pg := postgresFixture(t)
	defaultConn := db.Connect()
	original := *defaultConn
	*defaultConn = *pg
	t.Cleanup(func() { *defaultConn = original })
	conn := setup(t)
	Register()
	t.Cleanup(func() {
		agenteventservice.RegisterWithdrawalHook(nil)
		agenteventservice.RegisterExpiryHook(nil)
	})
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}, &agentEvents.Intent{}, &agentEvents.ReplayState{}); err != nil {
		t.Fatal(err)
	}
	agent := agents.Entity{UserId: 27, TokenPrefix: "agt_expiry_race", Enabled: 1}
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
	post := posts.Entity{Id: 902, TopicId: 901, PostNo: 1, UserId: 900, Content: "public source"}
	topic := topics.Entity{Id: post.TopicId, UserId: post.UserId, FirstPostId: post.Id, Status: 1}
	event := agentEvents.Entity{ID: delivery.EventID, InstanceID: "inst_test", AgentID: agent.UserId, Seq: 1, SourceIntentID: "stale-expiry-source", Type: "agent.mentioned", ActorID: post.UserId, TopicID: topic.Id, PostID: post.Id, Reasons: []string{"mention"}, OccurredAt: time.Now().UTC().Add(-2 * time.Hour), ExpiresAt: time.Now().UTC().Add(-time.Hour)}
	for _, row := range []any{&topic, &post, &event} {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	selected, resume := make(chan struct{}), make(chan struct{})
	type withdrawalKey struct{}
	var once sync.Once
	if err := conn.Callback().Query().After("gorm:query").Register("test:stale-expiry-selection", func(tx *gorm.DB) {
		if tx.Statement.Table != event.TableName() || tx.Statement.Context.Value(withdrawalKey{}) == true {
			return
		}
		if _, batch := tx.Statement.Dest.(*[]agentEvents.Entity); !batch {
			return
		}
		once.Do(func() {
			close(selected)
			select {
			case <-resume:
			case <-ctx.Done():
			}
		})
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Query().Remove("test:stale-expiry-selection") })
	result := make(chan error, 1)
	go func() { result <- agenteventservice.Cleanup() }()
	select {
	case <-selected:
	case <-ctx.Done():
		close(resume)
		t.Fatal(ctx.Err())
	}
	withdrawErr := conn.WithContext(context.WithValue(ctx, withdrawalKey{}, true)).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&post).Error; err != nil {
			return err
		}
		return agenteventservice.WithdrawContentTx(tx, post.TopicId, post.Id)
	})
	close(resume)
	cleanupErr := <-result
	if withdrawErr != nil || cleanupErr != nil {
		t.Fatalf("withdraw=%v, cleanup=%v", withdrawErr, cleanupErr)
	}
	stored, err := agentWebhook.GetTx(conn, event.InstanceID, delivery.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Body != "" || stored.Status != agentWebhook.Cancelled || stored.Reason != "withdrawn" {
		t.Fatalf("stale expiry selection overwrote permanent withdrawal: %#v", stored)
	}
	if err := conn.Where("id = ?", event.ID).Take(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.WithdrawnAt == nil || event.ActorID != 0 || event.PostID != 0 || event.TopicID != 0 {
		t.Fatalf("cleanup erased withdrawal or retained references: %#v", event)
	}
}

package agentwriteservice

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentWrites"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentcommentservice"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func writeFixture(t *testing.T) (*gorm.DB, context.Context) {
	t.Helper()
	t.Setenv("YOURTJ_AGENT_INSTANCE_ID", "write-policy")
	t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "epoch")
	t.Setenv("YOURTJ_AGENT_API_ENABLED", "true")
	t.Setenv("YOURTJ_AGENT_PRODUCER_ENABLED", "true")
	conn, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := conn.DB()
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err := conn.AutoMigrate(&users.EntityComplete{}, &users.BlockEntity{}, &agents.Entity{}, &topics.Entity{}, &posts.Entity{}, &agentWrites.Entry{}, &agentEvents.Entity{}, &pageConfig.Entity{}); err != nil {
		t.Fatal(err)
	}
	for _, row := range []any{
		&users.EntityComplete{Id: 1, Username: "policybot", ActorType: users.ActorTypeBot},
		&agents.Entity{UserId: 1, TokenPrefix: "policy", TokenHash: "hash", Enabled: agents.StatusEnabled},
		&topics.Entity{Id: 10, UserId: 1, Status: 1},
	} {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	return conn, WithOptions(context.Background(), Options{Operation: "post", TargetID: 10, Key: "reply", Digest: "one", CredentialHash: "hash"})
}

func TestBeginTxRechecksCommentPolicyAfterPreflight(t *testing.T) {
	for _, scope := range []string{"topic", "global"} {
		t.Run(scope, func(t *testing.T) {
			conn, ctx := writeFixture(t)
			// A request has passed controller preflight; an administrator commits a ban
			// while moderation is still in flight, before the content transaction begins.
			if scope == "topic" {
				if err := conn.Model(&topics.Entity{}).Where("id = ?", 10).Update("agent_comment_disabled", true).Error; err != nil {
					t.Fatal(err)
				}
			} else {
				if err := conn.Create(&pageConfig.Entity{PageType: pageConfig.AgentCommentPolicy, Config: `{"allowAgentComments":false}`}).Error; err != nil {
					t.Fatal(err)
				}
			}
			err := conn.Transaction(func(tx *gorm.DB) error { _, _, err := BeginTx(ctx, tx, 1); return err })
			if !errors.Is(err, agentcommentservice.ErrAgentCommentDisabled) {
				t.Fatalf("new write after %s policy ban: %v", scope, err)
			}
			var count int64
			conn.Model(&agentWrites.Entry{}).Count(&count)
			if count != 0 {
				t.Fatal("refused write consumed its idempotency key")
			}
		})
	}
}

func TestCommittedReplayIgnoresLaterCommentBan(t *testing.T) {
	conn, ctx := writeFixture(t)
	if err := conn.Transaction(func(tx *gorm.DB) error {
		reservation, _, err := BeginTx(ctx, tx, 1)
		if err != nil {
			return err
		}
		return agentWrites.CompleteTx(tx, reservation.Entry, 10, 20)
	}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = 10").Update("agent_comment_disabled", true).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error {
		_, replay, err := BeginTx(ctx, tx, 1)
		if !errors.Is(err, ErrReplay) || replay == nil || replay.PostID != 20 {
			return fmt.Errorf("committed replay changed: %#v %w", replay, err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestFinishTxPreservesEachPendingReplyDepth(t *testing.T) {
	conn, _ := writeFixture(t)
	now := time.Now().Add(-time.Minute)
	if err := conn.Model(&agents.Entity{}).Where("user_id = 1").Updates(map[string]any{"events_enabled": true, "events_enabled_at": now, "subscription_generation": 1}).Error; err != nil {
		t.Fatal(err)
	}
	for _, row := range []any{
		&users.EntityComplete{Id: 2, Username: "sourcebot", ActorType: users.ActorTypeBot},
		&posts.Entity{Id: 100, TopicId: 10, PostNo: 1, UserId: 2, Content: "source", AgentEventDepth: 3},
		&agentEvents.Entity{ID: "source", InstanceID: "write-policy", AgentID: 1, Seq: 1, SourceIntentID: "intent", SubscriptionGeneration: 1, Type: "forum.topic_created", TopicID: 10, PostID: 100, PostNo: 1, ActorID: 2, ActorType: "bot", Reasons: []string{"topic_created"}, ExpiresAt: time.Now().Add(time.Hour)},
	} {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	for n := uint64(2); n <= 3; n++ {
		ctx := WithOptions(context.Background(), Options{Operation: "post", TargetID: 10, Key: fmt.Sprintf("reply%d", n), Digest: "one", CredentialHash: "hash", SourceEventID: "source"})
		err := conn.Transaction(func(tx *gorm.DB) error {
			reservation, _, err := BeginTx(ctx, tx, 1)
			if err != nil {
				return err
			}
			p := posts.Entity{Id: 100 + n, TopicId: 10, PostNo: n, UserId: 1, Content: "pending reply", ProcessStatus: posts.ProcessStatusPending}
			if err := tx.Create(&p).Error; err != nil {
				return err
			}
			return FinishTx(ctx, tx, 1, reservation, 10, p.Id)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	var reply posts.Entity
	if err := conn.First(&reply, 102).Error; err != nil {
		t.Fatal(err)
	}
	if reply.AgentEventDepth != 4 {
		t.Fatalf("first pending reply lost depth after second response: %d", reply.AgentEventDepth)
	}
	var event agentEvents.Entity
	if err := conn.Where("id = ?", "source").Take(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.ResultingPostID != 103 {
		t.Fatalf("latest result not replaced: %#v", event)
	}
}

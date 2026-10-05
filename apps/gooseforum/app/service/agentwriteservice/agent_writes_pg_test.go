package agentwriteservice

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentWrites"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPostgreSQLBroadcastWritersUseOneParticipantLockOrder(t *testing.T) {
	for _, linked := range []bool{false, true} {
		t.Run(fmt.Sprintf("sourceLinked=%t", linked), func(t *testing.T) { testPostgreSQLBroadcastWriters(t, linked) })
	}
}

func testPostgreSQLBroadcastWriters(t *testing.T, linked bool) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("PostgreSQL DSN not configured")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("agent_broadcast_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec("DROP SCHEMA " + schema + " CASCADE"); pool, _ := admin.DB(); _ = pool.Close() })
	if parsed, err := url.Parse(dsn); err == nil && (parsed.Scheme == "postgres" || parsed.Scheme == "postgresql") {
		q := parsed.Query()
		q.Set("search_path", schema)
		parsed.RawQuery = q.Encode()
		dsn = parsed.String()
	} else {
		dsn += " search_path=" + schema
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool, _ := conn.DB(); _ = pool.Close() })
	if err := conn.AutoMigrate(&users.EntityComplete{}, &users.BlockEntity{}, &agents.Entity{}, &topics.Entity{}, &posts.Entity{}, &postRevisions.Entity{}, &agentWrites.Entry{}, &agentEvents.Entity{}, &agentEvents.Publication{}, &agentEvents.Intent{}, &pageConfig.Entity{}, &taskQueue.Entity{}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("YOURTJ_AGENT_INSTANCE_ID", "broadcast-pg")
	t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "epoch")
	t.Setenv("YOURTJ_AGENT_API_ENABLED", "true")
	t.Setenv("YOURTJ_AGENT_PRODUCER_ENABLED", "true")
	now := time.Now().Add(-time.Hour)
	for id := uint64(1); id <= 2; id++ {
		for _, row := range []any{&users.EntityComplete{Id: id, Username: fmt.Sprintf("bot%d", id), ActorType: users.ActorTypeBot}, &agents.Entity{UserId: id, TokenPrefix: fmt.Sprintf("token%d", id), TokenHash: "hash", Enabled: 1, EventsEnabled: true, EventsEnabledAt: &now, SubscriptionGeneration: 1, EventTypes: `["forum.post_created"]`}, &topics.Entity{Id: id * 10, UserId: id, Status: 1}} {
			if err := conn.Create(row).Error; err != nil {
				t.Fatal(err)
			}
		}
	}

	if linked {
		for id := uint64(1); id <= 2; id++ {
			human := users.EntityComplete{Id: id + 10, Username: fmt.Sprintf("human%d", id)}
			source := posts.Entity{Id: 100 + id, TopicId: id * 10, PostNo: 1, UserId: human.Id, Content: "human source"}
			event := agentEvents.Entity{ID: fmt.Sprintf("source%d", id), InstanceID: "broadcast-pg", AgentID: id, Seq: 1, SourceIntentID: fmt.Sprintf("intent%d", id), SubscriptionGeneration: 1, Type: "forum.topic_created", TopicID: id * 10, PostID: source.Id, PostNo: 1, ActorID: human.Id, ActorType: "human", Reasons: []string{"topic_created"}, ExpiresAt: time.Now().Add(time.Hour)}
			for _, row := range []any{&human, &source, &event} {
				if err := conn.Create(row).Error; err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	// The old path reaches its first subscriber query only after locking its own
	// user and Agent. Rendezvous here deterministically exposes the inversion.
	// The corrected path queries candidates before taking any participant lock.
	var arrived atomic.Int32
	release := make(chan struct{})
	if err := conn.Callback().Query().After("gorm:query").Register("test:broadcast-rendezvous", func(tx *gorm.DB) {
		if tx.Statement.Table == "agents" && strings.Contains(tx.Statement.SQL.String(), "events_enabled") {
			if arrived.Add(1) == 2 {
				close(release)
			}
			select {
			case <-release:
			case <-time.After(5 * time.Second):
				tx.AddError(fmt.Errorf("broadcast lock rendezvous timed out"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer conn.Callback().Query().Remove("test:broadcast-rendezvous")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	done := make(chan error, 2)
	for id := uint64(1); id <= 2; id++ {
		go func(id uint64) {
			options := Options{Operation: "post", TargetID: id * 10, Key: "reply", Digest: "one", CredentialHash: "hash"}
			if linked {
				options.SourceEventID = fmt.Sprintf("source%d", id)
			}
			request := WithOptions(ctx, options)
			done <- conn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				reservation, _, err := BeginTx(request, tx, id)
				if err != nil {
					return err
				}
				post := posts.Entity{TopicId: id * 10, UserId: id, PostNo: 2, Content: "concurrent public reply"}
				if err := tx.Create(&post).Error; err != nil {
					return err
				}
				if err := tx.Create(&postRevisions.Entity{PostId: post.Id, Version: 1, EditorId: id, Content: post.Content}).Error; err != nil {
					return err
				}
				return FinishTx(request, tx, id, reservation, post.TopicId, post.Id)
			})
		}(id)
	}
	for n := 0; n < 2; n++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent broadcast write: %v", err)
		}
	}
	var written []posts.Entity
	if err := conn.Where("post_no = ?", 2).Find(&written).Error; err != nil {
		t.Fatal(err)
	}
	if len(written) != 2 {
		t.Fatalf("committed replies: %d", len(written))
	}
	for _, p := range written {
		want := 0
		if linked {
			want = 1
		}
		if p.AgentEventDepth != want {
			t.Fatalf("persisted depth %d want %d", p.AgentEventDepth, want)
		}
	}

}

package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/migration"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupBotMaintenancePostgreSQL(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("PostgreSQL DSN not configured")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("bot_maintenance_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec("DROP SCHEMA " + schema + " CASCADE"); pool, _ := admin.DB(); _ = pool.Close() })
	parsed, err := url.Parse(dsn)
	if err == nil && (parsed.Scheme == "postgres" || parsed.Scheme == "postgresql") {
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
	if err := conn.AutoMigrate(migration.SchemaModels()...); err != nil {
		t.Fatal(err)
	}
	defaultConn := db.Connect()
	previous := *defaultConn
	*defaultConn = *conn
	t.Cleanup(func() { *defaultConn = previous })
	return conn
}

func TestPostgreSQLBotMaintenancePreservesRunnerRepliesAndSerializesEditors(t *testing.T) {
	conn := setupBotMaintenancePostgreSQL(t)
	const id = uint64(1173600)
	setupBotMaintenance(t, id)
	t.Setenv("YOURTJ_AGENT_INSTANCE_ID", "maintenance-pg")
	t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "epoch")
	t.Setenv("YOURTJ_AGENT_API_ENABLED", "true")
	t.Setenv("YOURTJ_AGENT_PRODUCER_ENABLED", "true")
	now := time.Now().Add(-time.Hour)
	subscriber := users.EntityComplete{Id: id - 10, Username: "maintenance-subscriber", ActorType: users.ActorTypeBot, IsActivated: users.ActivationSuccess}
	if err := conn.Create(&subscriber).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&agents.Entity{UserId: subscriber.Id, TokenPrefix: "subscriber-pg", Enabled: 1, EventsEnabled: true, EventsEnabledAt: &now, EventTypes: `["forum.post_created","forum.topic_created"]`}).Error; err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	const replies = 8
	for i := 0; i < replies; i++ {
		start := make(chan struct{})
		results := make(chan component.Response, 2)
		version := uint64(i + 1)
		go func() {
			<-start
			results <- AgentEditTopic(component.BetterRequest[BotEditTopicReq]{UserId: id, Context: ctx, Params: BotEditTopicReq{TopicId: id + 5, Title: fmt.Sprintf("Corrected bot title %d", version), Content: fmt.Sprintf("Corrected bot content version %d with enough words", version), CategoryId: []uint64{id + 4}, ExpectedRevisionNo: maintenanceVersion(version)}})
		}()
		go func() {
			<-start
			results <- AgentCreatePost(component.BetterRequest[AgentCreatePostReq]{UserId: id + 3, Context: ctx, Params: AgentCreatePostReq{TopicId: id + 5, Content: fmt.Sprintf("Runner reply %d with enough words", version), IdempotencyKey: fmt.Sprintf("runner-%d", version)}})
		}()
		close(start)
		for n := 0; n < 2; n++ {
			select {
			case result := <-results:
				if result.Data.Code != component.SUCCESS {
					t.Fatalf("concurrent result: %+v", result)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		}
	}
	topic := topics.Get(id + 5)
	if topic.PostCount != replies+1 || topic.PostSeq != replies+1 {
		t.Fatalf("runner statistics reverted: %+v", topic)
	}
	start := make(chan struct{})
	results := make(chan component.Response, 2)
	for _, actor := range []uint64{id, id + 1} {
		go func(actor uint64) { <-start; results <- editBotTopic(actor, id, replies+1) }(actor)
	}
	close(start)
	success, conflict := 0, 0
	for n := 0; n < 2; n++ {
		select {
		case result := <-results:
			if result.Data.Code == component.SUCCESS {
				success++
			} else if result.Code == http.StatusConflict {
				conflict++
			} else {
				t.Fatalf("editor result: %+v", result)
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("same-version results success=%d conflict=%d", success, conflict)
	}
	revisions := postRevisions.ListByPostId(id + 6)
	if len(revisions) != replies+2 {
		t.Fatalf("lost/duplicated versions: %d", len(revisions))
	}
	for i, r := range revisions {
		if r.Version != uint64(i+1) {
			t.Fatalf("version %d at %d", r.Version, i)
		}
	}
	var persistedReplies int64
	if err := conn.Model(&posts.Entity{}).Where("topic_id = ? AND post_no > 1", id+5).Count(&persistedReplies).Error; err != nil || persistedReplies != replies {
		t.Fatalf("reply count=%d err=%v", persistedReplies, err)
	}
}

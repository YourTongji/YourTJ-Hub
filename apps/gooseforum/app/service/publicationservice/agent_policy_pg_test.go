package publicationservice_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/migration"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentcommentservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agenteventservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// An approval and an Agent write on different topics still share the bot and
// global policy rows. They must acquire those rows in the same order.
func TestPostgreSQLAgentApprovalAndWriteSharePolicyLockOrder(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("PostgreSQL DSN not configured")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("agent_approval_%d", time.Now().UnixNano())
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
	if err := conn.AutoMigrate(migration.SchemaModels()...); err != nil {
		t.Fatal(err)
	}
	// Review uses the default connection. Swap only this serial test's handle,
	// restoring the original before the isolated PostgreSQL pool is closed.
	defaultConn := db.Connect()
	original := *defaultConn
	*defaultConn = *conn
	t.Cleanup(func() { *defaultConn = original })
	t.Setenv("YOURTJ_AGENT_INSTANCE_ID", "approval-pg")
	t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "epoch")
	t.Setenv("YOURTJ_AGENT_API_ENABLED", "true")
	t.Setenv("YOURTJ_AGENT_PRODUCER_ENABLED", "true")
	now := time.Now().Add(-time.Hour)
	rows := []any{
		&users.EntityComplete{Id: 1, Username: "approval-bot", ActorType: users.ActorTypeBot},
		&users.EntityComplete{Id: 2, Username: "approval-human"},
		&agents.Entity{UserId: 1, TokenPrefix: "approval-token", TokenHash: "hash", Enabled: 1, EventsEnabled: true, EventsEnabledAt: &now, EventTypes: `["forum.post_created"]`},
		&pageConfig.Entity{PageType: pageConfig.AgentCommentPolicy, Config: `{"allowAgentComments":true}`},
		&topics.Entity{Id: 10, UserId: 2, Status: 1, FirstPostId: 100, PostCount: 2, PostSeq: 2},
		&topics.Entity{Id: 20, UserId: 2, Status: 1, FirstPostId: 200, PostCount: 1, PostSeq: 1},
		&posts.Entity{Id: 100, TopicId: 10, UserId: 2, PostNo: 1, Content: "first"},
		&posts.Entity{Id: 200, TopicId: 20, UserId: 2, PostNo: 1, Content: "first"},
		&posts.Entity{Id: 201, TopicId: 10, UserId: 1, PostNo: 2, Content: "pending reply", ProcessStatus: posts.ProcessStatusPending, LatestRevisionId: 1},
		&postRevisions.Entity{Id: 1, PostId: 201, Version: 1, EditorId: 1, Content: "pending reply", ProcessStatus: posts.ProcessStatusPending},
	}
	for _, row := range rows {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	type approvalContextKey struct{}
	participantLock := make(chan struct{})
	var once sync.Once
	if err := conn.Callback().Query().Before("gorm:query").Register("test:approval-participant-lock", func(tx *gorm.DB) {
		if tx.Statement.Table == "users" && tx.Statement.Context.Value(approvalContextKey{}) == true {
			if _, locked := tx.Statement.Clauses["FOR"]; locked {
				once.Do(func() { close(participantLock) })
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Query().Remove("test:approval-participant-lock") })
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	writerLocked := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		results <- conn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if _, err := agenteventservice.PrepareWriteCaptureTx(tx, 1, 20, ""); err != nil {
				return err
			}
			close(writerLocked)
			select {
			case <-participantLock:
			case <-ctx.Done():
				return ctx.Err()
			}
			return agentcommentservice.CheckNewCommentTx(tx, 20)
		})
	}()
	select {
	case <-writerLocked:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	go func() {
		results <- publicationservice.Review(context.WithValue(ctx, approvalContextKey{}, true), 1, moderationDecision.ActionAllow, "", 2)
	}()
	for i := 0; i < 2; i++ {
		if err := <-results; err != nil {
			t.Errorf("approval/write lock order: %v", err)
		}
	}
}

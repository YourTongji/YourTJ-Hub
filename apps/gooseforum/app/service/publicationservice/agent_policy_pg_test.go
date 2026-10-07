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
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentcommentservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agenteventservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/postservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/publicationservice"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// An approval and an Agent write on different topics still share the bot and
// global policy rows. They must acquire those rows in the same order.
func TestPostgreSQLAgentApprovalAndWriteSharePolicyLockOrder(t *testing.T) {
	conn := setupAgentPolicyPostgreSQL(t)
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

func setupAgentPolicyPostgreSQL(t *testing.T) *gorm.DB {
	t.Helper()
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
	return conn
}

// A source event may refer to the owner's earlier member post while that owner
// submits a new persona reply. Both paths must lock content before the owner.
func TestPostgreSQLPersonaReplyAndAgentSourceShareLockOrder(t *testing.T) {
	for _, path := range []string{"direct", "pending", "pending edit", "review", "direct edit"} {
		t.Run(path, func(t *testing.T) {
			conn := setupAgentPolicyPostgreSQL(t)
			t.Setenv("YOURTJ_AGENT_INSTANCE_ID", "persona-pg")
			t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "epoch")
			t.Setenv("YOURTJ_AGENT_PRODUCER_ENABLED", "true")
			rows := []any{
				&users.EntityComplete{Id: 1, Username: "persona-bot", ActorType: users.ActorTypeBot},
				&users.EntityComplete{Id: 2, Username: "persona-owner"},
				&agents.Entity{UserId: 1, TokenPrefix: "persona-token", TokenHash: "hash", Enabled: 1},
				&identity.Persona{UID: "persona-owner", Name: "Test persona", AvatarSeed: "seed"},
				&identity.Binding{OwnerID: 2, PersonaUID: "persona-owner"},
				&topics.Entity{Id: 10, UserId: 2, Status: 1, FirstPostId: 100, PostCount: 1, PostSeq: 1},
				&posts.Entity{Id: 100, TopicId: 10, UserId: 2, PostNo: 1, Content: "Earlier public member post"},
				&agentEvents.Entity{ID: "source-member-event", InstanceID: "persona-pg", AgentID: 1, ActorID: 2, TopicID: 10, PostID: 100},
			}
			if path == "pending edit" || path == "review" || path == "direct edit" {
				rows = append(rows,
					&posts.Entity{Id: 201, TopicId: 10, UserId: 2, PostNo: 2, PersonaUID: "persona-owner", IsAnonymous: true, Content: "An earlier persona reply", LatestRevisionId: 1, ProcessStatus: posts.ProcessStatusPending},
					&postRevisions.Entity{Id: 1, PostId: 201, Version: 1, EditorId: 2, Content: "A pending persona revision", ProcessStatus: posts.ProcessStatusPending},
				)
			}
			for _, row := range rows {
				if err := conn.Create(row).Error; err != nil {
					t.Fatal(err)
				}
			}
			// Explicit fixture IDs do not advance PostgreSQL's identity sequence.
			if path == "pending edit" {
				if err := conn.Exec("SELECT setval(pg_get_serial_sequence('post_revisions', 'id'), 1, true)").Error; err != nil {
					t.Fatal(err)
				}
			}
			type sourceContextKey struct{}
			ownerBeforeTopic := make(chan struct{})
			agentBeforeUsers := make(chan struct{})
			var writerOnce, agentOnce sync.Once
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			waitBeforeTopic := func(tx *gorm.DB) {
				if tx.Statement.Table == "topics" && tx.Statement.Context.Value(sourceContextKey{}) != true {
					writerOnce.Do(func() {
						close(ownerBeforeTopic)
						select {
						case <-agentBeforeUsers:
						case <-ctx.Done():
						}
					})
				}
			}
			if err := conn.Callback().Update().Before("gorm:update").Register("test:persona-topic-lock", waitBeforeTopic); err != nil {
				t.Fatal(err)
			}
			if err := conn.Callback().Query().Before("gorm:query").Register("test:persona-source-lock", func(tx *gorm.DB) {
				if _, locked := tx.Statement.Clauses["FOR"]; !locked {
					return
				}
				if tx.Statement.Table == "users" && tx.Statement.Context.Value(sourceContextKey{}) == true {
					agentOnce.Do(func() { close(agentBeforeUsers) })
				}
				waitBeforeTopic(tx)
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = conn.Callback().Update().Remove("test:persona-topic-lock")
				_ = conn.Callback().Query().Remove("test:persona-source-lock")
			})
			results := make(chan error, 2)
			go func() {
				topic := topics.Entity{Id: 10, UserId: 2, Status: 1, FirstPostId: 100}
				reply := posts.Entity{TopicId: 10, UserId: 2, PersonaUID: "persona-owner", IsAnonymous: true, Content: "A new anonymous persona reply"}
				switch path {
				case "direct":
					results <- postservice.CreateTopicPost(&reply, topic)
				case "pending":
					results <- publicationservice.Submit(ctx, &topic, &reply)
				case "pending edit":
					reply.Id, reply.PostNo = 201, 2
					results <- publicationservice.Submit(ctx, &topic, &reply)
				case "review":
					results <- publicationservice.Review(ctx, 1, moderationDecision.ActionAllow, "", 2)
				case "direct edit":
					reply.Id, reply.PostNo = 201, 2
					results <- conn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
						if err := postservice.LockPersonaContentTx(tx, &reply); err != nil {
							return err
						}
						if err := posts.SaveTx(tx, &reply); err != nil {
							return err
						}
						return agenteventservice.CapturePublicTx(tx, &reply)
					})
				}
			}()
			select {
			case <-ownerBeforeTopic:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			go func() {
				results <- conn.WithContext(context.WithValue(ctx, sourceContextKey{}, true)).Transaction(func(tx *gorm.DB) error {
					_, err := agenteventservice.PrepareWriteCaptureTx(tx, 1, 10, "source-member-event")
					return err
				})
			}()
			for i := 0; i < 2; i++ {
				if err := <-results; err != nil {
					t.Errorf("persona/source lock order: %v", err)
				}
			}
		})
	}
}

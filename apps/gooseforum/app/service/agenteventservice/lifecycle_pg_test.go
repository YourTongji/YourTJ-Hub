package agenteventservice

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func lifecyclePostgreSQL(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin := stdlib.OpenDB(*cfg)
	schema := fmt.Sprintf("agent_lifecycle_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE"); _ = admin.Close() })
	cfg.RuntimeParams["search_path"] = schema
	cfg.RuntimeParams["statement_timeout"] = "10000"
	pool := stdlib.OpenDB(*cfg)
	pool.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = pool.Close() })
	conn, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defaultConn := db.Connect()
	original := *defaultConn
	*defaultConn = *conn
	t.Cleanup(func() { *defaultConn = original })
	return conn
}

func TestPostgreSQLCancelledIntentCannotResurrectAfterDeleteRestore(t *testing.T) {
	for _, missingRevision := range []bool{false, true} {
		t.Run(map[bool]string{false: "materialization", true: "failure_diagnostic"}[missingRevision], func(t *testing.T) {
			lifecyclePostgreSQL(t)
			conn, post, agent := setup(t)
			if err := agents.UpdateColumns(conn, agent.UserId, map[string]any{"webhook_enabled": true, "webhook_endpoint": "https://example.com/hook", "endpoint_generation": 1}); err != nil {
				t.Fatal(err)
			}
			deliveries := 0
			RegisterDeliveryHook(func(*gorm.DB, agentEvents.Entity, uint64) error { deliveries++; return nil })
			t.Cleanup(func() { RegisterDeliveryHook(nil) })
			if err := conn.Transaction(func(tx *gorm.DB) error { return CapturePublicTx(tx, &post) }); err != nil {
				t.Fatal(err)
			}
			intent, task := pendingLifecycleTask(t, conn, post.Id)
			read, resume := make(chan struct{}), make(chan struct{})
			type workerKey struct{}
			var once sync.Once
			if err := conn.Callback().Query().After("gorm:query").Register("test:stale-pending-intent", func(tx *gorm.DB) {
				if tx.Statement.Table == intent.TableName() && tx.Statement.Context.Value(workerKey{}) == true {
					once.Do(func() {
						close(read)
						<-resume
					})
				}
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = conn.Callback().Query().Remove("test:stale-pending-intent") })
			ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), workerKey{}, true), 12*time.Second)
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- HandleTask(ctx, &task) }()
			select {
			case <-read:
			case <-ctx.Done():
				close(resume)
				t.Fatal(ctx.Err())
			}
			// The actual lifecycle transaction cancels a pending occurrence even
			// when it has not materialized an event or delivery yet.
			deleteErr := conn.Transaction(func(tx *gorm.DB) error {
				if err := tx.Delete(&post).Error; err != nil {
					return err
				}
				return WithdrawContentTx(tx, post.TopicId, post.Id)
			})
			if deleteErr == nil {
				deleteErr = conn.Unscoped().Model(&post).UpdateColumn("deleted_at", nil).Error
			}
			if deleteErr == nil && missingRevision {
				deleteErr = conn.Where("post_id = ?", post.Id).Delete(&postRevisions.Entity{}).Error
			}
			close(resume)
			workerErr := <-result
			if deleteErr != nil {
				t.Fatal(deleteErr)
			}
			if workerErr != nil && !errors.Is(workerErr, gorm.ErrRecordNotFound) {
				t.Fatalf("worker: %v", workerErr)
			}
			var stored agentEvents.Intent
			if err := conn.Where("id = ?", intent.ID).Take(&stored).Error; err != nil {
				t.Fatal(err)
			}
			if stored.Status != "cancelled" || stored.ActorID != 0 || stored.LastError != "source_withdrawn" {
				t.Errorf("stale worker resurrected cancelled intent: %#v", stored)
			}
			var count int64
			if err := conn.Model(&agentEvents.Entity{}).Where("source_intent_id = ?", intent.ID).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 || deliveries != 0 {
				t.Errorf("cancelled occurrence emitted %d events and %d deliveries", count, deliveries)
			}
		})
	}
}

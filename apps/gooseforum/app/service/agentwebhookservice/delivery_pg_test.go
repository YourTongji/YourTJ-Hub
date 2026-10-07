package agentwebhookservice

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentWebhook"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func postgresFixture(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin := stdlib.OpenDB(*config)
	schema := fmt.Sprintf("agent_webhook_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE"); _ = admin.Close() })
	config.RuntimeParams["search_path"] = schema
	config.RuntimeParams["statement_timeout"] = "10000"
	pool := stdlib.OpenDB(*config)
	pool.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = pool.Close() })
	conn, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&agents.Entity{}, &agentWebhook.Delivery{}, &agentWebhook.Attempt{}, &taskQueue.Entity{}); err != nil {
		t.Fatal(err)
	}
	return conn
}
func TestPostgreSQLAgentWebhookDoubleClaimDueTimeAndFencing(t *testing.T) {
	conn := postgresFixture(t)
	task := taskQueue.Entity{Type: TaskType, ScheduleGroup: "12", TaskJson: `{"instanceId":"inst_pg","deliveryId":1}`}
	if err := taskQueue.CreateTx(conn, &task); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	type claim struct {
		row   taskQueue.Entity
		owned bool
		err   error
	}
	results := make(chan claim, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			row, owned, err := taskQueue.ClaimTaskWithDB(conn, task.Id)
			results <- claim{row, owned, err}
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	var claimed taskQueue.Entity
	count := 0
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		if result.owned {
			count++
			claimed = result.row
		}
	}
	if count != 1 {
		t.Fatalf("double claim winners=%d", count)
	}
	due := time.Now().Add(time.Hour)
	if err := conn.Transaction(func(tx *gorm.DB) error {
		ok, err := taskQueue.TransitionOwnedTx(tx, task.Id, claimed.LeaseToken, taskQueue.StatusRetrying, &due, "receiver_retryable")
		if err != nil {
			return err
		}
		if !ok {
			t.Fatal("lost current owner")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, owned, err := taskQueue.ClaimTaskWithDB(conn, task.Id); err != nil || owned {
		t.Fatalf("future job claimed=%v err=%v", owned, err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error {
		ok, err := taskQueue.TransitionOwnedTx(tx, task.Id, claimed.LeaseToken, taskQueue.StatusSuccess, nil, "")
		if ok {
			t.Fatal("old worker overwrote delayed task")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	// Scheduling recovery changes due time; a new claim changes fencing identity.
	if err := conn.Model(&taskQueue.Entity{}).Where("id = ?", task.Id).Update("next_run_at", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	newer, owned, err := taskQueue.ClaimTaskWithDB(conn, task.Id)
	if err != nil || !owned {
		t.Fatalf("reclaim=%v err=%v", owned, err)
	}
	if newer.LeaseToken == claimed.LeaseToken {
		t.Fatal("fencing token reused")
	}
	if err := conn.Transaction(func(tx *gorm.DB) error {
		ok, err := taskQueue.TransitionOwnedTx(tx, task.Id, claimed.LeaseToken, taskQueue.StatusSuccess, nil, "")
		if ok {
			t.Fatal("old lease overwrote new owner")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func TestPostgreSQLWebhookOutboxUniqueAcrossWorkers(t *testing.T) {
	conn := postgresFixture(t)
	now := time.Now()
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			results <- conn.Transaction(func(tx *gorm.DB) error {
				row := agentWebhook.Delivery{InstanceID: "inst_pg", EventID: "evt_same_source", AgentID: 12, EndpointGeneration: 4, SchemaVersion: 1, Status: agentWebhook.Pending, Body: `{"id":"evt_same_source"}`, Round: 1, Deadline: now.Add(24 * time.Hour), ExpiresAt: now.Add(7 * 24 * time.Hour)}
				return enqueueTx(tx, &row)
			})
		}()
	}
	close(start)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	var deliveries, tasks int64
	if err := conn.Model(&agentWebhook.Delivery{}).Count(&deliveries).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&taskQueue.Entity{}).Count(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	if deliveries != 1 || tasks != 1 {
		t.Fatalf("outbox duplicates deliveries=%d tasks=%d", deliveries, tasks)
	}
	rolledBack := errors.New("rollback injection")
	if err := conn.Transaction(func(tx *gorm.DB) error {
		row := agentWebhook.Delivery{InstanceID: "inst_pg", EventID: "evt_rollback", AgentID: 12, EndpointGeneration: 4, SchemaVersion: 1, Status: agentWebhook.Pending, Body: `{}`, Round: 1, Deadline: now.Add(time.Hour), ExpiresAt: now.Add(7 * 24 * time.Hour)}
		if err := enqueueTx(tx, &row); err != nil {
			return err
		}
		return rolledBack
	}); !errors.Is(err, rolledBack) {
		t.Fatal(err)
	}
	if err := conn.Model(&agentWebhook.Delivery{}).Count(&deliveries).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&taskQueue.Entity{}).Count(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	if deliveries != 1 || tasks != 1 {
		t.Fatalf("rollback leaked delivery/task %d/%d", deliveries, tasks)
	}
}

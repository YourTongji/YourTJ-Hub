package agentWrites

import (
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPostgreSQLConcurrentWriteReservationWaitsForCommittedResult(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		dsn = os.Getenv("TEST_PG_DSN")
	}
	if dsn == "" {
		t.Skip("PostgreSQL DSN not configured")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("agent_writes_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec("DROP SCHEMA " + schema + " CASCADE"); pool, _ := admin.DB(); pool.Close() })
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
	t.Cleanup(func() { pool, _ := conn.DB(); pool.Close() })
	if err := conn.AutoMigrate(&Entry{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	seed := Entry{InstanceID: "concurrent", AgentID: 1, Operation: "post", TargetID: 12, RequestKey: "reply:one", Digest: "same", ExpiresAt: now.Add(time.Hour)}
	first := conn.Begin()
	defer first.Rollback()
	one := seed
	if replay, err := ReserveTx(first, &one, now); err != nil || replay != nil {
		t.Fatalf("first reserve %v %v", replay, err)
	}
	if err := CompleteTx(first, &one, 12, 45); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- conn.Transaction(func(tx *gorm.DB) error {
			close(started)
			two := seed
			replay, err := ReserveTx(tx, &two, now)
			if err != nil {
				return err
			}
			if replay == nil || replay.PostID != 45 {
				return fmt.Errorf("missing committed replay: %#v", replay)
			}
			return nil
		})
	}()
	<-started
	select {
	case err := <-done:
		t.Fatalf("second creator escaped uncommitted reservation: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if err := first.Commit().Error; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("second reservation did not recover after commit")
	}
	var count int64
	if err := conn.Model(&Entry{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("reservation count %d %v", count, err)
	}
}

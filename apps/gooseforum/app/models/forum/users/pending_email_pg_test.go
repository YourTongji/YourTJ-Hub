package users

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// A registration keeps its claim transaction open while a different connection
// attempts to stage the same email. Checks across the two columns must serialize.
func TestPendingEmailClaimsOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	schema := fmt.Sprintf("email_claim_%d", time.Now().UnixNano())
	if err := conn.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Exec("DROP SCHEMA " + schema + " CASCADE") })
	// Table() on both handles keeps this test away from other test schemas.
	table := schema + ".users"
	if err := conn.Exec("CREATE TABLE " + table + " (id bigint PRIMARY KEY, email text UNIQUE NOT NULL, pending_email text NOT NULL DEFAULT '', pending_email_at timestamptz)").Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Table(table).Create(map[string]any{"id": 1, "email": "old@example.test"}).Error; err != nil {
		t.Fatal(err)
	}
	registration := conn.Begin()
	t.Cleanup(func() { registration.Rollback() })
	if registration.Error != nil {
		t.Fatal(registration.Error)
	}
	// Occupancy uses the normal users table name via a transaction-local path.
	if err := registration.Exec("SET LOCAL search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	target := "shared@example.test"
	if err := CheckEmailClaimTx(registration, target, 0); err != nil {
		t.Fatal(err)
	}

	staged := make(chan error, 1)
	go func() { staged <- stagePendingEmail(conn.Table(table), 1, target, time.Now()) }()
	var stageErr error
	finishedEarly := false
	select {
	case stageErr = <-staged:
		finishedEarly = true
	case <-time.After(200 * time.Millisecond):
		// The competing claim waits for the registration commit.
	}
	if err := registration.Table(table).Create(map[string]any{"id": 2, "email": target}).Error; err != nil {
		t.Fatal(err)
	}
	if err := registration.Commit().Error; err != nil {
		t.Fatal(err)
	}
	if !finishedEarly {
		select {
		case stageErr = <-staged:
		case <-time.After(5 * time.Second):
			t.Fatal("staging did not finish after registration committed")
		}
	}
	if !errors.Is(stageErr, ErrEmailOccupied) {
		t.Fatalf("competing staging error = %v, want ErrEmailOccupied", stageErr)
	}
	var claims int64
	if err := conn.Table(table).Where("email = ? OR pending_email = ?", target, target).Count(&claims).Error; err != nil {
		t.Fatal(err)
	}
	if claims != 1 {
		t.Fatalf("email has %d claims, want one", claims)
	}
}

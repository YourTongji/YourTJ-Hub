package taskQueue

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var errLostCommitReceipt = errors.New("injected lost commit receipt")

type lostReceiptPool struct{ *sql.DB }

func (p lostReceiptPool) BeginTx(ctx context.Context, options *sql.TxOptions) (gorm.ConnPool, error) {
	tx, err := p.DB.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	return &lostReceiptTx{tx}, nil
}

type lostReceiptTx struct{ *sql.Tx }

func (tx lostReceiptTx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return err
	}
	return errLostCommitReceipt
}

func exerciseEnqueueReceipt(t *testing.T, conn *gorm.DB, dialect func(gorm.ConnPool) gorm.Dialector) {
	t.Helper()
	if err := conn.AutoMigrate(&Entity{}); err != nil {
		t.Fatal(err)
	}
	rejected := errors.New("injected insert rejection")
	if err := conn.Callback().Create().After("gorm:create").Register("test:queue_reject", func(tx *gorm.DB) { _ = tx.AddError(rejected) }); err != nil {
		t.Fatal(err)
	}
	rejectedTask := &Entity{Type: "email.rejected", TaskJson: "{}"}
	err := createWithAcceptance(conn, rejectedTask)
	if rejectedTask.Id == 0 {
		t.Fatal("failure injection must occur after the SQL insert")
	}
	if !errors.Is(err, ErrNotEnqueued) || !errors.Is(err, rejected) {
		t.Fatalf("rejection receipt: %v", err)
	}
	if err := conn.Callback().Create().Remove("test:queue_reject"); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := conn.Model(&Entity{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("rejected task visible: %d %v", count, err)
	}
	if err := createWithAcceptance(conn, &Entity{Type: "email.accepted", TaskJson: "{}"}); err != nil {
		t.Fatal(err)
	}
	pool, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	uncertain, err := gorm.Open(dialect(&lostReceiptPool{pool}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	err = createWithAcceptance(uncertain, &Entity{Type: "email.uncertain", TaskJson: "{}"})
	if !errors.Is(err, errLostCommitReceipt) || errors.Is(err, ErrNotEnqueued) {
		t.Fatalf("commit outcome incorrectly classified as rejection: %v", err)
	}
	if err := conn.Model(&Entity{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("commit really accepted both tasks: %d %v", count, err)
	}
}

func TestEnqueueAcceptanceReceiptOnSQLite(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := conn.DB()
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	exerciseEnqueueReceipt(t, conn, func(pool gorm.ConnPool) gorm.Dialector { return sqlite.Dialector{Conn: pool} })
}

func TestEnqueueAcceptanceReceiptOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := conn.DB()
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	schema := fmt.Sprintf("enqueue_receipt_%d", time.Now().UnixNano())
	if err := conn.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Exec("DROP SCHEMA " + schema + " CASCADE") })
	if err := conn.Exec("SET search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	exerciseEnqueueReceipt(t, conn, func(pool gorm.ConnPool) gorm.Dialector { return postgres.New(postgres.Config{Conn: pool}) })
}

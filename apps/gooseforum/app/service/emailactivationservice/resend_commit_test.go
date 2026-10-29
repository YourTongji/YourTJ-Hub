package emailactivationservice

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

var errLostResendCommit = errors.New("injected lost resend commit acknowledgement")

type resendCommitPool struct{ *sql.DB }

func (p *resendCommitPool) BeginTx(ctx context.Context, options *sql.TxOptions) (gorm.ConnPool, error) {
	tx, err := p.DB.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	return &resendCommitTx{tx}, nil
}

type resendCommitTx struct{ *sql.Tx }

func (tx *resendCommitTx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return err
	}
	return errLostResendCommit
}

func TestResendUnknownCommitKeepsAcceptedCharge(t *testing.T) {
	setupResend(t)
	conn := db.Connect()
	pool, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	originalPool, originalStatementPool := conn.ConnPool, conn.Statement.ConnPool
	restore := func() { conn.ConnPool = originalPool; conn.Statement.ConnPool = originalStatementPool }
	t.Cleanup(restore)
	conn.ConnPool = &resendCommitPool{pool}
	conn.Statement.ConnPool = conn.ConnPool
	user := users.EntityComplete{Id: testUserID.Add(1), Username: "commit-receipt", Email: "commit-receipt@example.test", IsActivated: users.ActivationPending}
	result, sendErr := Resend(user)
	restore()
	if !errors.Is(sendErr, errLostResendCommit) {
		t.Fatalf("expected lost COMMIT receipt: %v", sendErr)
	}
	var count int64
	if err := conn.Model(&taskQueue.Entity{}).Where("task_json LIKE ?", "%commit-receipt@example.test%").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("the task must actually have committed: count=%d error=%v", count, err)
	}
	t.Cleanup(func() { conn.Where("task_json LIKE ?", "%commit-receipt@example.test%").Delete(&taskQueue.Entity{}) })
	if result.RemainingToday != 2 {
		t.Fatalf("accepted task was refunded after uncertain COMMIT: remaining=%d", result.RemainingToday)
	}
}

package taskQueue

import (
	"errors"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
)

// ErrNotEnqueued proves that no commit was attempted. A commit error is an
// unknown outcome: it must never cause a caller to refund accepted work.
var ErrNotEnqueued = errors.New("task was not enqueued")

// CreateWithAcceptance gives callers a receipt boundary without changing the
// task payload or the ordinary Create API. Errors before COMMIT are definite
// rejection; errors from COMMIT deliberately retain their uncertain outcome.
func CreateWithAcceptance(entity *Entity) error {
	return createWithAcceptance(db.Connect(), entity)
}

func createWithAcceptance(conn *gorm.DB, entity *Entity) error {
	tx := conn.Begin()
	if tx.Error != nil {
		return errors.Join(ErrNotEnqueued, tx.Error)
	}
	defer func() { _ = tx.Rollback().Error }()
	if err := CreateTx(tx, entity); err != nil {
		return errors.Join(ErrNotEnqueued, err)
	}
	return tx.Commit().Error
}

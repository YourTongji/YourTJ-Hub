package dbconnect

import "gorm.io/gorm"

// Savepoint issues an explicit SAVEPOINT statement for tx.
//
// Use this instead of gorm's DB.SavePoint: that helper cannot report failure.
// gorm.io/gorm v1.31.2 routes it through db.AddError(savePointer.SavePoint(db, name)),
// and both drivers in use (gorm.io/driver/postgres v1.6.3 and
// github.com/glebarez/sqlite v1.11.0) implement the dialector hook as
// `tx.Exec("SAVEPOINT " + name); return nil`, so the Exec error is discarded and
// `.Error` is always nil. Any "savepoint unavailable" branch written against the
// gorm API is therefore dead code (issue #1183, item 7). Every call site passes
// a constant savepoint name, never request data.
func Savepoint(tx *gorm.DB, name string) error {
	return tx.Exec("SAVEPOINT " + name).Error
}

// RollbackToSavepoint issues an explicit ROLLBACK TO SAVEPOINT statement.
// It shares Savepoint's rationale: gorm's DB.RollbackTo discards the driver
// error as well, so a `.Error` check on it can never fail.
func RollbackToSavepoint(tx *gorm.DB, name string) error {
	return tx.Exec("ROLLBACK TO SAVEPOINT " + name).Error
}

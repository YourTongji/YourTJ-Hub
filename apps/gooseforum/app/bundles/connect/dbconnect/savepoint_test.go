package dbconnect

import (
	"testing"
)

// A finished transaction is the smallest reproducible "savepoint unavailable"
// state: every statement inside it fails. gorm's own SavePoint/RollbackTo still
// report nil there because both drivers discard the Exec error, which is why the
// failure branches of call sites written against the gorm API were unreachable
// (issue #1183, item 7). The helpers must report the driver error instead, so a
// future replacement with the gorm API is caught here.
func TestSavepointHelpersReportDriverFailure(t *testing.T) {
	tx := Connect().Begin()
	if tx.Error != nil {
		t.Fatalf("begin: %v", tx.Error)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback: %v", err)
	}

	if err := tx.SavePoint("probe").Error; err != nil {
		t.Fatalf("gorm DB.SavePoint unexpectedly reported a driver failure (%v): the helper rationale changed, re-check every call site", err)
	}
	if err := Savepoint(tx, "probe"); err == nil {
		t.Fatal("Savepoint must report the driver failure")
	}
	if err := RollbackToSavepoint(tx, "probe"); err == nil {
		t.Fatal("RollbackToSavepoint must report the driver failure")
	}
}

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

// The pending-email switch commits an address without taking the per-email
// claim lock. When it lands between this transaction's claim check and its
// insert, the cross-column unique index fires on email; the conflict must be
// attributed to the email, not reported as a username collision.
func TestCreateVerifiedAccountTxAttributesEmailConflictOnPostgreSQL(t *testing.T) {
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
	// One connection keeps the session search_path for both the setup and the
	// registration transaction below.
	sqlDB.SetMaxOpenConns(1)
	other, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	otherDB, err := other.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = otherDB.Close() })

	schema := fmt.Sprintf("verified_account_%d", time.Now().UnixNano())
	if err := conn.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Exec("DROP SCHEMA " + schema + " CASCADE") })
	if err := conn.Exec("SET search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&EntityComplete{}); err != nil {
		t.Fatal(err)
	}

	target := "2351111@tongji.edu.cn"
	var injectErr error
	injected := false
	if err := conn.Callback().Create().Before("gorm:create").Register("test:claim_email_mid_insert", func(db *gorm.DB) {
		if injected || db.Statement.Table != tableName {
			return
		}
		injected = true
		// A separate connection commits the same address mid-transaction,
		// like CompletePendingEmailSwitch does outside the claim lock.
		injectErr = other.Table(schema + "." + tableName).
			Create(MakeUser("email_squatter", "Password123", target)).Error
	}); err != nil {
		t.Fatal(err)
	}

	applicant := MakeUser("applicant", "Password123", target)
	err = conn.Transaction(func(tx *gorm.DB) error {
		return CreateVerifiedAccountTx(tx, applicant, -1)
	})
	if injectErr != nil {
		t.Fatalf("inject conflicting email: %v", injectErr)
	}
	if !injected {
		t.Fatal("conflicting email was not injected before the insert")
	}
	if !errors.Is(err, ErrEmailOccupied) {
		t.Fatalf("duplicate key on email reported as %v, want ErrEmailOccupied", err)
	}
}

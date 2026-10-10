package users

import (
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// SQLite serializes writers, so the email-claim recheck path is exercised on
// PostgreSQL (verified_account_pg_test.go); here only the plain username
// collision is attributed.
func TestCreateVerifiedAccountTxReportsUsernameCollision(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := conn.AutoMigrate(&EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(MakeUser("taken_name", "Password123", "owner@example.test")).Error; err != nil {
		t.Fatal(err)
	}
	applicant := MakeUser("taken_name", "Password123", "applicant@example.test")
	err = conn.Transaction(func(tx *gorm.DB) error {
		return CreateVerifiedAccountTx(tx, applicant, -1)
	})
	if !errors.Is(err, ErrUsernameOccupied) {
		t.Fatalf("duplicate username reported as %v, want ErrUsernameOccupied", err)
	}
}

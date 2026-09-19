package campus

import (
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestReplacementTimestampOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sql, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sql.Close(); err != nil {
			t.Error(err)
		}
	})
	tx := db.Begin()
	defer tx.Rollback()
	if err := tx.AutoMigrate(&Binding{}); err != nil {
		t.Fatal(err)
	}
	store := Store{DB: tx}
	before := time.Now().Add(-24 * time.Hour).UTC().Truncate(time.Second)
	binding := Binding{UserID: 998123, IdentityKey: "old-school-identity", Revision: "original", Sealed: "ciphertext", CreatedAt: before}
	if err := store.Replace(binding, ""); err != nil {
		t.Fatal(err)
	}
	binding.Revision = "renewed"
	if err := store.Replace(binding, "original"); err != nil {
		t.Fatal(err)
	}
	stored, err := store.Get(binding.UserID)
	if err != nil || !stored.CreatedAt.Equal(before) {
		t.Fatalf("same identity timestamp: %v, %v", stored.CreatedAt, err)
	}
	binding.IdentityKey = "new-school-identity"
	binding.Revision = "replaced"
	if err := store.Replace(binding, "renewed"); err != nil {
		t.Fatal(err)
	}
	stored, err = store.Get(binding.UserID)
	if err != nil || !stored.CreatedAt.After(before) {
		t.Fatalf("replacement timestamp: %v, %v", stored.CreatedAt, err)
	}
}

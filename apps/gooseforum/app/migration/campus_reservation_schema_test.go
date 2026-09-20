package migration

import (
	"errors"
	"os"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/campus"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestSchemaCampusReservationsUpgradeOnSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sql, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sql.Close() })
	assertCampusReservationUpgrade(t, db)
}

func TestSchemaCampusReservationsUpgradeOnPostgreSQL(t *testing.T) {
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
	t.Cleanup(func() { _ = sql.Close() })
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	if err := tx.Exec("CREATE SCHEMA campus_reservation_upgrade; SET LOCAL search_path TO campus_reservation_upgrade").Error; err != nil {
		t.Fatal(err)
	}
	assertCampusReservationUpgrade(t, tx)
}

func assertCampusReservationUpgrade(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&campus.Binding{}); err != nil {
		t.Fatal(err)
	}
	legacy := campus.Binding{UserID: 42, IdentityKey: "legacy-fingerprint", Revision: "old", Sealed: "old-encrypted-credentials"}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(SchemaModels()...); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := campus.BackfillIdentityReservations(db); err != nil {
			t.Fatal(err)
		}
	}
	store := campus.Store{DB: db}
	if err := store.DeleteForUser(legacy.UserID); err != nil {
		t.Fatal(err)
	}
	if err := campus.BackfillIdentityReservations(db); err != nil {
		t.Fatal(err)
	}
	if err := store.ReserveSignup(legacy.IdentityKey); !errors.Is(err, campus.ErrIdentityUsed) {
		t.Fatalf("legacy identity became registrable: %v", err)
	}
	columns, err := db.Migrator().ColumnTypes(&campus.IdentityReservation{})
	if err != nil || len(columns) != 1 || columns[0].Name() != "identity_key" {
		t.Fatalf("reservation contains extra identifying data: %v", err)
	}
	var count int64
	if err := db.Model(&campus.IdentityReservation{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("backfill count=%d err=%v", count, err)
	}
}

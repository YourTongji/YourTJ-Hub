package migration

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSQLiteMigrationLockReleasesLease(t *testing.T) {
	db := openSQLiteMigrationLockDB(t)

	unlock, err := acquireSQLiteMigrationLock(db)
	if err != nil {
		t.Fatalf("acquire sqlite lock: %v", err)
	}
	unlock()

	var owner string
	if err := db.Raw("SELECT owner FROM app_migration_lock WHERE id = ?", versionedMigrationLockID).Scan(&owner).Error; err != nil {
		t.Fatalf("read sqlite lock owner: %v", err)
	}
	if owner != "" {
		t.Fatalf("sqlite migration lock owner = %q after release, want empty", owner)
	}
}

func TestSQLiteMigrationLockTakesOverExpiredLease(t *testing.T) {
	db := openSQLiteMigrationLockDB(t)
	unlock, err := acquireSQLiteMigrationLock(db)
	if err != nil {
		t.Fatalf("acquire sqlite lock: %v", err)
	}
	unlock()
	if err := db.Exec("UPDATE app_migration_lock SET owner = ?, expires_at = ? WHERE id = ?", "expired-owner", time.Now().Add(-time.Second).UnixNano(), versionedMigrationLockID).Error; err != nil {
		t.Fatalf("seed expired sqlite lock: %v", err)
	}

	secondUnlock, err := acquireSQLiteMigrationLockWithWait(db, 20*time.Millisecond)
	if err != nil {
		t.Fatalf("acquire expired sqlite lock: %v", err)
	}
	secondUnlock()
}

func TestSQLiteMigrationLockDoesNotTakeLiveFractionalLease(t *testing.T) {
	db := openSQLiteMigrationLockDB(t)
	unlock, err := acquireSQLiteMigrationLock(db)
	if err != nil {
		t.Fatalf("acquire sqlite lock: %v", err)
	}
	unlock()
	if err := db.Exec("UPDATE app_migration_lock SET owner = ?, expires_at = ? WHERE id = ?", "live-owner", time.Now().Add(500*time.Millisecond).UnixNano(), versionedMigrationLockID).Error; err != nil {
		t.Fatalf("seed live sqlite lock: %v", err)
	}

	if _, err := acquireSQLiteMigrationLockWithWait(db, 20*time.Millisecond); err == nil {
		t.Fatal("acquired a live sqlite lease")
	}
	if err := db.Exec("UPDATE app_migration_lock SET owner = '', expires_at = 0 WHERE id = ?", versionedMigrationLockID).Error; err != nil {
		t.Fatalf("clear live sqlite lock: %v", err)
	}
}

func openSQLiteMigrationLockDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "migration-lock.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite db: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close sqlite db: %v", err)
		}
	})
	return db
}

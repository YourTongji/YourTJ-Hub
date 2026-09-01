package migration

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSQLiteMigrationLockReleasesLease(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:migration-lock?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

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

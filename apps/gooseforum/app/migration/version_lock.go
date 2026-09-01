package migration

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
)

const (
	versionedMigrationLockID            = 1
	versionedMigrationLockLease         = 30 * time.Second
	versionedMigrationLockWait          = 30 * time.Second
	versionedMigrationLockRetry         = 100 * time.Millisecond
	versionedMigrationAdvisoryKey int64 = 0x596f7572544a4d
)

var versionedMigrationProcessMu sync.Mutex

// acquireVersionedMigrationLock serializes the versioned data migration
// runner across both goroutines in one process and independent app instances.
// PostgreSQL uses a session-scoped advisory lock; SQLite uses a durable row
// lease because it has no advisory-lock primitive.
func acquireVersionedMigrationLock() (func(), error) {
	versionedMigrationProcessMu.Lock()

	db := dbconnect.Connect()
	if db == nil {
		versionedMigrationProcessMu.Unlock()
		return nil, errors.New("migration: database connection is nil")
	}

	var unlock func()
	var err error
	switch db.Dialector.Name() {
	case "postgres":
		unlock, err = acquirePostgresMigrationLock(db)
	case "sqlite":
		unlock, err = acquireSQLiteMigrationLock(db)
	default:
		err = fmt.Errorf("migration: unsupported database dialect %q", db.Dialector.Name())
	}
	if err != nil {
		versionedMigrationProcessMu.Unlock()
		return nil, err
	}

	return sync.OnceFunc(func() {
		unlock()
		versionedMigrationProcessMu.Unlock()
	}), nil
}

func acquirePostgresMigrationLock(db *gorm.DB) (func(), error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("migration: get postgres pool: %w", err)
	}
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		return nil, fmt.Errorf("migration: pin postgres connection: %w", err)
	}
	if _, err := conn.ExecContext(context.Background(), "SELECT pg_advisory_lock($1)", versionedMigrationAdvisoryKey); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("migration: acquire postgres advisory lock: %w", err)
	}

	return sync.OnceFunc(func() {
		if _, err := conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", versionedMigrationAdvisoryKey); err != nil {
			slog.Warn("migration: release postgres advisory lock failed", "err", err)
		}
		if err := conn.Close(); err != nil {
			slog.Warn("migration: close pinned postgres connection failed", "err", err)
		}
	}), nil
}

func acquireSQLiteMigrationLock(db *gorm.DB) (func(), error) {
	if err := db.Exec(`
CREATE TABLE IF NOT EXISTS app_migration_lock (
		id INTEGER PRIMARY KEY,
		owner TEXT NOT NULL DEFAULT '',
		expires_at DATETIME NOT NULL DEFAULT '1970-01-01T00:00:00Z'
)`).Error; err != nil {
		return nil, fmt.Errorf("migration: create sqlite lock table: %w", err)
	}
	if err := db.Exec(`INSERT OR IGNORE INTO app_migration_lock (id) VALUES (?)`, versionedMigrationLockID).Error; err != nil {
		return nil, fmt.Errorf("migration: seed sqlite lock row: %w", err)
	}

	owner := fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	deadline := time.Now().Add(versionedMigrationLockWait)
	for {
		now := time.Now().UTC()
		expiresAt := now.Add(versionedMigrationLockLease)
		result := db.Exec(`
UPDATE app_migration_lock
SET owner = ?, expires_at = ?
WHERE id = ? AND (owner = '' OR expires_at <= ?)`,
			owner,
			expiresAt.Format(time.RFC3339Nano),
			versionedMigrationLockID,
			now.Format(time.RFC3339Nano),
		)
		if result.Error != nil {
			return nil, fmt.Errorf("migration: acquire sqlite lock: %w", result.Error)
		}
		if result.RowsAffected == 1 {
			return startSQLiteLockLease(db, owner), nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("migration: sqlite lock still held after %s", versionedMigrationLockWait)
		}
		time.Sleep(versionedMigrationLockRetry)
	}
}

func startSQLiteLockLease(db *gorm.DB, owner string) func() {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(versionedMigrationLockLease / 3)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				now := time.Now().UTC()
				result := db.Exec(`UPDATE app_migration_lock SET expires_at = ? WHERE id = ? AND owner = ?`,
					now.Add(versionedMigrationLockLease).Format(time.RFC3339Nano),
					versionedMigrationLockID,
					owner,
				)
				if result.Error != nil {
					slog.Warn("migration: renew sqlite lock failed", "err", result.Error)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return sync.OnceFunc(func() {
		cancel()
		<-done
		if err := db.Exec(`UPDATE app_migration_lock SET owner = '', expires_at = ? WHERE id = ? AND owner = ?`,
			time.Now().UTC().Format(time.RFC3339Nano), versionedMigrationLockID, owner).Error; err != nil {
			slog.Warn("migration: release sqlite lock failed", "err", err)
		}
	})
}

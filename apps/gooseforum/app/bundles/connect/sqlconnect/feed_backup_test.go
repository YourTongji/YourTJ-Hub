package sqlconnect

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"path/filepath"
	"testing"
)

func TestSQLiteBackupExcludesFeedRawAndKeepsSource(t *testing.T) {
	source, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{"CREATE TABLE feed_serve_log (id TEXT PRIMARY KEY,user_id INTEGER)", "INSERT INTO feed_serve_log VALUES ('trace',12)", "CREATE TABLE feed_seen_state (user_id INTEGER,topic_id INTEGER,expires_at TEXT)", "INSERT INTO feed_seen_state VALUES (12,31,'2026-11-07')", "CREATE TABLE business (id INTEGER PRIMARY KEY)", "INSERT INTO business VALUES (42)"} {
		if err = source.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(t.TempDir(), "backup.db")
	if err = backupSQLite(source, path); err != nil {
		t.Fatal(err)
	}
	copyDB, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		db    *gorm.DB
		table string
		want  int64
	}{{source, "feed_serve_log", 1}, {copyDB, "feed_serve_log", 0}, {copyDB, "business", 1}, {source, "feed_seen_state", 1}, {copyDB, "feed_seen_state", 0}} {
		var n int64
		if err = check.db.Table(check.table).Count(&n).Error; err != nil {
			t.Fatal(err)
		}
		if n != check.want {
			t.Fatalf("%s count=%d want %d", check.table, n, check.want)
		}
	}
}

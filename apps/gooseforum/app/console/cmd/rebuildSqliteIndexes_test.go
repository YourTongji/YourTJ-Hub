package cmd

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/glebarez/sqlite"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

type rebuildIndexFixture struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `gorm:"index:idx_rebuild_fixture_name"`
	Value string `gorm:"index:idx_rebuild_fixture_value"`
}

func (rebuildIndexFixture) TableName() string { return "rebuild_index_fixture" }

type rebuildIndexLegacyFixture struct {
	ID  uint   `gorm:"primaryKey"`
	Old string `gorm:"column:old;index:idx_rebuild_fixture_name"`
	New string `gorm:"column:new"`
}

func (rebuildIndexLegacyFixture) TableName() string { return "rebuild_index_fixture" }

type rebuildIndexChangedFixture struct {
	ID  uint   `gorm:"primaryKey"`
	New string `gorm:"column:new;index:idx_rebuild_fixture_name"`
}

type rebuildIndexRollbackFixture struct {
	ID  uint   `gorm:"primaryKey"`
	Old string `gorm:"column:old;index:idx_rebuild_fixture_name"`
}

func (rebuildIndexRollbackFixture) TableName() string { return "rebuild_index_fixture" }

func (rebuildIndexChangedFixture) TableName() string { return "rebuild_index_fixture" }

func openRebuildIndexTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "indexes.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})
	return db
}

func TestRebuildSQLiteIndexes(t *testing.T) {
	db := openRebuildIndexTestDB(t)
	var err error
	if err = db.AutoMigrate(&rebuildIndexFixture{}); err != nil {
		t.Fatal(err)
	}
	if err = db.Exec("CREATE INDEX idx_rebuild_fixture_manual ON rebuild_index_fixture (id)").Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS rebuild_index_fixture") })
	before, err := indexNames(db)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, _, err := rebuildSQLiteIndexes(db, []any{&rebuildIndexFixture{}}); err != nil {
		t.Fatal(err)
	}
	after, err := indexNames(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) || !after["idx_rebuild_fixture_name"] || !after["idx_rebuild_fixture_manual"] {
		t.Fatalf("indexes after rebuild = %v, want managed and manual indexes retained", after)
	}
}

func TestRebuildSQLiteIndexesRollsBackOnCreateFailure(t *testing.T) {
	db := openRebuildIndexTestDB(t)
	var err error
	if err = db.AutoMigrate(&rebuildIndexRollbackFixture{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS rebuild_index_fixture") })
	var before string
	if err = db.Raw("SELECT sql FROM sqlite_master WHERE name='idx_rebuild_fixture_name'").Scan(&before).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, _, err = rebuildSQLiteIndexes(db, []any{&rebuildIndexChangedFixture{}}); err == nil {
		t.Fatal("expected create failure for stale schema")
	}
	var after string
	if err = db.Raw("SELECT sql FROM sqlite_master WHERE name='idx_rebuild_fixture_name'").Scan(&after).Error; err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("index SQL after rollback = %q, want %q", after, before)
	}
}

func TestRebuildSQLiteIndexesRollsBackOnDropFailure(t *testing.T) {
	db := openRebuildIndexTestDB(t)
	if err := db.AutoMigrate(&rebuildIndexFixture{}); err != nil {
		t.Fatal(err)
	}
	var before map[string]bool
	var err error
	if before, err = indexNames(db); err != nil {
		t.Fatal(err)
	}
	const callback = "test:rebuild-sqlite-index-drop-failure"
	drops := 0
	if err := db.Callback().Raw().Before("gorm:raw").Register(callback, func(tx *gorm.DB) {
		if strings.HasPrefix(tx.Statement.SQL.String(), "DROP INDEX") {
			drops++
		}
		if drops == 2 {
			_ = tx.AddError(errors.New("injected drop failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	_, _, _, err = rebuildSQLiteIndexes(db, []any{&rebuildIndexFixture{}})
	if removeErr := db.Callback().Raw().Remove(callback); removeErr != nil {
		t.Fatal(removeErr)
	}
	if err == nil || drops != 2 {
		t.Fatalf("rebuild error = %v after %d drops, want injected second-drop failure", err, drops)
	}
	after, err := indexNames(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) || !after["idx_rebuild_fixture_name"] || !after["idx_rebuild_fixture_value"] {
		t.Fatalf("indexes after failed drop = %v, want transaction rollback to restore both", after)
	}
}

func TestRebuildSQLiteIndexesUsesCurrentModelDefinition(t *testing.T) {
	db := openRebuildIndexTestDB(t)
	if err := db.AutoMigrate(&rebuildIndexLegacyFixture{}); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := rebuildSQLiteIndexes(db, []any{&rebuildIndexChangedFixture{}}); err != nil {
		t.Fatal(err)
	}
	var sql string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE name='idx_rebuild_fixture_name'").Scan(&sql).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sql, "new") {
		t.Fatalf("rebuilt index SQL = %q, want current model column new", sql)
	}
}

func TestRebuildSQLiteIndexesKeepsSameNamedIndexOnOtherTable(t *testing.T) {
	db := openRebuildIndexTestDB(t)
	if err := db.Exec("CREATE TABLE manual_index_table (id INTEGER PRIMARY KEY, value TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE INDEX idx_rebuild_fixture_name ON manual_index_table (value)").Error; err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := rebuildSQLiteIndexes(db, []any{&rebuildIndexChangedFixture{}}); err != nil {
		t.Fatal(err)
	}
	var table string
	if err := db.Raw("SELECT tbl_name FROM sqlite_master WHERE name='idx_rebuild_fixture_name'").Scan(&table).Error; err != nil {
		t.Fatal(err)
	}
	if table != "manual_index_table" {
		t.Fatalf("same-named index table = %q, want manual_index_table", table)
	}
}

func TestRebuildSQLiteIndexesReportsDeclaredButMissing(t *testing.T) {
	db := openRebuildIndexTestDB(t)
	if err := db.AutoMigrate(&rebuildIndexLegacyFixture{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("DROP INDEX idx_rebuild_fixture_name").Error; err != nil {
		t.Fatal(err)
	}
	_, rebuilt, missing, err := rebuildSQLiteIndexes(db, []any{&rebuildIndexLegacyFixture{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rebuilt) != 0 {
		t.Fatalf("rebuilt indexes = %d, want 0 when the only declared index is missing", len(rebuilt))
	}
	if len(missing) != 1 || missing[0] != "rebuild_index_fixture.idx_rebuild_fixture_name" {
		t.Fatalf("missing indexes = %v, want [rebuild_index_fixture.idx_rebuild_fixture_name]", missing)
	}
}

func TestRebuildSQLiteIndexesRejectsMigrationOffWithoutMutation(t *testing.T) {
	db := dbconnect.Connect()
	if !dbconnect.IsSqlite() {
		t.Skip("test database is not SQLite")
	}
	if err := db.AutoMigrate(&rebuildIndexFixture{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE INDEX idx_rebuild_fixture_manual ON rebuild_index_fixture (id)").Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS rebuild_index_fixture") })
	before, err := indexNames(db)
	if err != nil {
		t.Fatal(err)
	}
	previous := preferences.GetString("db.migration", "on")
	preferences.Set("db.migration", "off")
	t.Cleanup(func() { preferences.Set("db.migration", previous) })
	cmd := &cobra.Command{}
	cmd.Flags().Bool("yes", true, "")
	if err = runRebuildSQLiteIndexes(cmd, nil); err == nil {
		t.Fatal("expected migration-off preflight to reject rebuild")
	}
	after, err := indexNames(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) || !after["idx_rebuild_fixture_name"] || !after["idx_rebuild_fixture_manual"] {
		t.Fatalf("indexes after migration-off rejection = %v, want unchanged indexes", after)
	}
}

func indexNames(db *gorm.DB) (map[string]bool, error) {
	var names []string
	err := db.Raw("SELECT name FROM sqlite_master WHERE type='index'").Scan(&names).Error
	result := make(map[string]bool, len(names))
	for _, name := range names {
		result[name] = true
	}
	return result, err
}

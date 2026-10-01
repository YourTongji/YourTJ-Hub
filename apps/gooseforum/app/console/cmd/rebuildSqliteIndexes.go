package cmd

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/setting"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/migration"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"slices"
)

type sqliteIndex struct {
	Name    string
	TblName string
	Model   any `gorm:"-"`
}

func init() {
	cmd := &cobra.Command{
		Use:   "rebuild-sqlite-indexes",
		Short: "Rebuild model-managed SQLite indexes",
		RunE:  runRebuildSQLiteIndexes,
	}
	cmd.Flags().Bool("yes", false, "Confirm rebuilding model-managed SQLite indexes")
	appendCommand(cmd)
}

func runRebuildSQLiteIndexes(cmd *cobra.Command, _ []string) error {
	totalStart := time.Now()
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		return fmt.Errorf("refusing to rebuild indexes without --yes")
	}
	if !dbconnect.IsSqlite() {
		return fmt.Errorf("default database is not sqlite")
	}
	if !setting.UseMigration() {
		return fmt.Errorf("refusing to rebuild SQLite indexes while db.migration is off")
	}
	db := dbconnect.Connect()
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(1)
	beforeCount, rebuilt, missing, err := rebuildSQLiteIndexes(db, migration.SchemaModels())
	if err != nil {
		return err
	}
	fmt.Printf("default sqlite indexes before rebuild: %d\n", beforeCount)
	fmt.Printf("rebuild complete: %d model-managed indexes recreated\n", len(rebuilt))
	for _, name := range missing {
		fmt.Printf("declared but missing index skipped: %s\n", name)
	}
	// VACUUM reclaims the pages freed by the dropped indexes; it cannot run
	// inside a transaction, so it executes after the rebuild has committed.
	if err = db.Exec("VACUUM").Error; err != nil {
		return fmt.Errorf("vacuum default sqlite db: %w", err)
	}
	fmt.Printf("total duration: %s\n", time.Since(totalStart).Round(time.Millisecond))
	return nil
}

// rebuildSQLiteIndexes recreates only indexes declared by SchemaModels in one
// SQLite transaction, leaving manually maintained indexes untouched. It also
// reports declared indexes that are missing from the database so operators
// are not left assuming a full rebuild happened.
func rebuildSQLiteIndexes(db *gorm.DB, models []any) (int, []sqliteIndex, []string, error) {
	managed := make(map[string]any)
	var cache sync.Map
	for _, model := range models {
		s, err := schema.Parse(model, &cache, db.NamingStrategy)
		if err != nil {
			return 0, nil, nil, err
		}
		for _, index := range s.ParseIndexes() {
			managed[s.Table+"\x00"+index.Name] = model
		}
	}

	var found []sqliteIndex
	var indexes []sqliteIndex
	var missing []string
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw(`SELECT name, tbl_name FROM sqlite_master WHERE type = 'index' AND name NOT LIKE 'sqlite_autoindex%'`).Scan(&found).Error; err != nil {
			return err
		}
		foundKeys := make(map[string]bool, len(found))
		for _, index := range found {
			foundKeys[index.TblName+"\x00"+index.Name] = true
			if model, ok := managed[index.TblName+"\x00"+index.Name]; ok {
				index.Model = model
				indexes = append(indexes, index)
			}
		}
		for key := range managed {
			if !foundKeys[key] {
				table, name, _ := strings.Cut(key, "\x00")
				missing = append(missing, table+"."+name)
			}
		}
		slices.Sort(missing)
		for _, index := range indexes {
			if err := tx.Exec("DROP INDEX " + sqliteQuoteIdent(index.Name)).Error; err != nil {
				return fmt.Errorf("drop index %s: %w", index.Name, err)
			}
		}
		for _, index := range indexes {
			if err := tx.Migrator().CreateIndex(index.Model, index.Name); err != nil {
				return fmt.Errorf("rebuild index %s: %w", index.Name, err)
			}
			var exists int
			if err := tx.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?", index.Name).Scan(&exists).Error; err != nil {
				return err
			}
			if exists != 1 {
				return fmt.Errorf("rebuild index %s: verification failed", index.Name)
			}
		}
		return nil
	})
	return len(found), indexes, missing, err
}

func sqliteQuoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

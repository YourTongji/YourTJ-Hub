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
	beforeCount, rebuilt, err := rebuildSQLiteIndexes(db, migration.SchemaModels())
	if err != nil {
		return err
	}
	fmt.Printf("default sqlite indexes before rebuild: %d\n", beforeCount)
	fmt.Printf("rebuilt default sqlite indexes, rebuilt %d model-managed indexes\n", len(rebuilt))
	fmt.Printf("total duration: %s\n", time.Since(totalStart).Round(time.Millisecond))
	return nil
}

// rebuildSQLiteIndexes recreates only indexes declared by SchemaModels in one
// SQLite transaction, leaving manually maintained indexes untouched.
func rebuildSQLiteIndexes(db *gorm.DB, models []any) (int, []sqliteIndex, error) {
	managed := make(map[string]any)
	var cache sync.Map
	for _, model := range models {
		s, err := schema.Parse(model, &cache, db.NamingStrategy)
		if err != nil {
			return 0, nil, err
		}
		for _, index := range s.ParseIndexes() {
			managed[s.Table+"\x00"+index.Name] = model
		}
	}

	var found []sqliteIndex
	var indexes []sqliteIndex
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw(`SELECT name, tbl_name FROM sqlite_master WHERE type = 'index' AND name NOT LIKE 'sqlite_autoindex%'`).Scan(&found).Error; err != nil {
			return err
		}
		for _, index := range found {
			if model, ok := managed[index.TblName+"\x00"+index.Name]; ok {
				index.Model = model
				indexes = append(indexes, index)
			}
		}
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
	return len(found), indexes, err
}

func sqliteQuoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

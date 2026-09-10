package cmd

import (
	"fmt"
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/migration"
	"github.com/spf13/cobra"
)

func init() {
	appendCommand(&cobra.Command{
		Use:   "migrate",
		Short: "Run database schema and versioned data migrations",
		Args:  cobra.NoArgs,
		RunE:  runMigrate,
	})
}

// runMigrate applies migrations so a deployment can run them explicitly before
// starting serve. Deferred (retry-later) and lock-held states are non-fatal:
// they preserve the historical log-and-proceed behavior and exit successfully
// with an actionable message rather than failing a release pipeline.
func runMigrate(_ *cobra.Command, _ []string) error {
	err := migration.M()
	if err == nil {
		fmt.Println("Database migrations completed.")
		return nil
	}
	if migration.Deferred(err) {
		fmt.Println("Database migrations deferred (will retry on next run):", err)
		return nil
	}
	slog.Error("database migrations failed", "err", err)
	return fmt.Errorf("run migrations: %w", err)
}

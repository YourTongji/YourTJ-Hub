package console

import (
	"log/slog"
	"os"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/closer"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/console/cmd"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/migration"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gooseforum",
	Short: "GooseForum command line tools",
	Long:  `GooseForum`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// `serve` runs migrations itself, behind the startup gate, so it is
		// skipped here; every other command keeps the historical behavior of
		// migrating synchronously before touching the database.
		//
		// For `serve` this starts the event bus before migrations run. That
		// inversion is safe only while no data migration publishes events —
		// if a future backfill publishes, it must move behind the startup gate
		// (see serve.go runStartup) instead of running before the bus is up.
		if cmd.Name() != "serve" {
			runMigrationForCommand()
		}
		// 初始化并启动事件总线
		eventbus.Start(eventhandlers.Handlers()...)
	},
	// Run: runWeb,
}

// runMigrationForCommand applies migrations before a CLI command runs.
// Sentinels (retry-later, lock held by another instance) are non-fatal and
// preserve the historical log-and-proceed behavior; any hard migration failure
// aborts with a message and a non-zero exit so the command never runs against
// a partial schema or incomplete data migration.
func runMigrationForCommand() {
	err := migration.M()
	if err == nil || migration.Deferred(err) {
		if err != nil {
			slog.Warn("migration deferred; continuing", "err", err)
		}
		return
	}
	slog.Error("migration failed", "err", err)
	os.Exit(1)
}

func init() {
	rootCmd.AddCommand(CmdServe)
	rootCmd.AddCommand(cmd.GetCommands()...)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	// Close registered resources before any CheckErr os.Exit (the defers would
	// be skipped by cobra.CheckErr's os.Exit(1) on error paths).
	if closeErr := closer.CloseAll(); closeErr != nil {
		slog.Error("closer: resources closed with errors", "err", closeErr)
	}
	cobra.CheckErr(err)
}

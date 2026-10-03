package console

import (
	"encoding/json"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/buildinfo"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print compiled release metadata as JSON",
		Args:  cobra.NoArgs,
		// Reading identity must not migrate a database or start background services.
		PersistentPreRun: func(*cobra.Command, []string) {},
		RunE: func(command *cobra.Command, _ []string) error {
			return json.NewEncoder(command.OutOrStdout()).Encode(buildinfo.Get())
		},
	})
}

package console

import (
	"encoding/json"
	"os"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/buildinfo"
	"github.com/spf13/cobra"
)

func init() {
	var output string
	command := &cobra.Command{
		Use:   "version",
		Short: "Print compiled release metadata as JSON",
		Args:  cobra.NoArgs,
		// Reading identity must not migrate a database or start background services.
		PersistentPreRun: func(*cobra.Command, []string) {},
		RunE: func(command *cobra.Command, _ []string) error {
			if output != "" {
				data, err := json.Marshal(buildinfo.Get())
				if err != nil {
					return err
				}
				return os.WriteFile(output, append(data, '\n'), 0o600)
			}
			return json.NewEncoder(command.OutOrStdout()).Encode(buildinfo.Get())
		},
	}
	command.Flags().StringVar(&output, "output", "", "Write JSON to a file independently of application logs")
	rootCmd.AddCommand(command)
}

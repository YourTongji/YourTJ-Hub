package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/feedservice"
	"github.com/spf13/cobra"
)

func init() {
	appendCommand(&cobra.Command{Use: "feed-report", Short: "Print bounded anonymous feed metrics and experiment intervals", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		result, err := feedservice.GetSummary(c.Context())
		if err != nil {
			return err
		}
		return json.NewEncoder(c.OutOrStdout()).Encode(result)
	}})
	appendCommand(&cobra.Command{Use: "feed-explain TOPIC_ID", Short: "Explain current public-topic scores without contributor identifiers", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil || id == 0 {
			return fmt.Errorf("expected a positive numeric topic ID")
		}
		result, err := feedservice.ExplainRank(c.Context(), id)
		if err != nil {
			return err
		}
		return json.NewEncoder(c.OutOrStdout()).Encode(result)
	}})
	appendCommand(&cobra.Command{Use: "feed-rebuild", Short: "Request a bounded rebuild by the serving feed worker", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		if err := feedservice.RequestRebuild(c.Context()); err != nil {
			return err
		}
		_, err := fmt.Fprintln(c.OutOrStdout(), "Feed rebuild requested; serve is the only scorer.")
		return err
	}})
	appendCommand(&cobra.Command{Use: "feed-replay SAMPLE_ID", Short: "Replay a retained finite candidate sample with the current weights", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
		result, err := feedservice.ReplaySample(c.Context(), args[0], feedconfig.Current().Weights)
		if err != nil {
			return err
		}
		return json.NewEncoder(c.OutOrStdout()).Encode(result)
	}})
}

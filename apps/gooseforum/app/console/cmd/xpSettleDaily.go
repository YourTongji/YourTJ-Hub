package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/xpservice"
	"github.com/spf13/cobra"
)

func init() {
	settleDailyXPCmd := &cobra.Command{
		Use:   "xp-settle-daily",
		Short: "Preview or settle the daily base XP reward for one Shanghai calendar date",
		Long: `Settle the +5 XP daily_base rule for exactly one Asia/Shanghai date.

The command defaults to dry-run; pass --apply and --start-date to write XP.
Past dates report that historical freeze and site-setting state is unavailable.
The root command still runs normal database migrations before this command and
starts the process event bus, so dry-run prevents XP ledger writes but is not a
whole-process read-only invocation.`,
		Args: cobra.NoArgs,
		RunE: runSettleDailyXP,
	}
	settleDailyXPCmd.Flags().String("date", "", "Settlement date in exact YYYY-MM-DD form (Asia/Shanghai)")
	settleDailyXPCmd.Flags().Bool("apply", false, "Write XP awards (required for any settlement writes)")
	settleDailyXPCmd.Flags().Bool("dry-run", false, "Preview candidates without XP ledger writes (default unless --apply is passed)")
	settleDailyXPCmd.Flags().String("start-date", "", "Required apply boundary; reject settlement dates before this YYYY-MM-DD date")
	settleDailyXPCmd.Flags().Int("batch-size", xpservice.DefaultDailySettlementBatchSize, "Maximum users read per ID-cursor batch")
	_ = settleDailyXPCmd.MarkFlagRequired("date")
	appendCommand(settleDailyXPCmd)
}

type dailySettlementCLIOutput struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	xpservice.DailySettlementStats
}

func runSettleDailyXP(cmd *cobra.Command, _ []string) error {
	date, _ := cmd.Flags().GetString("date")
	apply, _ := cmd.Flags().GetBool("apply")
	dryRunFlag, _ := cmd.Flags().GetBool("dry-run")
	startDate, _ := cmd.Flags().GetString("start-date")
	batchSize, _ := cmd.Flags().GetInt("batch-size")
	stats := xpservice.DailySettlementStats{Date: date, DryRun: !apply}

	if err := validateDailySettlementCommand(date, startDate, apply, dryRunFlag, time.Now()); err != nil {
		stats.Failed++
		if outputErr := writeDailySettlementOutput(cmd, "failed", stats, err); outputErr != nil {
			return errors.Join(err, fmt.Errorf("write xp-settle-daily summary: %w", outputErr))
		}
		return err
	}

	baseContext := cmd.Context()
	if baseContext == nil {
		baseContext = context.Background()
	}
	ctx, stop := signal.NotifyContext(baseContext, os.Interrupt, syscall.SIGTERM)
	defer stop()

	report, err := xpservice.New(dbconnect.ConnectContext(ctx)).SettleDailyBase(ctx, date, startDate, !apply, batchSize)
	if err != nil {
		if outputErr := writeDailySettlementOutput(cmd, "failed", report, err); outputErr != nil {
			return errors.Join(fmt.Errorf("xp-settle-daily: %w", err), fmt.Errorf("write summary: %w", outputErr))
		}
		return fmt.Errorf("xp-settle-daily: %w", err)
	}
	status := "completed"
	if !apply {
		status = "dry_run"
	} else if len(report.Warnings) > 0 {
		status = "completed_with_warnings"
	}
	if err := writeDailySettlementOutput(cmd, status, report, nil); err != nil {
		return fmt.Errorf("write xp-settle-daily summary: %w", err)
	}
	return nil
}

func validateDailySettlementCommand(date, startDate string, apply, dryRun bool, now time.Time) error {
	if apply && dryRun {
		return errors.New("--apply and --dry-run cannot be used together")
	}
	if apply && strings.TrimSpace(startDate) == "" {
		return errors.New("--start-date is required with --apply")
	}
	settlementDate, err := xpservice.ParseDailySettlementDate(date)
	if err != nil {
		return err
	}
	location := settlementDate.Location()
	localNow := now.In(location)
	today := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location)
	if settlementDate.After(today) {
		return fmt.Errorf("%w: %s", xpservice.ErrFutureSettlementDate, date)
	}
	if strings.TrimSpace(startDate) != "" {
		boundary, err := xpservice.ParseDailySettlementDate(startDate)
		if err != nil {
			return fmt.Errorf("invalid --start-date: %w", err)
		}
		if settlementDate.Before(boundary) {
			return fmt.Errorf("settlement date %s is before --start-date %s", date, startDate)
		}
	}
	return nil
}

func writeDailySettlementOutput(cmd *cobra.Command, status string, stats xpservice.DailySettlementStats, runErr error) error {
	output := dailySettlementCLIOutput{Status: status, DailySettlementStats: stats}
	if runErr != nil {
		output.Error = runErr.Error()
	}
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

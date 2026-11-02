package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/xpservice"
	"github.com/spf13/cobra"
)

func TestValidateDailySettlementCommandRequiresExplicitApplyBoundary(t *testing.T) {
	now := time.Date(2026, 10, 26, 1, 0, 0, 0, time.UTC)
	if err := validateDailySettlementCommand("2026-10-26", "", false, false, now); err != nil {
		t.Fatalf("default dry-run validation: %v", err)
	}
	if err := validateDailySettlementCommand("2026-10-26", "", true, false, now); err == nil {
		t.Fatal("--apply without --start-date was accepted")
	}
	if err := validateDailySettlementCommand("2026-10-26", "2026-10-27", true, false, now); err == nil {
		t.Fatal("--apply before the activation boundary was accepted")
	}
	if err := validateDailySettlementCommand("2026-10-26", "2026-10-26", true, false, now); err != nil {
		t.Fatalf("apply on the boundary date: %v", err)
	}
	if err := validateDailySettlementCommand("2026-10-27", "2026-10-26", true, false, now); err == nil {
		t.Fatal("future settlement date was accepted")
	}
	if err := validateDailySettlementCommand("2026-10-26T00:00:00+08:00", "", false, true, now); err == nil {
		t.Fatal("non-date timestamp was accepted")
	}
	if err := validateDailySettlementCommand("2026-10-26", "", true, true, now); err == nil {
		t.Fatal("--apply and --dry-run together were accepted")
	}
}

func TestDailySettlementCLIOutputIsStructuredAndAggregateOnly(t *testing.T) {
	buffer := new(bytes.Buffer)
	command := &cobra.Command{}
	command.SetOut(buffer)
	stats := xpservice.DailySettlementStats{
		Date: "2026-10-26", DryRun: true, Scanned: 20, Eligible: 12,
		Candidates: 9, AlreadySettled: 3, Skipped: 8,
	}
	if err := writeDailySettlementOutput(command, "dry_run", stats, nil); err != nil {
		t.Fatal(err)
	}
	output := buffer.String()
	for _, field := range []string{`"status": "dry_run"`, `"date": "2026-10-26"`, `"scanned": 20`, `"candidates": 9`, `"already_settled": 3`} {
		if !strings.Contains(output, field) {
			t.Errorf("CLI output missing %s: %s", field, output)
		}
	}
	if strings.Contains(output, "user_id") || strings.Contains(output, "120") {
		t.Fatalf("CLI output contains an account identifier: %s", output)
	}
}

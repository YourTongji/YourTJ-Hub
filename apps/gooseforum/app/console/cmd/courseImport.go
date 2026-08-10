package cmd

import (
	"fmt"

	"github.com/leancodebox/GooseForum/app/service/courseservice"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "course-import",
		Short: "Import the course catalog from a manifest package (courses/instructors/offerings JSONL)",
		Args:  cobra.ExactArgs(1),
		RunE:  runCourseImport,
	}
	cmd.Flags().Bool("dry-run", false, "validate and canonicalize only, do not write to the database")
	appendCommand(cmd)
}

func runCourseImport(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	report, err := courseservice.ImportCatalog(cmd.Context(), args[0], dryRun)
	if err != nil {
		return fmt.Errorf("course catalog import: %w", err)
	}
	fmt.Printf("course-import: source=%s manifest=%s dryRun=%v\n",
		report.Source, report.ManifestHash, report.DryRun)
	fmt.Printf("  total=%d inserted=%d updated=%d quarantined=%d skipped=%d errors=%d\n",
		report.TotalLines, report.Inserted, report.Updated, report.Quarantined, report.Skipped, len(report.Errors))
	for _, e := range report.Errors {
		fmt.Printf("  [%s] %s\n", e.Entity, e.Reason)
	}
	return nil
}

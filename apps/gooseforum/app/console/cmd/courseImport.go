package cmd

import (
	"fmt"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
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

	reviewsCmd := &cobra.Command{
		Use:   "reviews",
		Short: "Import legacy course reviews from a manifest package (reviews.jsonl)",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runCourseImportReviews,
	}
	reviewsCmd.Flags().String("manifest", "", "path to the reviews manifest package (alternative to positional argument)")
	reviewsCmd.Flags().Bool("dry-run", false, "validate manifest and parse rows only, do not write to the database")
	cmd.AddCommand(reviewsCmd)
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

func runCourseImportReviews(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	manifestPath, _ := cmd.Flags().GetString("manifest")
	if manifestPath == "" && len(args) == 1 {
		manifestPath = args[0]
	}
	if manifestPath == "" {
		return fmt.Errorf("manifest path is required (pass --manifest or a positional argument)")
	}
	report, err := courseservice.ImportReviews(cmd.Context(), manifestPath, dryRun)
	if err != nil {
		return fmt.Errorf("course reviews import: %w", err)
	}
	fmt.Printf("course-import reviews: source=%s manifest=%s dryRun=%v\n",
		report.Source, report.ManifestHash, report.DryRun)
	fmt.Printf("  total=%d inserted=%d updated=%d quarantined=%d skipped=%d errors=%d\n",
		report.TotalLines, report.Inserted, report.Updated, report.Quarantined, report.Skipped, len(report.Errors))
	for _, e := range report.Errors {
		fmt.Printf("  [%s] %s\n", e.Entity, e.Reason)
	}
	return nil
}

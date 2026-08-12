package cmd

import (
	"context"
	"fmt"

	"github.com/leancodebox/GooseForum/app/service/searchservice"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "reconcile-course-search",
		Short: "Reconcile the Meilisearch course index against PostgreSQL (dry-run by default)",
		RunE:  runReconcileCourseSearch,
	}
	cmd.Flags().Bool("dry-run", true, "only report drift, do not write")
	appendCommand(cmd)
}

func runReconcileCourseSearch(cmd *cobra.Command, _ []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	fmt.Printf("Reconciling Meilisearch course index (dry-run=%v)...\n", dryRun)
	result, err := searchservice.ReconcileCourseIndex(context.Background())
	if err != nil {
		return fmt.Errorf("reconcile course search: %w", err)
	}
	fmt.Printf("Course search reconciled: %d docs in index, %d courses in PG.\n", result.IndexedDocs, result.PGCourses)
	return nil
}

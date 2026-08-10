package cmd

import (
	"context"
	"fmt"

	"github.com/leancodebox/GooseForum/app/service/searchservice"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "rebuild-course-search",
		Short: "Rebuild the Meilisearch course index from PostgreSQL",
		RunE:  runRebuildCourseSearch,
	}
	appendCommand(cmd)
}

func runRebuildCourseSearch(_ *cobra.Command, _ []string) error {
	fmt.Println("Rebuilding Meilisearch course index...")
	result, err := searchservice.BuildCourseIndex(context.Background())
	if err != nil {
		return fmt.Errorf("rebuild Meilisearch course index: %w", err)
	}
	fmt.Printf("Meilisearch course index rebuilt: processed %d courses, failed %d, batches %d.\n",
		result.ProcessedCount, result.FailedCount, result.TotalBatches)
	return nil
}

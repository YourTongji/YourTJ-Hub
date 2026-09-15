package cmd

import (
	"fmt"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "rebuild-course-search",
		Short: "Rebuild the Meilisearch course index from PostgreSQL",
		RunE:  runRebuildCourseSearch,
	}
	cmd.Flags().Bool("in-place", false, "Refresh projection fields without emptying the live index")
	appendCommand(cmd)
}

func runRebuildCourseSearch(cmd *cobra.Command, _ []string) error {
	fmt.Println("Rebuilding Meilisearch course index...")
	inPlace, err := cmd.Flags().GetBool("in-place")
	if err != nil {
		return err
	}
	build := searchservice.BuildCourseIndex
	if inPlace {
		build = searchservice.RefreshCourseIndex
	}
	result, err := build(cmd.Context())
	if err != nil {
		return fmt.Errorf("rebuild Meilisearch course index: %w", err)
	}
	fmt.Printf("Meilisearch course index rebuilt: processed %d courses, failed %d, batches %d.\n",
		result.ProcessedCount, result.FailedCount, result.TotalBatches)
	return nil
}

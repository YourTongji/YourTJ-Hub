package cmd

import (
	"fmt"

	"github.com/leancodebox/GooseForum/app/service/searchservice"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "rebuild-search-index",
		Short: "Rebuild the Meilisearch topic, user and category indexes",
		RunE:  runRebuildSearchIndex,
	}
	appendCommand(cmd)
}

func runRebuildSearchIndex(_ *cobra.Command, _ []string) error {
	fmt.Println("Rebuilding Meilisearch topic index...")
	topicResult, err := searchservice.BuildMeilisearchIndex()
	if err != nil {
		return fmt.Errorf("rebuild Meilisearch topic index: %w", err)
	}
	fmt.Printf("Meilisearch topic index rebuilt: processed %d topics.\n", topicResult.ProcessedCount)

	userResult, err := searchservice.BuildUserIndex()
	if err != nil {
		return fmt.Errorf("rebuild Meilisearch user index: %w", err)
	}
	fmt.Printf("Meilisearch user index rebuilt: processed %d users.\n", userResult.ProcessedCount)

	categoryResult, err := searchservice.BuildCategoryIndex()
	if err != nil {
		return fmt.Errorf("rebuild Meilisearch category index: %w", err)
	}
	fmt.Printf("Meilisearch category index rebuilt: processed %d categories.\n", categoryResult.ProcessedCount)
	return nil
}

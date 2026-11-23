package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func init() {
	cmd := &cobra.Command{
		Use:   "backfill-images",
		Short: "Backfill topic image_urls from first post markdown",
		RunE:  runBackfillImages,
	}
	appendCommand(cmd)
}

// runBackfillImages 保留逐项处理，但查询或保存失败必须使维护命令失败。
func runBackfillImages(cmd *cobra.Command, args []string) error {
	all, err := topics.GetLatestPublished(10000)
	if err != nil {
		return fmt.Errorf("list topics for image backfill: %w", err)
	}
	updated, failed := 0, 0
	for _, topic := range all {
		if topic == nil || topic.Id == 0 {
			continue
		}
		firstPost, err := posts.GetWithContext(cmd.Context(), topic.FirstPostId)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			failed++
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "read first post failed topic %d: %v\n", topic.Id, err)
			continue
		}
		images := markdown2html.ExtractImageURLs(firstPost.Content)
		if len(images) > 3 {
			images = images[:3]
		}
		if len(topic.ImageUrls) == len(images) && strings.Join(topic.ImageUrls, "|") == strings.Join(images, "|") {
			continue
		}
		if err := topics.UpdateImageURLs(topic.Id, images); err != nil {
			failed++
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "save failed topic %d: %v\n", topic.Id, err)
			continue
		}
		updated++
	}
	if failed > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "image backfill incomplete: updated %d/%d topics, failed %d\n", updated, len(all), failed)
		return fmt.Errorf("image backfill failed for %d/%d topics", failed, len(all))
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "backfilled image_urls for %d/%d topics\n", updated, len(all))
	return nil
}

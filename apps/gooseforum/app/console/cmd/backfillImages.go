package cmd

import (
	"fmt"
	"strings"

	"github.com/leancodebox/GooseForum/app/http/controllers/markdown2html"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "backfill-images",
		Short: "Backfill topic image_urls from first post markdown",
		Run:   runBackfillImages,
	}
	appendCommand(cmd)
}

func runBackfillImages(cmd *cobra.Command, args []string) {
	all, _ := topics.GetLatestPublished(10000)
	updated := 0
	for _, topic := range all {
		if topic == nil || topic.Id == 0 {
			continue
		}
		firstPost := posts.Get(topic.FirstPostId)
		if firstPost.Id == 0 {
			continue
		}
		images := markdown2html.ExtractImageURLs(firstPost.Content)
		if len(images) > 3 {
			images = images[:3]
		}
		if len(topic.ImageUrls) == len(images) && strings.Join(topic.ImageUrls, "|") == strings.Join(images, "|") {
			continue
		}
		topic.ImageUrls = images
		if err := topics.Save(topic); err != nil {
			fmt.Println("save failed topic", topic.Id, ":", err)
			continue
		}
		updated++
	}
	fmt.Printf("backfilled image_urls for %d/%d topics\n", updated, len(all))
}

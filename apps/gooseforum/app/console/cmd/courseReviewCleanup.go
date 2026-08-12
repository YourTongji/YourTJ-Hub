package cmd

import (
	"fmt"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "course-review-cleanup",
		Short: "Clean up course reviews past the isolation window (privacy B3)",
		RunE:  runCourseReviewCleanup,
	}
	appendCommand(cmd)
}

// runCourseReviewCleanup 手动触发课评隔离窗口清理：清空删除超过 30 天的
// deleted 课评正文并断开作者关联，行保留（status 仍为 deleted）供审计。
func runCourseReviewCleanup(_ *cobra.Command, _ []string) error {
	cleaned, err := courseservice.CleanupExpiredReviewsAll()
	if err != nil {
		return fmt.Errorf("course review cleanup: %w", err)
	}
	fmt.Printf("course review cleanup finished: cleaned=%d (content cleared, author link removed)\n", cleaned)
	return nil
}

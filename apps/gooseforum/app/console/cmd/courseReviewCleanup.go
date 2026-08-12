package cmd

import (
	"fmt"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "course-review-cleanup",
		Short: "Clean up course reviews past the deletion isolation window (B3 privacy)",
		RunE:  runCourseReviewCleanup,
	}
	appendCommand(cmd)
}

// runCourseReviewCleanup 手动触发一次课评清理（issue #175 B3）：
// 清理 status=deleted 超隔离窗口的行（清空正文、断开作者、释放占位）。
// 与 cron 定时任务同一入口，可直接同步执行。
func runCourseReviewCleanup(_ *cobra.Command, _ []string) error {
	cleaned, err := courseservice.CleanupDeletedReviews(courseservice.ReviewCleanupRetentionDays * 24)
	if err != nil {
		return fmt.Errorf("course review cleanup: %w", err)
	}
	fmt.Printf("course review cleanup finished: %d reviews anonymized.\n", cleaned)
	return nil
}

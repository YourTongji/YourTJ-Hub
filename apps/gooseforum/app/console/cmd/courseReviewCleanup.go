package cmd

import (
	"fmt"
	"time"

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
// 注意单位（spec F2-CLI）：RetentionDays 必须乘以 24*time.Hour 才是时长；
// 此前 *24 作为 time.Duration 是 720 纳秒，会把全部 deleted 行（含窗口内）
// 一并清掉。
func runCourseReviewCleanup(_ *cobra.Command, _ []string) error {
	cleaned, err := courseservice.CleanupDeletedReviews(
		time.Duration(courseservice.ReviewCleanupRetentionDays) * 24 * time.Hour)
	if err != nil {
		return fmt.Errorf("course review cleanup: %w", err)
	}
	fmt.Printf("course review cleanup finished: %d reviews anonymized.\n", cleaned)
	return nil
}

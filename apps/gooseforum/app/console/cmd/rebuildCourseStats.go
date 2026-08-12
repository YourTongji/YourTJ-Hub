package cmd

import (
	"fmt"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/course"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "rebuild-course-stats",
		Short: "Rebuild course/offering review stats projections from the review fact table",
		RunE:  runRebuildCourseStats,
	}
	appendCommand(cmd)
}

func runRebuildCourseStats(_ *cobra.Command, _ []string) error {
	if err := course.RebuildAllCourseStats(); err != nil {
		return fmt.Errorf("rebuild course stats: %w", err)
	}
	conn := db.Connect()
	var courseStatsCount int64
	if err := conn.Model(&course.CourseStatsEntity{}).Count(&courseStatsCount).Error; err != nil {
		return fmt.Errorf("count course stats: %w", err)
	}
	var offeringStatsCount int64
	if err := conn.Model(&course.OfferingStatsEntity{}).Count(&offeringStatsCount).Error; err != nil {
		return fmt.Errorf("count offering stats: %w", err)
	}
	fmt.Printf("course/offering review stats rebuilt: %d course stats rows, %d offering stats rows.\n",
		courseStatsCount, offeringStatsCount)
	return nil
}

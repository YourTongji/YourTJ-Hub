package courseservice

import (
	"context"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
)

// TaskTypeCourseStatsRebuild 是课程统计重建 worker 的任务类型前缀。
// 后台任务替换 rebuild-course-stats CLI：管理页“重建课程统计”按钮入队，worker 消费。
const TaskTypeCourseStatsRebuild = "course-stats."

// EnqueueCourseStatsRebuildTask 入队一次全量课程统计重建任务（去重：已有 pending/retrying 则跳过）。
func EnqueueCourseStatsRebuildTask() error {
	var count int64
	if err := dbconnect.Connect().Table((&taskQueue.Entity{}).TableName()).
		Where("type LIKE ?", TaskTypeCourseStatsRebuild+"%").
		Where("status IN ?", []int{taskQueue.StatusPending, taskQueue.StatusRetrying}).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return taskQueue.Create(&taskQueue.Entity{
		Type:     TaskTypeCourseStatsRebuild + "rebuild",
		TaskJson: "{}",
	})
}

// RunCourseStatsRebuildTask worker 处理：全量重建课程/offering 统计投影。
// 以 review 事实表为准重新聚合，事务内清空后重插，中途失败整体回滚。
func RunCourseStatsRebuildTask(_ context.Context, _ *taskQueue.Entity) error {
	return course.RebuildAllCourseStats()
}

// RecoverCourseStatsRebuildTasks 启动时恢复统计重建 worker 前缀下崩溃遗留的 Running 任务。
func RecoverCourseStatsRebuildTasks() error {
	return taskQueue.RecoverStaleRunning(TaskTypeCourseStatsRebuild, taskQueue.LeaseDuration)
}

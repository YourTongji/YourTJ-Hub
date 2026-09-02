package courseservice

import (
	"context"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"gorm.io/gorm"
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

// EnqueueCourseStatsRebuildTaskTx 在业务事务内入队全量课程统计重建任务
// （transaction-bound outbox：任务行与业务写入同事务提交，崩溃前不产生任务）。
// 去重：已有 pending/retrying 则跳过。合并/撤销等改变评价集合的事务内使用，
// 避免"业务已提交而任务入队失败"导致统计永久陈旧（review P2）。
func EnqueueCourseStatsRebuildTaskTx(tx *gorm.DB) error {
	var count int64
	if err := tx.Table((&taskQueue.Entity{}).TableName()).
		Where("type LIKE ?", TaskTypeCourseStatsRebuild+"%").
		Where("status IN ?", []int{taskQueue.StatusPending, taskQueue.StatusRetrying}).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return taskQueue.CreateTx(tx, &taskQueue.Entity{
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

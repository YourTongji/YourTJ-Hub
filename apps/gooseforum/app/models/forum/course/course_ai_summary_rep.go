package course

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---- AI 课程总结缓存（B7, issue #181） ----

// GetCourseAiSummary 读取课程 AI 总结缓存；无缓存时返回零值实体（Id==0）。
func GetCourseAiSummary(courseId uint64) (entity CourseAiSummaryEntity) {
	dbconnect.Connect().Table("course_ai_summary").
		Where("course_id = ?", courseId).
		First(&entity)
	return
}

// GetCourseAiSummaryTx 事务内读取课程 AI 总结缓存。
func GetCourseAiSummaryTx(tx *gorm.DB, courseId uint64) (entity CourseAiSummaryEntity, err error) {
	err = tx.Table("course_ai_summary").Where("course_id = ?", courseId).First(&entity).Error
	return
}

// UpsertCourseAiSummary 覆盖写入课程 AI 总结缓存（UPSERT，幂等）。
func UpsertCourseAiSummary(entity *CourseAiSummaryEntity) error {
	return dbconnect.Connect().Table("course_ai_summary").
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "course_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"summary_json", "model", "prompt_version", "generated_at", "status"}),
		}).
		Create(entity).Error
}

// DeleteCourseAiSummaryTx 事务内删除课程 AI 总结缓存（评价变更时随事务失效，
// 保证 summary 不会引用已删除/隐藏/修改的评价内容）。
func DeleteCourseAiSummaryTx(tx *gorm.DB, courseId uint64) error {
	return tx.Table("course_ai_summary").
		Where(queryopt.Eq("course_id", courseId)).
		Delete(&CourseAiSummaryEntity{}).Error
}

// DeleteCourseAiSummary 删除课程 AI 总结缓存（评价被清理/课程删除时失效）。
func DeleteCourseAiSummary(courseId uint64) error {
	return dbconnect.Connect().Table("course_ai_summary").
		Where(queryopt.Eq("course_id", courseId)).
		Delete(&CourseAiSummaryEntity{}).Error
}

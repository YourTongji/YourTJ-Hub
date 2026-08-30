package course

import (
	"encoding/json"
	"log/slog"

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
			DoUpdates: clause.AssignmentColumns([]string{"summary_json", "model", "prompt_version", "generated_at"}),
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

// ListCourseAiSummaryKeywords 批量读取多门课程的 AI 总结高频关键词（issue #331 R3）。
// 返回 course_id -> keywords 映射；未触发过或 summary_json 为空/无法解析的课程省略
// （对应的 map key 不存在，调用方按空处理）。供目录列表一次性取推荐卡片 #标签，避免 N+1。
func ListCourseAiSummaryKeywords(courseIds []uint64) map[uint64][]string {
	result := make(map[uint64][]string)
	if len(courseIds) == 0 {
		return result
	}
	var list []CourseAiSummaryEntity
	if err := dbconnect.Connect().Table("course_ai_summary").
		Where("course_id IN ?", courseIds).
		Where("summary_json <> ''").
		Find(&list).Error; err != nil {
		slog.Warn("ListCourseAiSummaryKeywords: 查询失败", "courseIds", courseIds, "err", err)
		return result
	}
	for _, s := range list {
		var payload struct {
			Keywords []string `json:"keywords"`
		}
		if err := json.Unmarshal([]byte(s.SummaryJson), &payload); err != nil {
			continue
		}
		if len(payload.Keywords) > 0 {
			result[s.CourseId] = payload.Keywords
		}
	}
	return result
}

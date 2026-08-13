package course

import "time"

// CourseAiSummaryEntity AI 课程总结缓存（B7，issue #181）。
// 每门课程至多一条；summary_json 存 LLM 输出的结构化总结（text 列，PG/SQLite 兼容）。
// 缓存不过期，?refresh=true 强制重生成覆盖；provider/模型切换后由刷新重建。
type CourseAiSummaryEntity struct {
	CourseId      uint64    `gorm:"column:course_id;primaryKey;not null;" json:"courseId"`
	SummaryJson   string    `gorm:"column:summary_json;type:text;not null;default:'';" json:"summaryJson"`
	Model         string    `gorm:"column:model;type:varchar(128);not null;default:'';" json:"model"`
	PromptVersion string    `gorm:"column:prompt_version;type:varchar(64);not null;default:'';" json:"promptVersion"`
	GeneratedAt   time.Time `gorm:"column:generated_at;not null;" json:"generatedAt"`
}

func (CourseAiSummaryEntity) TableName() string {
	return "course_ai_summary"
}

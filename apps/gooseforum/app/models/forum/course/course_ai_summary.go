package course

import "time"

// CourseAiSummaryEntity AI 课程总结缓存（B7，issue #181）。
// 每门课程至多一条；summary_json 存 LLM 输出的结构化总结（text 列，PG/SQLite 兼容）。
// status=generated 时 summary_json 为有效总结；status=insufficient 表示已评估过
// 但评价不足（无 summary_json），评价不变时不再重复生成（前端 check 模式直读）。
// 缓存语义（已接受）：不过期、不因 provider/模型切换自动失效，仅由评价写路径
// （create/update/delete/hide/show/导入）DeleteCourseAiSummaryTx 失效；
// 课程/开课实例隐藏时 GetAiSummary/CheckAiSummary 入口直接 404，不会对外服务。
// ?refresh=true 强制重生成覆盖。status 列低基数（2 值）且查询全部按 course_id
// 主键访问，不建索引（review nit）。
type CourseAiSummaryEntity struct {
	CourseId      uint64    `gorm:"column:course_id;primaryKey;not null;" json:"courseId"`
	SummaryJson   string    `gorm:"column:summary_json;type:text;not null;default:'';" json:"summaryJson"`
	Model         string    `gorm:"column:model;type:varchar(128);not null;default:'';" json:"model"`
	PromptVersion string    `gorm:"column:prompt_version;type:varchar(64);not null;default:'';" json:"promptVersion"`
	GeneratedAt   time.Time `gorm:"column:generated_at;not null;" json:"generatedAt"`
	// Status 行状态：generated（有效总结，默认，兼容存量行）/ insufficient（评价不足已评估）。
	// AutoMigrate 对存量库加列时以 DEFAULT 'generated' 回填，旧缓存行语义不变。
	Status string `gorm:"column:status;type:varchar(16);not null;default:'generated';comment:summary row status;" json:"status"`
}

// AI 总结缓存行状态。
const (
	AiSummaryRowStatusGenerated    = "generated"
	AiSummaryRowStatusInsufficient = "insufficient"
)

func (CourseAiSummaryEntity) TableName() string {
	return "course_ai_summary"
}

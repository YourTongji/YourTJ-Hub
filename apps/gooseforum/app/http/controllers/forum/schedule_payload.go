package forum

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

// ScheduleProps 排课器页 props（对应 course.schedule）。
// 排课器为纯客户端交互工具，课程数据走 /api/pk/* JSON API 异步加载；
// 节次作息表（第 N 节开始/结束时间）由管理端配置，SSR 直接注入 props，
// 未配置时回退内置默认作息（defaultconfig 12 节表）。
type ScheduleProps struct {
	SectionTimes []pageConfig.ScheduleSectionTime `json:"sectionTimes"`
}

package forum

// ScheduleProps 排课器页 props（对应 course.schedule）。
// 排课器为纯客户端交互工具，数据全部走 /api/pk/* JSON API 异步加载，SSR 仅提供空壳
// （与 CourseReviewModerationPageProps 同款空壳模式）。
type ScheduleProps struct{}

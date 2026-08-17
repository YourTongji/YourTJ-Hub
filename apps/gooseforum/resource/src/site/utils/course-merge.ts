// 无限滚动合并工具：按课程 id 去重追加，避免翻页边界重复渲染。
// 与首页 mergeTopics 同构（HomePage.vue），抽出为纯函数便于单测。
import type { CourseSummaryPayload } from '@gooseforum/client'

export function mergeCourses(current: CourseSummaryPayload[], incoming: CourseSummaryPayload[]): CourseSummaryPayload[] {
  const seen = new Set(current.map((course) => course.id))
  return [...current, ...incoming.filter((course) => !seen.has(course.id))]
}

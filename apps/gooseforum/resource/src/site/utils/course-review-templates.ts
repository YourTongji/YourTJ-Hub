// 课程评价写评模板（参考上游 YourTJCourse-Serverless 的 TemplateSelector）。
// 模板正文为 Markdown 结构提示，用户选择后可继续编辑；正文标题沿用中文
// （平台以中文为主，正文属可编辑内容而非 UI 文案），模板名/描述走 i18n 键。
export interface CourseReviewTemplate {
  id: string
  nameKey: string
  descriptionKey: string
  content: string
}

export const COURSE_REVIEW_TEMPLATES: CourseReviewTemplate[] = [
  {
    id: 'comprehensive',
    nameKey: 'courseDetailPage.templates.comprehensive.name',
    descriptionKey: 'courseDetailPage.templates.comprehensive.description',
    content: '## 课程内容\n\n## 教学方式\n\n## 作业与考核\n\n## 收获与建议\n',
  },
  {
    id: 'quick',
    nameKey: 'courseDetailPage.templates.quick.name',
    descriptionKey: 'courseDetailPage.templates.quick.description',
    content: '**总体评价：**\n\n**优点：**\n-\n\n**缺点：**\n-\n\n**建议：**\n',
  },
  {
    id: 'teacher-focused',
    nameKey: 'courseDetailPage.templates.teacherFocused.name',
    descriptionKey: 'courseDetailPage.templates.teacherFocused.description',
    content: '## 教学态度\n\n## 授课风格\n\n## 师生互动\n\n## 总体印象\n',
  },
  {
    id: 'exam-focused',
    nameKey: 'courseDetailPage.templates.examFocused.name',
    descriptionKey: 'courseDetailPage.templates.examFocused.description',
    content: '## 考试形式\n\n## 考试难度\n\n## 备考建议\n\n## 给分情况\n',
  },
  {
    id: 'workload',
    nameKey: 'courseDetailPage.templates.workload.name',
    descriptionKey: 'courseDetailPage.templates.workload.description',
    content: '## 课时安排\n\n## 作业量\n\n## 项目/实验\n\n## 时间投入\n',
  },
  {
    id: 'blank',
    nameKey: 'courseDetailPage.templates.blank.name',
    descriptionKey: 'courseDetailPage.templates.blank.description',
    content: '',
  },
]

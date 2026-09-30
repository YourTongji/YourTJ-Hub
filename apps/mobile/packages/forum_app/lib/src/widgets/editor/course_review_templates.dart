/// Markdown templates kept in sync with the Web course review templates.
class CourseReviewTemplate {
  const CourseReviewTemplate(this.id, this.content);

  final String id;
  final String content;
}

const List<CourseReviewTemplate> courseReviewTemplates = <CourseReviewTemplate>[
  CourseReviewTemplate(
    'comprehensive',
    '## 课程内容\n\n## 教学方式\n\n## 作业与考核\n\n## 收获与建议\n',
  ),
  CourseReviewTemplate(
    'quick',
    '**总体评价：**\n\n**优点：**\n-\n\n**缺点：**\n-\n\n**建议：**\n',
  ),
  CourseReviewTemplate(
    'teacher-focused',
    '## 教学态度\n\n## 授课风格\n\n## 师生互动\n\n## 总体印象\n',
  ),
  CourseReviewTemplate(
    'exam-focused',
    '## 考试形式\n\n## 考试难度\n\n## 备考建议\n\n## 给分情况\n',
  ),
  CourseReviewTemplate(
    'workload',
    '## 课时安排\n\n## 作业量\n\n## 项目/实验\n\n## 时间投入\n',
  ),
  CourseReviewTemplate('blank', ''),
];

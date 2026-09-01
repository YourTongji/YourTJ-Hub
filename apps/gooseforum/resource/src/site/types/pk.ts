// PK 排课器（/schedule）数据类型定义。
//
// 对齐上游 YourTJCourse-Serverless scheduler（utils/myInterface.ts）的数据契约，
// 与 Epic #172 PRD §5.4.2（PK 数据域）/ §5.4.4（PK 13 端点）的字段语义。
// 后端端点契约见 #187；前端消费的字段以本文件为唯一来源。

/** 一次上课安排（一个时间段）。对应 PkCourseDetail.arrangementInfo 元素。 */
export interface PkArrangement {
  /** 人类可读文本，如 "[1-8周] 周一第3-4节 同济楼A201" */
  arrangementText: string
  /** 星期 1-7 */
  occupyDay: number
  /** 节次数组（连续），如 [3,4]，范围 1-12 */
  occupyTime: number[]
  /** 周次数组（展开），如 [1,2,...,8] */
  occupyWeek: number[]
  /** 教室 */
  occupyRoom: string
  /** "教师名(工号)"，多教师逗号分隔 */
  teacherAndCode: string
}

/** 教师。对应 PkTeacher。 */
export interface PkTeacher {
  teacherName: string
  teacherCode: string
}

/** 课程详情 = 一个教学班（一门课的一个授课实例）。对应 PkCourseDetail。 */
export interface PkCourseDetail {
  arrangementInfo: PkArrangement[]
  /** 校区（多校区用"、"拼接） */
  campus: string
  /** 教学班课号（含班号后缀，如 "122004.01"） */
  code: string
  /** 教学班 id（pk_course_detail.id）：P13 课评摘要 by-offering 精准定位直查键。 */
  teachingClassId?: number
  /** 排他标志：仅专业课/必修为 true（P5 courses-by-major 语义） */
  isExclusive?: boolean
  /** 0 未选 / 1 备选 / 2 已选 */
  status?: number
  teachers: PkTeacher[]
  teachingLanguage: string
}

/** 课程信息（必修列表 / 选修列表 / 搜索结果通用）。对应 courseInfo。 */
export interface PkCourse {
  courseName: string
  /** 保留课程名（CSV/XLS 导出用） */
  courseNameReserved: string
  /** 基础课号（无班号） */
  courseCode: string
  /** '必' 必修 / '选' 选修 / '跨' 跨学科 / '查' 搜索 */
  courseType: string
  /** 开课院系 */
  faculty?: string
  /** 学分（浮点，x.0 / x.5） */
  credit: number
  courseNature?: string[]
  campus?: string[]
  status: number
  /** 注意：这里是 string[]（与 PkStagedCourse.teacher 的 PkTeacher[] 不同） */
  teacher: string[]
  courseDetail: PkCourseDetail[]
  /** 年级（必修课按年级分组展示） */
  grade?: number
}

/** 备选 / 已选课程。对应 stagedCourse。 */
export interface PkStagedCourse {
  courseCode: string
  courseName: string
  courseNameReserved: string
  credit: number
  courseType: string
  courseNature?: string[]
  teacher: PkTeacher[]
  status: number
  courseDetail: PkCourseDetail[]
}

/** 学期→年级→专业 选择三元组。对应 baseInfoTriplet。 */
export interface PkMajorSelection {
  calendarId: number | undefined
  grade: number | undefined
  major: string | undefined
}

/** 选修课类型。对应 optionalCourseType。 */
export interface PkOptionalType {
  courseLabelId: number
  courseLabelName: string
}

/** occupied 12×7 三维数组的单元（一个格子里的一个课程）。对应 occupyCell。 */
export interface PkOccupyCell {
  code: string
  courseName: string
  occupyWeek: number[]
}

/** 课表条目（平铺，渲染时转网格）。对应 courseOnTable。
 * v2：结构化字段（occupyWeek/teacherAndCode/arrangementText/occupyRoom）随重建写入，
 * 渲染不再从 showText 反解（showText 保留用于详情卡兜底展示）。 */
export interface PkCourseOnTable {
  showText: string
  courseName: string
  code: string
  occupyTime: number[]
  occupyDay: number
  /** 该安排的周次数组（v2；单周视图过滤用，sanitize 容忍缺失）。 */
  occupyWeek?: number[]
  teacherAndCode?: string
  arrangementText?: string
  occupyRoom?: string
}

/** 鼠标点击的课程。对应 clickedCourseInfo。 */
export interface PkClickedCourse {
  courseCode: string
  courseName: string
  teacherCode?: string
  teacherName?: string
}

/** CSV 一行（一门课的一个时间段）。对应 csvCourse。 */
export interface PkCsvCourse {
  courseName: string
  occupyDay: number | string
  start: number | string
  end: number | string
  teacherName: string
  occupyRoom: string
  occucpyWeek: string
}

/** XLS 一行（一门课）。对应 xlsCourse。 */
export interface PkXlsCourse {
  code: string
  courseName: string
  teacherName: string
}

/** P1 /api/pk/calendars：最近 8 个学期。
 * startDate/endDate 为可选学期起止日期（YYYY-MM-DD；后端未配置时为 null），
 * 排课器「当前周次」定位与学期日期条展示用。 */
export interface PkCalendar {
  calendarId: number
  calendarName: string
  startDate?: string | null
  endDate?: string | null
}

/** 自定义占位事件（v2 多方案）：用户手工标注的不可用时段（如「有事」）。 */
export interface PkCustomEvent {
  id: string
  label: string
  /** 星期 1-7 */
  day: number
  /** 节次集合（1-12） */
  sections: number[]
  /** 周次集合 */
  weeks: number[]
}

/** 排课方案（v2）：每套方案独立持有已选/备选课程与自定义占位。 */
export interface PkPlan {
  id: string
  name: string
  createdAt: number
  stagedCourses: PkStagedCourse[]
  /** 已选班级课号（含班号） */
  selectedCourses: string[]
  customEvents: PkCustomEvent[]
}

/** 周次视图状态（持久化）：week 为 null 表示「全部周次」堆叠视图。 */
export interface PkWeekView {
  week: number | null
  /** 「当前周次」开关（需学期起始日期；无日期时 UI 禁用） */
  useCurrent: boolean
}

/** P3 /api/pk/grades：某学期可选年级（上游 gradeList 为纯数字数组）。 */
export type PkGrade = number

/** P4 /api/pk/majors：年级→专业。 */
export interface PkMajor {
  code: string
  name: string
}

/** P2 /api/pk/campuses、/faculties 字典项。 */
export interface PkDictItem {
  code: string
  name: string
}

/** P11 /api/pk/latest-update：fetchlog 最近同步日期。 */
export interface PkLatestUpdate {
  latestSyncAt: string | null
  msg?: string
}

/** P10 /api/pk/courses-by-time 响应（timeslot 未就绪时降级 LIKE，auxiliaryReady=false）。 */
export interface PkCoursesByTimeResult {
  courses: PkCourse[]
  /** false = timeslot 索引未就绪，结果来自 arrangeInfoText LIKE 降级 */
  auxiliaryReady: boolean
}

/** P12 /api/pk/course-info-sync 请求参数。 */
export interface PkCourseInfoSyncInput {
  calendarId: number
  /** 专业课代码（isExclusive=true 的已选） */
  majorCourseCodes: string[]
  /** 其他课程代码（通识/选修已选） */
  otherCourseCodes: string[]
  majorInfo: { grade: number; code: string }
}

/** P12 /api/pk/course-info-sync 响应：{ courseCode: [courseDetail] } 字典。 */
export type PkCourseInfoSyncResult = Record<string, PkCourseDetail[]>

/** P13 /api/pk/course-review-brief 请求参数。 */
export interface PkCourseReviewBriefInput {
  courseCode: string
  teacherName: string
  /** 可选：限定教学班课号只在该学期内匹配（跨学期班号复用时不串学期）。 */
  calendarId?: number
  /** 可选：教学班直查键（course_offering.teaching_class_id）；提供时按班精准定位，缺省回退 courseCode+teacherName 匹配。 */
  teachingClassId?: number
}

/** P13 /api/pk/course-review-brief 响应（复用课评 API 语义）。 */
export interface PkCourseReviewBrief {
  /** Hub 课程目录主键（/courses/:courseId 详情页跳转用）；未匹配课评目录时为 0。 */
  courseId?: number
  ratingAvg?: number | null
  reviewCount: number
  /** 各教学班的 offering 级课评摘要（class_code 匹配；无匹配时为空数组）。 */
  classes?: PkReviewBriefClass[]
  reviews?: Array<{
    id: number
    rating?: number
    content: string
    helpfulCount?: number
    authorName?: string
  }>
}

/** P13 教学班级课评摘要项：按 Hub offering（class_code 匹配）聚合。 */
export interface PkReviewBriefClass {
  /** 教学班课号，与 course_offering.class_code 对齐（如 11000101）。 */
  classCode: string
  /** Hub 开课实例主键，供 /courses/:courseId?offeringId=:offeringId 聚焦该班评价。 */
  offeringId: number
  teachers: string[]
  ratingAvg?: number | null
  reviewCount: number
}

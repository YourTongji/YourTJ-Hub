// PK 排课器 API 客户端（Epic #172 PRD §5.4.4 的 13 端点）。
//
// 端点契约见 Issue #187；前端按统一信封 `{code, msg, data}` 对接：
// 成功 code === 0（Hub 惯例）或 200（上游一系统惯例），负载在 `data`。
// 所有函数走 `readPkResponse`，错误抛 `Error`（msg 优先，fallback 兜底）。

import { i18n } from './i18n'
import type {
  PkCalendar,
  PkCourse,
  PkCourseDetail,
  PkCourseInfoSyncInput,
  PkCourseInfoSyncResult,
  PkCourseReviewBrief,
  PkCourseReviewBriefInput,
  PkCoursesByTimeResult,
  PkDictItem,
  PkGrade,
  PkLatestUpdate,
  PkMajor,
  PkOptionalType,
} from '@/site/types/pk'

// ---- OpenAPI 契约 wire 形状（与 packages/api-contract 对齐；前端类型见 site/types/pk.ts）----

/** P2 校区契约项：campusId/campusName。 */
interface PkCampusWireItem {
  campusId: string
  campusName: string
}

/** P2 院系契约项：facultyId/facultyName。 */
interface PkFacultyWireItem {
  facultyId: string
  facultyName: string
}

/** P5 courses-by-major 契约项：教学班字段为 courses（前端 PkCourse 用 courseDetail）。 */
interface PkCourseByMajorWireItem {
  courseCode: string
  courseName: string
  faculty: string
  facultyI18n: string
  credit: number
  grade: number
  courseNature: string[]
  courses: PkCourseDetail[]
}

/** P7 courses-by-nature 契约项：data 为分组数组，courses 元素带课程信息。 */
interface PkNatureCourseWireItem {
  campus: string[]
  courseCode: string
  courseName: string
  faculty: string
  facultyI18n: string
  credit: number
  courseLabelName: string
  crossDiscipline: boolean
}

interface PkCourseByNatureWireItem {
  courseLabelId: number
  courseLabelIds: number[]
  courseLabelName: string
  crossDiscipline: boolean
  courses: PkNatureCourseWireItem[]
}

/** P9 course-search 契约：data = { courses, sizeLimit }。 */
interface PkSearchCourseWireItem {
  courseCode: string
  courseName: string
  faculty: string
  facultyI18n: string
  courseNature: string[]
  campus: string[]
  campus_list: string[]
  credit: number
}

interface PkSearchResultWire {
  courses: PkSearchCourseWireItem[]
  sizeLimit: number
}

interface PkEnvelope<T> {
  code: number
  msg?: string
  data?: T
}

function t(key: string): string {
  return i18n.global.t(key)
}

/** 解析 PK 信封：成功（code 0/200）返回 data，失败抛 Error。 */
async function readPkResponse<T>(response: Response, fallback: string): Promise<T> {
  const data = (await response.json().catch(() => undefined)) as PkEnvelope<T> | undefined
  if (response.status === 429) {
    throw new Error(data?.msg || fallback)
  }
  const code = data?.code
  if (code !== undefined && code !== 0 && code !== 200) {
    throw new Error(data?.msg || fallback)
  }
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`)
  }
  if (!data) {
    throw new Error(fallback)
  }
  return data.data as T
}

async function postPk<T>(url: string, body: unknown, fallback: string): Promise<T> {
  let response: Response
  try {
    response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
  } catch {
    throw new Error(fallback)
  }
  return readPkResponse<T>(response, fallback)
}

async function getPk<T>(url: string, fallback: string): Promise<T> {
  let response: Response
  try {
    response = await fetch(url)
  } catch {
    throw new Error(fallback)
  }
  return readPkResponse<T>(response, fallback)
}

// ---- 端点函数（P1-P13）----

/** P1 最近 8 个学期。 */
export function getPkCalendars(): Promise<PkCalendar[]> {
  return getPk<PkCalendar[]>('/api/pk/calendars', t('api.pkCalendarsFailed'))
}

/** P2 校区列表。契约 data = [{ campusId, campusName }]，映射为 {code,name}。 */
export function getPkCampuses(): Promise<PkDictItem[]> {
  return getPk<PkCampusWireItem[]>('/api/pk/campuses', t('api.pkCampusesFailed')).then((list) =>
    list.map((item) => ({ code: item.campusId, name: item.campusName })),
  )
}

/** P2 院系列表。契约 data = [{ facultyId, facultyName }]，映射为 {code,name}。 */
export function getPkFaculties(): Promise<PkDictItem[]> {
  return getPk<PkFacultyWireItem[]>('/api/pk/faculties', t('api.pkFacultiesFailed')).then((list) =>
    list.map((item) => ({ code: item.facultyId, name: item.facultyName })),
  )
}

/** P3 某学期可选年级。契约 data = { gradeList: [...] }。 */
export function getPkGrades(calendarId: number): Promise<PkGrade[]> {
  return postPk<{ gradeList: PkGrade[] }>('/api/pk/grades', { calendarId }, t('api.pkGradesFailed')).then(
    (result) => result.gradeList,
  )
}

/** P4 年级→专业。 */
export function getPkMajors(grade: number, calendarId: number): Promise<PkMajor[]> {
  return postPk<PkMajor[]>('/api/pk/majors', { grade, calendarId }, t('api.pkMajorsFailed'))
}

/** P5 专业课表（必修来源，验收标准 1）。契约教学班字段为 courses，映射为 courseDetail。 */
export function getPkCoursesByMajor(
  grade: number,
  code: string,
  calendarId: number,
): Promise<PkCourse[]> {
  return postPk<PkCourseByMajorWireItem[]>(
    '/api/pk/courses-by-major',
    { grade, code, calendarId },
    t('api.pkCoursesByMajorFailed'),
  ).then((list) =>
    list.map((item) => ({
      courseCode: item.courseCode,
      courseName: item.courseName,
      courseNameReserved: item.courseName,
      courseType: '必',
      faculty: item.faculty,
      credit: item.credit,
      courseNature: item.courseNature,
      status: 0,
      teacher: [],
      courseDetail: item.courses,
      grade: item.grade,
    })),
  )
}

/** P6 通识/选修类型（coursenature_by_calendar 优先）。 */
export function getPkOptionalTypes(calendarId: number): Promise<PkOptionalType[]> {
  return postPk<PkOptionalType[]>('/api/pk/optional-types', { calendarId }, t('api.pkOptionalTypesFailed'))
}

/** P7 按性质查课程。契约 data 为分组数组，扁平化为课程列表并带上性质标签。 */
export function getPkCoursesByNature(calendarId: number, ids: number[]): Promise<PkCourse[]> {
  return postPk<PkCourseByNatureWireItem[]>(
    '/api/pk/courses-by-nature',
    { calendarId, ids },
    t('api.pkCoursesByNatureFailed'),
  ).then((groups) =>
    groups.flatMap((group) =>
      group.courses.map((item) => ({
        courseCode: item.courseCode,
        courseName: item.courseName,
        courseNameReserved: item.courseName,
        courseType: '选',
        faculty: item.faculty,
        credit: item.credit,
        courseNature: [group.courseLabelName],
        campus: item.campus,
        status: 0,
        teacher: [],
        courseDetail: [],
      })),
    ),
  )
}

/** P8 批量课程详情字典（选修/备选来源，验收标准 1）。 */
export function getPkCourseDetails(
  calendarId: number,
  courseCodes: string[],
): Promise<Record<string, PkCourseDetail[]>> {
  return postPk<Record<string, PkCourseDetail[]>>(
    '/api/pk/course-details',
    { calendarId, courseCodes },
    t('api.pkCourseDetailsFailed'),
  )
}

/** P9 高级检索（搜索来源，验收标准 1）。 */
export interface PkCourseSearchInput {
  calendarId: number
  courseName?: string
  courseCode?: string
  teacherCode?: string
  teacherName?: string
  campus?: string
  faculty?: string
}

export function searchPkCourses(input: PkCourseSearchInput): Promise<PkCourse[]> {
  return postPk<PkSearchResultWire>('/api/pk/course-search', input, t('api.pkCourseSearchFailed')).then((result) =>
    result.courses.map((item) => ({
      courseCode: item.courseCode,
      courseName: item.courseName,
      courseNameReserved: item.courseName,
      courseType: '查',
      faculty: item.faculty,
      credit: item.credit,
      courseNature: item.courseNature,
      campus: item.campus,
      status: 0,
      teacher: [],
      courseDetail: [],
    })),
  )
}

/** P10 时间段查课（timeslot 未就绪时 LIKE 降级 + auxiliaryReady:false）。 */
export function getPkCoursesByTime(
  calendarId: number,
  day: number,
  section: number,
): Promise<PkCoursesByTimeResult> {
  return postPk<PkCoursesByTimeResult>(
    '/api/pk/courses-by-time',
    { calendarId, day, section },
    t('api.pkCoursesByTimeFailed'),
  )
}

/** P11 数据过期校验。契约 data 为裸字符串（YYYY-MM-DD）或 null。 */
export function getPkLatestUpdate(): Promise<PkLatestUpdate> {
  return getPk<string | null>('/api/pk/latest-update', t('api.pkLatestUpdateFailed')).then((value) => ({
    latestSyncAt: value,
  }))
}

/** P12 增量同步最新（保留已选；isExclusive 仅 major 课程回传）。 */
export function syncPkCourseInfo(input: PkCourseInfoSyncInput): Promise<PkCourseInfoSyncResult> {
  return postPk<PkCourseInfoSyncResult>('/api/pk/course-info-sync', input, t('api.pkCourseInfoSyncFailed'))
}

/** P13 排课器弹窗课评摘要（复用课评 API 语义）。 */
export function getPkCourseReviewBrief(input: PkCourseReviewBriefInput): Promise<PkCourseReviewBrief> {
  const query = new URLSearchParams({ courseCode: input.courseCode })
  if (input.teacherName) query.set('teacherName', input.teacherName)
  if (input.calendarId) query.set('calendarId', String(input.calendarId))
  if (input.teachingClassId) query.set('teachingClassId', String(input.teachingClassId))
  return getPk<PkCourseReviewBrief>(
    `/api/pk/course-review-brief?${query.toString()}`,
    t('api.pkCourseReviewBriefFailed'),
  )
}

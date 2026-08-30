// 排课器组合式 store（模块级单例）—— v2 多方案 + 容忍式冲突。
//
// 数据模型（对齐 USTC 排课器交互模型）：
// - `pk.plans`（PkPlan[]）+ `pk.activePlanId` 是课程数据的唯一持久化来源；每套方案
//   独立持有已选/备选课程（stagedCourses/selectedCourses）与自定义占位事件。
// - `occupied` / `timeTableData` 不再持久化：由 rebuildScheduleFromStaged 从方案数据
//   派生（加载/切换方案/同步/增删课程时重建），压缩 localStorage 占用。
// - v1 迁移：存在旧键（pk.stagedCourses / pk.selectedCourses）且无 pk.plans 时，
//   包装为单个「方案一」并激活；旧键只读不删（回滚到旧版本仍读到迁移前快照）。
// - 冲突语义（容忍式）：stageCourse 总是入表，返回入表前的冲突列表由 UI 标注
//   （课表 ⚠ / 列表红标 / 统计计数共用 deriveConflicts 同一判据），不再弹窗阻断。
// - 学期/年级/专业任一变更清空所有方案（防跨学期污染，沿用上游语义）。

import { reactive } from 'vue'
import { i18n } from '@/runtime/i18n'
import {
  createEmptyOccupied,
  deriveConflicts,
  findConflicts,
  getCourseBaseCode,
  CUSTOM_EVENT_CODE_PREFIX,
  insertOccupied,
  isClassOfCourse,
  isSameCourse,
  type PkConflictItem,
} from '@/site/utils/pkConflict'
import { MAX_WEEK, maxRowsForCalendar } from '@/site/utils/pkArrange'
import type {
  PkArrangement,
  PkCalendar,
  PkClickedCourse,
  PkCourse,
  PkCourseDetail,
  PkCourseOnTable,
  PkCustomEvent,
  PkMajorSelection,
  PkOccupyCell,
  PkOptionalType,
  PkPlan,
  PkStagedCourse,
  PkTeacher,
  PkWeekView,
} from '@/site/types/pk'

/** localStorage 键（带 pk. 前缀避免与论坛其他状态冲突）。 */
const STORAGE_KEYS = {
  majorSelected: 'pk.majorSelected',
  /** v1 旧键（只读迁移源，不再写入；回滚兼容）。 */
  legacyStagedCourses: 'pk.stagedCourses',
  legacySelectedCourses: 'pk.selectedCourses',
  /** v2 键。 */
  plans: 'pk.plans',
  activePlanId: 'pk.activePlanId',
  weekView: 'pk.weekView',
  updateTime: 'pk.updateTime',
} as const

export const COURSE_STATUS = {
  UNSELECTED: 0,
  STAGED: 1,
  SELECTED: 2,
} as const

interface CommonLists {
  /** 必修课（按年级分组展示） */
  compulsoryCourses: PkCourse[]
  optionalTypes: PkOptionalType[]
  /** 选修课（按类型分组） */
  optionalCourses: PkCourse[]
  searchCourses: PkCourse[]
  /** 备选课程池（当前激活方案的镜像，与 plan.stagedCourses 同引用） */
  stagedCourses: PkStagedCourse[]
  /** 已选班级课号（当前激活方案的镜像，与 plan.selectedCourses 同引用） */
  selectedCourses: string[]
}

interface ScheduleState {
  majorSelected: PkMajorSelection
  plans: PkPlan[]
  activePlanId: string
  commonLists: CommonLists
  clickedCourseInfo: PkClickedCourse
  /** 派生态：由激活方案重建（不持久化）。 */
  occupied: PkOccupyCell[][][]
  timeTableData: PkCourseOnTable[]
  weekView: PkWeekView
  /** 学期字典（含起止日期；会话缓存不持久化，MajorSelector 加载后写入）。 */
  calendars: PkCalendar[]
  flags: {
    majorNotChanged: boolean
    isDataOutdated: boolean
  }
  updateTime: string
  latestUpdateTime: string
}

function createEmptyCommonLists(): CommonLists {
  return {
    compulsoryCourses: [],
    optionalTypes: [],
    optionalCourses: [],
    searchCourses: [],
    stagedCourses: [],
    selectedCourses: [],
  }
}

function createInitialState(): ScheduleState {
  const plan = createEmptyPlan()
  return {
    majorSelected: { calendarId: undefined, grade: undefined, major: undefined },
    plans: [plan],
    activePlanId: plan.id,
    commonLists: createEmptyCommonLists(),
    clickedCourseInfo: { courseCode: '', courseName: '', teacherCode: '', teacherName: '' },
    occupied: createEmptyOccupied(),
    timeTableData: [],
    weekView: { week: null, useCurrent: false },
    calendars: [],
    flags: { majorNotChanged: false, isDataOutdated: false },
    updateTime: '',
    latestUpdateTime: '',
  }
}

// ---- id 生成 ----

let planSeq = 0
function genId(prefix: string): string {
  planSeq += 1
  return `${prefix}_${Date.now().toString(36)}_${planSeq.toString(36)}`
}

function nextPlanName(): string {
  return i18n.global.t('schedule.planDefaultName', { n: state.plans.length + 1 })
}

function createEmptyPlan(name?: string): PkPlan {
  return {
    id: genId('plan'),
    name: name ?? i18n.global.t('schedule.planDefaultName', { n: 1 }),
    createdAt: Date.now(),
    stagedCourses: [],
    selectedCourses: [],
    customEvents: [],
  }
}

// ---- sanitize（损坏即清理/回退）----

function safeParseJson<T = unknown>(value: string | null): T | undefined {
  if (!value) return undefined
  try {
    return JSON.parse(value) as T
  } catch {
    return undefined
  }
}

function ensureArray<T = unknown>(value: unknown): T[] {
  return Array.isArray(value) ? (value as T[]) : []
}

function normalizeStringList(value: unknown): string[] {
  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed || trimmed === '[]' || trimmed === '[""]') return []
    if (trimmed.startsWith('[') && trimmed.endsWith(']')) {
      const parsed = safeParseJson<unknown>(trimmed)
      if (parsed !== undefined) return normalizeStringList(parsed)
    }
    return trimmed.split(',').map((item) => item.trim()).filter(Boolean)
  }
  if (Array.isArray(value)) {
    return value.flatMap((item) => normalizeStringList(item)).map((item) => item.trim()).filter(Boolean)
  }
  return []
}

function normalizeCredit(value: unknown): number {
  const n = typeof value === 'number' ? value : typeof value === 'string' ? Number(value.trim()) : NaN
  return Number.isFinite(n) ? n : 0
}

function sanitizeTeachers(raw: unknown): PkTeacher[] {
  return ensureArray<Record<string, unknown>>(raw)
    .map((item) => ({
      teacherName: typeof item?.teacherName === 'string' ? item.teacherName : '',
      teacherCode: typeof item?.teacherCode === 'string' ? item.teacherCode : '',
    }))
    .filter((item) => item.teacherName || item.teacherCode)
}

function sanitizeArrangementInfo(raw: unknown): PkArrangement[] {
  return ensureArray<Record<string, unknown>>(raw)
    .map((item) => ({
      arrangementText: typeof item?.arrangementText === 'string' ? item.arrangementText : '',
      occupyDay: typeof item?.occupyDay === 'number' ? item.occupyDay : 0,
      occupyTime: ensureArray<number>(item?.occupyTime).filter(
        (slot) => typeof slot === 'number' && slot >= 1 && slot <= 12,
      ),
      occupyWeek: ensureArray<number>(item?.occupyWeek).filter(
        (week) => typeof week === 'number' && week >= 1 && week <= MAX_WEEK,
      ),
      occupyRoom: typeof item?.occupyRoom === 'string' ? item.occupyRoom : '',
      teacherAndCode: typeof item?.teacherAndCode === 'string' ? item.teacherAndCode : '',
    }))
    .filter((item) => item.occupyDay >= 1 && item.occupyDay <= 7 && item.occupyTime.length > 0)
}

function sanitizeCourseDetail(raw: unknown): PkCourseDetail[] {
  return ensureArray<Record<string, unknown>>(raw)
    .map((detail) => ({
      arrangementInfo: sanitizeArrangementInfo(detail?.arrangementInfo),
      campus: normalizeStringList(detail?.campus).join('、'),
      code: typeof detail?.code === 'string' ? detail.code : '',
      isExclusive: typeof detail?.isExclusive === 'boolean' ? detail.isExclusive : undefined,
      status: typeof detail?.status === 'number' ? detail.status : 0,
      teachers: sanitizeTeachers(detail?.teachers),
      teachingLanguage: typeof detail?.teachingLanguage === 'string' ? detail.teachingLanguage : '',
    }))
    .filter((detail) => detail.code)
}

function sanitizeMajorSelected(value: unknown): PkMajorSelection {
  const input = value && typeof value === 'object' ? (value as Record<string, unknown>) : {}
  return {
    calendarId: typeof input.calendarId === 'number' ? input.calendarId : undefined,
    grade: typeof input.grade === 'number' ? input.grade : undefined,
    major: typeof input.major === 'string' ? input.major : undefined,
  }
}

function sanitizeStagedCourse(raw: unknown): PkStagedCourse {
  const input = raw && typeof raw === 'object' ? (raw as Record<string, unknown>) : {}
  return {
    courseCode: typeof input.courseCode === 'string' ? input.courseCode : '',
    courseName: typeof input.courseName === 'string' ? input.courseName : '',
    courseNameReserved: typeof input.courseNameReserved === 'string' ? input.courseNameReserved : '',
    credit: normalizeCredit(input.credit),
    courseType: typeof input.courseType === 'string' ? input.courseType : '',
    courseNature: normalizeStringList(input.courseNature),
    teacher: sanitizeTeachers(input.teacher),
    status: typeof input.status === 'number' ? input.status : 0,
    courseDetail: sanitizeCourseDetail(input.courseDetail),
  }
}


function sanitizeCustomEvent(raw: unknown): PkCustomEvent {
  const input = raw && typeof raw === 'object' ? (raw as Record<string, unknown>) : {}
  return {
    id: typeof input.id === 'string' ? input.id : genId('evt'),
    label: typeof input.label === 'string' ? input.label : '',
    day: typeof input.day === 'number' && input.day >= 1 && input.day <= 7 ? input.day : 0,
    sections: [...new Set(ensureArray<number>(input.sections))]
      .filter((sec) => typeof sec === 'number' && sec >= 1 && sec <= 12)
      .sort((a, b) => a - b),
    weeks: [...new Set(ensureArray<number>(input.weeks))]
      .filter((week) => typeof week === 'number' && week >= 1 && week <= MAX_WEEK)
      .sort((a, b) => a - b),
  }
}

function sanitizePlan(raw: unknown): PkPlan {
  const input = raw && typeof raw === 'object' ? (raw as Record<string, unknown>) : {}
  const plan: PkPlan = {
    id: typeof input.id === 'string' && input.id ? input.id : genId('plan'),
    name: typeof input.name === 'string' && input.name.trim() ? input.name.trim() : i18n.global.t('schedule.planDefaultName', { n: 1 }),
    createdAt: typeof input.createdAt === 'number' ? input.createdAt : Date.now(),
    stagedCourses: ensureArray<unknown>(input.stagedCourses).map(sanitizeStagedCourse).filter((c) => c.courseCode),
    selectedCourses: ensureArray<unknown>(input.selectedCourses).filter((item): item is string => typeof item === 'string'),
    customEvents: ensureArray<unknown>(input.customEvents).map(sanitizeCustomEvent).filter((ev) => ev.day >= 1 && ev.sections.length > 0),
  }
  return plan
}

function sanitizeWeekView(raw: unknown): PkWeekView | undefined {
  if (!raw || typeof raw !== 'object') return undefined
  const input = raw as Record<string, unknown>
  const week =
    input.week === null
      ? null
      : typeof input.week === 'number' && input.week >= 1 && input.week <= MAX_WEEK
        ? Math.floor(input.week)
        : null
  return { week, useCurrent: input.useCurrent === true }
}

function sanitizeOptionalTypes(raw: unknown): PkOptionalType[] {
  return ensureArray<Record<string, unknown>>(raw)
    .map((item) => ({
      courseLabelId: typeof item?.courseLabelId === 'number' ? item.courseLabelId : 0,
      courseLabelName: typeof item?.courseLabelName === 'string' ? item.courseLabelName : '',
    }))
    .filter((item) => item.courseLabelId > 0 && item.courseLabelName)
}

function sanitizeCourseCollection(raw: unknown): PkCourse[] {
  return ensureArray<Record<string, unknown>>(raw)
    .map((item) => ({
      ...item,
      courseCode: typeof item?.courseCode === 'string' ? item.courseCode : '',
      courseName: typeof item?.courseName === 'string' ? item.courseName : '',
      courseNameReserved: typeof item?.courseNameReserved === 'string' ? item.courseNameReserved : '',
      courseType: typeof item?.courseType === 'string' ? item.courseType : '',
      faculty: typeof item?.faculty === 'string' ? item.faculty : '',
      credit: normalizeCredit(item?.credit),
      courseNature: normalizeStringList(item?.courseNature),
      campus: normalizeStringList(item?.campus),
      status: typeof item?.status === 'number' ? item.status : 0,
      teacher: normalizeStringList(item?.teacher),
      grade: typeof item?.grade === 'number' ? item.grade : undefined,
      courseDetail: sanitizeCourseDetail(item?.courseDetail),
    }))
    .filter((item) => item.courseCode || item.courseDetail.length > 0)
}

// ---- 派生重建（v2 核心：occupied/timeTableData 由方案数据重建）----

export interface RebuiltSchedule {
  occupied: PkOccupyCell[][][]
  timeTableData: PkCourseOnTable[]
}

/** 自定义占位事件在课表/占用表中的展示行（每个节次一行，便于网格渲染）。 */
function customEventTableRows(event: PkCustomEvent): PkCourseOnTable[] {
  const code = `${CUSTOM_EVENT_CODE_PREFIX}${event.id}`
  return event.sections.map((section) => ({
    showText: event.label,
    courseName: event.label,
    code,
    occupyTime: [section],
    occupyDay: event.day,
    occupyWeek: [...event.weeks],
  }))
}

function customEventArrangements(event: PkCustomEvent): PkArrangement[] {
  return [
    {
      arrangementText: '',
      occupyDay: event.day,
      occupyTime: [...event.sections],
      occupyWeek: [...event.weeks],
      occupyRoom: '',
      teacherAndCode: '',
    },
  ]
}

/**
 * 从方案数据（备选课程 + 自定义占位）重建占用表与课表行。
 * 纯函数：迁移自 v1 applySyncedCourses 的重建逻辑并泛化到任意方案。
 */
export function rebuildScheduleFromStaged(
  staged: PkStagedCourse[],
  customEvents: PkCustomEvent[],
): RebuiltSchedule {
  let occupied = createEmptyOccupied()
  const timeTableData: PkCourseOnTable[] = []

  for (const course of staged) {
    for (const detail of course.courseDetail) {
      if (detail.status !== COURSE_STATUS.STAGED && detail.status !== COURSE_STATUS.SELECTED) continue
      for (const arrangement of detail.arrangementInfo) {
        timeTableData.push({
          showText: `${arrangement.teacherAndCode} ${course.courseNameReserved}(${detail.code}) ${arrangement.arrangementText}`,
          courseName: course.courseNameReserved || course.courseName,
          code: detail.code,
          occupyTime: [...arrangement.occupyTime],
          occupyDay: arrangement.occupyDay,
          occupyWeek: [...arrangement.occupyWeek],
          teacherAndCode: arrangement.teacherAndCode,
          arrangementText: arrangement.arrangementText,
          occupyRoom: arrangement.occupyRoom,
        })
      }
      occupied = insertOccupied(occupied, detail.arrangementInfo, detail.code, course.courseNameReserved || course.courseName)
    }
  }

  for (const event of customEvents) {
    const code = `${CUSTOM_EVENT_CODE_PREFIX}${event.id}`
    timeTableData.push(...customEventTableRows(event))
    occupied = insertOccupied(occupied, customEventArrangements(event), code, event.label)
  }

  return { occupied, timeTableData }
}

// ---- 模块级单例 ----

const state = reactive<ScheduleState>(createInitialState())

function writeStorage(key: string, value: unknown): void {
  try {
    window.localStorage.setItem(key, JSON.stringify(value))
  } catch {
    // localStorage 可能不可用（隐私/受限浏览模式），静默忽略。
  }
}

function readStorage(key: string): string | null {
  try {
    return window.localStorage.getItem(key)
  } catch {
    return null
  }
}

function removeStorage(key: string): void {
  try {
    window.localStorage.removeItem(key)
  } catch {
    // 忽略
  }
}

function readTimeTableRows(): number {
  return maxRowsForCalendar(state.majorSelected.calendarId)
}

// ---- 方案操作（内部）----

function activePlan(): PkPlan {
  const plan = state.plans.find((item) => item.id === state.activePlanId)
  if (plan) return plan
  return state.plans[0] ?? createEmptyPlan()
}

/** 把激活方案的课程数据镜像到 commonLists（同引用），并重建派生态。 */
function syncActiveView(): void {
  const plan = activePlan()
  state.commonLists.stagedCourses = plan.stagedCourses
  state.commonLists.selectedCourses = plan.selectedCourses
  const rebuilt = rebuildScheduleFromStaged(plan.stagedCourses, plan.customEvents)
  state.occupied = rebuilt.occupied
  state.timeTableData = rebuilt.timeTableData
}

/** 从课表/占用/已选中移除一门课（按班级课号；作用于激活方案）。 */
function removeCourseFromSchedule(classCode: string): void {
  const plan = activePlan()
  const input = String(classCode ?? '').trim()
  // 入参可能是基础课号（退课按 courseCode 传入，如 '122004'）或班级课号
  // （'122004.01' / '12200401'）。getCourseBaseCode 对无点号的基础课号会误裁
  // 后两位（'122004'→'1220'），故先与备选池精确匹配，命中即为完整基础课号。
  const base = plan.stagedCourses.some((course) => course.courseCode === input)
    ? input
    : getCourseBaseCode(input)
  plan.stagedCourses = plan.stagedCourses.filter((course) => course.courseCode !== base)
  state.commonLists.stagedCourses = plan.stagedCourses
  plan.selectedCourses = plan.selectedCourses.filter((code) => !isClassOfCourse(code, base))
  state.commonLists.selectedCourses = plan.selectedCourses
  syncActiveView()
}

/** 追加课程到课表（同基础课号先替换，再入表、更新占用与备选状态）。 */
function appendToTimeTable(payload: PkCourseDetail): void {
  const plan = activePlan()
  const sameCodeCourse = state.timeTableData.find((course) => isSameCourse(course.code, payload.code))

  // 规定相同课号的课只能有一个：先移除旧的。
  if (sameCodeCourse) {
    state.timeTableData = state.timeTableData.filter((course) => !isSameCourse(course.code, payload.code))
    state.occupied = deleteOccupedByCode(state.occupied, sameCodeCourse.code)
    const staged = plan.stagedCourses.find(
      (course) => course.courseCode === getCourseBaseCode(payload.code),
    )
    if (staged) {
      const oldDetail = staged.courseDetail.find((detail) => isSameCourse(detail.code, sameCodeCourse.code))
      if (oldDetail) {
        // 旧班若已保存（status=2），从已选列表移除，避免导出残留旧班时间。
        if (oldDetail.status === COURSE_STATUS.SELECTED) {
          plan.selectedCourses = plan.selectedCourses.filter((code) => code !== oldDetail.code)
          state.commonLists.selectedCourses = plan.selectedCourses
        }
        oldDetail.status = COURSE_STATUS.UNSELECTED
      }
    }
  }

  for (const arrangement of payload.arrangementInfo) {
    state.timeTableData.push({
      showText: `${arrangement.teacherAndCode} ${state.clickedCourseInfo.courseName}(${payload.code}) ${arrangement.arrangementText}`,
      courseName: state.clickedCourseInfo.courseName,
      code: payload.code,
      occupyTime: [...arrangement.occupyTime],
      occupyDay: arrangement.occupyDay,
      occupyWeek: [...arrangement.occupyWeek],
      teacherAndCode: arrangement.teacherAndCode,
      arrangementText: arrangement.arrangementText,
      occupyRoom: arrangement.occupyRoom,
    })
  }

  state.occupied = insertOccupied(
    state.occupied,
    payload.arrangementInfo,
    payload.code,
    state.clickedCourseInfo.courseName,
  )

  payload.status = COURSE_STATUS.STAGED
  const stagedCourse = plan.stagedCourses.find(
    (course) => course.courseCode === getCourseBaseCode(payload.code),
  )
  if (stagedCourse) {
    stagedCourse.status = COURSE_STATUS.STAGED
    stagedCourse.teacher = payload.teachers
    // 同步班级状态（saveSelectedCourses 依赖 detail.status 判定待选/已选）。
    const matched = stagedCourse.courseDetail.find((detail) => isSameCourse(detail.code, payload.code))
    if (matched) matched.status = COURSE_STATUS.STAGED
  }
}

/** 按班级课号从占用表移除（custom 伪课号精确匹配；真实课号走基础课号归一）。 */
function deleteOccupedByCode(occupied: PkOccupyCell[][][], code: string): PkOccupyCell[][][] {
  if (code.startsWith(CUSTOM_EVENT_CODE_PREFIX)) {
    return occupied.map((row) => row.map((cell) => cell.filter((item) => item.code !== code)))
  }
  return occupied.map((row) =>
    row.map((cell) => cell.filter((item) => getCourseBaseCode(item.code) !== getCourseBaseCode(code))),
  )
}

// ---- 对外 API ----

export interface StageCourseResult {
  added: boolean
  /** 容忍式冲突：入表前与已占课程的冲突列表（同课换班不计），供 UI 标注。 */
  conflicts?: PkConflictItem[]
}

export interface ScheduleStats {
  courseCount: number
  totalCredit: number
  totalHours: number
  conflictCount: number
}

export function useScheduleStore() {
  function setMajorInfo(payload: PkMajorSelection): void {
    state.majorSelected = { ...payload }
    state.flags.majorNotChanged = false
  }

  function setCompulsoryCourses(payload: PkCourse[]): void {
    state.commonLists.compulsoryCourses = sanitizeCourseCollection(payload)
    state.flags.majorNotChanged = true
  }

  function setOptionalTypes(payload: PkOptionalType[]): void {
    state.commonLists.optionalTypes = sanitizeOptionalTypes(payload)
  }

  function setOptionalCourses(payload: PkCourse[]): void {
    state.commonLists.optionalCourses = sanitizeCourseCollection(payload)
  }

  function setSearchedCourses(payload: PkCourse[]): void {
    state.commonLists.searchCourses = sanitizeCourseCollection(payload)
  }

  function pushStagedCourse(payload: PkStagedCourse): void {
    const sanitized = sanitizeStagedCourse(payload)
    if (!sanitized.courseCode) return
    const plan = activePlan()
    plan.stagedCourses = [...plan.stagedCourses, sanitized]
    state.commonLists.stagedCourses = plan.stagedCourses
  }

  function popStagedCourse(courseCode: string): void {
    removeCourseFromSchedule(courseCode)
    state.clickedCourseInfo = { courseCode: '', courseName: '', teacherCode: '', teacherName: '' }
  }

  function setClickedCourseInfo(payload: PkClickedCourse): void {
    state.clickedCourseInfo = { ...payload }
  }

  /** 学期/年级/专业任一变更：清空所有方案（防跨学期污染）。 */
  function clearStagedAndSelectedCourses(): void {
    for (const plan of state.plans) {
      plan.stagedCourses = []
      plan.selectedCourses = []
      plan.customEvents = []
    }
    state.clickedCourseInfo = { courseCode: '', courseName: '', teacherCode: '', teacherName: '' }
    syncActiveView()
  }

  /**
   * 容忍式加课：总是入表并返回入表前的冲突列表（同课换班不计），
   * 由 UI 决定如何标注（课表 ⚠ / 列表红标 / flash 提示），不再阻断。
   */
  function stageCourse(payload: PkCourseDetail): StageCourseResult {
    const conflicts = findConflicts(payload, state.occupied).filter(
      (conflict) => !isSameCourse(conflict.code, payload.code),
    )
    appendToTimeTable(payload)
    return { added: true, conflicts }
  }

  /** 保存课表：激活方案内所有待选（status=1）班级升为已选（status=2）。 */
  function saveSelectedCourses(): void {
    const plan = activePlan()
    const selected = [...plan.selectedCourses]
    for (const course of plan.stagedCourses) {
      if (course.status !== COURSE_STATUS.STAGED) continue
      course.status = COURSE_STATUS.SELECTED
      for (const detail of course.courseDetail) {
        if (detail.status === COURSE_STATUS.STAGED) {
          detail.status = COURSE_STATUS.SELECTED
          if (!selected.includes(detail.code)) selected.push(detail.code)
        } else if (detail.status === COURSE_STATUS.SELECTED) {
          detail.status = COURSE_STATUS.UNSELECTED
          const index = selected.indexOf(detail.code)
          if (index >= 0) selected.splice(index, 1)
        }
      }
    }
    plan.selectedCourses = selected
    state.commonLists.selectedCourses = selected
  }

  // ---- 方案 CRUD ----

  function createPlan(): PkPlan {
    const plan = createEmptyPlan(nextPlanName())
    state.plans = [...state.plans, plan]
    return plan
  }

  function switchPlan(planId: string): void {
    if (!state.plans.some((plan) => plan.id === planId)) return
    state.activePlanId = planId
    state.clickedCourseInfo = { courseCode: '', courseName: '', teacherCode: '', teacherName: '' }
    syncActiveView()
    solidify()
  }

  function deletePlan(planId: string): void {
    const remaining = state.plans.filter((plan) => plan.id !== planId)
    if (remaining.length === 0) {
      // 删最后一个：自动建空方案，保证始终至少一个。
      const fresh = createEmptyPlan(nextPlanName())
      state.plans = [fresh]
      state.activePlanId = fresh.id
    } else {
      state.plans = remaining
      if (state.activePlanId === planId) state.activePlanId = remaining[0].id
    }
    state.clickedCourseInfo = { courseCode: '', courseName: '', teacherCode: '', teacherName: '' }
    syncActiveView()
    solidify()
  }

  /** 清空当前方案（保留方案壳）。 */
  function clearActivePlan(): void {
    const plan = activePlan()
    plan.stagedCourses = []
    plan.selectedCourses = []
    plan.customEvents = []
    state.clickedCourseInfo = { courseCode: '', courseName: '', teacherCode: '', teacherName: '' }
    syncActiveView()
    solidify()
  }

  // ---- 自定义占位事件 ----

  function addCustomEvent(input: { label: string; day: number; sections: number[]; weeks: number[] }): PkCustomEvent | null {
    const event = sanitizeCustomEvent(input)
    if (event.day < 1 || event.sections.length === 0 || event.weeks.length === 0) return null
    const plan = activePlan()
    plan.customEvents = [...plan.customEvents, event]
    syncActiveView()
    solidify()
    return event
  }

  function updateCustomEvent(id: string, patch: Partial<Omit<PkCustomEvent, 'id'>>): boolean {
    const plan = activePlan()
    const index = plan.customEvents.findIndex((event) => event.id === id)
    if (index < 0) return false
    const merged = sanitizeCustomEvent({ ...plan.customEvents[index], ...patch })
    if (merged.day < 1 || merged.sections.length === 0 || merged.weeks.length === 0) return false
    plan.customEvents = plan.customEvents.map((event, i) => (i === index ? merged : event))
    syncActiveView()
    solidify()
    return true
  }

  function removeCustomEvent(id: string): void {
    const plan = activePlan()
    plan.customEvents = plan.customEvents.filter((event) => event.id !== id)
    syncActiveView()
    solidify()
  }
  // ---- 周次视图 ----

  function setWeekView(view: PkWeekView): void {
    state.weekView = sanitizeWeekView(view) ?? { week: null, useCurrent: false }
    writeStorage(STORAGE_KEYS.weekView, state.weekView)
  }

  /** 学期字典写入（MajorSelector 加载 P1 后回填，供周次定位/日期条消费）。 */
  function setCalendars(payload: PkCalendar[]): void {
    state.calendars = Array.isArray(payload) ? payload : []
  }

  // ---- 同步 ----

  function setUpdateTime(payload: string): void {
    state.updateTime = payload
    writeStorage(STORAGE_KEYS.updateTime, payload)
  }

  function setLatestUpdateTime(payload: string): void {
    state.latestUpdateTime = payload
  }

  function setDataOutdated(payload: boolean): void {
    state.flags.isDataOutdated = payload
  }

  /** 同步最新数据（清空所有方案课程缓存并更新时间；保留方案壳）。 */
  function syncLatestData(): void {
    removeStorage(STORAGE_KEYS.plans)
    removeStorage(STORAGE_KEYS.activePlanId)
    for (const plan of state.plans) {
      plan.stagedCourses = []
      plan.selectedCourses = []
      plan.customEvents = []
    }
    state.clickedCourseInfo = { courseCode: '', courseName: '', teacherCode: '', teacherName: '' }
    syncActiveView()
    state.updateTime = state.latestUpdateTime
    writeStorage(STORAGE_KEYS.updateTime, state.updateTime)
    state.flags.isDataOutdated = false
  }


  /**
   * P12 同步结果应用到全部方案（跨方案并集请求后调用）：
   * 各方案按课号命中替换课程详情，保留该方案的班级排课状态；updateTime 全局推进。
   */
  function applySyncToAllPlans(detailsByCode: Record<string, PkCourseDetail[]>): void {
    for (const plan of state.plans) {
      plan.stagedCourses = plan.stagedCourses
        .map((course) => {
          const details = detailsByCode[course.courseCode]
          if (!details || details.length === 0) return course
          return {
            ...course,
            courseDetail: details.map((detail) => ({
              ...detail,
              status: course.courseDetail.find((old) => old.code === detail.code)?.status ?? 0,
            })),
          }
        })
        .filter((course) => course.courseCode)
    }
    syncActiveView()

    state.updateTime = state.latestUpdateTime
    state.flags.isDataOutdated = false
    solidify()
  }

  /** 持久化关键状态到 localStorage（v2：plans/activePlanId/weekView；派生态不再写入）。 */
  function solidify(): void {
    writeStorage(STORAGE_KEYS.majorSelected, state.majorSelected)
    writeStorage(STORAGE_KEYS.plans, state.plans)
    writeStorage(STORAGE_KEYS.activePlanId, state.activePlanId)
    writeStorage(STORAGE_KEYS.weekView, state.weekView)
  }

  /** 从 localStorage 恢复（v1 旧键迁移 → v2；损坏即回退空方案）。 */
  function loadSolidify(): void {
    const majorSelected = safeParseJson(readStorage(STORAGE_KEYS.majorSelected))
    if (majorSelected) state.majorSelected = sanitizeMajorSelected(majorSelected)

    const plansRaw = safeParseJson(readStorage(STORAGE_KEYS.plans))
    if (Array.isArray(plansRaw)) {
      const plans = ensureArray<unknown>(plansRaw).map(sanitizePlan)
      state.plans = plans.length > 0 ? plans : [createEmptyPlan()]
    } else {
      // v1 → v2 迁移：旧单方案数据包装为「方案一」；旧键只读保留（回滚兼容）。
      const legacyStaged = safeParseJson(readStorage(STORAGE_KEYS.legacyStagedCourses))
      const legacySelected = safeParseJson(readStorage(STORAGE_KEYS.legacySelectedCourses))
      const plan = createEmptyPlan(i18n.global.t('schedule.planDefaultName', { n: 1 }))
      if (Array.isArray(legacyStaged)) {
        plan.stagedCourses = ensureArray<unknown>(legacyStaged).map(sanitizeStagedCourse).filter((course) => course.courseCode)
      }
      if (Array.isArray(legacySelected)) {
        plan.selectedCourses = ensureArray<unknown>(legacySelected).filter((item): item is string => typeof item === 'string')
      }
      state.plans = [plan]
    }

    const activeId = safeParseJson<string>(readStorage(STORAGE_KEYS.activePlanId))
    state.activePlanId =
      typeof activeId === 'string' && state.plans.some((plan) => plan.id === activeId)
        ? activeId
        : state.plans[0].id

    const weekView = safeParseJson(readStorage(STORAGE_KEYS.weekView))
    state.weekView = sanitizeWeekView(weekView) ?? { week: null, useCurrent: false }

    syncActiveView()
  }

  /** 仅恢复同步时间（不触发课程缓存校验）。 */
  function loadSolidifyTime(): void {
    const updateTime = readStorage(STORAGE_KEYS.updateTime)
    if (updateTime) state.updateTime = updateTime
  }

  function isMajorSelected(): boolean {
    return Boolean(
      state.majorSelected.calendarId &&
        state.majorSelected.grade !== undefined &&
        state.majorSelected.major,
    )
  }

  /** 统计（当前方案）：门数 / 总学分 / 总学时（Σ 节数×周数）/ 冲突门数。 */
  function stats(): ScheduleStats {
    const plan = activePlan()
    const conflicts = deriveConflicts(state.occupied)
    let courseCount = 0
    let totalCredit = 0
    let totalHours = 0
    for (const course of plan.stagedCourses) {
      const arranged = course.courseDetail.filter(
        (detail) => detail.status === COURSE_STATUS.STAGED || detail.status === COURSE_STATUS.SELECTED,
      )
      if (arranged.length === 0) continue
      courseCount += 1
      totalCredit += Number(course.credit ?? 0)
      for (const detail of arranged) {
        for (const arrangement of detail.arrangementInfo) {
          totalHours += arrangement.occupyTime.length * arrangement.occupyWeek.length
        }
      }
    }
    const conflictCount = [...conflicts.keys()].filter(
      (key) => !key.startsWith(CUSTOM_EVENT_CODE_PREFIX),
    ).length
    return { courseCount, totalCredit, totalHours, conflictCount }
  }

  /** 已选学分统计（已选/专业/通识；兼容旧课表头展示）。 */
  function creditSummary(): { selectedTotal: number; selectedMajor: number; selectedGeneral: number } {
    const plan = activePlan()
    const selectedBases = new Set(
      plan.selectedCourses.map((code) => getCourseBaseCode(code)),
    )
    let selectedTotal = 0
    let selectedMajor = 0
    let selectedGeneral = 0

    for (const course of plan.stagedCourses) {
      if (!selectedBases.has(course.courseCode)) continue
      const credit = Number(course.credit ?? 0)
      selectedTotal += credit
      if (course.courseType === '选' || course.courseType === '跨') {
        selectedGeneral += credit
      } else if (isExclusiveCourse(course)) {
        selectedMajor += credit
      } else {
        selectedGeneral += credit
      }
    }
    return { selectedTotal, selectedMajor, selectedGeneral }
  }

  function isExclusiveCourse(course: PkStagedCourse): boolean {
    return course.courseDetail.some((detail) => detail.isExclusive === true)
  }

  return {
    // 直接暴露 reactive state（对齐 shell-state.ts 惯例）；组件按约定只读消费，
    // 数据修改一律走下方操作方法。运行时 Vue proxy 仍会拦截误写。
    state,
    readTimeTableRows,
    setMajorInfo,
    setCompulsoryCourses,
    setOptionalTypes,
    setOptionalCourses,
    setSearchedCourses,
    pushStagedCourse,
    popStagedCourse,
    setClickedCourseInfo,
    clearStagedAndSelectedCourses,
    stageCourse,
    saveSelectedCourses,
    createPlan,
    switchPlan,
    deletePlan,
    clearActivePlan,
    addCustomEvent,
    updateCustomEvent,
    removeCustomEvent,
    setWeekView,
    setCalendars,
    setUpdateTime,
    setLatestUpdateTime,
    setDataOutdated,
    syncLatestData,
    applySyncToAllPlans,
    solidify,
    loadSolidify,
    loadSolidifyTime,
    isMajorSelected,
    stats,
    creditSummary,
  }
}

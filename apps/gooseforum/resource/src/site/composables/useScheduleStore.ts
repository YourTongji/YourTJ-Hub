// 排课器组合式 store（模块级单例）。
//
// 对齐上游 scheduler store 的状态模型与 localStorage 持久化（majorSelected /
// stagedCourses / selectedCourses / occupied / timeTableData / updateTime），
// 但采用模块级 reactive 单例（项目惯例：home-feed-mode.ts / shell-state.ts），
// 并带完整 sanitize 防御：JSON 解析失败或字段非法即回退默认值（验收标准 5）。
//
// 冲突处理增强（验收标准 2）：加入课表时若冲突，返回冲突项由 UI 决定
// 「强制替换 / 放弃」；强制替换 = 移除冲突课程后重新入表。

import { reactive } from 'vue'
import { i18n } from '@/runtime/i18n'
import {
  canAddCourse,
  createEmptyOccupied,
  deleteOccupied,
  findConflicts,
  getCourseBaseCode,
  insertOccupied,
  isClassOfCourse,
  isSameCourse,
  type PkConflictItem,
} from '@/site/utils/pkConflict'
import { maxRowsForCalendar } from '@/site/utils/pkArrange'
import type {
  PkArrangement,
  PkClickedCourse,
  PkCourse,
  PkCourseDetail,
  PkCourseOnTable,
  PkMajorSelection,
  PkOccupyCell,
  PkOptionalType,
  PkStagedCourse,
  PkTeacher,
} from '@/site/types/pk'

/** localStorage 键（带 pk. 前缀避免与论坛其他状态冲突）。 */
const STORAGE_KEYS = {
  majorSelected: 'pk.majorSelected',
  stagedCourses: 'pk.stagedCourses',
  selectedCourses: 'pk.selectedCourses',
  occupied: 'pk.occupied',
  timeTableData: 'pk.timeTableData',
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
  /** 备选课程池 */
  stagedCourses: PkStagedCourse[]
  /** 已选班级课号（含班号） */
  selectedCourses: string[]
}

interface ScheduleState {
  majorSelected: PkMajorSelection
  commonLists: CommonLists
  clickedCourseInfo: PkClickedCourse
  occupied: PkOccupyCell[][][]
  timeTableData: PkCourseOnTable[]
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
  return {
    majorSelected: { calendarId: undefined, grade: undefined, major: undefined },
    commonLists: createEmptyCommonLists(),
    clickedCourseInfo: { courseCode: '', courseName: '', teacherCode: '', teacherName: '' },
    occupied: createEmptyOccupied(),
    timeTableData: [],
    flags: { majorNotChanged: false, isDataOutdated: false },
    updateTime: '',
    latestUpdateTime: '',
  }
}

// ---- sanitize（损坏即清理/回退，验收标准 5）----

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
      occupyWeek: ensureArray<number>(item?.occupyWeek).filter((week) => typeof week === 'number'),
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

function sanitizeTimeTableData(raw: unknown): PkCourseOnTable[] {
  return ensureArray<Record<string, unknown>>(raw)
    .map((item) => ({
      showText: typeof item?.showText === 'string' ? item.showText : '',
      courseName: typeof item?.courseName === 'string' ? item.courseName : '',
      code: typeof item?.code === 'string' ? item.code : '',
      occupyTime: ensureArray<number>(item?.occupyTime).filter(
        (slot) => typeof slot === 'number' && slot >= 1 && slot <= 12,
      ),
      occupyDay:
        typeof item?.occupyDay === 'number' && item.occupyDay >= 1 && item.occupyDay <= 7
          ? item.occupyDay
          : 0,
    }))
    .filter((item) => item.code && item.courseName && item.occupyDay > 0 && item.occupyTime.length > 0)
}

function sanitizeOccupied(raw: unknown): PkOccupyCell[][][] {
  const rows = ensureArray<unknown[]>(raw)
  if (rows.length !== 12) return createEmptyOccupied()
  return rows.map((row) => {
    const cols = ensureArray<unknown[]>(row)
    if (cols.length !== 7) return Array.from({ length: 7 }, () => [] as PkOccupyCell[])
    return cols.map((cell) =>
      ensureArray<Record<string, unknown>>(cell)
        .map((item) => ({
          code: typeof item?.code === 'string' ? item.code : '',
          courseName: typeof item?.courseName === 'string' ? item.courseName : '',
          occupyWeek: ensureArray<number>(item?.occupyWeek).filter((week) => typeof week === 'number'),
        }))
        .filter((item) => item.code),
    )
  })
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

// ---- 内部操作 ----

/** 从课表/占用/已选/备选池中移除一门课（入参为基础课号或班级课号）。 */
function removeCourseFromSchedule(classCode: string): void {
  const input = String(classCode ?? '').trim()
  // 入参可能是基础课号（退课按 courseCode 传入，如 '122004'）或班级课号
  // （'122004.01' / '12200401'）。getCourseBaseCode 对无点号的基础课号会误裁
  // 后两位（'122004'→'1220'），故先与备选池精确匹配，命中即为完整基础课号。
  const base = state.commonLists.stagedCourses.some((course) => course.courseCode === input)
    ? input
    : getCourseBaseCode(input)
  state.commonLists.stagedCourses = state.commonLists.stagedCourses.filter(
    (course) => course.courseCode !== base,
  )
  state.commonLists.selectedCourses = state.commonLists.selectedCourses.filter(
    (code) => !isClassOfCourse(code, base),
  )
  state.timeTableData = state.timeTableData.filter((course) => !isClassOfCourse(course.code, base))
  // deleteOccupied 会对入参再走一次 getCourseBaseCode，基础课号会被误裁，
  // 这里直接按已归一化的 base 比较占用格。
  state.occupied = state.occupied.map((row) =>
    row.map((cell) => cell.filter((item) => getCourseBaseCode(item.code) !== base)),
  )
}

/** 追加课程到课表（同基础课号先替换，再入表、更新占用与备选状态）。 */
function appendToTimeTable(payload: PkCourseDetail): void {
  const sameCodeCourse = state.timeTableData.find((course) => isSameCourse(course.code, payload.code))

  // 规定相同课号的课只能有一个：先移除旧的。
  if (sameCodeCourse) {
    state.timeTableData = state.timeTableData.filter((course) => !isSameCourse(course.code, payload.code))
    state.occupied = deleteOccupied(state.occupied, sameCodeCourse.code)
    const staged = state.commonLists.stagedCourses.find(
      (course) => course.courseCode === getCourseBaseCode(payload.code),
    )
    if (staged) {
      const oldDetail = staged.courseDetail.find((detail) => isSameCourse(detail.code, sameCodeCourse.code))
      if (oldDetail) {
        // 旧班若已保存（status=2），从已选列表移除，避免导出残留旧班时间。
        if (oldDetail.status === COURSE_STATUS.SELECTED) {
          state.commonLists.selectedCourses = state.commonLists.selectedCourses.filter(
            (code) => code !== oldDetail.code,
          )
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
      occupyTime: arrangement.occupyTime,
      occupyDay: arrangement.occupyDay,
    })
  }

  state.occupied = insertOccupied(
    state.occupied,
    payload.arrangementInfo,
    payload.code,
    state.clickedCourseInfo.courseName,
  )

  payload.status = COURSE_STATUS.STAGED
  const stagedCourse = state.commonLists.stagedCourses.find(
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

// ---- 对外 API ----

export interface StageCourseResult {
  added: boolean
  conflicts?: PkConflictItem[]
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
    state.commonLists.stagedCourses = [...state.commonLists.stagedCourses, sanitized]
  }

  function popStagedCourse(courseCode: string): void {
    removeCourseFromSchedule(courseCode)
    state.clickedCourseInfo = { courseCode: '', courseName: '', teacherCode: '', teacherName: '' }
  }

  function setClickedCourseInfo(payload: PkClickedCourse): void {
    state.clickedCourseInfo = { ...payload }
  }

  function clearStagedAndSelectedCourses(): void {
    state.commonLists.stagedCourses = []
    state.commonLists.selectedCourses = []
    state.timeTableData = []
    state.occupied = createEmptyOccupied()
    state.clickedCourseInfo = { courseCode: '', courseName: '', teacherCode: '', teacherName: '' }
  }

  /**
   * 尝试将某教学班加入课表。
   * 无冲突 → 直接加入并返回 { added: true }；
   * 有冲突 → 不加入，返回 { added: false, conflicts } 由 UI 决定「强制替换/放弃」。
   */
  function stageCourse(payload: PkCourseDetail): StageCourseResult {
    const check = canAddCourse(payload.arrangementInfo, state.occupied, payload.code)
    if (check.canAdd) {
      appendToTimeTable(payload)
      return { added: true }
    }
    return { added: false, conflicts: findConflicts(payload, state.occupied) }
  }

  /** 强制替换：移除所有冲突课程后把目标课程加入课表。 */
  function forceReplaceCourse(payload: PkCourseDetail): boolean {
    const conflicts = findConflicts(payload, state.occupied)
    for (const conflict of conflicts) {
      // 跳过候选课自身的旧班（同基础课号）：由 appendToTimeTable 的隐式替换处理，
      // 否则 removeCourseFromSchedule 会把候选课程整门从 stagedCourses 移除。
      if (isSameCourse(conflict.code, payload.code)) continue
      removeCourseFromSchedule(conflict.code)
    }
    appendToTimeTable(payload)
    return true
  }

  /** 保存课表：所有待选（status=1）班级升为已选（status=2）。 */
  function saveSelectedCourses(): void {
    for (const course of state.commonLists.stagedCourses) {
      if (course.status !== COURSE_STATUS.STAGED) continue
      course.status = COURSE_STATUS.SELECTED
      for (const detail of course.courseDetail) {
        if (detail.status === COURSE_STATUS.STAGED) {
          detail.status = COURSE_STATUS.SELECTED
          state.commonLists.selectedCourses = [...state.commonLists.selectedCourses, detail.code]
        } else if (detail.status === COURSE_STATUS.SELECTED) {
          detail.status = COURSE_STATUS.UNSELECTED
          state.commonLists.selectedCourses = state.commonLists.selectedCourses.filter(
            (code) => code !== detail.code,
          )
        }
      }
    }
  }

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

  /** 同步最新数据（清空课程缓存并更新时间）。 */
  function syncLatestData(): void {
    removeStorage(STORAGE_KEYS.stagedCourses)
    removeStorage(STORAGE_KEYS.selectedCourses)
    removeStorage(STORAGE_KEYS.occupied)
    removeStorage(STORAGE_KEYS.timeTableData)
    state.commonLists.stagedCourses = []
    state.commonLists.selectedCourses = []
    state.timeTableData = []
    state.occupied = createEmptyOccupied()
    state.clickedCourseInfo = { courseCode: '', courseName: '', teacherCode: '', teacherName: '' }
    state.updateTime = state.latestUpdateTime
    writeStorage(STORAGE_KEYS.updateTime, state.updateTime)
    state.flags.isDataOutdated = false
  }


  /**
   * 应用 P12 course-info-sync 返回的最新课程（增量保留已选，验收「同步最新」）。
   * 用新详情替换 stagedCourses，并按保留的排课状态（待选/已选）重建课表与占用表。
   */
  function applySyncedCourses(newStaged: PkStagedCourse[]): void {
    const sanitized = newStaged.map(sanitizeStagedCourse)
    state.commonLists.stagedCourses = sanitized

    let occupied = createEmptyOccupied()
    const nextTimeTable: PkCourseOnTable[] = []
    for (const course of sanitized) {
      for (const detail of course.courseDetail) {
        if (detail.status !== COURSE_STATUS.STAGED && detail.status !== COURSE_STATUS.SELECTED) continue
        for (const arrangement of detail.arrangementInfo) {
          nextTimeTable.push({
            showText: `${arrangement.teacherAndCode} ${course.courseNameReserved}(${detail.code}) ${arrangement.arrangementText}`,
            courseName: course.courseNameReserved,
            code: detail.code,
            occupyTime: arrangement.occupyTime,
            occupyDay: arrangement.occupyDay,
          })
        }
        occupied = insertOccupied(occupied, detail.arrangementInfo, detail.code, course.courseNameReserved)
      }
    }
    state.occupied = occupied
    state.timeTableData = nextTimeTable

    state.updateTime = state.latestUpdateTime
    writeStorage(STORAGE_KEYS.stagedCourses, state.commonLists.stagedCourses)
    writeStorage(STORAGE_KEYS.selectedCourses, state.commonLists.selectedCourses)
    writeStorage(STORAGE_KEYS.occupied, state.occupied)
    writeStorage(STORAGE_KEYS.timeTableData, state.timeTableData)
    writeStorage(STORAGE_KEYS.updateTime, state.updateTime)
    state.flags.isDataOutdated = false
  }

  /** 持久化全部关键状态到 localStorage。 */
  function solidify(): void {
    writeStorage(STORAGE_KEYS.majorSelected, state.majorSelected)
    writeStorage(STORAGE_KEYS.stagedCourses, state.commonLists.stagedCourses)
    writeStorage(STORAGE_KEYS.selectedCourses, state.commonLists.selectedCourses)
    writeStorage(STORAGE_KEYS.occupied, state.occupied)
    writeStorage(STORAGE_KEYS.timeTableData, state.timeTableData)
  }

  /** 从 localStorage 恢复关键状态（损坏即清理，验收标准 5）。 */
  function loadSolidify(): void {
    const majorSelected = safeParseJson(readStorage(STORAGE_KEYS.majorSelected))
    if (majorSelected) state.majorSelected = sanitizeMajorSelected(majorSelected)

    const stagedCourses = safeParseJson(readStorage(STORAGE_KEYS.stagedCourses))
    if (stagedCourses) {
      state.commonLists.stagedCourses = ensureArray<unknown>(stagedCourses).map(sanitizeStagedCourse)
    }
    const selectedCourses = safeParseJson(readStorage(STORAGE_KEYS.selectedCourses))
    if (selectedCourses) {
      state.commonLists.selectedCourses = ensureArray<unknown>(selectedCourses).filter(
        (item) => typeof item === 'string',
      )
    }
    const occupied = safeParseJson(readStorage(STORAGE_KEYS.occupied))
    if (occupied) state.occupied = sanitizeOccupied(occupied)
    const timeTableData = safeParseJson(readStorage(STORAGE_KEYS.timeTableData))
    if (timeTableData) state.timeTableData = sanitizeTimeTableData(timeTableData)
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

  /** 已选学分统计（已选/专业/通识）。 */
  function creditSummary(): { selectedTotal: number; selectedMajor: number; selectedGeneral: number } {
    const selectedBases = new Set(
      state.commonLists.selectedCourses.map((code) => getCourseBaseCode(code)),
    )
    let selectedTotal = 0
    let selectedMajor = 0
    let selectedGeneral = 0

    for (const course of state.commonLists.stagedCourses) {
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
    forceReplaceCourse,
    saveSelectedCourses,
    setUpdateTime,
    setLatestUpdateTime,
    setDataOutdated,
    syncLatestData,
    applySyncedCourses,
    solidify,
    loadSolidify,
    loadSolidifyTime,
    isMajorSelected,
    creditSummary,
  }
}

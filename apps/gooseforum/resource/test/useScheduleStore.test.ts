import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { i18n } from '../src/runtime/i18n'
import { MAX_PLANS, useScheduleStore } from '../src/site/composables/useScheduleStore'
import {
  CUSTOM_EVENT_CODE_PREFIX,
  conflictBaseOf,
  createEmptyOccupied,
  deriveConflicts,
  insertOccupied,
  getCourseBaseCode,
} from '../src/site/utils/pkConflict'
import { currentWeekForDate, formatWeeksText } from '../src/site/utils/pkArrange'
import type { PkArrangement, PkCourseDetail, PkPlan, PkStagedCourse } from '../src/site/types/pk'

/** 展开的周次区间（如 range(1, 8) = [1..8]），对齐「1-8周」语义。 */
function range(from: number, to: number): number[] {
  return Array.from({ length: to - from + 1 }, (_, i) => from + i)
}

/** 与 store 同源的默认方案名（如「方案 {n}」），断言名称不依赖测试环境 locale。 */
function planName(n: number): string {
  return i18n.global.t('schedule.planDefaultName', { n })
}

function makeDetail(code: string, day: number, time: number[], weeks: number[]): PkCourseDetail {
  return {
    code,
    campus: '',
    teachers: [],
    teachingLanguage: '',
    arrangementInfo: [
      {
        arrangementText: '[1-8周] 周一 3-4节 A101',
        occupyDay: day,
        occupyTime: time,
        // 测试约定：传 [start, end] 两元素视为区间展开（对齐「1-8周」语义）。
        occupyWeek: weeks.length === 2 && weeks[1] > weeks[0] ? range(weeks[0], weeks[1]) : weeks,
        occupyRoom: 'A101',
        teacherAndCode: '张三(T001)',
      },
    ],
  }
}

function makeStaged(courseCode: string, details: PkCourseDetail[], credit = 3): PkStagedCourse {
  return {
    courseCode,
    courseName: courseCode,
    courseNameReserved: courseCode,
    credit,
    courseType: '必',
    teacher: [],
    status: 0,
    courseDetail: details,
  }
}

function makeStorage(initial: Record<string, string> = {}) {
  const storage = new Map<string, string>(Object.entries(initial))
  return {
    localStorage: {
      getItem: vi.fn((key: string) => storage.get(key) ?? null),
      setItem: vi.fn((key: string, value: string) => storage.set(key, value)),
      removeItem: vi.fn((key: string) => storage.delete(key)),
    },
  }
}

/** 读取 mock localStorage 已写入的 JSON。 */
function writtenJson(calls: ReturnType<typeof vi.fn>, key: string): any {
  for (const call of calls.mock.calls) {
    if (call[0] === key) {
      try {
        return JSON.parse(call[1])
      } catch {
        return undefined
      }
    }
  }
  return undefined
}

describe('useScheduleStore（v2 多方案 + 容忍式冲突）', () => {
  beforeEach(() => {
    vi.stubGlobal('window', makeStorage())
    const store = useScheduleStore()
    store.clearStagedAndSelectedCourses()
    store.setMajorInfo({ calendarId: undefined, grade: undefined, major: undefined })
    // 重置为单方案初始态（清掉上一用例遗留的多方案，再删最后一个触发自动补建）。
    while (store.state.plans.length > 1) {
      store.deletePlan(store.state.plans[0].id)
    }
    store.deletePlan(store.state.plans[0].id)
    store.setWeekView({ week: null, useCurrent: false })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  test('stageCourse 无冲突时加入课表并更新占用', () => {
    const store = useScheduleStore()
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3, 4], [1, 8])]))

    const result = store.stageCourse(makeDetail('122004.01', 1, [3, 4], [1, 8]))

    expect(result.added).toBe(true)
    expect(result.conflicts).toHaveLength(0)
    expect(store.state.timeTableData).toHaveLength(1)
    expect(store.state.occupied[2][0]).toHaveLength(1)
    // 排入课表后班级状态为待选（1）
    expect(store.state.commonLists.stagedCourses[0].courseDetail[0].status).toBe(1)
    // v2 结构化字段随入表写入（周次过滤渲染用）。
    expect(store.state.timeTableData[0].occupyWeek).toEqual([1, 2, 3, 4, 5, 6, 7, 8])
    expect(store.state.timeTableData[0].occupyRoom).toBe('A101')
  })

  test('容忍式冲突：冲突课仍入表并返回冲突列表（不再阻断）', () => {
    const store = useScheduleStore()
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3], [1, 8])]))
    store.stageCourse(makeDetail('122004.01', 1, [3], [1, 8]))

    store.setClickedCourseInfo({ courseCode: '122005', courseName: '英语' })
    store.pushStagedCourse(makeStaged('122005', [makeDetail('122005.01', 1, [3], [1, 8])]))
    const result = store.stageCourse(makeDetail('122005.01', 1, [3], [1, 8]))

    // 容忍式：加入成功，同时返回冲突标注。
    expect(result.added).toBe(true)
    expect(result.conflicts).toHaveLength(1)
    expect(result.conflicts?.[0].code).toBe('122004.01')
    // 两门课都入表。
    expect(store.state.timeTableData).toHaveLength(2)
    expect(store.state.occupied[2][0]).toHaveLength(2)
    // 冲突派生：两门课互相标注。
    const conflicts = deriveConflicts(store.state.occupied)
    expect(conflicts.get('122004')?.map((item) => item.code)).toContain('122005.01')
    expect(conflicts.get('122005')?.map((item) => item.code)).toContain('122004.01')
  })

  test('周次不重叠不判冲突（同天同节不同周）', () => {
    const store = useScheduleStore()
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3], [1, 8])]))
    store.stageCourse(makeDetail('122004.01', 1, [3], [1, 8]))

    store.setClickedCourseInfo({ courseCode: '122005', courseName: '英语' })
    store.pushStagedCourse(makeStaged('122005', [makeDetail('122005.01', 1, [3], [9, 16])]))
    const result = store.stageCourse(makeDetail('122005.01', 1, [3], [9, 16]))

    expect(result.added).toBe(true)
    expect(result.conflicts).toHaveLength(0)
    expect(deriveConflicts(store.state.occupied).size).toBe(0)
  })

  test('同课换班（同基础课号）不判冲突且隐式替换旧班', () => {
    const store = useScheduleStore()
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(
      makeStaged('122004', [makeDetail('122004.01', 1, [3], [1, 8]), makeDetail('122004.02', 2, [3], [1, 8])]),
    )
    store.stageCourse(makeDetail('122004.01', 1, [3], [1, 8]))
    const result = store.stageCourse(makeDetail('122004.02', 2, [3], [1, 8]))

    expect(result.added).toBe(true)
    expect(result.conflicts).toHaveLength(0)
    // 旧班被替换：课表只剩新班。
    expect(store.state.timeTableData.map((course) => course.code)).toEqual(['122004.02'])
    expect(deriveConflicts(store.state.occupied).size).toBe(0)
  })

  test('saveSelectedCourses 后班级进入已选列表', () => {
    const store = useScheduleStore()
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3, 4], [1, 8])]))
    store.stageCourse(makeDetail('122004.01', 1, [3, 4], [1, 8]))

    store.saveSelectedCourses()

    expect(store.state.commonLists.selectedCourses).toEqual(['122004.01'])
    expect(store.state.commonLists.stagedCourses[0].status).toBe(2)
  })

  test('saveSelectedCourses 无可保存课程时为空操作（不抛错）', () => {
    const store = useScheduleStore()

    expect(() => store.saveSelectedCourses()).not.toThrow()
    expect(store.state.commonLists.selectedCourses).toEqual([])
  })

  test('损坏的 localStorage 恢复后回退空方案', () => {
    vi.stubGlobal('window', {
      localStorage: {
        getItem: vi.fn(() => '{ 损坏的JSON'),
        setItem: vi.fn(),
        removeItem: vi.fn(),
      },
    })
    const store = useScheduleStore()
    store.loadSolidify()
    expect(store.state.plans).toHaveLength(1)
    expect(store.state.commonLists.stagedCourses).toEqual([])
    expect(store.state.occupied).toHaveLength(12)
    expect(store.state.timeTableData).toEqual([])
  })

  test('v1 → v2 迁移：旧键包装为单方案且旧键只读保留', () => {
    const legacyStaged = [makeStaged('122004', [makeDetail('122004.01', 1, [3, 4], [1, 8])])]
    const storage = makeStorage({
      'pk.stagedCourses': JSON.stringify(legacyStaged),
      'pk.selectedCourses': JSON.stringify(['122004.01']),
      // 无 pk.plans → 触发迁移。
    })
    vi.stubGlobal('window', storage)
    const store = useScheduleStore()
    store.loadSolidify()

    // 迁移为单方案。
    expect(store.state.plans).toHaveLength(1)
    expect(store.state.commonLists.stagedCourses[0].courseCode).toBe('122004')
    expect(store.state.commonLists.selectedCourses).toEqual(['122004.01'])
    // 派生态从方案数据重建（迁移源班级 status=0 → 无课表行）。
    expect(store.state.timeTableData).toEqual([])
    // 旧键未被删除。
    expect(storage.localStorage.removeItem).not.toHaveBeenCalledWith('pk.stagedCourses')
  })

  test('合法 localStorage 恢复（v2 plans 键）', () => {
    const store = useScheduleStore()
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3, 4], [1, 8])]))
    store.stageCourse(makeDetail('122004.01', 1, [3, 4], [1, 8]))
    store.solidify()

    const restored = useScheduleStore()
    restored.loadSolidify()
    expect(restored.state.plans).toHaveLength(1)
    expect(restored.state.commonLists.stagedCourses[0].courseCode).toBe('122004')
    // 班级 status=1（待选）→ 重建后占用表/课表就绪。
    expect(restored.state.occupied[2][0][0].code).toBe('122004.01')
    expect(restored.state.timeTableData[0].occupyWeek).toEqual([1, 2, 3, 4, 5, 6, 7, 8])
  })

  test('方案 CRUD：新增/切换/删除/清空', () => {
    const store = useScheduleStore()
    const first = store.state.plans[0]

    // 方案一加课。
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3], [1, 8])]))
    store.stageCourse(makeDetail('122004.01', 1, [3], [1, 8]))
    expect(store.state.commonLists.stagedCourses).toHaveLength(1)

    // 新建方案二并切换：镜像清空。
    const second = store.createPlan()!
    store.switchPlan(second.id)
    expect(store.state.activePlanId).toBe(second.id)
    expect(store.state.commonLists.stagedCourses).toHaveLength(0)
    expect(store.state.timeTableData).toHaveLength(0)

    // 方案二加另一门课。
    store.setClickedCourseInfo({ courseCode: 'G100', courseName: '通识' })
    store.pushStagedCourse(makeStaged('G100', [makeDetail('G100.01', 2, [5], [1, 16])]))
    store.stageCourse(makeDetail('G100.01', 2, [5], [1, 16]))

    // 切回方案一：数据恢复。
    store.switchPlan(first.id)
    expect(store.state.commonLists.stagedCourses[0].courseCode).toBe('122004')

    // 删除当前方案一：自动激活剩余方案。
    store.deletePlan(first.id)
    expect(store.state.plans.map((plan) => plan.id)).toEqual([second.id])
    expect(store.state.activePlanId).toBe(second.id)
    expect(store.state.commonLists.stagedCourses[0].courseCode).toBe('G100')

    // 清空当前方案：课程清空但方案壳保留。
    store.clearActivePlan()
    expect(store.state.plans).toHaveLength(1)
    expect(store.state.commonLists.stagedCourses).toHaveLength(0)
    expect(store.state.timeTableData).toHaveLength(0)
  })

  test('删除最后一个方案自动建空方案', () => {
    const store = useScheduleStore()
    const only = store.state.plans[0]
    store.deletePlan(only.id)
    expect(store.state.plans).toHaveLength(1)
    expect(store.state.plans[0].id).not.toBe(only.id)
    expect(store.state.activePlanId).toBe(store.state.plans[0].id)
  })

  test('删除唯一方案后自动补建回到「方案 1」（序号不再自增）', () => {
    const store = useScheduleStore()
    expect(store.state.plans).toHaveLength(1)
    expect(store.state.plans[0].name).toBe(planName(1))

    // 删除唯一的「方案 1」→ 自动补建仍为「方案 1」，而非「方案 2」。
    store.deletePlan(store.state.plans[0].id)
    expect(store.state.plans).toHaveLength(1)
    expect(store.state.plans[0].name).toBe(planName(1))

    // 再次删除 → 序号继续回落，不再循环自增。
    store.deletePlan(store.state.plans[0].id)
    expect(store.state.plans[0].name).toBe(planName(1))
  })

  test('新增方案序号取现存最大值 +1：删除中间方案后不与现存方案重名', () => {
    const store = useScheduleStore()
    const first = store.state.plans[0]
    expect(first.name).toBe(planName(1))

    const second = store.createPlan()
    expect(second).not.toBeNull()
    expect(second!.name).toBe(planName(2))

    const third = store.createPlan()
    expect(third).not.toBeNull()
    expect(third!.name).toBe(planName(3))

    // 已有 方案1/2/3，删除中间的「方案 2」→ 剩余 方案1/3 → 再新增应为「方案 4」。
    store.deletePlan(second!.id)
    const fourth = store.createPlan()
    expect(fourth).not.toBeNull()
    expect(fourth!.name).toBe(planName(4))
    // 新方案不与现存方案重名。
    expect(store.state.plans.map((plan) => plan.name).filter((name) => name === fourth!.name)).toHaveLength(1)
  })

  test('方案数量达到上限后 createPlan 返回 null（不再创建）', () => {
    const store = useScheduleStore()
    const created: PkPlan[] = []
    // 预置到 MAX_PLANS 个方案（含初始 1 个）。
    for (let i = 1; i < MAX_PLANS; i++) {
      const plan = store.createPlan()
      expect(plan).not.toBeNull()
      created.push(plan!)
    }
    expect(store.state.plans).toHaveLength(MAX_PLANS)

    // 到达上限：拒绝新增且不改变状态。
    expect(store.createPlan()).toBeNull()
    expect(store.state.plans).toHaveLength(MAX_PLANS)

    // 删除一个后可继续新增。
    store.deletePlan(created[0].id)
    expect(store.state.plans).toHaveLength(MAX_PLANS - 1)
    expect(store.createPlan()).not.toBeNull()
    expect(store.state.plans).toHaveLength(MAX_PLANS)
  })

  test('学期/年级/专业变更清空所有方案（防跨学期污染）', () => {
    const store = useScheduleStore()
    store.createPlan()
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3], [1, 8])]))
    store.stageCourse(makeDetail('122004.01', 1, [3], [1, 8]))
    store.addCustomEvent({ label: '有事', day: 5, sections: [1, 2], weeks: [1, 2] })

    store.clearStagedAndSelectedCourses()

    for (const plan of store.state.plans) {
      expect(plan.stagedCourses).toEqual([])
      expect(plan.customEvents).toEqual([])
    }
    expect(store.state.timeTableData).toEqual([])
  })

  test('自定义占位事件：增删参与冲突派生', () => {
    const store = useScheduleStore()
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3], [1, 8])]))
    store.stageCourse(makeDetail('122004.01', 1, [3], [1, 8]))

    const event = store.addCustomEvent({ label: '有事', day: 1, sections: [3], weeks: [2, 3] })
    expect(event).not.toBeNull()
    // custom 事件进占用表并参与冲突。
    expect(store.state.occupied[2][0]).toHaveLength(2)
    const conflicts = deriveConflicts(store.state.occupied)
    const customKey = `${CUSTOM_EVENT_CODE_PREFIX}${event!.id}`
    expect(conflicts.get(customKey)?.[0].code).toBe('122004.01')
    expect(conflicts.get('122004')?.[0].code).toBe(customKey)

    // stats 的冲突计数只统计真实课程。
    expect(store.stats().conflictCount).toBe(1)

    store.removeCustomEvent(event!.id)
    expect(store.state.occupied[2][0]).toHaveLength(1)
    expect(deriveConflicts(store.state.occupied).size).toBe(0)
  })

  test('stats：门数/学分/学时/冲突数', () => {
    const store = useScheduleStore()
    // 122004：2 节 × 8 周 = 16 学时；G100：1 节 × 16 周 = 16 学时。
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3, 4], [1, 8])], 4))
    store.stageCourse(makeDetail('122004.01', 1, [3, 4], [1, 8]))

    store.setClickedCourseInfo({ courseCode: 'G100', courseName: '通识' })
    store.pushStagedCourse(makeStaged('G100', [makeDetail('G100.01', 2, [5], [1, 16])], 2))
    store.stageCourse(makeDetail('G100.01', 2, [5], [1, 16]))

    // 未排课的课程不计入门数。
    store.pushStagedCourse(makeStaged('G200', [makeDetail('G200.01', 3, [1], [1, 16])], 1))

    const summary = store.stats()
    expect(summary.courseCount).toBe(2)
    expect(summary.totalCredit).toBe(6)
    expect(summary.totalHours).toBe(32)
    expect(summary.conflictCount).toBe(0)
  })

  test('popStagedCourse 从备选/已选/课表/占用四处移除课程', () => {
    const store = useScheduleStore()
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3, 4], [1, 8])]))
    store.stageCourse(makeDetail('122004.01', 1, [3, 4], [1, 8]))
    store.saveSelectedCourses()

    store.popStagedCourse('122004')

    expect(store.state.commonLists.stagedCourses).toEqual([])
    expect(store.state.commonLists.selectedCourses).toEqual([])
    expect(store.state.timeTableData).toEqual([])
    expect(store.state.occupied[2][0]).toEqual([])
  })

  test('学分统计正确区分专业/通识', () => {
    const store = useScheduleStore()
    store.setMajorInfo({ calendarId: 119, grade: 2024, major: '123' })
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse({
      ...makeStaged('122004', [makeDetail('122004.01', 1, [3], [1, 8])], 4),
      courseDetail: [{ ...makeDetail('122004.01', 1, [3], [1, 8]), isExclusive: true }],
    })
    store.stageCourse({ ...makeDetail('122004.01', 1, [3], [1, 8]), isExclusive: true })

    store.setClickedCourseInfo({ courseCode: 'G100', courseName: '通识' })
    store.pushStagedCourse({
      ...makeStaged('G100', [makeDetail('G100.01', 2, [3], [1, 8])], 2),
      courseType: '选',
    })
    store.stageCourse(makeDetail('G100.01', 2, [3], [1, 8]))

    store.saveSelectedCourses()
    const summary = store.creditSummary()
    expect(summary.selectedTotal).toBe(6)
    expect(summary.selectedMajor).toBe(4)
    expect(summary.selectedGeneral).toBe(2)
  })

  test('solidify 持久化 plans/activePlanId/weekView（不再写派生态）', () => {
    const storage = makeStorage()
    vi.stubGlobal('window', storage)
    const store = useScheduleStore()
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3], [1, 8])]))
    store.stageCourse(makeDetail('122004.01', 1, [3], [1, 8]))
    store.setWeekView({ week: 3, useCurrent: false })
    store.solidify()

    const plans = writtenJson(storage.localStorage.setItem, 'pk.plans')
    expect(Array.isArray(plans)).toBe(true)
    expect(plans[0].stagedCourses[0].courseCode).toBe('122004')
    // 旧派生态键不再写入。
    const writtenKeys = storage.localStorage.setItem.mock.calls.map((call) => call[0])
    expect(writtenKeys).not.toContain('pk.occupied')
    expect(writtenKeys).not.toContain('pk.timeTableData')
    // weekView 持久化。
    const weekView = writtenJson(storage.localStorage.setItem, 'pk.weekView')
    expect(weekView).toEqual({ week: 3, useCurrent: false })
  })

  test('applySyncToAllPlans 保留各方案班级状态', () => {
    const store = useScheduleStore()
    const first = store.state.plans[0]
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3], [1, 8])]))
    store.stageCourse(makeDetail('122004.01', 1, [3], [1, 8])) // status=1

    const second = store.createPlan()!
    store.switchPlan(second.id)
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3], [1, 8])]))

    // 同步返回同课号新详情（班级相同时）。
    store.applySyncToAllPlans({
      '122004': [{ ...makeDetail('122004.01', 1, [5], [1, 8]), status: 0 }],
    })

    const firstPlan = store.state.plans.find((plan) => plan.id === first.id)!
    const secondPlan = store.state.plans.find((plan) => plan.id === second.id)!
    // 方案一保留排课状态（status=1），方案二未排（status=0）。
    expect(firstPlan.stagedCourses[0].courseDetail[0].status).toBe(1)
    expect(secondPlan.stagedCourses[0].courseDetail[0].status).toBe(0)
    // 新时间生效。
    expect(firstPlan.stagedCourses[0].courseDetail[0].arrangementInfo[0].occupyTime).toEqual([5])
  })

  test('同课换班按班级课号精确替换：旧班不复活', () => {
    const store = useScheduleStore()
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(
      makeStaged('122004', [
        makeDetail('122004.01', 1, [3, 4], [1, 8]),
        makeDetail('122004.02', 1, [5, 6], [1, 8]),
      ]),
    )

    store.stageCourse(makeDetail('122004.01', 1, [3, 4], [1, 8]))
    store.stageCourse(makeDetail('122004.02', 1, [5, 6], [1, 8]))

    const details = store.state.commonLists.stagedCourses[0].courseDetail
    expect(details[0].status).toBe(0)
    expect(details[1].status).toBe(1)

    // 切走再切回（触发纯重建）：只有新班在表。
    const other = store.createPlan()!
    store.switchPlan(other.id)
    store.switchPlan(store.state.plans.find((plan) => plan.stagedCourses.length > 0)!.id)
    expect(store.state.timeTableData.map((row) => row.code)).toEqual(['122004.02'])
  })

  test('syncLatestData 仅推进同步时间：课程与自定义占位保留', () => {
    const store = useScheduleStore()
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3], [1, 8])]))
    store.stageCourse(makeDetail('122004.01', 1, [3], [1, 8]))
    store.addCustomEvent({ label: '有事', day: 2, sections: [3], weeks: [1, 2] })

    store.setLatestUpdateTime('2026-08-30T00:00:00Z')
    store.syncLatestData()

    expect(store.state.updateTime).toBe('2026-08-30T00:00:00Z')
    expect(store.state.flags.isDataOutdated).toBe(false)
    expect(store.state.plans[0].stagedCourses).toHaveLength(1)
    expect(store.state.plans[0].customEvents).toHaveLength(1)
    // 课表行 = 1 门课程 + 1 个占位事件（各一行），全部保留。
    expect(store.state.timeTableData).toHaveLength(2)
  })

  test('getCourseBaseCode 兼容两种班号格式', () => {
    expect(getCourseBaseCode('122004.01')).toBe('122004')
    expect(getCourseBaseCode('12200401')).toBe('122004')
  })
})

describe('周次工具', () => {
  test('currentWeekForDate 计算与边界', () => {
    // 2025-09-08 是周一。
    expect(currentWeekForDate('2025-09-08', new Date('2025-09-08T10:00:00'))).toBe(1)
    expect(currentWeekForDate('2025-09-08', new Date('2025-09-14T23:59:59'))).toBe(1)
    expect(currentWeekForDate('2025-09-08', new Date('2025-09-15T00:00:00'))).toBe(2)
    expect(currentWeekForDate('2025-09-08', new Date('2025-12-28T00:00:00'))).toBe(16)
    // 超出 16 周支持范围：学期外返回 null（不夹取，避免学期结束后永远停在第 16 周）。
    expect(currentWeekForDate('2025-09-08', new Date('2026-06-01T00:00:00'))).toBeNull()
    // 学期前：null。
    expect(currentWeekForDate('2025-09-08', new Date('2025-09-01T00:00:00'))).toBeNull()
    // endDate：学期最后一天仍算校内（含当日），次日为 null。
    expect(currentWeekForDate('2025-09-08', new Date('2025-12-28T10:00:00'), '2025-12-28')).toBe(16)
    expect(currentWeekForDate('2025-09-08', new Date('2025-12-29T00:00:00'), '2025-12-28')).toBeNull()
    // 非法日期：null。
    expect(currentWeekForDate('not-a-date')).toBeNull()
  })

  test('formatWeeksText 区间与枚举', () => {
    expect(formatWeeksText([1, 2, 3, 4])).toBe('1-4')
    expect(formatWeeksText([1, 3, 5, 7])).toBe('1,3,5,7')
    expect(formatWeeksText([2])).toBe('2')
    expect(formatWeeksText([1, 2, 3, 5, 6, 9])).toBe('1-3,5-6,9')
    expect(formatWeeksText([])).toBe('')
    expect(formatWeeksText(undefined)).toBe('')
  })
})

describe('deriveConflicts 判据一致性', () => {
  const arr = (day: number, week: number[]): PkArrangement => ({
    arrangementText: '',
    occupyDay: day,
    occupyTime: [3],
    occupyWeek: week,
    occupyRoom: '',
    teacherAndCode: '',
  })

  test('同格多课按周次交集互标，同课不标', () => {
    let occupied = createEmptyOccupied()
    occupied = insertOccupied(occupied, [arr(1, [1, 2])], '122004.01', '高数')
    occupied = insertOccupied(occupied, [arr(1, [2, 3])], '122005.01', '英语')
    // 122006 周次与另两门不重叠 → 不参与冲突。
    occupied = insertOccupied(occupied, [arr(1, [5])], '122006.01', '体育')
    const conflicts = deriveConflicts(occupied)
    expect(conflicts.size).toBe(2)
    expect(conflicts.get('122004')?.map((item) => item.code)).toEqual(['122005.01'])
    expect(conflicts.get('122005')?.map((item) => item.code)).toEqual(['122004.01'])
    expect(conflicts.has('122006')).toBe(false)
  })

  test('conflictBaseOf：custom 原样、真实课号归一到基础课号', () => {
    expect(conflictBaseOf('122004.01')).toBe('122004')
    expect(conflictBaseOf('12200401')).toBe('122004')
    expect(conflictBaseOf(`${CUSTOM_EVENT_CODE_PREFIX}evt_1`)).toBe(`${CUSTOM_EVENT_CODE_PREFIX}evt_1`)
  })

  test('custom 伪课号不被 getCourseBaseCode 误裁', () => {
    let occupied = createEmptyOccupied()
    occupied = insertOccupied(occupied, [arr(2, [1])], `${CUSTOM_EVENT_CODE_PREFIX}evt_1`, '有事')
    occupied = insertOccupied(occupied, [arr(2, [1])], '122004.01', '高数')
    const conflicts = deriveConflicts(occupied)
    expect(conflicts.get(`${CUSTOM_EVENT_CODE_PREFIX}evt_1`)?.[0].code).toBe('122004.01')
    expect(conflicts.get('122004')?.[0].code).toBe(`${CUSTOM_EVENT_CODE_PREFIX}evt_1`)
  })
})

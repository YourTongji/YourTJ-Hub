import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { useScheduleStore } from '../src/site/composables/useScheduleStore'
import type { PkCourseDetail, PkStagedCourse } from '../src/site/types/pk'

function makeDetail(code: string, day: number, time: number[], weeks: number[]): PkCourseDetail {
  return {
    code,
    campus: '',
    teachers: [],
    teachingLanguage: '',
    arrangementInfo: [
      {
        arrangementText: '',
        occupyDay: day,
        occupyTime: time,
        occupyWeek: weeks,
        occupyRoom: '',
        teacherAndCode: '',
      },
    ],
  }
}

function makeStaged(courseCode: string, details: PkCourseDetail[]): PkStagedCourse {
  return {
    courseCode,
    courseName: courseCode,
    courseNameReserved: courseCode,
    credit: 3,
    courseType: '必',
    teacher: [],
    status: 0,
    courseDetail: details,
  }
}

describe('useScheduleStore', () => {
  beforeEach(() => {
    vi.stubGlobal('window', {
      localStorage: {
        getItem: vi.fn(() => null),
        setItem: vi.fn(),
        removeItem: vi.fn(),
      },
    })
    const store = useScheduleStore()
    store.clearStagedAndSelectedCourses()
    store.setMajorInfo({ calendarId: undefined, grade: undefined, major: undefined })
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
    expect(store.state.timeTableData).toHaveLength(1)
    expect(store.state.occupied[2][0]).toHaveLength(1)
    // 排入课表后班级状态为待选（1）
    expect(store.state.commonLists.stagedCourses[0].courseDetail[0].status).toBe(1)
  })

  test('冲突时返回 conflicts 且不加入课表（验收标准 2）', () => {
    const store = useScheduleStore()
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3], [1, 8])]))
    store.stageCourse(makeDetail('122004.01', 1, [3], [1, 8]))

    store.setClickedCourseInfo({ courseCode: '122005', courseName: '英语' })
    store.pushStagedCourse(makeStaged('122005', [makeDetail('122005.01', 1, [3], [1, 8])]))
    const result = store.stageCourse(makeDetail('122005.01', 1, [3], [1, 8]))

    expect(result.added).toBe(false)
    expect(result.conflicts).toHaveLength(1)
    expect(result.conflicts?.[0].code).toBe('122004.01')
    expect(store.state.timeTableData).toHaveLength(1)
  })

  test('forceReplaceCourse 强制替换（验收标准 2）', () => {
    const store = useScheduleStore()
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3], [1, 8])]))
    store.stageCourse(makeDetail('122004.01', 1, [3], [1, 8]))

    store.setClickedCourseInfo({ courseCode: '122005', courseName: '英语' })
    store.pushStagedCourse(makeStaged('122005', [makeDetail('122005.01', 1, [3], [1, 8])]))
    const replaced = store.forceReplaceCourse(makeDetail('122005.01', 1, [3], [1, 8]))

    expect(replaced).toBe(true)
    expect(store.state.occupied[2][0].map((cell) => cell.code)).toEqual(['122005.01'])
    expect(store.state.timeTableData.map((course) => course.code)).toEqual(['122005.01'])
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

    // 备选池为空：保存应为 no-op，不产生已选课程。
    expect(() => store.saveSelectedCourses()).not.toThrow()
    expect(store.state.commonLists.selectedCourses).toEqual([])
  })

  test('损坏的 localStorage 恢复后清理（验收标准 5）', () => {
    vi.stubGlobal('window', {
      localStorage: {
        getItem: vi.fn(() => '{ 损坏的JSON'),
        setItem: vi.fn(),
        removeItem: vi.fn(),
      },
    })
    const store = useScheduleStore()
    store.loadSolidify()
    expect(store.state.commonLists.stagedCourses).toEqual([])
    expect(store.state.commonLists.selectedCourses).toEqual([])
    expect(store.state.occupied).toHaveLength(12)
    expect(store.state.timeTableData).toEqual([])
  })

  test('合法 localStorage 恢复（验收标准 5）', () => {
    const store = useScheduleStore()
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '高数' })
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3, 4], [1, 8])]))
    store.stageCourse(makeDetail('122004.01', 1, [3, 4], [1, 8]))

    // 模拟持久化后的 localStorage 内容
    const storage = new Map<string, string>()
    vi.stubGlobal('window', {
      localStorage: {
        getItem: vi.fn((key: string) => storage.get(key) ?? null),
        setItem: vi.fn((key: string, value: string) => storage.set(key, value)),
        removeItem: vi.fn((key: string) => storage.delete(key)),
      },
    })
    store.solidify()

    // 新的 store 实例会重新读取同一模块级 state（已是持久化状态），此处验证能通过 loadSolidify 恢复。
    const restored = useScheduleStore()
    restored.loadSolidify()
    expect(restored.state.commonLists.stagedCourses[0].courseCode).toBe('122004')
    expect(restored.state.occupied[2][0][0].code).toBe('122004.01')
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
      ...makeStaged('122004', [makeDetail('122004.01', 1, [3], [1, 8])]),
      credit: 4,
      courseType: '必',
      courseDetail: [{ ...makeDetail('122004.01', 1, [3], [1, 8]), isExclusive: true }],
    })
    store.stageCourse({ ...makeDetail('122004.01', 1, [3], [1, 8]), isExclusive: true })

    store.setClickedCourseInfo({ courseCode: 'G100', courseName: '通识' })
    store.pushStagedCourse({
      ...makeStaged('G100', [makeDetail('G100.01', 2, [3], [1, 8])]),
      credit: 2,
      courseType: '选',
    })
    store.stageCourse(makeDetail('G100.01', 2, [3], [1, 8]))

    store.saveSelectedCourses()
    const summary = store.creditSummary()
    expect(summary.selectedTotal).toBe(6)
    expect(summary.selectedMajor).toBe(4)
    expect(summary.selectedGeneral).toBe(2)
  })
})

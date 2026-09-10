// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'

import { i18n } from '../src/runtime/i18n'

import ScheduleCellPicker from '../src/site/components/schedule/ScheduleCellPicker.vue'
import { useScheduleStore } from '../src/site/composables/useScheduleStore'
import { getPkCourseDetails, getPkCoursesByTime } from '../src/runtime/pk-api'
import type { PkCourse, PkCourseDetail, PkCourseOnTable, PkStagedCourse } from '../src/site/types/pk'

vi.mock('../src/runtime/pk-api', () => ({
  getPkCoursesByTime: vi.fn(async () => ({ courses: [], auxiliaryReady: true })),
  getPkCourseDetails: vi.fn(async () => ({})),
}))

function makeDetail(code: string, day: number, time: number[]): PkCourseDetail {
  return {
    code,
    campus: '四平路校区',
    teachers: [{ teacherCode: 'T1', teacherName: '张三' }],
    teachingLanguage: '中文',
    arrangementInfo: [
      {
        arrangementText: `周${day} ${time.join('-')}节`,
        occupyDay: day,
        occupyTime: time,
        occupyWeek: [1, 8],
        occupyRoom: 'A101',
        teacherAndCode: '张三(T1)',
      },
    ],
  }
}

function makeStaged(courseCode: string, details: PkCourseDetail[]): PkStagedCourse {
  return {
    courseCode,
    courseName: courseCode,
    courseNameReserved: `课程${courseCode}`,
    credit: 3,
    courseType: '必',
    teacher: [],
    status: 0,
    courseDetail: details,
  }
}

function makeCourse(courseCode: string): PkCourse {
  return {
    courseCode,
    courseName: `课程${courseCode}`,
    courseNameReserved: `课程${courseCode}`,
    credit: 3,
    courseType: '必',
    status: 0,
    teacher: [],
    courseDetail: [],
  }
}

function mountPicker(
  day: number,
  section: number,
  options?: { replacingCourse?: PkCourseOnTable | null },
): VueWrapper {
  return mount(ScheduleCellPicker, {
    props: {
      open: true,
      day,
      section,
      replacingCourse: options?.replacingCourse,
    },
    global: {
      plugins: [i18n],
    },
    // reka-ui DialogPortal 渲染到 body：断言/交互都走 document（与 schedule-course-picker.test.ts 一致）。
    attachTo: document.body,
  })
}

/** 当前弹窗内的候选课程行。 */
function dialogRows(): HTMLElement[] {
  return [...document.querySelectorAll<HTMLElement>('[role="dialog"] li')]
}

/** 点击弹窗内第 index 个候选课程。 */
async function clickCandidate(index: number): Promise<void> {
  // reka-ui DialogPortal 内容异步挂载：先等渲染完成再查询按钮。
  await flushPromises()
  const buttons = document.querySelectorAll<HTMLElement>('[role="dialog"] li button')
  buttons[index]?.click()
  await flushPromises()
}

afterEach(() => {
  document.body.innerHTML = ''
  vi.clearAllMocks()
})

describe('ScheduleCellPicker 时段备选课程选择框', () => {
  beforeEach(() => {
    const store = useScheduleStore()
    store.clearStagedAndSelectedCourses()
    store.setCompulsoryCourses([])
    store.setMajorInfo({ calendarId: 121, grade: 2025, major: '00301' })
    vi.mocked(getPkCoursesByTime).mockResolvedValue({ courses: [], auxiliaryReady: true })
    vi.mocked(getPkCourseDetails).mockResolvedValue({})
  })

  test('只展示「该天该节次有课」的教学班', async () => {
    const store = useScheduleStore()
    store.pushStagedCourse(
      makeStaged('122004', [
        makeDetail('122004.01', 1, [3, 4]), // 周一 3-4 节 → 命中 day=1 section=3
        makeDetail('122004.02', 2, [3, 4]), // 周二 3-4 节 → 不命中
      ]),
    )

    const wrapper = mountPicker(1, 3)
    await flushPromises()

    const rows = dialogRows()
    expect(rows).toHaveLength(1)
    expect(rows[0].textContent).toContain('122004.01')
    expect(rows[0].textContent).not.toContain('122004.02')
  })

  test('该时段无备选课程时显示空态', async () => {
    const wrapper = mountPicker(5, 1)
    await flushPromises()
    // happy-dom 下 i18n 检测为 en，断言英文空态文案。
    expect(document.querySelector('[role="dialog"]')?.textContent).toContain('No staged courses at this time')
  })

  test('计划内课程在备选池和同时段全校列表中置顶，其他课程保持原序', async () => {
    const store = useScheduleStore()
    store.pushStagedCourse(makeStaged('122004', [makeDetail('122004.01', 1, [3])]))
    store.pushStagedCourse(makeStaged('122005', [makeDetail('122005.01', 1, [3])]))
    store.pushStagedCourse(makeStaged('122006', [makeDetail('122006.01', 1, [3])]))
    store.setCompulsoryCourses([makeCourse('122005'), makeCourse('CHEM101')])
    vi.mocked(getPkCoursesByTime).mockResolvedValue({
      courses: [makeCourse('BIO101'), makeCourse('CHEM101'), makeCourse('PHY101')],
      auxiliaryReady: true,
    })

    mountPicker(1, 3)
    await flushPromises()

    const stagedNames = dialogRows().map((row) => row.querySelector('span')?.textContent?.trim())
    expect(stagedNames).toEqual(['课程122005', '课程122004', '课程122006'])

    const dialogText = document.querySelector('[role="dialog"]')?.textContent ?? ''
    expect(dialogText.indexOf('课程CHEM101')).toBeLessThan(dialogText.indexOf('课程BIO101'))
    expect(dialogText.indexOf('课程BIO101')).toBeLessThan(dialogText.indexOf('课程PHY101'))
  })

  test('点击候选课程：无冲突时加入课表并关闭弹窗', async () => {
    const store = useScheduleStore()
    const detail = makeDetail('122004.01', 1, [3, 4])
    store.pushStagedCourse(makeStaged('122004', [detail]))
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '课程122004' })

    const wrapper = mountPicker(1, 3)
    await clickCandidate(0)
    await flushPromises()

    // 无冲突：教学班排入课表（timeTableData + occupied）、班级状态置为备选。
    expect(store.state.timeTableData).toHaveLength(1)
    expect(store.state.timeTableData[0].code).toBe('122004.01')
    expect(store.state.occupied[2][0]).toHaveLength(1)
    expect(store.state.commonLists.stagedCourses[0].courseDetail[0].status).toBe(1)
    // emit staged 并关闭弹窗，未 emit conflict。
    expect(wrapper.emitted('staged')).toHaveLength(1)
    expect(wrapper.emitted('close')).toHaveLength(1)
    expect(wrapper.emitted('conflict')).toBeUndefined()
  })

  test('点击候选课程：即使上次打开的是别的课程，课表行也用候选课程信息（review P1）', async () => {
    const store = useScheduleStore()
    const detail = makeDetail('122004.01', 1, [3, 4])
    store.pushStagedCourse(makeStaged('122004', [detail]))
    // 故意把 clickedCourseInfo 设成另一门课：候选课程名/代码必须覆盖陈旧上下文。
    store.setClickedCourseInfo({ courseCode: '999999', courseName: '上次打开的课程' })

    const wrapper = mountPicker(1, 3)
    await clickCandidate(0)
    await flushPromises()

    // 课表行与占用格使用候选课程名（而非上次打开的课程名）。
    expect(store.state.timeTableData[0].courseName).toBe('课程122004')
    expect(store.state.timeTableData[0].code).toBe('122004.01')
    expect(store.state.occupied[2][0][0].courseName).toBe('课程122004')
    expect(store.state.clickedCourseInfo.courseCode).toBe('122004')
    expect(store.state.clickedCourseInfo.courseName).toBe('课程122004')
  })

  test('点击候选课程：冲突时 emit conflict 并关闭弹窗', async () => {
    const store = useScheduleStore()
    const detail = makeDetail('122004.01', 1, [3, 4])
    store.pushStagedCourse(makeStaged('122004', [detail]))
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '课程122004' })
    // 先占用周一 3-4 节，再点候选必冲突。
    const occupied = store.state.occupied
    const other = makeDetail('999999.01', 1, [3, 4])
    occupied[2][0] = [
      {
        code: other.code,
        courseName: '其他课',
        occupyWeek: [1, 8],
      },
    ]

    const wrapper = mountPicker(1, 3)
    await clickCandidate(0)
    await flushPromises()

    expect(wrapper.emitted('conflict')).toHaveLength(1)
    expect(wrapper.emitted('staged')).toBeUndefined()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  test('从该时段全校选修课 API 加载课程并支持展开班级后加入课表', async () => {
    const electiveCourse: PkCourse = {
      courseCode: 'BIO101',
      courseName: '生物学概论',
      courseNameReserved: '生物学概论',
      credit: 2,
      courseType: '选',
      faculty: '生命科学与技术学院',
      campus: '四平路校区',
      status: 0,
      teacher: [],
      courseDetail: [],
    }

    const electiveDetail = makeDetail('BIO101.01', 2, [3, 4])

    vi.mocked(getPkCoursesByTime).mockResolvedValue({
      courses: [electiveCourse],
      auxiliaryReady: true,
    })
    vi.mocked(getPkCourseDetails).mockResolvedValue({
      BIO101: [electiveDetail],
    })

    const wrapper = mountPicker(2, 3)
    await flushPromises()

    // 验证调用了 getPkCoursesByTime(calendarId=121, day=2, section=2)
    // 节次 3 对应第 2 大节（3-4节）
    expect(getPkCoursesByTime).toHaveBeenCalledWith(121, 2, 2)

    // 对话框中能看到该课程
    const dialog = document.querySelector('[role="dialog"]')
    expect(dialog?.textContent).toContain('生物学概论')
    expect(dialog?.textContent).toContain('BIO101')

    // 点击展开班级按钮
    const expandBtn = [...dialog?.querySelectorAll('button') || []].find((btn) =>
      btn.textContent?.includes('View classes') || btn.textContent?.includes('查看开课班级'),
    )
    expect(expandBtn).toBeDefined()
    expandBtn?.click()
    await flushPromises()

    // 验证调用了 getPkCourseDetails
    expect(getPkCourseDetails).toHaveBeenCalledWith(121, ['BIO101'])

    // 能看到班级 BIO101.01
    expect(dialog?.textContent).toContain('BIO101.01')

    // 点击加入课表
    const addBtn = [...dialog?.querySelectorAll('button') || []].find((btn) =>
      btn.textContent?.includes('Add to schedule') || btn.textContent?.includes('加入课表'),
    )
    expect(addBtn).toBeDefined()
    addBtn?.click()
    await flushPromises()

    const store = useScheduleStore()
    expect(store.state.timeTableData.some((c) => c.code === 'BIO101.01')).toBe(true)
    expect(wrapper.emitted('staged')).toHaveLength(1)
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  test('同时段替换模式：点击新班级自动替换原课程并触发 replaced 事件', async () => {
    const store = useScheduleStore()
    // 先把原课程 122004.01 排入课表
    const oldDetail = makeDetail('122004.01', 1, [3, 4])
    store.pushStagedCourse(makeStaged('122004', [oldDetail]))
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '旧课程' })
    store.stageCourse(oldDetail)
    store.solidify()
    expect(store.state.timeTableData).toHaveLength(1)

    // 新课程
    const newDetail = makeDetail('233005.01', 1, [3, 4])
    store.pushStagedCourse(makeStaged('233005', [newDetail]))

    const replacingCourse: PkCourseOnTable = {
      ...store.state.timeTableData[0],
      courseName: '旧课程',
      code: '122004.01',
    }

    const wrapper = mountPicker(1, 3, { replacingCourse })
    await flushPromises()

    // 弹窗中应排除正在被替换的班级 122004.01，只显示 233005.01
    const rows = dialogRows()
    expect(rows).toHaveLength(1)
    expect(rows[0].textContent).toContain('233005.01')
    expect(rows[0].textContent).not.toContain('122004.01')

    // 点击替换
    await clickCandidate(0)
    await flushPromises()

    // 原课程已被移除，新课程排入课表
    expect(store.state.timeTableData.some((c) => c.code === '122004.01')).toBe(false)
    expect(store.state.timeTableData.some((c) => c.code === '233005.01')).toBe(true)

    // 触发 replaced 和 close 事件
    expect(wrapper.emitted('replaced')).toHaveLength(1)
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})

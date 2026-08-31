// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import { i18n } from '../src/runtime/i18n'

import ScheduleTimeTable from '../src/site/components/schedule/ScheduleTimeTable.vue'
import { useScheduleStore } from '../src/site/composables/useScheduleStore'
import type { PkCourseDetail, PkStagedCourse } from '../src/site/types/pk'

function makeDetail(code: string, day: number, time: number[], weeks: number[]): PkCourseDetail {
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
        occupyWeek: weeks,
        occupyRoom: 'A101',
        teacherAndCode: '张三(T1)',
      },
    ],
  }
}

function makeStaged(courseCode: string, name: string, details: PkCourseDetail[]): PkStagedCourse {
  return {
    courseCode,
    courseName: courseCode,
    courseNameReserved: name,
    credit: 3,
    courseType: '必',
    teacher: [],
    status: 0,
    courseDetail: details,
  }
}

function mountTable() {
  return mount(ScheduleTimeTable, {
    global: {
      plugins: [i18n],
    },
    attachTo: document.body,
  })
}

/** 同一格内的课程块：course 块是 role=button 且 aria-label=课名（空格 td 无课名 label）。 */
function cellBlocks(nameA: string, nameB: string): HTMLElement[] {
  const blocks = [...document.querySelectorAll<HTMLElement>('[role="button"]')]
  return blocks.filter((el) => {
    const label = el.getAttribute('aria-label') ?? ''
    return label.includes(nameA) || label.includes(nameB)
  })
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('ScheduleTimeTable 同格多课渲染', () => {
  beforeEach(() => {
    const store = useScheduleStore()
    store.clearStagedAndSelectedCourses()
    store.setWeekView({ week: null, useCurrent: false })
  })

  test('全部周次下同格两门课竖向堆叠（不再横向细条）', async () => {
    const store = useScheduleStore()
    // 同周一 3-4 节、周次重叠（真冲突，容忍式同格）。
    const detailA = makeDetail('100A.01', 1, [3, 4], [1, 8])
    const detailB = makeDetail('100B.01', 1, [3, 4], [1, 8])
    store.pushStagedCourse(makeStaged('100A', '课程甲', [detailA]))
    store.pushStagedCourse(makeStaged('100B', '课程乙', [detailB]))
    // 真实流程：点课/弹窗选班先写 clickedCourseInfo，appendToTimeTable 以此命名课表行。
    store.setClickedCourseInfo({ courseCode: '100A', courseName: '课程甲' })
    store.stageCourse(detailA)
    store.setClickedCourseInfo({ courseCode: '100B', courseName: '课程乙' })
    store.stageCourse(detailB)
    store.solidify()

    mountTable()
    await flushPromises()

    const blocks = cellBlocks('课程甲', '课程乙')
    expect(blocks).toHaveLength(2)
    // 两块同属一个格容器，容器为纵向 flex（竖排），不再 flex-row（横排细条）。
    const container = blocks[0].parentElement
    expect(container).toBeTruthy()
    expect(container!.className).toContain('flex-col')
    expect(container!.className).not.toContain('flex-row')
  })

  test('单双周同位共存：不判冲突且两门课都在该位置显示', async () => {
    const store = useScheduleStore()
    // 同周一 1-2 节：甲单周、乙双周——周次无交集，不构成冲突。
    const detailA = makeDetail('200A.01', 1, [1, 2], [1, 3, 5, 7])
    const detailB = makeDetail('200B.01', 1, [1, 2], [2, 4, 6, 8])
    store.pushStagedCourse(makeStaged('200A', '单周课', [detailA]))
    store.pushStagedCourse(makeStaged('200B', '双周课', [detailB]))
    store.setClickedCourseInfo({ courseCode: '200A', courseName: '单周课' })
    store.stageCourse(detailA)
    store.setClickedCourseInfo({ courseCode: '200B', courseName: '双周课' })
    store.stageCourse(detailB)
    store.solidify()

    mountTable()
    await flushPromises()

    // 两门课都渲染在同一格。
    const blocks = cellBlocks('单周课', '双周课')
    expect(blocks).toHaveLength(2)
    // 周次无交集：无 ⚠ 冲突角标（title=Conflict，happy-dom 下 i18n 为 en）。
    expect(document.querySelector('[title="Conflict"]')).toBeNull()
    expect(store.stats().conflictCount).toBe(0)
  })
})

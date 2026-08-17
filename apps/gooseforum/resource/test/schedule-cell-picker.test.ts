// @vitest-environment happy-dom
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'

import { i18n } from '../src/runtime/i18n'

import ScheduleCellPicker from '../src/site/components/schedule/ScheduleCellPicker.vue'
import { useScheduleStore } from '../src/site/composables/useScheduleStore'
import type { PkCourseDetail, PkStagedCourse } from '../src/site/types/pk'

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

function mountPicker(day: number, section: number): VueWrapper {
  return mount(ScheduleCellPicker, {
    props: { open: true, day, section },
    global: {
      plugins: [i18n],
      // Teleport 内容直接渲染进 wrapper，便于断言弹窗内容与交互。
      stubs: { teleport: true },
    },
  })
}

describe('ScheduleCellPicker 时段备选课程选择框', () => {
  beforeEach(() => {
    const store = useScheduleStore()
    store.clearStagedAndSelectedCourses()
    store.setMajorInfo({ calendarId: 121, grade: 2025, major: '00301' })
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

    const rows = wrapper.findAll('li')
    expect(rows).toHaveLength(1)
    expect(rows[0].text()).toContain('122004.01')
    expect(rows[0].text()).not.toContain('122004.02')
  })

  test('该时段无备选课程时显示空态', async () => {
    const wrapper = mountPicker(5, 1)
    await flushPromises()
    // happy-dom 下 i18n 检测为 en，断言英文空态文案。
    expect(wrapper.text()).toContain('No staged courses at this time')
  })

  test('点击候选课程：无冲突时加入课表并关闭弹窗', async () => {
    const store = useScheduleStore()
    const detail = makeDetail('122004.01', 1, [3, 4])
    store.pushStagedCourse(makeStaged('122004', [detail]))
    store.setClickedCourseInfo({ courseCode: '122004', courseName: '课程122004' })

    const wrapper = mountPicker(1, 3)
    await flushPromises()
    await wrapper.find('li button').trigger('click')
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
    await flushPromises()
    await wrapper.find('li button').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('conflict')).toHaveLength(1)
    expect(wrapper.emitted('staged')).toBeUndefined()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})

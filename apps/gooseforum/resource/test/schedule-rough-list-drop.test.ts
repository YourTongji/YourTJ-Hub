// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'

import { i18n } from '../src/runtime/i18n'
import ScheduleRoughList from '../src/site/components/schedule/ScheduleRoughList.vue'
import { useScheduleStore } from '../src/site/composables/useScheduleStore'
import type { PkCourseDetail, PkStagedCourse } from '../src/site/types/pk'

function makeDetail(code: string): PkCourseDetail {
  return {
    code,
    campus: '四平路校区',
    teachers: [{ teacherCode: 'T1', teacherName: '张伟' }],
    teachingLanguage: '中文',
    arrangementInfo: [
      {
        arrangementText: '星期一3-4节[1-8周] 四平路校区 A101',
        occupyDay: 1,
        occupyTime: [3, 4],
        occupyWeek: [1, 8],
        occupyRoom: 'A101',
        teacherAndCode: '张伟(T1)',
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

/** 预置一门已选课程并持久化（模拟用户此前已保存的课表）。 */
function seedSelectedCourse(courseCode: string) {
  const store = useScheduleStore()
  store.setClickedCourseInfo({ courseCode, courseName: `课程${courseCode}` })
  store.pushStagedCourse(makeStaged(courseCode, [makeDetail(`${courseCode}.01`)]))
  store.stageCourse(makeDetail(`${courseCode}.01`))
  store.saveSelectedCourses()
  store.solidify()
}

/** 确认弹窗经 DialogPortal 渲染到 body，从 document 查询（不依赖 locale 文案）。 */
async function clickDialogButton(selector: string) {
  const button = document.querySelector<HTMLButtonElement>(selector)
  expect(button).not.toBeNull()
  button!.click()
  await flushPromises()
}

describe('ScheduleRoughList 退课持久化', () => {
  let mounted: VueWrapper | null = null

  beforeEach(() => {
    localStorage.clear()
    const store = useScheduleStore()
    store.clearStagedAndSelectedCourses()
    store.setMajorInfo({ calendarId: 121, grade: 2025, major: '00301' })
    seedSelectedCourse('122004')
    mounted = null
  })

  afterEach(() => {
    mounted?.unmount()
    mounted = null
    // DialogPortal 渲染到 body，卸载后清掉 portal 残留，避免串扰后续测试。
    document.body.innerHTML = ''
  })

  test('确认退课后立即写入 localStorage，重进页面课程不复活', async () => {
    // 回归：confirmDrop 曾漏调 solidify，删除只停留在内存，
    // 页面重挂载时 loadSolidify 用旧的 localStorage 覆盖内存，课程"复活"。
    const store = useScheduleStore()
    expect(localStorage.getItem('pk.plans')).toContain('122004')

    mounted = mount(ScheduleRoughList, { global: { plugins: [i18n] } })
    await flushPromises()

    // 点击课程行的退课按钮（status=2 时为退课），打开确认弹窗。
    const dropButton = mounted.find('li button.gf-button-ghost')
    expect(dropButton.exists()).toBe(true)
    await dropButton.trigger('click')
    await flushPromises()
    await clickDialogButton('.gf-button-danger')

    // 内存状态已移除，且 localStorage 同步移除（退课立即持久化）。
    expect(store.state.commonLists.stagedCourses).toEqual([])
    expect(store.state.commonLists.selectedCourses).toEqual([])
    expect(localStorage.getItem('pk.plans')).not.toContain('122004')
    expect(localStorage.getItem('pk.plans')).not.toContain('122004.01')

    // 模拟刷新/重进：loadSolidify 用持久化状态覆盖内存，退掉的课程不得复活。
    store.pushStagedCourse(makeStaged('999999', [makeDetail('999999.01')]))
    store.loadSolidify()
    expect(store.state.commonLists.stagedCourses.map((c) => c.courseCode)).not.toContain('122004')
    expect(store.state.commonLists.selectedCourses).not.toContain('122004.01')
  })

  test('取消退课不改动内存与 localStorage', async () => {
    const store = useScheduleStore()

    mounted = mount(ScheduleRoughList, { global: { plugins: [i18n] } })
    await flushPromises()

    await mounted.find('li button.gf-button-ghost').trigger('click')
    await flushPromises()
    await clickDialogButton('.gf-button-md.gf-button-ghost')

    expect(store.state.commonLists.stagedCourses.map((c) => c.courseCode)).toEqual(['122004'])
    expect(localStorage.getItem('pk.plans')).toContain('122004')
  })
})

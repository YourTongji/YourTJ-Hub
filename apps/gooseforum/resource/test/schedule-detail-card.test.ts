// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import { i18n } from '../src/runtime/i18n'
import ScheduleDetailCard from '../src/site/components/schedule/ScheduleDetailCard.vue'
import type { PkCourseOnTable } from '../src/site/types/pk'

vi.mock('../src/runtime/pk-api', () => ({
  getPkCourseReviewBrief: vi.fn(async () => ({
    courseId: 101,
    ratingAvg: 4.8,
    reviewCount: 15,
  })),
}))

const mockCourse: PkCourseOnTable = {
  courseName: '高等数学',
  code: 'MATH101.01',
  occupyDay: 2,
  occupyTime: [3, 4],
  occupyWeek: [1, 16],
  showText: '高数老师(T01) 高等数学(MATH101.01) 周二 3-4节 嘉定一教101',
  teacherAndCode: '高数老师(T01)',
  arrangementText: '周二 3-4节 嘉定一教101',
  occupyRoom: '嘉定一教101',
}

afterEach(() => {
  document.body.innerHTML = ''
  vi.clearAllMocks()
})

describe('ScheduleDetailCard 课程详情卡', () => {
  beforeEach(() => {
    i18n.global.locale.value = 'zh'
  })

  test('点击同时段替换按钮时触发 replace 与 close 事件', async () => {
    const wrapper = mount(ScheduleDetailCard, {
      props: { course: mockCourse },
      global: {
        plugins: [i18n],
      },
      attachTo: document.body,
    })
    await flushPromises()

    const dialog = document.querySelector('[role="dialog"]')
    expect(dialog?.textContent).toContain('高等数学')
    expect(dialog?.textContent).toContain('MATH101.01')

    // 查找同时段替换按钮
    const replaceBtn = [...dialog?.querySelectorAll('button') || []].find((btn) =>
      btn.textContent?.includes('同时段替换') || btn.textContent?.includes('Replace course'),
    )
    expect(replaceBtn).toBeDefined()
    replaceBtn?.click()
    await flushPromises()

    expect(wrapper.emitted('replace')).toHaveLength(1)
    expect(wrapper.emitted('replace')?.[0]).toEqual([mockCourse])
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})

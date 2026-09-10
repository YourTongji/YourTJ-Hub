// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'

import { i18n } from '../src/runtime/i18n'
import ScheduleRoughList from '../src/site/components/schedule/ScheduleRoughList.vue'
import { useScheduleStore } from '../src/site/composables/useScheduleStore'
import type { PkStagedCourse } from '../src/site/types/pk'

function makeCourse(code: string): PkStagedCourse {
  return {
    courseCode: code,
    courseName: code,
    courseNameReserved: `课程${code}`,
    credit: 2,
    courseType: '选',
    teacher: [],
    status: 0,
    courseDetail: [],
  }
}

describe('Schedule 移动端回归', () => {
  let wrapper: VueWrapper | null = null

  beforeEach(() => {
    localStorage.clear()
    const store = useScheduleStore()
    store.clearStagedAndSelectedCourses()
    store.pushStagedCourse(makeCourse('M001'))
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = null
    document.body.innerHTML = ''
  })

  test('移动端候选课程列表交给页面原生纵向滚动，桌面才启用内部滚动，点按课程仍可打开', async () => {
    wrapper = mount(ScheduleRoughList, { global: { plugins: [i18n] } })
    await flushPromises()

    const list = wrapper.get('ul')
    expect(list.classes()).toContain('lg:overflow-y-auto')
    expect(list.classes()).toContain('lg:overscroll-contain')
    expect(list.classes()).not.toContain('overflow-y-auto')
    expect(list.classes()).not.toContain('overscroll-contain')

    await wrapper.get('li button.group').trigger('click')
    expect(wrapper.emitted('openDetail')).toHaveLength(1)
  })
})

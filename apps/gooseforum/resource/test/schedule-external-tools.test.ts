// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'

import { i18n } from '../src/runtime/i18n'
import ScheduleExternalToolsTip from '../src/site/components/schedule/ScheduleExternalToolsTip.vue'

describe('ScheduleExternalToolsTip', () => {
  let wrapper: VueWrapper | null = null

  beforeEach(() => {
    i18n.global.locale.value = 'zh'
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = null
    document.body.innerHTML = ''
  })

  test('renders compact trigger button with hammer icon and without ping dot', () => {
    wrapper = mount(ScheduleExternalToolsTip, {
      global: { plugins: [i18n] },
      attachTo: document.body,
    })

    const trigger = wrapper.find('[data-testid="schedule-external-tools-trigger"]')
    expect(trigger.exists()).toBe(true)
    expect(trigger.text()).toContain('其他工具')

    // No blinking/ping beacon dot
    const pingDot = trigger.find('.animate-ping')
    expect(pingDot.exists()).toBe(false)

    // Contains accessible aria-label
    expect(trigger.attributes('aria-label')).toBe('其他选课与排课工具')
  })

  test('clicking trigger opens popover bubble with 2 external tools', async () => {
    wrapper = mount(ScheduleExternalToolsTip, {
      global: { plugins: [i18n] },
      attachTo: document.body,
    })

    const trigger = wrapper.find('[data-testid="schedule-external-tools-trigger"]')
    await trigger.trigger('click')
    await flushPromises()

    // Popover portal renders to document.body
    const popover = document.querySelector('[data-testid="schedule-external-tools-popover"]')
    expect(popover).toBeTruthy()
    expect(popover?.textContent).toContain('其他选课与排课工具')
    expect(popover?.textContent).toContain('同济同学开发的实用辅助工具')

    const links = popover?.querySelectorAll('a')
    expect(links?.length).toBe(2)

    // Tool 1: 同济排课助手
    const tool1 = links![0]
    expect(tool1.getAttribute('href')).toBe('https://xk.xialing.icu/')
    expect(tool1.getAttribute('target')).toBe('_blank')
    expect(tool1.getAttribute('rel')).toBe('noopener noreferrer')
    expect(tool1.textContent).toContain('同济排课助手')
    expect(tool1.textContent).toContain('xk.xialing.icu')
    expect(tool1.textContent).toContain('使用老版排课模拟器，模拟一系统体验')

    // Tool 2: 通济-模拟选课系统
    const tool2 = links![1]
    expect(tool2.getAttribute('href')).toBe('https://course.f1justin.com/')
    expect(tool2.getAttribute('target')).toBe('_blank')
    expect(tool2.getAttribute('rel')).toBe('noopener noreferrer')
    expect(tool2.textContent).toContain('通济-模拟选课系统')
    expect(tool2.textContent).toContain('course.f1justin.com')
    expect(tool2.textContent).toContain('强大的课程筛选工具')

    // No truncate, uses break-words
    for (const link of links!) {
      const desc = link.querySelector('p')
      expect(desc?.className).toContain('break-words')
      expect(desc?.className).not.toContain('truncate')
    }
  })

  test('supports english locale', async () => {
    i18n.global.locale.value = 'en'
    wrapper = mount(ScheduleExternalToolsTip, {
      global: { plugins: [i18n] },
      attachTo: document.body,
    })

    const trigger = wrapper.find('[data-testid="schedule-external-tools-trigger"]')
    expect(trigger.text()).toContain('Other Tools')

    await trigger.trigger('click')
    await flushPromises()

    const popover = document.querySelector('[data-testid="schedule-external-tools-popover"]')
    expect(popover?.textContent).toContain('Tongji Course Scheduler')
    expect(popover?.textContent).toContain('Tongji Course Explorer')
  })
})

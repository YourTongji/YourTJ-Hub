// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'

import { i18n } from '../src/runtime/i18n'

const getPkCalendars = vi.fn()
const getPkGrades = vi.fn()
const getPkMajors = vi.fn()
vi.mock('../src/runtime/pk-api', () => ({
  getPkCalendars: () => getPkCalendars(),
  getPkGrades: (calendarId: number) => getPkGrades(calendarId),
  getPkMajors: (grade: number, calendarId: number) => getPkMajors(grade, calendarId),
}))

import ScheduleConfigSection from '../src/site/components/schedule/ScheduleConfigSection.vue'
import { useScheduleStore } from '../src/site/composables/useScheduleStore'

describe('ScheduleConfigSection 折叠/展开交互与状态持久化', () => {
  let mounted: VueWrapper | null = null

  beforeEach(() => {
    vi.resetAllMocks()
    localStorage.clear()
    const store = useScheduleStore()
    store.setConfigCollapsed(false)
    store.state.majorSelected = {
      calendarId: 121,
      grade: 2024,
      major: '080901',
      majorName: '计算机科学与技术',
    }
    store.state.calendars = [
      { calendarId: 121, calendarName: '2025-2026学年第2学期' },
    ]
    getPkCalendars.mockResolvedValue([{ calendarId: 121, calendarName: '2025-2026学年第2学期' }])
    getPkGrades.mockResolvedValue([2024])
    getPkMajors.mockResolvedValue([{ code: '080901', name: '计算机科学与技术' }])
  })

  afterEach(() => {
    mounted?.unmount()
    mounted = null
  })

  test('默认状态：未保存收起偏好时，默认呈现展开状态', async () => {
    mounted = mount(ScheduleConfigSection, {
      global: { plugins: [i18n] },
    })
    await flushPromises()

    expect(mounted.find('[data-testid="schedule-config-expanded-wrap"]').exists()).toBe(true)
    expect(mounted.find('[data-testid="schedule-config-collapsed-bar"]').exists()).toBe(false)
  })

  test('收起与展开交互：调用 collapse 后显示摘要条，点击摘要条恢复展开', async () => {
    mounted = mount(ScheduleConfigSection, {
      global: { plugins: [i18n] },
    })
    await flushPromises()

    // 触发收起
    const vm = mounted.vm as any
    vm.collapse()
    await flushPromises()

    const collapsedBar = mounted.find('[data-testid="schedule-config-collapsed-bar"]')
    expect(collapsedBar.exists()).toBe(true)
    const store = useScheduleStore()
    const planName = store.state.plans.find((p) => p.id === store.state.activePlanId)?.name || 'Plan 1'
    expect(collapsedBar.text()).toContain(planName)
    expect(collapsedBar.text()).toContain('2025-2026学年第2学期')
    expect(collapsedBar.text()).toContain('2024')
    expect(collapsedBar.text()).toContain('计算机科学与技术')
    expect(localStorage.getItem('goose:scheduleConfigCollapsed')).toBe('1')

    // 点击收起条展开
    await collapsedBar.trigger('click')
    await flushPromises()

    expect(mounted.find('[data-testid="schedule-config-expanded-wrap"]').exists()).toBe(true)
    expect(localStorage.getItem('goose:scheduleConfigCollapsed')).toBe('0')
  })

  test('键盘交互：支持 Enter 和 Space 展开', async () => {
    mounted = mount(ScheduleConfigSection, {
      global: { plugins: [i18n] },
    })
    await flushPromises()

    const vm = mounted.vm as any
    vm.collapse()
    await flushPromises()

    const collapsedBar = mounted.find('[data-testid="schedule-config-collapsed-bar"]')
    expect(collapsedBar.exists()).toBe(true)

    // 按 Enter 键展开
    await collapsedBar.trigger('keydown.enter')
    await flushPromises()
    expect(mounted.find('[data-testid="schedule-config-expanded-wrap"]').exists()).toBe(true)

    // 再次收起并按 Space 键展开
    vm.collapse()
    await flushPromises()
    const collapsedBar2 = mounted.find('[data-testid="schedule-config-collapsed-bar"]')
    await collapsedBar2.trigger('keydown.space')
    await flushPromises()
    expect(mounted.find('[data-testid="schedule-config-expanded-wrap"]').exists()).toBe(true)
  })

  test('持久化记忆：存储为 1 时，挂载自动保持收起状态', async () => {
    localStorage.setItem('goose:scheduleConfigCollapsed', '1')

    mounted = mount(ScheduleConfigSection, {
      global: { plugins: [i18n] },
    })
    await flushPromises()

    expect(mounted.find('[data-testid="schedule-config-collapsed-bar"]').exists()).toBe(true)
    expect(mounted.find('[data-testid="schedule-config-expanded-wrap"]').exists()).toBe(false)
  })

  test('持久化记忆：存储为 1 时无论是否已选专业均保持收起记忆，切页面不自动展开', async () => {
    localStorage.setItem('goose:scheduleConfigCollapsed', '1')
    const store = useScheduleStore()
    store.state.majorSelected = {
      calendarId: 121,
      grade: 2024,
      major: undefined,
    }

    mounted = mount(ScheduleConfigSection, {
      global: { plugins: [i18n] },
    })
    await flushPromises()

    expect(mounted.find('[data-testid="schedule-config-collapsed-bar"]').exists()).toBe(true)
    expect(mounted.find('[data-testid="schedule-config-expanded-wrap"]').exists()).toBe(false)
  })

  test('切页面仿真：收起配置后切页面（组件卸载重挂载），依然保持收起状态', async () => {
    // 首次进入排课页
    mounted = mount(ScheduleConfigSection, {
      global: { plugins: [i18n] },
    })
    await flushPromises()

    // 用户主动点击收起
    const vm = mounted.vm as any
    vm.collapse()
    await flushPromises()
    expect(mounted.find('[data-testid="schedule-config-collapsed-bar"]').exists()).toBe(true)
    expect(localStorage.getItem('goose:scheduleConfigCollapsed')).toBe('1')

    // 用户切换到其他页面（模拟排课器组件卸载）
    mounted.unmount()
    mounted = null

    // 用户切回排课页面（重新挂载排课器组件）
    mounted = mount(ScheduleConfigSection, {
      global: { plugins: [i18n] },
    })
    await flushPromises()

    // 验证切页面回来后依然严格保持收起记忆，绝不自动展开
    expect(mounted.find('[data-testid="schedule-config-collapsed-bar"]').exists()).toBe(true)
    expect(mounted.find('[data-testid="schedule-config-expanded-wrap"]').exists()).toBe(false)
  })

  test('点击 SchedulePlanBar 中的收起按钮触发折叠', async () => {
    mounted = mount(ScheduleConfigSection, {
      global: { plugins: [i18n] },
    })
    await flushPromises()

    const collapseBtn = mounted.find('button[aria-label="' + i18n.global.t('schedule.collapseSettings') + '"]')
    expect(collapseBtn.exists()).toBe(true)
    await collapseBtn.trigger('click')
    await flushPromises()

    expect(mounted.find('[data-testid="schedule-config-collapsed-bar"]').exists()).toBe(true)
  })

  test('点击 ScheduleMajorSelector 中的快捷收起按钮触发折叠', async () => {
    mounted = mount(ScheduleConfigSection, {
      global: { plugins: [i18n] },
    })
    await flushPromises()

    const doneButtons = mounted.findAll('button')
    const doneBtn = doneButtons.find((b) => b.text().includes(i18n.global.t('schedule.collapseSettingsDone')))
    expect(doneBtn).toBeDefined()
    await doneBtn!.trigger('click')
    await flushPromises()

    expect(mounted.find('[data-testid="schedule-config-collapsed-bar"]').exists()).toBe(true)
  })
})


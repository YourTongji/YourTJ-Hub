// @vitest-environment happy-dom
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import { i18n } from '../src/runtime/i18n'
import SchedulePlanBar from '../src/site/components/schedule/SchedulePlanBar.vue'
import { MAX_PLANS, useScheduleStore } from '../src/site/composables/useScheduleStore'

function makeStorage() {
  const storage = new Map<string, string>()
  // 保留 happy-dom 的 DOM 构造器（test-utils trigger 依赖 window.Event/MouseEvent），仅替换 localStorage。
  const win = globalThis.window as unknown as Record<string, unknown>
  const dom = Object.fromEntries(
    Object.getOwnPropertyNames(win)
      .filter((key) => typeof win[key] === 'function')
      .map((key) => [key, win[key]]),
  )
  return {
    ...dom,
    localStorage: {
      getItem: vi.fn((k: string) => storage.get(k) ?? null),
      setItem: vi.fn((k: string, v: string) => storage.set(k, v)),
      removeItem: vi.fn((k: string) => storage.delete(k)),
    },
  }
}

beforeEach(() => {
  vi.stubGlobal('window', makeStorage())
  vi.stubGlobal('ResizeObserver', class { observe() {} unobserve() {} disconnect() {} })
  vi.stubGlobal('matchMedia', () => ({ matches: false, addEventListener: () => {}, removeEventListener: () => {} }))
  const store = useScheduleStore()
  store.clearStagedAndSelectedCourses()
  while (store.state.plans.length > 1) store.deletePlan(store.state.plans[0].id)
  store.deletePlan(store.state.plans[0].id) // resets to single fresh 方案 1
})

function mountBar() {
  return mount(SchedulePlanBar, { global: { plugins: [i18n] } })
}

async function openMenu(wrapper: ReturnType<typeof mountBar>) {
  await wrapper.find('button[aria-label="More actions"]').trigger('click')
  await nextTick()
}

describe('SchedulePlanBar 方案管理', () => {
  test('重命名方案：菜单 → 弹窗 → 保存生效', async () => {
    const wrapper = mountBar()
    const store = useScheduleStore()

    await openMenu(wrapper)
    const renameItem = wrapper.findAll('button.gf-menu-item').find((b) => b.text() === 'Rename plan')
    expect(renameItem).toBeDefined()
    await renameItem!.trigger('click')
    await flushPromises()

    // reka-ui Dialog 内容 teleport 到 document.body
    const input = document.body.querySelector<HTMLInputElement>('input[type="text"]')
    expect(input).not.toBeNull()
    input!.value = '新学期'
    input!.dispatchEvent(new Event('input'))

    const saveBtn = [...document.body.querySelectorAll('button')].find((b) => b.textContent?.trim() === 'Save')
    expect(saveBtn).toBeDefined()
    saveBtn!.click()
    await flushPromises()

    expect(store.state.plans[0].name).toBe('新学期')
  })

  test('创建副本：复制并激活新方案', async () => {
    const wrapper = mountBar()
    const store = useScheduleStore()

    await openMenu(wrapper)
    const dupItem = wrapper.findAll('button.gf-menu-item').find((b) => b.text() === 'Duplicate plan')
    expect(dupItem).toBeDefined()
    await dupItem!.trigger('click')
    await flushPromises()

    expect(store.state.plans.length).toBe(2)
    expect(store.state.activePlanId).toBe(store.state.plans[1].id)
  })

  test('达到方案上限时禁用创建副本', async () => {
    const store = useScheduleStore()
    while (store.state.plans.length < MAX_PLANS) store.createPlan()

    const wrapper = mountBar()
    await openMenu(wrapper)
    const dupItem = wrapper.findAll('button.gf-menu-item').find((b) => b.text() === 'Duplicate plan')
    expect(dupItem).toBeDefined()
    expect(dupItem!.attributes('disabled')).toBeDefined()
  })
})
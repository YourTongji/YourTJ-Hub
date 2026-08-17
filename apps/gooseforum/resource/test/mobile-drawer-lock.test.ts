// @vitest-environment happy-dom
import { afterEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import MobileDrawer from '../src/site/components/MobileDrawer.vue'

// reka-ui 的 body 模态锁（pointerEvents/overflow）是跨组件共享栈
// （useBodyScrollLock createSharedComposable），同文件多抽屉残留会污染断言，
// 故模态锁恢复测试独立成文件，保证锁栈干净。

function mountDrawer(overrides: Record<string, unknown> = {}) {
  return mount(MobileDrawer, {
    props: {
      open: true,
      primaryItems: [{ key: 'home', label: '首页', url: '/', active: false }],
      resourceItems: [],
      sidebarGroups: [],
      categoryItems: [],
      footer: { links: [], primary: [] },
      closeLabel: '关闭菜单',
      menuLabel: '菜单',
      resourcesLabel: '资源',
      categoriesLabel: '分类',
      sidebarIcon: () => null,
      ...overrides,
    },
    global: { plugins: [i18n] },
    attachTo: document.body,
  })
}

describe('MobileDrawer 跨断点模态锁恢复', () => {
  test('断点变化关闭抽屉后 body 模态锁解除（pointerEvents/overflow 恢复）', async () => {
    const listeners = new Set<(event: MediaQueryListEvent) => void>()
    const mql = {
      matches: false,
      media: '(min-width: 1024px)',
      addEventListener: (_type: string, fn: (event: MediaQueryListEvent) => void) => listeners.add(fn),
      removeEventListener: (_type: string, fn: (event: MediaQueryListEvent) => void) => listeners.delete(fn),
    }
    const matchMediaSpy = vi.spyOn(window, 'matchMedia').mockReturnValue(mql as unknown as MediaQueryList)

    const wrapper = mountDrawer()
    await flushPromises()
    await new Promise((r) => setTimeout(r, 50))
    // 抽屉打开期间 body 被 reka-ui 锁定
    expect(document.body.style.pointerEvents).toBe('none')
    expect(document.body.style.overflow).toBe('hidden')

    // 跨断点：触发 matchMedia change → drawer 发出 close；
    // MobileDrawer 是受控组件，父组件（AppShell）收到 close 后把 open 置 false
    // 并卸载抽屉（v-if），这里模拟父组件行为。
    mql.matches = true
    for (const fn of listeners) fn({ matches: true } as MediaQueryListEvent)
    await flushPromises()
    expect(wrapper.emitted('close')).toHaveLength(1)
    await wrapper.setProps({ open: false })
    await flushPromises()
    await new Promise((r) => setTimeout(r, 150))
    // 模态锁解除：body 恢复可交互
    expect(document.body.style.pointerEvents).toBe('')
    expect(document.body.style.overflow).toBe('')

    matchMediaSpy.mockRestore()
    wrapper.unmount()
  })

  afterEach(() => {
    document.body.style.pointerEvents = ''
    document.body.style.overflow = ''
  })
})

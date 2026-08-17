// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { LayoutPayload, WikiTreeNamespace } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import AppShell from '../src/site/components/AppShell.vue'
import MobileDrawer from '../src/site/components/MobileDrawer.vue'
import WikiSidebar from '../src/site/components/WikiSidebar.vue'

// reka-ui Dialog 在测试环境（happy-dom）可直接渲染，无需 stub 动画原语。

const wikiTree: WikiTreeNamespace[] = [
  {
    name: 'tongji-guide',
    label: '同济新手教程',
    nodes: [
      { kind: 'page', pageId: 1, path: 'tongji-guide/intro', title: '学校简介', active: true, children: [] },
      { kind: 'page', pageId: 2, path: 'tongji-guide/选课指南', title: '选课指南', active: false, children: [] },
    ],
  },
  {
    name: 'manual',
    label: '使用指南',
    nodes: [{ kind: 'page', pageId: 3, path: 'manual/start', title: '快速开始', active: false, children: [] }],
  },
]

function makeLayout(mode: 'forum' | 'wiki'): LayoutPayload {
  return {
    site: {
      name: 'Test Forum',
      description: '',
      logo: '',
      favicon: '',
      externalLinks: '',
      brandType: 'default',
      brandText: 'Test Forum',
      brandImage: '',
    },
    viewer: {
      id: 0,
      username: '',
      email: '',
      avatarUrl: '',
      isAuthenticated: false,
      canAccessAdmin: false,
      isModerator: false,
      requiresEmailVerification: false,
      adminPermissions: [],
    },
    header: [],
    sidebar: {
      main: [],
      resources: [],
      groups: [],
      categories: [],
      activeKey: 'wiki',
      mode,
      wikiTree: mode === 'wiki' ? wikiTree : undefined,
    },
    footer: { links: [], primary: [] },
    unread: { notifications: false, messages: false, moderationReports: false, latestNotificationType: '' },
    theme: { enabled: false, current: 'gf-light', themeColor: '#ffffff' },
  }
}


describe('WikiSidebar 导航事件', () => {
  test('点击页面链接发出 navigate 事件，且中文路径按段编码', async () => {
    const wrapper = mount(WikiSidebar, {
      props: { tree: wikiTree },
      global: { plugins: [i18n] },
    })
    const link = wrapper.get('a[href="/wiki/tongji-guide/%E9%80%89%E8%AF%BE%E6%8C%87%E5%8D%97"]')
    expect(link.text()).toBe('选课指南')
    await link.trigger('click')
    expect(wrapper.emitted('navigate')).toHaveLength(1)
  })
})

describe('MobileDrawer wiki 模式', () => {
  function mountDrawer(overrides: Record<string, unknown> = {}) {
    return mount(MobileDrawer, {
      props: {
        open: true,
        primaryItems: [{ key: 'wiki', label: 'Wiki', url: '/wiki', active: false }],
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
    })
  }

  test('wiki 模式下渲染命名空间 + 页面树（侧栏顶部胶囊含回到首页/Wiki 入口）', () => {
    const wrapper = mountDrawer({ wikiMode: true, wikiTree })
    const drawer = wrapper.get('[role="dialog"]')
    expect(drawer.text()).toContain('同济新手教程')
    expect(drawer.text()).toContain('使用指南')
    expect(drawer.text()).toContain('学校简介')
    expect(drawer.text()).toContain('快速开始')
    // 需求：wiki 侧边栏顶部胶囊提供「回到首页」与「Wiki」两个入口（桌面侧栏与移动抽屉一致）。
    expect(drawer.find('a[href="/"]').exists()).toBe(true)
    expect(drawer.find('a[href="/wiki"]').exists()).toBe(true)
  })

  test('活动页面保持高亮（与桌面侧栏一致）', () => {
    const wrapper = mountDrawer({ wikiMode: true, wikiTree })
    const activeLink = wrapper.get('a[href="/wiki/tongji-guide/intro"]')
    expect(activeLink.classes()).toContain('bg-info/10')
    expect(activeLink.classes()).toContain('text-primary')
  })

  test('点击 wiki 页面链接后关闭抽屉', async () => {
    const wrapper = mountDrawer({ wikiMode: true, wikiTree })
    await wrapper.get('a[href="/wiki/manual/start"]').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  test('空树时渲染空态文案', () => {
    const wrapper = mountDrawer({ wikiMode: true, wikiTree: [] })
    expect(wrapper.get('[role="dialog"]').text()).toContain(i18n.global.t('wiki.sidebarEmpty'))
  })

  test('非 wiki 模式不渲染 wiki 树', () => {
    const wrapper = mountDrawer({ wikiMode: false, wikiTree })
    const drawer = wrapper.get('[role="dialog"]')
    expect(drawer.text()).toContain('Wiki')
    expect(drawer.text()).not.toContain('同济新手教程')
  })

  // P2-7（review #320 第二轮）：forum 模式导航需要 nav landmark（旧实现 Motion as="nav"）。
  test('forum 模式渲染 nav landmark（aria-label=菜单）', () => {
    const wrapper = mountDrawer({
      wikiMode: false,
      primaryItems: [
        { key: 'home', label: '首页', url: '/', active: false },
        { key: 'messages', label: '消息', url: '/messages', active: false },
      ],
    })
    const nav = wrapper.get('nav[aria-label="菜单"]')
    expect(nav.find('a[href="/"]').text()).toBe('首页')
    expect(nav.find('a[href="/messages"]').text()).toBe('消息')
  })

  // P2-4（review #320 第二轮）：resize 到 lg 后抽屉仍 mounted/open，
  // reka-ui 继续 trap focus + body 锁定，必须监听 matchMedia 断点变化自动关闭。
  test('断点变化到桌面宽度时自动关闭抽屉', async () => {
    const listeners = new Set<(event: MediaQueryListEvent) => void>()
    const mql = {
      matches: false,
      media: '(min-width: 1024px)',
      addEventListener: (_type: string, fn: (event: MediaQueryListEvent) => void) => listeners.add(fn),
      removeEventListener: (_type: string, fn: (event: MediaQueryListEvent) => void) => listeners.delete(fn),
    }
    const matchMediaSpy = vi.spyOn(window, 'matchMedia').mockReturnValue(mql as unknown as MediaQueryList)

    const wrapper = mountDrawer({ wikiMode: false })
    expect(wrapper.emitted('close')).toBeUndefined()

    // 模拟跨过 lg 断点（如旋转/窗口放大）：触发 matchMedia change 事件
    mql.matches = true
    for (const fn of listeners) fn({ matches: true } as MediaQueryListEvent)
    await flushPromises()
    expect(wrapper.emitted('close')).toHaveLength(1)

    matchMediaSpy.mockRestore()
    wrapper.unmount()
  })

  // 建议项（review #320 第三轮）模态锁解除测试见 test/mobile-drawer-lock.test.ts
  // （reka-ui body 锁是跨组件共享栈，同文件多抽屉残留会污染断言，独立文件保证干净锁栈）。
})

describe('AppShell 抽屉接线', () => {
  test('wiki 布局下抽屉收到 wiki 树（移动端可浏览命名空间与页面）', async () => {
    const wrapper = mount(AppShell, {
      props: { layout: makeLayout('wiki') },
      global: { plugins: [i18n] },
    })
    await wrapper.get('button[aria-label="打开菜单"]').trigger('click')
    await flushPromises()
    const drawer = wrapper.get('[role="dialog"]')
    expect(drawer.text()).toContain('同济新手教程')
    expect(drawer.text()).toContain('学校简介')
  })

  test('forum 布局下抽屉不渲染 wiki 树', async () => {
    const wrapper = mount(AppShell, {
      props: { layout: makeLayout('forum') },
      global: { plugins: [i18n] },
    })
    await wrapper.get('button[aria-label="打开菜单"]').trigger('click')
    await flushPromises()
    const drawer = wrapper.get('[role="dialog"]')
    expect(drawer.text()).not.toContain('同济新手教程')
  })

  test('桌面侧栏在 wiki 模式下渲染 wiki 树（桌面行为不变）', () => {
    const wrapper = mount(AppShell, {
      props: { layout: makeLayout('wiki') },
      global: { plugins: [i18n] },
    })
    const sidebar = wrapper.get('aside[aria-label="Sidebar"]')
    expect(sidebar.text()).toContain('同济新手教程')
    expect(sidebar.text()).toContain('学校简介')
  })

  test('桌面侧栏在 forum 模式下不渲染 wiki 树', () => {
    const wrapper = mount(AppShell, {
      props: { layout: makeLayout('forum') },
      global: { plugins: [i18n] },
    })
    const sidebar = wrapper.get('aside[aria-label="Sidebar"]')
    expect(sidebar.text()).not.toContain('同济新手教程')
  })
})

beforeEach(() => {
  i18n.global.locale.value = 'zh'
  // happy-dom 共享 document：前序测试的抽屉可能残留 reka-ui body 模态锁，
  // 复位避免污染后续断言（与 i18n locale 复位同理）。
  document.body.style.pointerEvents = ''
  document.body.style.overflow = ''
  // happy-dom 默认窗口宽度 1024px，matchMedia('(min-width: 1024px)') 返回
  // matches=true；而 MobileDrawer 现在挂载时立即检查初始 matches（P2 第四轮修复），
  // 会把所有抽屉测试误判为桌面端并立即 close。这里 mock 移动端视口。
  const mobileMql = {
    matches: false,
    media: '(min-width: 1024px)',
    addEventListener: () => {},
    removeEventListener: () => {},
  }
  vi.spyOn(window, 'matchMedia').mockReturnValue(mobileMql as unknown as MediaQueryList)
})

afterEach(() => {
  i18n.global.locale.value = 'zh'
  vi.restoreAllMocks()
})

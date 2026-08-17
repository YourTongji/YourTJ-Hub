// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { LayoutPayload, WikiTreeNamespace } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import AppShell from '../src/site/components/AppShell.vue'
import MobileDrawer from '../src/site/components/MobileDrawer.vue'
import WikiSidebar from '../src/site/components/WikiSidebar.vue'

// motion-v 的 Motion/AnimatePresence 依赖浏览器动画 API，测试环境用纯渲染 stub 替代。
const MotionStub = { template: '<div><slot /></div>' }
const AnimatePresenceStub = { template: '<div><slot /></div>' }

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

const stubs = { Motion: MotionStub, AnimatePresence: AnimatePresenceStub }

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
      global: { plugins: [i18n], stubs },
    })
  }

  test('wiki 模式下渲染命名空间 + 页面树（侧边栏已无 wiki 首页块）', () => {
    const wrapper = mountDrawer({ wikiMode: true, wikiTree })
    const drawer = wrapper.get('[role="dialog"]')
    expect(drawer.text()).toContain('同济新手教程')
    expect(drawer.text()).toContain('使用指南')
    expect(drawer.text()).toContain('学校简介')
    expect(drawer.text()).toContain('快速开始')
    // 需求：wiki 侧边栏移除顶部 pb-2 首页块 → 抽屉内不再有 /wiki 首页链接。
    expect(drawer.find('a[href="/wiki"]').exists()).toBe(false)
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
})

describe('AppShell 抽屉接线', () => {
  test('wiki 布局下抽屉收到 wiki 树（移动端可浏览命名空间与页面）', async () => {
    const wrapper = mount(AppShell, {
      props: { layout: makeLayout('wiki') },
      global: { plugins: [i18n], stubs },
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
      global: { plugins: [i18n], stubs },
    })
    await wrapper.get('button[aria-label="打开菜单"]').trigger('click')
    await flushPromises()
    const drawer = wrapper.get('[role="dialog"]')
    expect(drawer.text()).not.toContain('同济新手教程')
  })

  test('桌面侧栏在 wiki 模式下渲染 wiki 树（桌面行为不变）', () => {
    const wrapper = mount(AppShell, {
      props: { layout: makeLayout('wiki') },
      global: { plugins: [i18n], stubs },
    })
    const sidebar = wrapper.get('aside[aria-label="Sidebar"]')
    expect(sidebar.text()).toContain('同济新手教程')
    expect(sidebar.text()).toContain('学校简介')
  })

  test('桌面侧栏在 forum 模式下不渲染 wiki 树', () => {
    const wrapper = mount(AppShell, {
      props: { layout: makeLayout('forum') },
      global: { plugins: [i18n], stubs },
    })
    const sidebar = wrapper.get('aside[aria-label="Sidebar"]')
    expect(sidebar.text()).not.toContain('同济新手教程')
  })
})

beforeEach(() => {
  i18n.global.locale.value = 'zh'
})

afterEach(() => {
  i18n.global.locale.value = 'zh'
})

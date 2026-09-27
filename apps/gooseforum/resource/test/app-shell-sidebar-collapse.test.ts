// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import { i18n } from '../src/runtime/i18n'
import AppShell from '../src/site/components/AppShell.vue'
import { resetShellSidebar, useShellSidebar } from '../src/runtime/shell-sidebar'
import type { LayoutPayload } from '@gooseforum/client'

const sidebarCollapsedStorageKey = 'goose:shell-sidebar-collapsed'

function minimalLayout(): LayoutPayload {
  return {
    site: {
      name: 'yourtj',
      description: '',
      logo: '',
      favicon: '',
      brandType: 'text',
      brandText: 'yourtj',
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
    sidebar: {
      main: [],
      resources: [],
      groups: [],
      categories: [{ id: 1, label: '论坛运营', url: '/c/1', color: '#10b981' }],
      activeKey: 'topics',
    },
    footer: { links: [], primary: [] },
    unread: { notifications: false, messages: false, moderationReports: false },
    theme: { enabled: false, current: 'gf-light', themeColor: '#fbfdff' },
    umamiEnabled: false,
  }
}

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }],
  })
}

function mountShell(router: Router): VueWrapper {
  return mount(AppShell, {
    props: { layout: minimalLayout() },
    global: { plugins: [i18n, router] },
    attachTo: document.body,
  })
}

function sidebarToggle(wrapper: VueWrapper) {
  // 折叠开关按 aria-controls 关联侧栏，避免与 navbar 其他图标按钮混淆
  return wrapper.get('button[aria-controls="goose-shell-sidebar"]')
}

function sidebar(wrapper: VueWrapper) {
  return wrapper.get('#goose-shell-sidebar')
}

// 前台外壳侧栏折叠（issue #798）：桌面宽度下侧栏可收起，让主内容列占满，
// 收起状态按 goose:* 偏好键记忆；收起时侧栏必须对键盘与辅助技术同样不可达
// （只做视觉隐藏会让 Tab 键仍能进入不可见链接）。
describe('AppShell 桌面侧栏折叠', () => {
  beforeEach(() => {
    window.localStorage.clear()
    resetShellSidebar()
  })

  afterEach(() => {
    document.body.innerHTML = ''
    vi.unstubAllGlobals()
  })

  test('默认展开：侧栏可访问，开关 aria-expanded=true', () => {
    const wrapper = mountShell(makeRouter())
    const toggle = sidebarToggle(wrapper)

    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(sidebar(wrapper).element.hasAttribute('inert')).toBe(false)
    expect(sidebar(wrapper).attributes('aria-hidden')).toBeUndefined()
    expect(wrapper.get('.gf-shell-main').classes()).not.toContain('gf-shell-main--collapsed')
    expect(window.localStorage.getItem(sidebarCollapsedStorageKey)).toBeNull()

    wrapper.unmount()
  })

  test('点击收起：轨道归零标记、侧栏 inert 且对辅助技术隐藏、写入偏好', async () => {
    const wrapper = mountShell(makeRouter())
    await sidebarToggle(wrapper).trigger('click')

    expect(sidebarToggle(wrapper).attributes('aria-expanded')).toBe('false')
    expect(wrapper.get('.gf-shell-main').classes()).toContain('gf-shell-main--collapsed')
    expect(sidebar(wrapper).element.hasAttribute('inert')).toBe(true)
    expect(sidebar(wrapper).attributes('aria-hidden')).toBe('true')
    expect(window.localStorage.getItem(sidebarCollapsedStorageKey)).toBe('1')

    wrapper.unmount()
  })

  test('再次点击展开：恢复初始状态并记录展开偏好', async () => {
    const wrapper = mountShell(makeRouter())
    await sidebarToggle(wrapper).trigger('click')
    await sidebarToggle(wrapper).trigger('click')

    expect(sidebarToggle(wrapper).attributes('aria-expanded')).toBe('true')
    expect(wrapper.get('.gf-shell-main').classes()).not.toContain('gf-shell-main--collapsed')
    expect(sidebar(wrapper).element.hasAttribute('inert')).toBe(false)
    expect(window.localStorage.getItem(sidebarCollapsedStorageKey)).toBe('0')

    wrapper.unmount()
  })

  test('持久化记忆：偏好为 1 时挂载即处于收起态', () => {
    window.localStorage.setItem(sidebarCollapsedStorageKey, '1')
    resetShellSidebar()

    const wrapper = mountShell(makeRouter())

    expect(wrapper.get('.gf-shell-main').classes()).toContain('gf-shell-main--collapsed')
    expect(sidebar(wrapper).element.hasAttribute('inert')).toBe(true)
    expect(sidebarToggle(wrapper).attributes('aria-expanded')).toBe('false')

    wrapper.unmount()
  })

  test('折叠开关带可读文案（供辅助技术与悬停提示）', () => {
    const wrapper = mountShell(makeRouter())

    expect(sidebarToggle(wrapper).attributes('title')).toBeTruthy()
    expect(sidebarToggle(wrapper).attributes('aria-label')).toBeTruthy()

    wrapper.unmount()
  })

  // 设计约定：navbar 左侧这个槽位在窄屏是抽屉入口、在桌面是侧栏折叠，
  // 图标必须是同一颗（展开/收起由 title、aria-expanded 与旋转标记表达），
  // 否则窗口跨断点缩放时图标会换脸。这里比对图标几何而非整个 svg 标记：
  // 折叠开关额外挂动效类，类名差异不属于「换图标」。
  test('折叠开关复用窄屏抽屉入口的图标', () => {
    const wrapper = mountShell(makeRouter())
    const drawerTrigger = wrapper.get(`button[aria-label="${String(i18n.global.t('shell.openMenu'))}"]`)
    const iconGeometry = (element: Element) => element.querySelector('svg')?.innerHTML

    expect(iconGeometry(sidebarToggle(wrapper).element)).toBeTruthy()
    expect(iconGeometry(sidebarToggle(wrapper).element)).toBe(iconGeometry(drawerTrigger.element))

    wrapper.unmount()
  })

  // 方向反馈靠图标旋转半圈（不换图标），旋转标记必须跟随状态，
  // 否则收起后图标会停在原位、动效只剩侧栏轨道在动。
  test('方向动效标记随状态切换', async () => {
    const wrapper = mountShell(makeRouter())
    const icon = () => sidebarToggle(wrapper).get('.gf-shell-sidebar-toggle-icon')
    const flipped = 'gf-shell-sidebar-toggle-icon--flipped'

    expect(icon().classes()).not.toContain(flipped)
    await sidebarToggle(wrapper).trigger('click')
    expect(icon().classes()).toContain(flipped)
    await sidebarToggle(wrapper).trigger('click')
    expect(icon().classes()).not.toContain(flipped)

    wrapper.unmount()
  })

  // 隐私模式/存储被拒时 localStorage 访问会抛错：必须回退到「展开」并且不把
  // 异常抛回调用方，否则一次折叠点击会让整个外壳崩掉。
  test('本地存储不可用时回退展开且不抛错', () => {
    const denied = () => { throw new Error('storage denied') }
    vi.stubGlobal('window', { localStorage: { getItem: denied, setItem: denied } })

    resetShellSidebar()
    const { sidebarCollapsed, setSidebarCollapsed } = useShellSidebar()
    expect(sidebarCollapsed.value).toBe(false)

    expect(() => setSidebarCollapsed(true)).not.toThrow()
    expect(sidebarCollapsed.value).toBe(true)

    vi.unstubAllGlobals()
    resetShellSidebar()
  })
})

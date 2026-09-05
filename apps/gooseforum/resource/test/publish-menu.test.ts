// @vitest-environment happy-dom
vi.mock('@/site/composables/useQuickPublish', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../src/site/composables/useQuickPublish')>()
  return {
    ...actual,
    loadQuickPublishModal: vi.fn(actual.loadQuickPublishModal),
  }
})

import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import type { LayoutPayload } from '@gooseforum/client'
import AppShell from '../src/site/components/AppShell.vue'
import PublishMenu from '../src/site/components/PublishMenu.vue'
import { i18n } from '../src/runtime/i18n'
import { useQuickPublish } from '../src/site/composables/useQuickPublish'
import * as quickPublishComposable from '../src/site/composables/useQuickPublish'
import { useShellState, resetShellState } from '../src/runtime/shell-state'

function makeLayout(isAuthenticated = true): LayoutPayload {
  return {
    viewer: {
      id: 1,
      username: 'test',
      email: 'test@example.com',
      avatarUrl: '',
      isAuthenticated,
      canAccessAdmin: false,
      isModerator: false,
      requiresEmailVerification: false,
      adminPermissions: [],
    },
    site: {
      name: 'GooseForum',
      description: '',
      logo: '',
      favicon: '',
      externalLinks: '',
      brandType: 'default',
      brandText: 'GooseForum',
      brandImage: '',
    },
    header: [],
    sidebar: {
      main: [],
      resources: [],
      groups: [],
      categories: [],
      activeKey: 'home',
      mode: 'forum',
    },
    categories: [],
    navigation: [],
    footer: { links: [], primary: [] },
    unread: { notifications: false, messages: false, moderationReports: false, latestNotificationType: '' },
    theme: { enabled: false, current: 'gf-light', themeColor: '#ffffff' },
  } as any
}

describe('PublishMenu 组件', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) }))
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })
  test('渲染桌面端 (navbar) 模式的原版 gf-button 触发按钮', () => {
    const wrapper = mount(PublishMenu, {
      props: { variant: 'navbar' },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    const trigger = wrapper.find('button')
    expect(trigger.exists()).toBe(true)
    expect(trigger.classes()).toContain('gf-button')
    expect(trigger.classes()).toContain('gf-button-primary')
    expect(trigger.text()).toContain(i18n.global.t('shell.publish'))
    expect(trigger.attributes('aria-haspopup')).toBe('true')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    wrapper.unmount()
  })

  test('渲染移动端 (fab) 模式的悬浮按钮', () => {
    const wrapper = mount(PublishMenu, {
      props: { variant: 'fab' },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    const trigger = wrapper.find('button')
    expect(trigger.exists()).toBe(true)
    expect(trigger.classes()).toContain('fixed')
    expect(trigger.classes()).toContain('h-14')
    expect(trigger.classes()).toContain('w-14')
    wrapper.unmount()
  })

  test('点击触发按钮展开菜单，点击非文章类型触发弹层发布', async () => {
    const { quickPublishOpen, quickPublishType } = useQuickPublish()
    quickPublishOpen.value = false

    const wrapper = mount(PublishMenu, {
      props: { variant: 'navbar' },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    const trigger = wrapper.get('button')
    await trigger.trigger('click')
    await flushPromises()

    expect(trigger.attributes('aria-expanded')).toBe('true')

    // 检查浮层内 3 个发布类型链接（发瞬间、提问题、写文章）
    const links = document.body.querySelectorAll<HTMLAnchorElement>('a[role="menuitem"]')
    expect(links.length).toBe(3)

    // 点击「发瞬间」（type=2），应触发弹层发布
    const thoughtLink = Array.from(links).find((l) => l.getAttribute('href')?.includes('thought'))
    expect(thoughtLink).toBeDefined()
    thoughtLink?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    await flushPromises()

    expect(quickPublishOpen.value).toBe(true)
    expect(quickPublishType.value).toBe(2)

    // 点击「提问题」（type=1），应触发弹层发布
    const questionLink = Array.from(links).find((l) => l.getAttribute('href')?.includes('question'))
    expect(questionLink).toBeDefined()
    questionLink?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    await flushPromises()

    expect(quickPublishOpen.value).toBe(true)
    expect(quickPublishType.value).toBe(1)

    wrapper.unmount()
  })

  test('pointerenter 悬停菜单项触发 QuickPublishModal 预加载', async () => {
    const preloadSpy = vi.mocked(quickPublishComposable.loadQuickPublishModal)
    preloadSpy.mockClear()

    const wrapper = mount(PublishMenu, {
      props: { variant: 'navbar' },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    const trigger = wrapper.get('button')
    await trigger.trigger('click')
    await flushPromises()

    // 弹层打开会自动聚焦首个菜单项（onOpenChange → focus），本身就会触发
    // @focusin 预热；这里断言 pointerenter 追加一次调用。
    const callsBeforeHover = preloadSpy.mock.calls.length
    const menuItem = document.body.querySelector<HTMLAnchorElement>('a[role="menuitem"]')
    expect(menuItem).not.toBeNull()
    menuItem?.dispatchEvent(new PointerEvent('pointerenter', { bubbles: true }))
    expect(preloadSpy.mock.calls.length).toBe(callsBeforeHover + 1)

    wrapper.unmount()
  })

  test('通过 shellState.isTopicPage 标记帖子详情页状态以控制移动端 FAB 显示', () => {
    const shellState = useShellState()
    expect(shellState.isTopicPage).toBe(false)

    shellState.isTopicPage = true
    expect(shellState.isTopicPage).toBe(true)

    resetShellState()
    expect(shellState.isTopicPage).toBe(false)
  })

  test('在排课器 (/schedule) 与课程页面 (/courses) 下移动端 FAB 自动隐去', async () => {
    const router = createRouter({
      history: createWebHistory(),
      routes: [
        { path: '/', component: { template: '<div>Home</div>' } },
        { path: '/schedule', component: { template: '<div>Schedule</div>' } },
        { path: '/courses', component: { template: '<div>Courses</div>' } },
        { path: '/courses/:id', component: { template: '<div>CourseDetail</div>' } },
      ],
    })

    await router.push('/')
    await router.isReady()

    const wrapper = mount(AppShell, {
      props: { layout: makeLayout(true) },
      global: { plugins: [i18n, router] },
      attachTo: document.body,
    })
    await flushPromises()

    // 1. 在普通页面（/），移动端 FAB 显示
    const fabHome = wrapper.findAllComponents(PublishMenu).find((c) => c.props('variant') === 'fab')
    expect(fabHome).toBeDefined()

    // 2. 导航到 /schedule，FAB 自动隐藏
    await router.push('/schedule')
    await flushPromises()
    const fabSchedule = wrapper.findAllComponents(PublishMenu).find((c) => c.props('variant') === 'fab')
    expect(fabSchedule).toBeUndefined()

    // 3. 导航到 /courses，FAB 自动隐藏
    await router.push('/courses')
    await flushPromises()
    const fabCourses = wrapper.findAllComponents(PublishMenu).find((c) => c.props('variant') === 'fab')
    expect(fabCourses).toBeUndefined()

    // 4. 导航到 /courses/123，FAB 自动隐藏
    await router.push('/courses/123')
    await flushPromises()
    const fabCourseDetail = wrapper.findAllComponents(PublishMenu).find((c) => c.props('variant') === 'fab')
    expect(fabCourseDetail).toBeUndefined()

    // 5. 导航回首页，FAB 恢复
    await router.push('/')
    await flushPromises()
    const fabHomeAgain = wrapper.findAllComponents(PublishMenu).find((c) => c.props('variant') === 'fab')
    expect(fabHomeAgain).toBeDefined()

    wrapper.unmount()
  })
})

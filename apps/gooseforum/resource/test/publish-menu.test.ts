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
    umamiEnabled: false,
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

  test('仅在首页（含 sort 查询）显示移动端 FAB', async () => {
    const pageStub = { template: '<div>Page</div>' }
    const router = createRouter({
      history: createWebHistory(),
      routes: [
        { path: '/', component: pageStub },
        { path: '/:pathMatch(.*)*', component: pageStub },
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

    const findFab = () => wrapper.findAllComponents(PublishMenu).find((c) => c.props('variant') === 'fab')

    expect(findFab()).toBeDefined()

    await router.push('/?sort=hot')
    await flushPromises()
    expect(findFab()).toBeDefined()

    await router.push('/?sort=popular')
    await flushPromises()
    expect(findFab()).toBeDefined()

    for (const path of [
      '/publish',
      '/p/post/123',
      '/topics/123',
      '/c/general/1',
      '/search',
      '/notifications',
      '/u/123',
      '/settings',
      '/courses',
      '/courses/123',
      '/schedule',
      '/wiki',
      '/wiki/guide/intro',
      '/campus',
      '/map',
    ]) {
      await router.push(path)
      await flushPromises()
      expect(findFab()).toBeUndefined()
    }

    wrapper.unmount()
  })
})

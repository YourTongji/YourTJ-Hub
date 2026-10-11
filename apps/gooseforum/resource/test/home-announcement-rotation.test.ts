// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { i18n, setLocale } from '../src/runtime/i18n'
import HomePage from '../src/site/pages/HomePage.vue'
import type { HomeProps, LayoutPayload } from '@gooseforum/client'

function announcementFixture(overrides: Partial<HomeProps['announcement']> = {}): HomeProps['announcement'] {
  return {
    enabled: true,
    html: '',
    publishedAt: new Date(Date.now() - 60_000).toISOString(),
    items: [
      { id: 'ann-1', title: '短公告', html: '<p>一行短公告</p>' },
      { id: 'ann-2', title: '长公告', html: '<p>长公告第一行</p><p>长公告第二行</p><p>长公告第三行</p>' },
      { id: 'ann-3', title: '', html: '<p>中等公告第一行</p><p>中等公告第二行</p>' },
    ],
    ...overrides,
  }
}

function homeProps(announcement: HomeProps['announcement'] = announcementFixture()): HomeProps {
  return {
    sort: 'latest',
    tabs: [{ key: 'latest', url: '/', active: true }],
    topics: [],
    pagination: { page: 1, nextPage: 0, hasNext: false, nextUrl: '' },
    announcement,
  }
}

function layoutFixture(): LayoutPayload {
  return {
    viewer: {
      id: 1,
      username: 'tester',
      email: 'tester@example.com',
      avatarUrl: '',
      isAuthenticated: true,
      canAccessAdmin: false,
      isModerator: false,
      requiresEmailVerification: false,
      adminPermissions: [],
    },
  } as unknown as LayoutPayload
}

function installBrowserMocks() {
  const mql = {
    matches: false,
    media: '',
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  }
  vi.stubGlobal('matchMedia', vi.fn(() => mql))
  vi.stubGlobal('IntersectionObserver', class { observe() {} unobserve() {} disconnect() {} })
  vi.stubGlobal('ResizeObserver', class { observe() {} unobserve() {} disconnect() {} })
}

function mountHome(announcement?: HomeProps['announcement']): VueWrapper {
  return mount(HomePage, {
    props: {
      layout: layoutFixture(),
      props: homeProps(announcement),
      pageUrl: '/',
    },
    global: {
      plugins: [i18n],
      stubs: {
        TopicList: true,
        TopicListFooter: true,
        EmptyState: true,
      },
    },
    attachTo: document.body,
  })
}

function visibleLayers(wrapper: VueWrapper) {
  return wrapper.findAll('[data-testid="announcement-item"]').filter((layer) => !layer.classes().includes('invisible'))
}

describe('首页公告轮播高度预留（issue #1102）', () => {
  beforeEach(() => {
    void setLocale('zh')
    installBrowserMocks()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    document.body.innerHTML = ''
  })

  test('展开态同时渲染全部条目层，仅激活层可见，其余层移出无障碍树与焦点序', () => {
    const wrapper = mountHome()

    const layers = wrapper.findAll('[data-testid="announcement-item"]')
    expect(layers).toHaveLength(3)
    const visible = visibleLayers(wrapper)
    expect(visible).toHaveLength(1)
    expect(visible[0]!.text()).toContain('一行短公告')

    const hidden = layers.filter((layer) => layer.classes().includes('invisible'))
    expect(hidden).toHaveLength(2)
    for (const layer of hidden) {
      expect(layer.attributes('aria-hidden')).toBe('true')
      expect(layer.attributes('inert')).toBeDefined()
    }
    // 激活层不带 inert/aria-hidden（可被屏幕阅读器与 Tab 到达）
    expect(visible[0]!.attributes('inert')).toBeUndefined()
    expect(visible[0]!.attributes('aria-hidden')).toBeUndefined()
    wrapper.unmount()
  })

  test('圆点切换只改可见性，层的 DOM 节点与数量不变（轮播高度恒定）', async () => {
    const wrapper = mountHome()
    const before = wrapper.findAll('[data-testid="announcement-item"]').map((layer) => layer.element)

    const dots = wrapper.get('[role="tablist"]').findAll('button')
    expect(dots).toHaveLength(3)
    await dots[1]!.trigger('click')

    const after = wrapper.findAll('[data-testid="announcement-item"]').map((layer) => layer.element)
    expect(after).toHaveLength(3)
    expect(after).toEqual(before)

    const visible = visibleLayers(wrapper)
    expect(visible).toHaveLength(1)
    expect(visible[0]!.text()).toContain('长公告第一行')
    expect(dots[1]!.attributes('aria-selected')).toBe('true')
    wrapper.unmount()
  })

  test('条目为空时回退到合并 html（既有单条行为不变）', () => {
    const wrapper = mountHome(announcementFixture({ items: [], html: '<p>聚合公告正文</p>' }))

    expect(wrapper.findAll('[data-testid="announcement-item"]')).toHaveLength(0)
    expect(wrapper.text()).toContain('聚合公告正文')
    expect(wrapper.find('[role="tablist"]').exists()).toBe(false)
    wrapper.unmount()
  })
})

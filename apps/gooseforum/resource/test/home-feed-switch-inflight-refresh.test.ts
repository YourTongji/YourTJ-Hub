// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { i18n, setLocale } from '../src/runtime/i18n'
import { homeFeedNavigation } from '../src/runtime/home-feed-navigation'
import HomePage from '../src/site/pages/HomePage.vue'
import type { HomeProps, LayoutPayload, PagePayload } from '@gooseforum/client'

const mocks = vi.hoisted(() => ({ fetchPage: vi.fn() }))

vi.mock('../src/runtime/router', () => ({ fetchPage: mocks.fetchPage }))

function homeProps(): HomeProps {
  return {
    sort: 'latest',
    tabs: [{ key: 'latest', url: '/', active: true }],
    topics: [{ id: 1, title: '在途前的旧帖' } as HomeProps['topics'][number]],
    pagination: { page: 1, nextPage: 0, hasNext: false, nextUrl: '' },
    announcement: { enabled: false, html: '', publishedAt: undefined, items: [] },
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

function homePagePayload(): PagePayload<HomeProps> {
  return {
    component: 'home.index',
    url: '/',
    layout: layoutFixture(),
    props: {
      ...homeProps(),
      topics: [
        { id: 1, title: '在途前的旧帖' } as HomeProps['topics'][number],
        { id: 2, title: '在途刷新带回的新帖' } as HomeProps['topics'][number],
      ],
    },
  } as unknown as PagePayload<HomeProps>
}

function createMemoryStorage(): Storage {
  const store = new Map<string, string>()
  return {
    get length() {
      return store.size
    },
    clear: () => store.clear(),
    getItem: (key: string) => (store.has(key) ? store.get(key)! : null),
    key: (index: number) => Array.from(store.keys())[index] ?? null,
    removeItem: (key: string) => void store.delete(key),
    setItem: (key: string, value: string) => void store.set(key, String(value)),
  } as Storage
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

function mountHome(): VueWrapper {
  return mount(HomePage, {
    props: {
      layout: layoutFixture(),
      props: homeProps(),
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

beforeEach(() => {
  void setLocale('zh')
  Object.defineProperty(window, 'localStorage', {
    configurable: true,
    value: createMemoryStorage(),
  })
  installBrowserMocks()
})

afterEach(() => {
  homeFeedNavigation.cancel()
  mocks.fetchPage.mockReset()
  vi.unstubAllGlobals()
  document.body.innerHTML = ''
})

test('话题流切换挂起/失败期间，在途刷新的结果不合并进当前列表', async () => {
  let resolveRefresh!: (payload: PagePayload<HomeProps>) => void
  mocks.fetchPage.mockImplementation(
    () => new Promise<PagePayload<HomeProps>>((resolve) => {
      resolveRefresh = resolve
    }),
  )
  const wrapper = mountHome()

  await wrapper.get('.gf-home-refresh-button').trigger('click')
  expect(wrapper.text()).toContain('正在刷新')

  const request = homeFeedNavigation.begin('/?sort=hot')
  await flushPromises()
  // 切换挂起时 feed 区显示加载中，列表被遮罩替代
  expect(wrapper.text()).toContain('加载中')

  // 刷新请求在切换挂起期间返回：结果必须被丢弃
  resolveRefresh!(homePagePayload())
  await flushPromises()

  // 切换失败：进入失败重试态
  homeFeedNavigation.fail(request, '/?sort=hot')
  await flushPromises()
  expect(wrapper.text()).toContain('重试')

  // 修复点：失败的切换不递增 revision，但挂起/失败期间的结果不得应用——
  // 未修复时会写入「已刷新到最新内容」并把新帖合并进列表
  expect(wrapper.text()).not.toContain('已刷新到最新内容')
  wrapper.unmount()
})


test('daily ranking labels track server readiness and retain the legacy rollback label', async () => {
  const wrapper = mountHome()
  await wrapper.setProps({ layout: { ...layoutFixture(), dailyRanking: true }, props: { ...homeProps(), tabs: [{key: 'popular', label: '今日热榜', url: '/?sort=popular', active: true}] } })
  expect(wrapper.get('a[href="/?sort=popular"]').text()).toBe('今日热榜')
  await wrapper.setProps({ layout: { ...layoutFixture(), dailyRanking: false } })
  expect(wrapper.get('a[href="/?sort=popular"]').text()).toBe('流行')
  wrapper.unmount()
})

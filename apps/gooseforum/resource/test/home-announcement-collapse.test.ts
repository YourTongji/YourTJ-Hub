// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { i18n, setLocale } from '../src/runtime/i18n'
import HomePage from '../src/site/pages/HomePage.vue'
import type { HomeProps, LayoutPayload } from '@gooseforum/client'

const COLLAPSE_KEY = 'goose:announcement:collapsed'
const READ_KEY = 'goose:announcement:last-read-published-at'
const PANEL_ID = 'gf-announcement-panel'

function announcementFixture(overrides: Partial<HomeProps['announcement']> = {}): HomeProps['announcement'] {
  return {
    enabled: true,
    html: '',
    publishedAt: new Date(Date.now() - 60_000).toISOString(),
    items: [
      { id: 'ann-1', title: '维护通知', html: '<p>今晚 23:00 系统维护</p>' },
      { id: 'ann-2', title: '新功能上线', html: '<p>首页支持折叠公告</p>' },
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

let storage: Storage

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

describe('首页公告栏折叠/展开（issue #799，方案 A：localStorage 持久化）', () => {
  beforeEach(() => {
    // 固定 zh：断言文案不依赖运行环境的浏览器语言探测结果
    void setLocale('zh')
    storage = createMemoryStorage()
    Object.defineProperty(window, 'localStorage', { configurable: true, value: storage })
    installBrowserMocks()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    document.body.innerHTML = ''
  })

  test('默认展开：保留既有轮播、指示点与铃铛；收起按钮带 aria-expanded/aria-controls', () => {
    const wrapper = mountHome()

    expect(wrapper.find('.gf-announcement-panel').exists()).toBe(true)
    expect(wrapper.find('.gf-prose-announcement').exists()).toBe(true)
    expect(wrapper.text()).toContain('维护通知')
    expect(wrapper.text()).toContain('今晚 23:00 系统维护')
    // 轮播指示点（tablist）仍渲染
    expect(wrapper.find('[role="tablist"]').exists()).toBe(true)

    const collapseButton = wrapper.get('[data-testid="announcement-collapse-button"]')
    expect(collapseButton.attributes('aria-expanded')).toBe('true')
    expect(collapseButton.attributes('aria-controls')).toBe(PANEL_ID)
    wrapper.unmount()
  })

  test('收起后公告 HTML 不再渲染（v-if 移出可达性树），标题行 + 展开控件保留，并持久化', async () => {
    const wrapper = mountHome()
    await wrapper.get('[data-testid="announcement-collapse-button"]').trigger('click')

    // v-if：公告正文彻底移除，屏幕阅读器与可达性树不再保留
    expect(wrapper.find('.gf-prose-announcement').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('今晚 23:00 系统维护')
    expect(wrapper.text()).not.toContain('首页支持折叠公告')
    // 可识别的标题行仍保留
    expect(wrapper.text()).toContain('公告')
    expect(wrapper.text()).toContain('维护通知')

    const expandButton = wrapper.get('[data-testid="announcement-expand-button"]')
    expect(expandButton.attributes('aria-expanded')).toBe('false')
    expect(expandButton.attributes('aria-controls')).toBe(PANEL_ID)
    // 键盘可达的控制仍在面板内
    expect(wrapper.find('.gf-announcement-panel button').exists()).toBe(true)

    expect(storage.getItem(COLLAPSE_KEY)).toBe('1')
    // 已读状态不受折叠影响（未写入已读时间戳）
    expect(storage.getItem(READ_KEY)).toBeNull()
    wrapper.unmount()
  })

  test('收起状态刷新页面后仍保持（localStorage 持久化）', async () => {
    const first = mountHome()
    await first.get('[data-testid="announcement-collapse-button"]').trigger('click')
    first.unmount()

    // 模拟刷新：重新挂载
    const second = mountHome()
    expect(second.find('[data-testid="announcement-expand-button"]').exists()).toBe(true)
    expect(second.find('[data-testid="announcement-collapse-button"]').exists()).toBe(false)
    second.unmount()
  })

  test('新公告未读时收起栏显示提示点；标记已读后提示消失但保持收起（互不耦合）', async () => {
    // 无已读记录 → 新公告未读
    const wrapper = mountHome()
    await wrapper.get('[data-testid="announcement-collapse-button"]').trigger('click')
    expect(wrapper.find('[data-testid="announcement-unread-dot"]').exists()).toBe(true)
    // 铃铛仍处于「未读提醒」态
    expect(wrapper.get('.gf-announcement-panel button[aria-pressed="true"]').exists()).toBe(true)

    // 直接收起状态下点铃铛标记已读：不重新展开，只熄灭提示
    await wrapper.get('.gf-announcement-panel button[aria-pressed="true"]').trigger('click')
    expect(wrapper.find('[data-testid="announcement-unread-dot"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="announcement-expand-button"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="announcement-expand-button"]').attributes('aria-expanded')).toBe('false')
    expect(storage.getItem(READ_KEY)).not.toBeNull()
    wrapper.unmount()
  })

  test('已读记录等于当前 publishedAt 时不再提示（不误报）', () => {
    const announcement = announcementFixture()
    storage.setItem(READ_KEY, String(Date.parse(announcement.publishedAt!)))
    const wrapper = mountHome(announcement)
    expect(wrapper.find('[data-testid="announcement-unread-dot"]').exists()).toBe(false)
    wrapper.unmount()
  })
})
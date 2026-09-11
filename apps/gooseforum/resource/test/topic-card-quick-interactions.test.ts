// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import TopicCardActions from '../src/site/components/TopicCardActions.vue'
import TopicList from '../src/site/components/TopicList.vue'
import type { TopicPayload } from '@gooseforum/client'
import zh from '../src/locales/zh'

vi.mock('@/runtime/api', () => ({
  likeTopic: vi.fn(),
  bookmarkTopic: vi.fn(),
}))

import { bookmarkTopic, likeTopic } from '@/runtime/api'

const i18n = createI18n({
  legacy: false,
  locale: 'zh',
  messages: { zh },
})

function baseTopic(overrides: Partial<TopicPayload> = {}): TopicPayload {
  return {
    id: 42,
    title: '卡片互动',
    description: '预览摘要',
    url: '/p/42',
    author: { id: 1, username: 'alice', avatarUrl: '' },
    participants: [],
    categories: [],
    replyCount: 7,
    viewCount: 100,
    likeCount: 5,
    pinWeight: 0,
    processStatus: 0,
    activityText: '刚刚',
    lastUpdateTime: '2026-09-11T00:00:00Z',
    contentType: 3,
    liked: false,
    bookmarked: false,
    ...overrides,
  }
}

let wrapper: VueWrapper | null = null

// happy-dom 下替换 location，断言访客点击互动后的登录跳转。
const realLocation = window.location

function mountActions(topic: TopicPayload) {
  wrapper = mount(TopicCardActions, {
    props: { topic },
    global: { plugins: [i18n] },
  })
  return wrapper
}

describe('TopicCardActions 卡片快捷互动（issue #380）', () => {
  beforeEach(() => {
    vi.mocked(likeTopic).mockReset()
    vi.mocked(bookmarkTopic).mockReset()
    // @ts-expect-error 测试替换 window.location
    delete window.location
    window.location = {
      href: 'http://localhost/',
      pathname: '/',
      search: '',
      hash: '',
    } as unknown as Location
  })

  afterEach(() => {
    window.location = realLocation
    wrapper?.unmount()
    wrapper = null
    vi.resetAllMocks()
  })

  it('渲染点赞计数、回复计数与评论入口', () => {
    const view = mountActions(baseTopic())
    const likeButton = view.find('button[title="点赞"]')
    expect(likeButton.text()).toBe('5')
    const commentLink = view.find('a[href="/p/42?reply=1"]')
    expect(commentLink.exists()).toBe(true)
    expect(commentLink.text()).toContain('7')
  })

  it('就地刷新时跟随 props 更新状态与计数', async () => {
    const view = mountActions(baseTopic())
    await view.setProps({ topic: baseTopic({ liked: true, likeCount: 9 }) })
    const likeButton = view.find('button[title="点赞"]')
    expect(likeButton.classes()).toContain('text-error')
    expect(likeButton.text()).toBe('9')
  })

  it('评论链接容错处理带 query 的 url', async () => {
    const view = mountActions(baseTopic({ url: '/p/42?from=hot' }))
    expect(view.find('a[href="/p/42?from=hot&reply=1"]').exists()).toBe(true)
  })

  it('成功互动回写共享 topic，重挂载实例读到最新状态（card↔table 切换场景）', async () => {
    const topic = baseTopic()
    vi.mocked(likeTopic).mockResolvedValueOnce(true)
    const view = mount(TopicList, {
      props: { topics: [topic], feedMode: 'card' },
      global: { plugins: [i18n], stubs: { TopicFeedPreview: true } },
    })
    await view
      .findComponent(TopicCardActions)
      .find('button[title="点赞"]')
      .trigger('click')
    await flushPromises()
    // 父级数组中的共享对象已被回写：重建实例（如切换视图）初始化即读到新状态。
    expect(topic.liked).toBe(true)
    expect(topic.likeCount).toBe(6)
  })

  it('点赞乐观更新 +1，成功后保持并调用 action=1', async () => {
    vi.mocked(likeTopic).mockResolvedValueOnce(true)
    const view = mountActions(baseTopic())
    const likeButton = view.find('button[title="点赞"]')
    await likeButton.trigger('click')
    expect(likeButton.text()).toBe('6')
    await flushPromises()
    expect(likeButton.text()).toBe('6')
    expect(likeTopic).toHaveBeenCalledWith(42, 1)
  })

  it('取消点赞乐观 -1 并调用 action=2', async () => {
    vi.mocked(likeTopic).mockResolvedValueOnce(true)
    const view = mountActions(baseTopic({ liked: true }))
    const likeButton = view.find('button[title="点赞"]')
    await likeButton.trigger('click')
    await flushPromises()
    expect(likeButton.text()).toBe('4')
    expect(likeTopic).toHaveBeenCalledWith(42, 2)
  })

  it('点赞失败回滚计数', async () => {
    vi.mocked(likeTopic).mockRejectedValueOnce(new Error('boom'))
    const view = mountActions(baseTopic())
    const likeButton = view.find('button[title="点赞"]')
    await likeButton.trigger('click')
    await flushPromises()
    expect(likeTopic).toHaveBeenCalledWith(42, 1)
    expect(likeButton.text()).toBe('5')
    expect(likeButton.classes()).not.toContain('text-error')
  })
  it('收藏乐观切换并在失败时回滚', async () => {
    vi.mocked(bookmarkTopic).mockRejectedValueOnce(new Error('boom'))
    const view = mountActions(baseTopic())
    const bookmarkButton = view.find('button[title="收藏"]')
    await bookmarkButton.trigger('click')
    await flushPromises()
    expect(bookmarkTopic).toHaveBeenCalledWith(42, 1)
    expect(bookmarkButton.attributes('aria-pressed')).toBe('false')
    expect(bookmarkButton.attributes('title')).toBe('收藏')
  })

  it('访客点击点赞引导登录且不调用接口', async () => {
    const view = mountActions(baseTopic({ liked: undefined, bookmarked: undefined }))
    await view.find('button[title="点赞"]').trigger('click')
    await flushPromises()
    expect(window.location.href).toBe('/login?redirect=%2F')
    expect(likeTopic).not.toHaveBeenCalled()
  })

  it('卡片视图渲染 stretched-link 与快捷互动条', () => {
    const view = mount(TopicList, {
      props: { topics: [baseTopic()], feedMode: 'card' },
      global: { plugins: [i18n], stubs: { TopicFeedPreview: true } },
    })
    const cardLink = view.find('a[aria-label="卡片互动"]')
    expect(cardLink.exists()).toBe(true)
    expect(cardLink.classes()).toContain('absolute')
    expect(cardLink.attributes('href')).toBe('/p/42')
    expect(view.findComponent(TopicCardActions).exists()).toBe(true)
  })

  it('表格视图不渲染快捷互动条', () => {
    const view = mount(TopicList, {
      props: { topics: [baseTopic()] },
      global: {
        plugins: [i18n],
        stubs: { TopicFeedPreview: true, TopicRow: true },
      },
    })
    expect(view.findComponent(TopicCardActions).exists()).toBe(false)
  })
})

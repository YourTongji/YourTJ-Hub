// @vitest-environment happy-dom
import { mount } from '@vue/test-utils'
import type { TopicPayload } from '@gooseforum/client'
import { createI18n } from 'vue-i18n'
import { afterEach, describe, expect, it, vi } from 'vitest'
import TopicFeedPreview from '../src/site/components/TopicFeedPreview.vue'
import TopicList from '../src/site/components/TopicList.vue'
import TopicRow from '../src/site/components/TopicRow.vue'

const topic = {
  id: 42,
  url: '/topic/42',
  title: '标题 😀',
  description: '摘要 🐦‍🔥',
  author: {
    id: 7,
    username: 'test-user',
    nickname: '昵称',
    avatarUrl: '',
    decoration: null,
    wornBadge: null,
    publicUid: null,
  },
  categories: [],
  participants: [],
  images: [],
  lastUpdateTime: new Date().toISOString(),
  replyCount: 1,
  viewCount: 10,
  pinWeight: 0,
  contentType: 0,
  processStatus: 0,
  unseen: false,
} as unknown as TopicPayload
let originalInnerWidth: number | undefined

function mountOptions() {
  return {
    global: {
      plugins: [createI18n({
        legacy: false,
        locale: 'en',
        messages: {
          en: {
            topicList: {
              columns: { replies: 'Replies', views: 'Views' },
              pinnedTopics: 'Pinned topics',
              pinned: 'Pinned',
              pinnedTopicsCount: '{count} pinned topics',
              expandPinnedTopics: 'Expand pinned topics',
              collapsePinnedTopics: 'Collapse pinned topics',
            },
          },
        },
      })],
      stubs: {
        AvatarStack: true,
        PersonaTag: true,
        UserAvatar: true,
        UserNickname: true,
      },
    },
  }
}

describe('home feed Twemoji coverage', () => {
  afterEach(() => {
    if (originalInnerWidth !== undefined) {
      Object.defineProperty(window, 'innerWidth', {
        configurable: true,
        value: originalInnerWidth,
      })
      originalInnerWidth = undefined
    }
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('renders emoji in the home list title and summary', () => {
    const wrapper = mount(TopicRow, {
      ...mountOptions(),
      attachTo: document.body,
      props: { topic, home: true },
    })

    expect(wrapper.findAll('img.twemoji').map((image) => image.attributes('alt'))).toEqual([
      '😀',
      '🐦‍🔥',
    ])
    wrapper.unmount()
  })

  it('renders emoji in the shared card and hover preview content', () => {
    const wrapper = mount(TopicFeedPreview, {
      ...mountOptions(),
      props: { topic, compact: true },
    })

    expect(wrapper.findAll('img.twemoji').map((image) => image.attributes('alt'))).toEqual([
      '😀',
      '🐦‍🔥',
    ])
    wrapper.unmount()
  })

  it('renders emoji in pinned topic titles and summary fallbacks', async () => {
    const pinnedTitle = { ...topic, id: 43, pinWeight: 1, title: 'Pinned 😀' }
    const pinnedSummary = {
      ...topic,
      id: 44,
      pinWeight: 2,
      title: '',
      description: 'Pinned summary 🐦‍🔥',
    }
    const wrapper = mount(TopicList, {
      ...mountOptions(),
      props: {
        topics: [pinnedTitle, pinnedSummary] as unknown as TopicPayload[],
        home: true,
        showPinned: true,
        feedMode: 'card',
      },
    })

    expect(wrapper.findAll('img.twemoji').map((image) => image.attributes('alt'))).toEqual([
      '😀',
    ])
    await wrapper.get('button').trigger('click')
    expect(wrapper.findAll('img.twemoji').map((image) => image.attributes('alt'))).toEqual([
      '😀',
      '🐦‍🔥',
    ])
    wrapper.unmount()
  })

  it('shows the same Twemoji in the delayed topic hover card', async () => {
    originalInnerWidth = window.innerWidth
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 1280 })
    vi.useFakeTimers()

    const wrapper = mount(TopicRow, {
      ...mountOptions(),
      attachTo: document.body,
      props: { topic, home: true },
    })
    vi.spyOn(wrapper.element, 'getBoundingClientRect').mockReturnValue({
      top: 24,
      right: 400,
      bottom: 72,
      left: 0,
      width: 400,
      height: 48,
      x: 0,
      y: 24,
      toJSON: () => ({}),
    } as DOMRect)

    await wrapper.trigger('mouseenter', { clientX: 180, clientY: 36 })
    await vi.advanceTimersByTimeAsync(800)

    expect(document.body.querySelectorAll('img.twemoji[alt="😀"]')).toHaveLength(2)
    expect(document.body.querySelectorAll('img.twemoji[alt="🐦‍🔥"]')).toHaveLength(2)

    wrapper.unmount()
  })
})

// @vitest-environment happy-dom
import { describe, expect, test } from 'vitest'
import { mount } from '@vue/test-utils'
import TopicFeedPreview from '@/site/components/TopicFeedPreview.vue'
import { i18n } from '../src/runtime/i18n'
import type { TopicPayload } from '@gooseforum/client'

function createTopic(overrides: Partial<TopicPayload> = {}): TopicPayload {
  return {
    id: 101,
    title: 'A titled moment',
    description: 'A short description',
    url: '/p/post/101',
    pinWeight: 0,
    processStatus: 0,
    author: { id: 1, username: 'tester', avatarUrl: '' },
    participants: [],
    categories: [{ id: 1, name: 'Moments', url: '/c/moments', color: '#3b82f6' }],
    replyCount: 1,
    viewCount: 10,
    activityText: 'now',
    lastUpdateTime: '2026-10-03T00:00:00Z',
    contentType: 2,
    images: [],
    ...overrides,
  }
}

function render(topic: TopicPayload, compact: boolean) {
  return mount(TopicFeedPreview, {
    props: { topic, compact, showStats: false },
    global: { plugins: [i18n] },
  })
}

function unreadDot(wrapper: ReturnType<typeof render>) {
  return wrapper.find('span[aria-hidden="true"].rounded-full')
}

describe('TopicFeedPreview unread indicator', () => {
  test('untitled unseen post places its indicator beside the excerpt without an empty heading row', () => {
    const wrapper = render(createTopic({ title: '', unseen: true }), true)

    expect(wrapper.find('h3').exists()).toBe(false)
    expect(wrapper.find('p').text()).toBe('A short description')
    expect(wrapper.find('p').element.contains(unreadDot(wrapper).element)).toBe(true)
  })

  test('untitled seen post has no heading or leftover indicator spacing', () => {
    const wrapper = render(createTopic({ title: '', unseen: false }), true)

    expect(wrapper.find('h3').exists()).toBe(false)
    expect(unreadDot(wrapper).exists()).toBe(false)
    expect(wrapper.find('p').text()).toBe('A short description')
  })

  test('titled unseen post keeps its heading and inline indicator', () => {
    const wrapper = render(createTopic({ unseen: true }), true)
    const heading = wrapper.get('h3')

    expect(heading.text()).toContain('A titled moment')
    expect(heading.element.contains(unreadDot(wrapper).element)).toBe(true)
    expect(wrapper.find('p').text()).toBe('A short description')
  })

  test('image-only untitled unseen post keeps its indicator in metadata on compact and hover surfaces', () => {
    const topic = createTopic({ title: '', description: '', unseen: true, images: ['/moment.png'] })

    for (const compact of [true, false]) {
      const wrapper = render(topic, compact)
      const metadata = wrapper.find('.unread-meta-dot').element.parentElement

      expect(wrapper.find('h3').exists()).toBe(false)
      expect(wrapper.find('p').exists()).toBe(false)
      expect(metadata?.textContent).toContain('tester')
      expect(wrapper.find('img[src="/moment.png"]').exists()).toBe(true)
    }
  })

  test('untitled unseen post without an image still shows its indicator beside the excerpt', () => {
    const wrapper = render(createTopic({ title: '', unseen: true, images: [] }), false)

    expect(wrapper.find('img[src="/moment.png"]').exists()).toBe(false)
    expect(wrapper.find('p').element.contains(unreadDot(wrapper).element)).toBe(true)
  })

  test('untitled post with no description or image keeps its indicator in metadata', () => {
    const wrapper = render(createTopic({ title: '', description: '', unseen: true, images: [] }), false)

    expect(wrapper.find('p').exists()).toBe(false)
    expect(wrapper.find('.unread-meta-dot').element.parentElement?.textContent).toContain('tester')
  })

  test('clearing unseen state removes the indicator from an untitled excerpt', async () => {
    const topic = createTopic({ title: '', unseen: true })
    const wrapper = render(topic, true)

    expect(unreadDot(wrapper).exists()).toBe(true)
    await wrapper.setProps({ topic: { ...topic, unseen: false } })
    expect(unreadDot(wrapper).exists()).toBe(false)
    expect(wrapper.find('h3').exists()).toBe(false)
    expect(wrapper.find('p').text()).toBe('A short description')
  })
})

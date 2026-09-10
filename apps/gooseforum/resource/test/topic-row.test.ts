// @vitest-environment happy-dom
import { describe, expect, test, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import TopicRow from '@/site/components/TopicRow.vue'
import { i18n } from '../src/runtime/i18n'
import type { TopicPayload } from '@gooseforum/client'

function createMockTopic(contentType: 0 | 1 | 2 | 3, overrides: Partial<TopicPayload> = {}): TopicPayload {
  return {
    id: 100,
    title: '测试瞬间标题',
    description: '测试内容描述',
    url: '/p/post/100',
    pinWeight: 0,
    processStatus: 0,
    author: {
      id: 1,
      username: 'tester',
      nickname: '测试者',
      avatarUrl: '/uploads/avatar.png',
    },
    participants: [
      {
        id: 1,
        username: 'tester',
        nickname: '测试者',
        avatarUrl: '/uploads/avatar.png',
      },
    ],
    categories: [
      {
        id: 1,
        name: '技术讨论',
        url: '/c/tech',
        color: '#3b82f6',
      },
    ],
    replyCount: 5,
    viewCount: 120,
    activityText: '10分钟前',
    lastUpdateTime: '2026-09-04T02:00:00Z',
    contentType,
    ...overrides,
  }
}

function previewCards() {
  return document.body.querySelectorAll('div.fixed.z-50')
}

describe('TopicRow.vue 列表行一致性验证', () => {
  test('瞬间（contentType: 2）在桌面端与移动端均呈现参与者头像和回复数，列对齐完整', () => {
    const topic = createMockTopic(2)
    const wrapper = mount(TopicRow, {
      props: {
        topic,
        home: true,
      },
      global: {
        plugins: [i18n],
      },
    })

    // 1. 验证瞬间 Badge 存在
    expect(wrapper.text()).toContain(i18n.global.t('publish.contentTypes.thought'))

    // 2. 验证 AvatarStack 参与者头像组件存在
    const avatarStacks = wrapper.findAllComponents({ name: 'AvatarStack' })
    expect(avatarStacks.length).toBe(2) // 移动端 + 桌面端各 1 个

    // 3. 验证回复数与浏览数均被渲染
    expect(wrapper.text()).toContain('5')
    expect(wrapper.text()).toContain('120')

    // 4. 验证桌面端各列结构完整（包含内容列、头像列、回复数列、浏览数列、活动时间列）
    const rootEl = wrapper.find('article')
    // 直接子 div 数量应为 5 列
    const cols = rootEl.findAll(':scope > div')
    expect(cols.length).toBe(5)
  })

  test('提问（contentType: 1）与文章（contentType: 3）均保持一致的 5 列布局', () => {
    for (const type of [1, 3] as const) {
      const topic = createMockTopic(type)
      const wrapper = mount(TopicRow, {
        props: {
          topic,
          home: true,
        },
        global: {
          plugins: [i18n],
        },
      })

      const cols = wrapper.find('article').findAll(':scope > div')
      expect(cols.length).toBe(5)

      const avatarStacks = wrapper.findAllComponents({ name: 'AvatarStack' })
      expect(avatarStacks.length).toBe(2)
    }
  })

  test('打开另一行预览时只保留最新的 Hover Card', async () => {
    vi.useFakeTimers()
    const first = mount(TopicRow, {
      props: {
        topic: createMockTopic(3, { id: 101, title: '第一篇文章', url: '/p/post/101' }),
        home: true,
      },
      global: { plugins: [i18n] },
    })
    const second = mount(TopicRow, {
      props: {
        topic: createMockTopic(3, { id: 102, title: '第二篇文章', url: '/p/post/102' }),
        home: true,
      },
      global: { plugins: [i18n] },
    })

    try {
      await first.get('article').trigger('mouseenter', { clientX: 360 })
      await vi.advanceTimersByTimeAsync(800)
      expect(previewCards()).toHaveLength(1)

      await second.get('article').trigger('mouseenter', { clientX: 420 })
      await vi.advanceTimersByTimeAsync(800)

      expect(previewCards()).toHaveLength(1)
      expect(previewCards()[0]?.textContent).toContain('第二篇文章')
    } finally {
      first.unmount()
      second.unmount()
      vi.clearAllTimers()
      vi.useRealTimers()
    }
  })

  test('窗口失焦时关闭 Hover Card', async () => {
    vi.useFakeTimers()
    const wrapper = mount(TopicRow, {
      props: {
        topic: createMockTopic(3),
        home: true,
      },
      global: { plugins: [i18n] },
    })

    try {
      await wrapper.get('article').trigger('mouseenter', { clientX: 360 })
      await vi.advanceTimersByTimeAsync(800)
      expect(previewCards()).toHaveLength(1)

      window.dispatchEvent(new Event('blur'))
      await wrapper.vm.$nextTick()

      expect(previewCards()).toHaveLength(0)
    } finally {
      wrapper.unmount()
      vi.clearAllTimers()
      vi.useRealTimers()
    }
  })
})

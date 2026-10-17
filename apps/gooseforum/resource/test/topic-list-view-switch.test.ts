// issue #1164：列表/卡片切换曾整表卸载重建（INP 263ms）。
// 这里锁住两点：切换复用两棵视图树的 DOM；信息流传导/推荐流锚点只在可见树上命中行节点。
// @vitest-environment happy-dom
import { expect, test, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import type { TopicPayload } from '@gooseforum/client'
import TopicList from '../src/site/components/TopicList.vue'
import { i18n } from '../src/runtime/i18n'
import { closeAllTopicPreviews, registerTopicPreview, unregisterTopicPreview } from '../src/runtime/topic-hover-preview'

function topic(id: number, feedTrace = `trace-${id}`): TopicPayload {
  return {
    id, title: `话题 ${id}`, description: '描述', url: `/p/${id}`,
    author: { id: 1, username: 'alice', avatarUrl: '' }, participants: [],
    categories: [], replyCount: 1, viewCount: 2, pinWeight: 0, processStatus: 0,
    contentType: 0, activityText: '刚刚', lastUpdateTime: '2026-10-01T00:00:00Z', feedTrace,
  }
}

function mountList(feedMode: 'table' | 'card', switchable = true) {
  return mount(TopicList, {
    props: { topics: [topic(1), topic(2)], feedMode, switchable },
    global: { plugins: [i18n] },
    // isConnected / getComputedStyle 需要真实挂载
    attachTo: document.body,
  })
}

test('切换视图复用两棵树的 DOM 节点，不整表卸载重建', async () => {
  const wrapper = mountList('table')
  try {
    const rowsBefore = wrapper.findAll('.gf-topic-row').map((row) => row.element)
    expect(rowsBefore).toHaveLength(2)

    await wrapper.setProps({ feedMode: 'card' })
    await nextTick()

    const rowsAfter = wrapper.findAll('.gf-topic-row').map((row) => row.element)
    expect(rowsAfter).toEqual(rowsBefore)
    expect(rowsBefore.every((element) => element.isConnected)).toBe(true)
    expect((wrapper.get('[data-feed-view="table"]').element as HTMLElement).style.display).toBe('none')
    expect((wrapper.get('[data-feed-view="card"]').element as HTMLElement).style.display).not.toBe('none')

    const cards = wrapper.findAll('.gf-card').map((card) => card.element)
    expect(cards).toHaveLength(2)

    await wrapper.setProps({ feedMode: 'table' })
    await nextTick()

    expect(wrapper.findAll('.gf-card').map((card) => card.element)).toEqual(cards)
    expect(cards.every((element) => element.isConnected)).toBe(true)
    expect((wrapper.get('[data-feed-view="card"]').element as HTMLElement).style.display).toBe('none')
    expect((wrapper.get('[data-feed-view="table"]').element as HTMLElement).style.display).not.toBe('none')
  } finally {
    wrapper.unmount()
  }
})

test('隐藏树剥离 data-feed-id / data-feed-trace，传导与锚点只命中可见行', async () => {
  const wrapper = mountList('table')
  try {
    const row = wrapper.findAll('.gf-topic-row')[0]!
    expect(row.attributes('data-feed-id')).toBe('1')
    expect(row.attributes('data-feed-trace')).toBe('trace-1')

    await wrapper.setProps({ feedMode: 'card' })
    await nextTick()

    expect(row.attributes('data-feed-id')).toBeUndefined()
    expect(row.attributes('data-feed-trace')).toBeUndefined()

    // 传导与锚点按文档顺序查询行节点：命中的必须都是可见卡片树里的行
    const tagged = wrapper.findAll('[data-feed-id]')
    expect(tagged).toHaveLength(2)
    for (const element of tagged) expect(element.element.closest('[data-feed-view="card"]')).not.toBeNull()

    const card = wrapper.findAll('.gf-card')[0]!
    expect(card.attributes('data-feed-id')).toBe('1')
    expect(card.attributes('data-feed-trace')).toBe('trace-1')
  } finally {
    wrapper.unmount()
  }
})

test('切换视图关闭表格行的悬停预览弹层', async () => {
  const close = vi.fn()
  registerTopicPreview(close)
  const wrapper = mountList('table')
  try {
    await wrapper.setProps({ feedMode: 'card' })
    await nextTick()
    expect(close).toHaveBeenCalled()
  } finally {
    unregisterTopicPreview(close)
    wrapper.unmount()
  }
})

test('没有切换入口的列表页不渲染卡片树，也不预建', async () => {
  const wrapper = mountList('table', false)
  try {
    expect(wrapper.findAll('.gf-card')).toHaveLength(0)
    await new Promise((resolve) => setTimeout(resolve, 350))
    await nextTick()
    expect(wrapper.findAll('.gf-card')).toHaveLength(0)
  } finally {
    wrapper.unmount()
  }
})

test('预建完成后切换（idle 回调路径）同样复用既有 DOM', async () => {
  const idleCallbacks: Array<() => void> = []
  vi.stubGlobal('requestIdleCallback', (cb: () => void) => { idleCallbacks.push(cb); return idleCallbacks.length })
  vi.stubGlobal('cancelIdleCallback', () => {})
  const wrapper = mountList('table')
  try {
    expect(wrapper.findAll('.gf-card')).toHaveLength(0)
    for (const callback of idleCallbacks.splice(0)) callback()
    await nextTick()
    expect(wrapper.findAll('.gf-card')).toHaveLength(2)

    const rows = wrapper.findAll('.gf-topic-row').map((row) => row.element)
    await wrapper.setProps({ feedMode: 'card' })
    await nextTick()
    expect(wrapper.findAll('.gf-topic-row').map((row) => row.element)).toEqual(rows)
  } finally {
    wrapper.unmount()
    vi.unstubAllGlobals()
  }
})

test('closeAllTopicPreviews 关闭全部已注册行级预览', () => {
  const first = vi.fn(); const second = vi.fn()
  registerTopicPreview(first); registerTopicPreview(second)
  closeAllTopicPreviews()
  expect(first).toHaveBeenCalledTimes(1)
  expect(second).toHaveBeenCalledTimes(1)
  unregisterTopicPreview(first); unregisterTopicPreview(second)
})

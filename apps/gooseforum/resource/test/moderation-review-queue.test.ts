// @vitest-environment happy-dom
import { afterEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount, shallowMount, type VueWrapper } from '@vue/test-utils'
import * as api from '../src/runtime/api'
import { i18n } from '../src/runtime/i18n'
import ModerationReviewQueue from '../src/site/components/ModerationReviewQueue.vue'
import ModerationPage from '../src/site/pages/ModerationPage.vue'
import type { LayoutPayload } from '@gooseforum/client'
import type { ReviewQueueItem } from '../src/admin/types'

const item = (id: number): ReviewQueueItem => ({
  id, revisionId: 100 + id, title: `待审话题 ${id}`, excerpt: '正文摘要', userId: 7, username: 'author', processStatus: 2,
  createdAt: '2026-10-01T10:00:00Z', images: ['/file/img/a.png'],
  aiReview: {
    id: 1, subjectType: 'topic', subjectId: id, authorId: 7, mode: 'enforce', policyRevision: 'r1', visionModel: 'vl', jevModel: 'jev',
    images: [], signals: {}, triggeredPolicies: ['violence'], evidenceStatus: 'complete', finalAction: 'review', appliedAction: 'review',
    errorKind: '', humanAction: '', latencyMs: 1, cost: 0, createdAt: '2026-10-01T10:00:00Z',
    reasons: [{ code: 'between_thresholds', policy: 'violence', score: 0.6, threshold: 0.5, limit: 0.9 }],
  },
})

const previews: VueWrapper[] = []
afterEach(() => {
  previews.splice(0).forEach(wrapper => wrapper.unmount())
  vi.restoreAllMocks()
})

describe('moderation workbench review queue (issue #975)', () => {
  test('lists scoped items with AI reasons and approves in one click', async () => {
    i18n.global.locale.value = 'zh'
    vi.spyOn(api, 'fetchModerationReviewQueue').mockResolvedValue({ items: [item(1), item(2)], total: 2, page: 1, pageSize: 20 })
    const action = vi.spyOn(api, 'moderationReviewAction').mockResolvedValue('success')
    const wrapper = mount(ModerationReviewQueue, { global: { plugins: [i18n] } })
    await flushPromises()
    expect(wrapper.text()).toContain('待审话题 1')
    expect(wrapper.text()).toContain('暴力血腥得分 0.60，在人工审核线 0.50 与拦截线 0.90 之间，进入人工审核。')
    expect(wrapper.find('img').attributes('src')).toBe('/file/img/a.png')

    await wrapper.findAll('button').find(b => b.text() === '通过')!.trigger('click')
    await flushPromises()
    expect(action).toHaveBeenCalledWith('topic', 1, true, 101)
    expect(wrapper.text()).not.toContain('待审话题 1')
    expect(wrapper.emitted('changed')).toHaveLength(1)
  })

  test('rejection needs a second click to confirm', async () => {
    i18n.global.locale.value = 'zh'
    vi.spyOn(api, 'fetchModerationReviewQueue').mockResolvedValue({ items: [item(3)], total: 1, page: 1, pageSize: 20 })
    const action = vi.spyOn(api, 'moderationReviewAction').mockResolvedValue('success')
    const wrapper = mount(ModerationReviewQueue, { global: { plugins: [i18n] } })
    await flushPromises()
    await wrapper.findAll('button').find(b => b.text() === '拒绝')!.trigger('click')
    expect(action).not.toHaveBeenCalled()
    await wrapper.findAll('button').find(b => b.text() === '确认拒绝')!.trigger('click')
    await flushPromises()
    expect(action).toHaveBeenCalledWith('topic', 3, false, 103)
    expect(wrapper.text()).toContain('没有待审核的内容')
  })

  test('items still being checked show the checking badge instead of old AI reasons', async () => {
    i18n.global.locale.value = 'zh'
    vi.spyOn(api, 'fetchModerationReviewQueue').mockResolvedValue({ items: [{ ...item(4), aiReview: undefined, aiChecking: true }], total: 1, page: 1, pageSize: 20 })
    const wrapper = mount(ModerationReviewQueue, { global: { plugins: [i18n] } })
    await flushPromises()
    expect(wrapper.text()).toContain('AI 检查中')
    expect(wrapper.text()).toContain('通常几秒内会自动公开或拒绝，你也可以现在处理。')
    expect(wrapper.text()).not.toContain('AI 转人工审核')
  })
})


describe('moderation image previews (issue #1091)', () => {
  test('opens the clicked image inline and preserves rejection confirmation, pagination and pending approval', async () => {
    i18n.global.locale.value = 'zh'
    const entry = { ...item(5), images: ['/file/img/a.png', '/file/img/b.png'] }
    const fetch = vi.spyOn(api, 'fetchModerationReviewQueue').mockResolvedValue({ items: [entry, item(6)], total: 30, page: 2, pageSize: 20 })
    let resolveAction!: (result: string) => void
    const action = vi.spyOn(api, 'moderationReviewAction').mockImplementation(() => new Promise(resolve => { resolveAction = resolve }))
    const wrapper = mount(ModerationReviewQueue, { global: { plugins: [i18n] }, attachTo: document.body })
    previews.push(wrapper)
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text() === '拒绝')!.trigger('click')
    const preview = wrapper.findAll('button').filter(button => button.find('img').exists())[1]!
    expect(preview).toBeDefined()
    expect(preview.attributes('type')).toBe('button')
    expect(preview.attributes('aria-label')).toBeTruthy()
    expect(wrapper.find('a[href="/file/img/b.png"]').exists()).toBe(false)
    await preview.trigger('click')
    await flushPromises()
    let dialog = document.querySelector('[role="dialog"]')!
    expect(dialog.querySelector('img')?.getAttribute('src')).toBe('/file/img/b.png')
    expect(dialog.textContent).toContain('2 / 2')
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft' }))
    await flushPromises()
    expect(dialog.querySelector('img')?.getAttribute('src')).toBe('/file/img/a.png')
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()
    expect(document.querySelector('[role="dialog"]')).toBeNull()
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(document.activeElement).toBe(preview.element)
    expect(wrapper.text()).toContain('确认拒绝')
    expect(action).not.toHaveBeenCalled()
    await wrapper.findAll('button').find(button => button.text() === '通过')!.trigger('click')
    await preview.trigger('click')
    await flushPromises()
    dialog = document.querySelector('[role="dialog"]')!
    expect(dialog).toBeTruthy()
    expect(wrapper.findAll('button').find(button => button.text() === '通过')!.attributes('disabled')).toBeDefined()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()
    expect(fetch).toHaveBeenCalledTimes(1)
    await wrapper.findAll('button').find(button => button.text() === '加载更多')!.trigger('click')
    await flushPromises()
    expect(fetch).toHaveBeenLastCalledWith('topic', 3, 20)
    resolveAction('success')
    await flushPromises()
    expect(action).toHaveBeenCalledWith('topic', 5, true, 105)
    expect(wrapper.emitted('changed')).toHaveLength(1)
  })
})


test('leave workspace is a separate homepage link that stays available on every tab', async () => {
  i18n.global.locale.value = 'zh'
  vi.spyOn(api, 'fetchModerationReports').mockResolvedValue({ items: [], nextCursor: 0, hasNext: false })
  vi.spyOn(api, 'fetchModerationLogs').mockResolvedValue({ items: [], nextCursor: 0, hasNext: false })
  const wrapper = shallowMount(ModerationPage, {
    props: {
      layout: {} as LayoutPayload,
      props: { topics: [], categoryTabs: [], pagination: { page: 1, nextPage: 2, hasNext: false, nextUrl: '' } },
    },
    global: { plugins: [i18n], stubs: { PageHeader: false } },
  })
  previews.push(wrapper)
  for (const tab of wrapper.findAll('main > div > button')) {
    await tab.trigger('click')
    const link = wrapper.find('header a[href="/"]')
    expect(link.exists()).toBe(true)
    expect(link.text()).toBe('离开工作台')
    expect(link.attributes('target')).toBeUndefined()
  }
})

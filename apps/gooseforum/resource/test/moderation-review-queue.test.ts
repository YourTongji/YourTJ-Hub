// @vitest-environment happy-dom
import { afterEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import * as api from '../src/runtime/api'
import { i18n } from '../src/runtime/i18n'
import ModerationReviewQueue from '../src/site/components/ModerationReviewQueue.vue'
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

afterEach(() => vi.restoreAllMocks())

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

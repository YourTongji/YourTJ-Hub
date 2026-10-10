// @vitest-environment happy-dom
import { afterEach, describe, expect, test } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import ReviewQueueAiDetails from '../src/admin/components/ReviewQueueAiDetails.vue'
import { i18n } from '../src/runtime/i18n'
import type { ReviewQueueItem } from '../src/admin/types'

const wrappers: VueWrapper[] = []
afterEach(() => wrappers.splice(0).forEach(wrapper => wrapper.unmount()))

const item: ReviewQueueItem = {
  id: 1, revisionId: 101, title: '待审', excerpt: '正文', userId: 7, username: 'author', processStatus: 2,
  createdAt: '2026-10-01T10:00:00Z', images: ['/file/img/a.png', '/file/img/b.png'],
  aiReview: {
    id: 1, subjectType: 'topic', subjectId: 1, authorId: 7, mode: 'enforce', policyRevision: 'r1', visionModel: 'vl', jevModel: 'jev',
    images: [{ status: 'ok', evidence: '保留 AI 图像证据' }], signals: {}, triggeredPolicies: ['violence'], evidenceStatus: 'complete', finalAction: 'review', appliedAction: 'review',
    errorKind: '', humanAction: '', latencyMs: 1, cost: 0, createdAt: '2026-10-01T10:00:00Z',
    reasons: [{ code: 'between_thresholds', policy: 'violence', score: 0.6, threshold: 0.5, limit: 0.9 }],
  },
}

describe('admin review image previews (issue #1091)', () => {
  test('opens clicked index inline and preserves AI reason and evidence DOM through close', async () => {
    i18n.global.locale.value = 'zh'
    const wrapper = mount(ReviewQueueAiDetails, { props: { item }, global: { plugins: [i18n] }, attachTo: document.body })
    wrappers.push(wrapper)
    const reason = wrapper.find('li').element
    const evidence = wrapper.find('p').element
    const preview = wrapper.findAll('button').filter(button => button.find('img').exists())[1]!
    expect(preview).toBeDefined()
    expect(preview.attributes('aria-label')).toBeTruthy()
    expect(wrapper.find('a[href="/file/img/b.png"]').exists()).toBe(false)
    await preview.trigger('click')
    await flushPromises()
    const dialog = document.querySelector('[role="dialog"]')!
    expect(dialog.querySelector('img')?.getAttribute('src')).toBe('/file/img/b.png')
    expect(dialog.textContent).toContain('2 / 2')
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft' }))
    await flushPromises()
    expect(dialog.querySelector('img')?.getAttribute('src')).toBe('/file/img/a.png')
    ;(dialog.querySelector('button[aria-label="关闭"]') as HTMLButtonElement).click()
    await flushPromises()
    expect(document.querySelector('[role="dialog"]')).toBeNull()
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(document.activeElement).toBe(preview.element)
    expect(wrapper.find('li').element).toBe(reason)
    expect(wrapper.find('p').element).toBe(evidence)
    expect(wrapper.text()).toContain('保留 AI 图像证据')
  })

  test('single-image preview has no pagination and Escape returns focus', async () => {
    const wrapper = mount(ReviewQueueAiDetails, { props: { item: { ...item, images: ['/file/img/a.png'] } }, global: { plugins: [i18n] }, attachTo: document.body })
    wrappers.push(wrapper)
    const preview = wrapper.find('button')
    expect(preview.exists()).toBe(true)
    await preview.trigger('click')
    await flushPromises()
    const dialog = document.querySelector('[role="dialog"]')!
    expect(dialog.querySelector('img')?.getAttribute('src')).toBe('/file/img/a.png')
    expect(dialog.querySelector('.font-mono')).toBeNull()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()
    expect(document.querySelector('[role="dialog"]')).toBeNull()
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(document.activeElement).toBe(preview.element)
  })
})

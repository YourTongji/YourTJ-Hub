// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, shallowMount, type VueWrapper } from '@vue/test-utils'
import type { LayoutPayload, WikiDetailProps } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import { bookmarkTopic, likeTopic, watchTopic } from '../src/runtime/api'
import WikiPage from '../src/site/pages/WikiPage.vue'
import WikiPageActions from '../src/site/components/WikiPageActions.vue'

const pushFlash = vi.hoisted(() => vi.fn())
vi.mock('../src/runtime/api', () => ({ bookmarkTopic: vi.fn(), likeTopic: vi.fn(), watchTopic: vi.fn() }))
vi.mock('../src/runtime/flash-message', () => ({ useFlashMessages: () => ({ push: pushFlash }) }))
vi.mock('../src/site/components/PostStream.vue', () => ({ default: { name: 'PostStream', props: ['interactions'], template: '<div />' } }))

function detail(active = false, topicId = 42): WikiDetailProps {
  return {
    page: {
      id: topicId, topicId, namespace: 'guide', path: 'guide/start', title: 'Guide',
      content: '<p>Test</p>', toc: [], updatedAt: '', likeCount: active ? 5 : 4,
      viewCount: 0, postCount: 0, liked: active, bookmarked: active, watched: active,
      canEdit: true, publishedRevisionNo: 1,
      editUrl: 'https://github.com/YourTongji/YourTJ-Wiki/edit/main/guide/start.md',
      historyUrl: 'https://github.com/YourTongji/YourTJ-Wiki/commits/main/guide/start.md',
    },
    contributors: [], hotTopics: [],
  }
}

let wrapper: VueWrapper
beforeEach(() => {
  vi.resetAllMocks()
  for (const api of [likeTopic, bookmarkTopic, watchTopic]) vi.mocked(api).mockResolvedValue(true)
  i18n.global.locale.value = 'zh'
})
afterEach(() => wrapper?.unmount())

function mountPage(active = false, authenticated = true) {
  wrapper = shallowMount(WikiPage, {
    props: { layout: { viewer: { isAuthenticated: authenticated, id: authenticated ? 7 : 0 } } as LayoutPayload, props: detail(active) },
    global: {
      plugins: [i18n], stubs: { WikiPageActions: false },
      directives: { 'code-copy': {}, 'code-highlight': {}, 'math-render': {}, 'content-enhancements': {} },
    },
  })
  expect(wrapper.findAllComponents(WikiPageActions)).toHaveLength(2)
}
const button = (bar: number, index: number) => wrapper.findAllComponents(WikiPageActions)[bar]!.findAll('button')[index]!
const streamState = () => wrapper.findComponent({ name: 'PostStream' }).props('interactions')
const cases = [
  { name: 'like', index: 0, field: 'isLiked', api: likeTopic, activeClass: 'text-error' },
  { name: 'bookmark', index: 1, field: 'isBookmarked', api: bookmarkTopic, activeClass: 'text-primary' },
  { name: 'watch', index: 2, field: 'isWatched', api: watchTopic, activeClass: 'text-success' },
] as const
function expectState(action: typeof cases[number], active: boolean, disabled = false) {
  for (const bar of [0, 1]) {
    expect(button(bar, action.index).classes().includes(action.activeClass)).toBe(active)
    expect(button(bar, action.index).attributes('disabled') !== undefined).toBe(disabled)
  }
  expect(streamState()[action.field]).toBe(active)
}
function deferred() {
  let resolve!: (value: boolean) => void
  let reject!: (reason: Error) => void
  const promise = new Promise<boolean>((yes, no) => { resolve = yes; reject = no })
  return { promise, resolve, reject }
}

for (const action of cases) {
  describe(action.name, () => {
    test.each([0, 1])('keeps both bars in sync when toggling from bar %i and back', async (from) => {
      mountPage()
      await button(from, action.index).trigger('click')
      await flushPromises()
      expectState(action, true)
      if (action.name === 'like') expect(streamState().likeCount).toBe(5)
      await button(1 - from, action.index).trigger('click')
      await flushPromises()
      expectState(action, false)
      expect(vi.mocked(action.api).mock.calls).toEqual([[42, 1], [42, 2]])
      expect(streamState().likeCount).toBe(4)
    })

    test.each([0, 1])('locks both bars while the request from bar %i is pending', async (from) => {
      const request = deferred()
      vi.mocked(action.api).mockReturnValueOnce(request.promise)
      mountPage()
      await button(from, action.index).trigger('click')
      expectState(action, true, true)
      await button(1 - from, action.index).trigger('click')
      expect(action.api).toHaveBeenCalledExactlyOnceWith(42, 1)
      for (const other of cases.filter(other => other !== action)) {
        expect(button(1 - from, other.index).attributes('disabled')).toBeUndefined()
      }
      request.resolve(true)
      await flushPromises()
      expectState(action, true)
    })

    for (const from of [0, 1]) {
      test.each([false, true])(`rolls back initial active=%s in both bars and the stream from bar ${from}`, async (active) => {
        const request = deferred()
        vi.mocked(action.api).mockReturnValueOnce(request.promise)
        mountPage(active)
        await button(from, action.index).trigger('click')
        expectState(action, !active, true)
        request.reject(new Error('Request rejected'))
        await flushPromises()
        expectState(action, active)
        expect(streamState().likeCount).toBe(active ? 5 : 4)
        expect(pushFlash).toHaveBeenCalledExactlyOnceWith('Request rejected', 'error')
        await button(1 - from, action.index).trigger('click')
        await flushPromises()
        expectState(action, !active)
        expect(vi.mocked(action.api).mock.calls).toEqual([[42, active ? 2 : 1], [42, active ? 2 : 1]])
      })
    }
  })
}

test('a failed like does not roll back a completed bookmark or pending watch', async () => {
  const like = deferred()
  const watch = deferred()
  vi.mocked(likeTopic).mockReturnValueOnce(like.promise)
  vi.mocked(watchTopic).mockReturnValueOnce(watch.promise)
  mountPage()
  await button(0, 0).trigger('click')
  await button(1, 1).trigger('click')
  await flushPromises()
  await button(1, 2).trigger('click')
  like.reject(new Error('Like rejected'))
  await flushPromises()
  expectState(cases[0], false)
  expectState(cases[1], true)
  expectState(cases[2], true, true)
  expect(streamState().likeCount).toBe(4)
  watch.resolve(true)
  await flushPromises()
  expectState(cases[2], true)
})

test.each(['success', 'failure'] as const)('ignores an old page request %s after new payload and a new pending request', async (result) => {
  const old = deferred()
  const current = deferred()
  vi.mocked(bookmarkTopic).mockReturnValueOnce(old.promise).mockReturnValueOnce(current.promise)
  mountPage()
  await button(0, 1).trigger('click')
  await wrapper.setProps({ props: detail(true, 99) })
  expectState(cases[1], true)
  await button(1, 1).trigger('click')
  expectState(cases[1], false, true)
  if (result === 'failure') old.reject(new Error('Old page failed'))
  else old.resolve(true)
  await flushPromises()
  expectState(cases[1], false, true)
  expect(pushFlash).not.toHaveBeenCalled()
  current.resolve(true)
  await flushPromises()
  expectState(cases[1], false)
  expect(vi.mocked(bookmarkTopic).mock.calls).toEqual([[42, 1], [99, 2]])
})

test('ignores a late failure after leaving the page', async () => {
  const request = deferred()
  vi.mocked(likeTopic).mockReturnValueOnce(request.promise)
  mountPage()
  await button(0, 0).trigger('click')
  wrapper.unmount()
  request.reject(new Error('Old page failed'))
  await flushPromises()
  expect(pushFlash).not.toHaveBeenCalled()
})

test('never shows a negative like count and preserves GitHub editing/history links', async () => {
  mountPage()
  const next = detail(true)
  next.page.likeCount = 0
  await wrapper.setProps({ props: next })
  await button(0, 0).trigger('click')
  await flushPromises()
  expect(streamState().likeCount).toBe(0)
  for (const bar of wrapper.findAllComponents(WikiPageActions)) {
    expect(bar.findAll('a').map(link => link.attributes('href'))).toEqual([next.page.editUrl, next.page.historyUrl])
    for (const link of bar.findAll('a')) expect(link.attributes('rel')).toBe('noopener noreferrer')
  }
})


test.each(cases)('preserves guest $name requests and error rollback in both bars', async (action) => {
  vi.mocked(action.api).mockRejectedValueOnce(new Error('Please sign in'))
  mountPage(false, false)
  await button(0, action.index).trigger('click')
  await flushPromises()
  expect(action.api).toHaveBeenCalledExactlyOnceWith(42, 1)
  expectState(action, false)
  expect(streamState().likeCount).toBe(4)
  expect(pushFlash).toHaveBeenCalledExactlyOnceWith('Please sign in', 'error')
})

// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { flushPromises, shallowMount, type VueWrapper } from '@vue/test-utils'
import type { LayoutPayload, WikiDetailProps } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import { likeTopic } from '../src/runtime/api'
import WikiPage from '../src/site/pages/WikiPage.vue'
import WikiPageActions from '../src/site/components/WikiPageActions.vue'
import PostStream from '../src/site/components/PostStream.vue'

vi.mock('../src/runtime/api', async (importOriginal) => ({
  ...await importOriginal<typeof import('../src/runtime/api')>(),
  likeTopic: vi.fn(),
  getPostWindow: vi.fn(async () => ({ posts: [], replyTargets: [], hasBefore: false, hasAfter: false, total: 0, maxPostNo: 0 })),
}))
vi.mock('../src/runtime/flash-message', () => ({ useFlashMessages: () => ({ push: vi.fn() }) }))
let wrapper: VueWrapper
const detail = (topicId: number, liked = false): WikiDetailProps => ({
  page: { id: topicId, topicId, namespace: 'guide', path: 'guide/test', title: 'Guide', content: '', toc: [], updatedAt: '',
    likeCount: liked ? 5 : 4, viewCount: 0, postCount: 0, liked, bookmarked: false, watched: false, canEdit: false, publishedRevisionNo: 1 },
  contributors: [], hotTopics: [],
})
afterEach(() => { wrapper?.unmount(); vi.clearAllMocks() })

test('real PostStream reflects optimistic state and rollback without echoing stale state into either bar', async () => {
  let reject!: (error: Error) => void
  vi.mocked(likeTopic).mockReturnValueOnce(new Promise((_resolve, no) => { reject = no })).mockResolvedValue(true)
  wrapper = shallowMount(WikiPage, {
    props: { layout: { viewer: { isAuthenticated: true, id: 7 } } as LayoutPayload, props: detail(42) },
    global: { plugins: [i18n], stubs: { WikiPageActions: false, PostStream: false }, directives: { 'code-copy': {}, 'code-highlight': {}, 'math-render': {}, 'content-enhancements': {} } },
  })
  const bars = wrapper.findAllComponents(WikiPageActions)
  const stream = wrapper.findComponent(PostStream)
  await flushPromises()
  await bars[0]!.findAll('button')[0]!.trigger('click')
  await flushPromises()
  expect(stream.props('interactions')).toMatchObject({ isLiked: true, likeCount: 5 })
  expect(stream.emitted('interaction-state')?.at(-1)?.[0]).toMatchObject({ isLiked: true, likeCount: 5 })
  reject(new Error('Failed'))
  await flushPromises()
  expect(stream.props('interactions')).toMatchObject({ isLiked: false, likeCount: 4 })
  expect(stream.emitted('interaction-state')?.at(-1)?.[0]).toMatchObject({ isLiked: false, likeCount: 4 })
  for (const bar of bars) expect(bar.findAll('button')[0]!.classes()).not.toContain('text-error')
  await bars[1]!.findAll('button')[0]!.trigger('click')
  await flushPromises()
  expect(vi.mocked(likeTopic).mock.calls).toEqual([[42, 1], [42, 1]])
  await wrapper.setProps({ props: detail(99) })
  await flushPromises()
  expect(stream.props('interactions')).toMatchObject({ isLiked: false, likeCount: 4 })
  expect(stream.emitted('interaction-state')?.at(-1)?.[0]).toMatchObject({ isLiked: false, likeCount: 4 })
  for (const bar of bars) expect(bar.findAll('button')[0]!.classes()).not.toContain('text-error')
})

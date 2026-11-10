// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { VNode } from 'vue'
import type { PagePayload, UserBookmarkPayload, UserProfileProps } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import { fetchPage } from '../src/runtime/router'
import UserPage from '../src/site/pages/UserPage.vue'
import TopicListFooter from '../src/site/components/TopicListFooter.vue'
import { layout, memberProfile } from './fixtures/profile'

vi.mock('../src/runtime/router', () => ({ fetchPage: vi.fn() }))
vi.mock('../src/runtime/content-updates', () => ({ useContentUpdates: vi.fn() }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))

const topic: UserBookmarkPayload = {
  id: 7, type: 'topic', topicId: 100, title: 'Saved topic', url: '/p/100',
  bookmarkedAt: '2026-11-01T00:00:00Z',
}
const post: UserBookmarkPayload = {
  id: 7, type: 'post', topicId: 200, postId: 555, postNo: 3,
  title: 'Saved reply', excerpt: 'Reply preview', url: '/p/200?postNo=3',
  bookmarkedAt: '2026-10-31T00:00:00Z',
}
const nextUrl = '/u/1024/bookmarks?cursor=next'
let wrapper: ReturnType<typeof mount> | undefined

beforeEach(() => vi.mocked(fetchPage).mockReset())
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
})

function mountPage(section: 'bookmarks' | 'activity', bookmarks: UserBookmarkPayload[]) {
  wrapper = mount(UserPage, {
    props: { layout, props: { ...memberProfile, section, activityTab: 'bookmarks', bookmarks,
      pagination: { page: 1, nextPage: 2, hasNext: true, nextUrl } } },
    global: { plugins: [i18n], stubs: { PrivateNoteEditor: true } },
  })
  return wrapper
}

async function append(bookmarks: UserBookmarkPayload[], hasNext = false) {
  vi.mocked(fetchPage).mockResolvedValue({
    props: { ...memberProfile, bookmarks,
      pagination: { page: 2, nextPage: hasNext ? 3 : 0, hasNext, nextUrl: hasNext ? nextUrl : '' } },
  } as PagePayload<UserProfileProps>)
  await wrapper!.findComponent(TopicListFooter).get('button').trigger('click')
  await flushPromises()
}

function bookmarkNodes(node: VNode): VNode[] {
  if (!node || typeof node !== 'object') return []
  const own = node.type === 'a' && [topic.url, post.url].includes(node.props?.href) ? [node] : []
  return [...own, ...(Array.isArray(node.children) ? node.children.flatMap(child => bookmarkNodes(child as VNode)) : [])]
}

for (const section of ['bookmarks', 'activity'] as const) {
  describe(`${section} bookmark identity`, () => {
    for (const [first, next] of [[topic, post], [post, topic]] as const) {
      test(`preserves ${first.type} then ${next.type} with the same numeric ID`, async () => {
        const page = mountPage(section, [first])
        await append([next])
        expect(bookmarkNodes(page.vm.$.subTree).map(node => node.props?.href)).toEqual([first.url, next.url])
        expect(page.text()).toContain('Saved topic')
        expect(page.text()).toContain('Saved reply')
      })
    }

    test('deduplicates repeated same-type rows without merging the other type', async () => {
      const page = mountPage(section, [topic])
      await append([topic, post, post])
      expect(bookmarkNodes(page.vm.$.subTree).map(node => node.props?.href)).toEqual([topic.url, post.url])
    })

    test('renders distinct stable keys for same-page collisions and preserves them on reorder', async () => {
      const page = mountPage(section, [topic, post])
      expect(bookmarkNodes(page.vm.$.subTree).map(node => node.key)).toEqual(['topic:7', 'post:7'])
      await page.setProps({ props: { ...page.props('props'), bookmarks: [post, topic] } })
      expect(bookmarkNodes(page.vm.$.subTree).map(node => node.key)).toEqual(['post:7', 'topic:7'])
    })

    test('retains the original row after a failed page and accepts a retry', async () => {
      const page = mountPage(section, [topic])
      vi.mocked(fetchPage).mockRejectedValueOnce(new Error('Page unavailable'))
      await page.findComponent(TopicListFooter).get('button').trigger('click')
      await flushPromises()
      expect(bookmarkNodes(page.vm.$.subTree).map(node => node.props?.href)).toEqual([topic.url])
      expect(page.text()).toContain(i18n.global.t('topicList.autoLoadFailed'))
      await append([post])
      expect(bookmarkNodes(page.vm.$.subTree).map(node => node.props?.href)).toEqual([topic.url, post.url])
    })

    test('keeps existing rows through an empty page and accepts the next cursor page', async () => {
      const page = mountPage(section, [topic])
      await append([], true)
      expect(bookmarkNodes(page.vm.$.subTree)).toHaveLength(1)
      await append([post])
      expect(bookmarkNodes(page.vm.$.subTree).map(node => node.props?.href)).toEqual([topic.url, post.url])
      expect(fetchPage).toHaveBeenCalledTimes(2)
    })
  })
}

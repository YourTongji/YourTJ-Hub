// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import UserPage from '../src/site/pages/UserPage.vue'
import TopicList from '../src/site/components/TopicList.vue'
import TopicListFooter from '../src/site/components/TopicListFooter.vue'
import { fetchPage } from '../src/runtime/router'
import { i18n } from '../src/runtime/i18n'
vi.mock('../src/runtime/router', () => ({ fetchPage: vi.fn() }))
class Stream extends EventTarget {
 static current: Stream
 close = vi.fn()
 constructor() { super(); Stream.current = this }
}
let wrapper: VueWrapper
const pagination = (nextUrl = '') => ({ page: 1, hasNext: !!nextUrl, nextPage: nextUrl ? 2 : 0, nextUrl })
const topic = (id: number, status = 2) => ({ id, processStatus: status, title: `topic-${id}` })
function profile(topics = [topic(1)], nextUrl = '/u/1/activity/topics?page=2', userId = 1) {
 return { user: { userId, username: 'owner', externalInformation: {} }, section: 'activity', activityTab: 'topics', topics, activities: [], likes: [], bookmarks: [], following: [], followers: [], badges: [], tabs: [], activityTabs: [], pagination: pagination(nextUrl) }
}
function payload(props: ReturnType<typeof profile>) { return { component: 'user.index', props } as any }
function rows() { return wrapper.findComponent(TopicList).props('topics') }
function changed() { Stream.current.dispatchEvent(new Event('content.changed')) }
beforeEach(() => { vi.stubGlobal('EventSource', Stream); vi.spyOn(document, 'hidden', 'get').mockReturnValue(false) })
afterEach(() => { wrapper?.unmount(); vi.restoreAllMocks(); vi.unstubAllGlobals(); vi.mocked(fetchPage).mockReset() })
function start() {
 wrapper = mount(UserPage, { props: { layout: { viewer: { id: 1, isAuthenticated: true } } as any, props: profile() as any }, global: { plugins: [i18n], stubs: { TopicList: true, TopicListFooter: true, PrivateNoteEditor: true, UserAvatar: true } } })
}
test('owner feed reconciles approved and rejected rows across loaded pages', async () => {
 start()
 vi.mocked(fetchPage).mockResolvedValueOnce(payload(profile([topic(2)], '')))
 wrapper.findComponent(TopicListFooter).vm.$emit('loadMore')
 await flushPromises()
 expect(rows().map((t: any) => t.id)).toEqual([1, 2])
 vi.mocked(fetchPage).mockResolvedValueOnce(payload(profile([topic(1, 0)], '/u/1/activity/topics?page=2')))
 vi.mocked(fetchPage).mockResolvedValueOnce(payload(profile([topic(3, 0)], '')))
 changed(); await flushPromises()
 expect(rows()).toEqual([topic(1, 0), topic(3, 0)])
 expect(wrapper.findComponent(TopicListFooter).props('pagination').hasNext).toBe(false)
})
test('late profile refresh cannot overwrite a newly selected profile', async () => {
 start()
 let complete!: (value: any) => void
 vi.mocked(fetchPage).mockReturnValueOnce(new Promise(resolve => { complete = resolve }))
 changed(); await flushPromises()
 expect(fetchPage).toHaveBeenCalledTimes(1)
 await wrapper.setProps({ props: profile([topic(9, 0)], '', 9) as any })
 complete(payload(profile([topic(1, 0)], ''))); await flushPromises()
 expect(rows()).toEqual([topic(9, 0)])
})
test('reconnect refreshes and the stream closes on unmount', async () => {
 start()
 vi.mocked(fetchPage).mockResolvedValueOnce(payload(profile([topic(1, 0)], '')))
 Stream.current.dispatchEvent(new Event('hello')); await flushPromises()
 expect(rows()).toEqual([topic(1, 0)])
 wrapper.unmount(); expect(Stream.current.close).toHaveBeenCalled()
})

test('an older pagination response cannot restore a rejected row after reconciliation', async () => {
 start()
 let complete!: (value: any) => void
 vi.mocked(fetchPage).mockReturnValueOnce(new Promise(resolve => { complete = resolve }))
 wrapper.findComponent(TopicListFooter).vm.$emit('loadMore')
 await flushPromises()
 vi.mocked(fetchPage).mockResolvedValueOnce(payload(profile([topic(1, 0)], '')))
 changed(); await flushPromises()
 complete(payload(profile([topic(2)], ''))); await flushPromises()
 expect(rows()).toEqual([topic(1, 0)])
})

test('failed reconciliation retains the loaded window and the next event retries', async () => {
 start()
 vi.mocked(fetchPage).mockResolvedValueOnce(payload(profile([topic(2)], '')))
 wrapper.findComponent(TopicListFooter).vm.$emit('loadMore'); await flushPromises()
 vi.mocked(fetchPage).mockResolvedValueOnce(payload(profile([topic(1, 0)])))
 vi.mocked(fetchPage).mockRejectedValueOnce(new Error('offline'))
 changed(); await flushPromises()
 expect(rows()).toEqual([topic(1), topic(2)])
 vi.mocked(fetchPage).mockResolvedValueOnce(payload(profile([topic(1, 0)], '')))
 changed(); await flushPromises()
 expect(rows()).toEqual([topic(1, 0)])
 expect(wrapper.findComponent(TopicListFooter).props('loadError')).toBe('')
})

// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n, setLocale } from '../src/runtime/i18n'
import { notificationText } from '../src/runtime/notification-text'
import type { NotificationPayload } from '@gooseforum/client'
const { fetchNotifications, markNotificationRead, markAllNotificationsRead } = vi.hoisted(() => ({
  fetchNotifications: vi.fn(),
  markNotificationRead: vi.fn(),
  markAllNotificationsRead: vi.fn(),
}))
vi.mock('../src/runtime/api', () => ({ fetchNotifications, markAllNotificationsRead, markNotificationRead }))
const { unreadSetNotifications, unreadRefresh, unreadClearNotifications } = vi.hoisted(() => ({
  unreadSetNotifications: vi.fn(),
  unreadRefresh: vi.fn(),
  unreadClearNotifications: vi.fn(),
}))
vi.mock('../src/runtime/unread-status', () => ({
  useUnreadStatus: () => ({ clearNotifications: unreadClearNotifications, setNotifications: unreadSetNotifications, refresh: unreadRefresh }),
}))
import NotificationsPage from '../src/site/pages/NotificationsPage.vue'
import { markAllNotificationsRead, markNotificationRead } from '../src/runtime/api'
afterEach(()=>vi.clearAllMocks())
function item(eventType='post_reply'): NotificationPayload {
 return {id:1,eventType,isRead:false,createdAt:'',title:'',content:'',actor:{id:1,username:'Alice'},topic:{id:42,title:'Campus',url:'/p/post/42/2'},payload:{actorId:1,topicTitle:'Campus'}}
}
test.each(['zh','en','ja','de'] as const)('complete public copy and direct mention priority in %s',async locale=>{
 await setLocale(locale)
 const n=item(); n.payload.hasDirectMention=true
 const text=notificationText(n,i18n.global.t)
 expect(text).toContain('Alice');expect(text).toContain('Campus');expect(text).not.toContain('notifications.')
 expect(text).not.toBe(notificationText(item(),i18n.global.t))
 expect(notificationText({...item('old'),title:'notifications.templates.unknown'},i18n.global.t)).toBe(i18n.global.t('notifications.fallback'))
 expect(notificationText({...item('old'),title:'Public account notice'},i18n.global.t)).toBe('Public account notice')
})
test('category and unread filter combine in server queries, including empty categories',async()=>{
 await setLocale('en')
 fetchNotifications.mockResolvedValue({items:[],nextCursor:0,hasNext:false,unreadCount:1})
 const wrapper=mount(NotificationsPage,{props:{layout:{} as any,props:{notifications:[item()],total:1,unreadCount:1,pagination:{page:1,nextPage:0,hasNext:false,nextUrl:''}}},global:{plugins:[i18n]}})
 await wrapper.findAll('button').find(button=>button.text()==='Mentions')!.trigger('click')
 await flushPromises()
 expect(fetchNotifications).toHaveBeenLastCalledWith('all',0,20,'mentions')
 await wrapper.get('input[type="checkbox"]').setValue(true)
 await flushPromises()
 expect(fetchNotifications).toHaveBeenLastCalledWith('unread',0,20,'mentions')
 expect(wrapper.text()).toContain('No unread notifications')
 wrapper.unmount()
})

// 同一次 flush 内既发生服务端 props 刷新、又发生用户点击时，点击必须生效：
// 一次点击就要加载被点分类的数据，且选中项与列表一致（issue #1183 第 3 项）。
test('a props refresh in the same flush as a category click still loads the clicked category',async()=>{
 await setLocale('en')
 fetchNotifications.mockResolvedValue({items:[],nextCursor:0,hasNext:false,unreadCount:1})
 const wrapper=mount(NotificationsPage,{props:{layout:{} as any,props:{category:'all',filter:'all',notifications:[item()],total:1,unreadCount:1,pagination:{page:1,nextPage:0,hasNext:false,nextUrl:''}}},global:{plugins:[i18n]}})
 fetchNotifications.mockClear()
 // 不 await：props 更新与随后的点击落在同一 flush。
 void wrapper.setProps({props:{category:'all',filter:'all',notifications:[item()],total:1,unreadCount:1,pagination:{page:1,nextPage:0,hasNext:false,nextUrl:''}}})
 await wrapper.findAll('button').find(button=>button.text()==='Likes')!.trigger('click')
 await flushPromises()
 expect(fetchNotifications).toHaveBeenLastCalledWith('all',0,20,'likes')
 expect(wrapper.findAll('button').find(button=>button.text()==='Likes')!.attributes('aria-pressed')).toBe('true')
 wrapper.unmount()
})

test('server-rendered category refresh restores both filters without re-fetching supplied rows', async()=>{
 await setLocale('en')
 const wrapper=mount(NotificationsPage,{props:{layout:{} as any,props:{category:'mentions',filter:'unread',notifications:[item()],total:1,unreadCount:1,pagination:{page:1,nextPage:0,hasNext:false,nextUrl:''}}},global:{plugins:[i18n]}})
 expect(wrapper.get('input[type="checkbox"]').element).toHaveProperty('checked',true)
 await wrapper.setProps({props:{category:'system',filter:'all',notifications:[item('system')],total:1,unreadCount:1,pagination:{page:1,nextPage:0,hasNext:false,nextUrl:''}}})
 await flushPromises()
 expect(wrapper.findAll('button').find(button=>button.text()==='System')!.attributes('aria-pressed')).toBe('true')
 expect(wrapper.get('input[type="checkbox"]').element).toHaveProperty('checked',false)
 expect(fetchNotifications).not.toHaveBeenCalled()
 wrapper.unmount()
})

test.each(['zh','en','ja','de'] as const)('all event copy has resolved placeholders in %s', async locale => {
 await setLocale(locale)
 for (const type of ['mention','post_reply','comment','like','follow','topic_post','wiki_updated','badge','system','review_pending','review_approved','review_rejected']) {
  const n=item(type); n.payload.metadata={badgeName:'Welcome'}
  const text=notificationText(n,i18n.global.t)
  expect(text, type).not.toMatch(/notifications\.|\{(?:actor|topic|badge)\}/)
  expect(text, type).not.toBe('查看通知详情')
 }
})

test('global mark-all applies to a category loaded while the request is pending', async () => {
 await setLocale('en')
 let finish!: () => void
 vi.mocked(markAllNotificationsRead).mockReturnValueOnce(new Promise<void>(resolve => { finish = resolve }))
 fetchNotifications.mockResolvedValue({items:[item('system')],nextCursor:0,hasNext:false,unreadCount:1})
 const wrapper=mount(NotificationsPage,{props:{layout:{} as any,props:{notifications:[item()],total:1,unreadCount:1,pagination:{page:1,nextPage:0,hasNext:false,nextUrl:''}}},global:{plugins:[i18n]}})
 await wrapper.findAll('button').find(button=>button.text()===i18n.global.t('notifications.markAllRead'))!.trigger('click')
 await wrapper.findAll('button').find(button=>button.text()==='System')!.trigger('click')
 await flushPromises()
 expect(wrapper.findAll('article button')).toHaveLength(1)
 finish(); await flushPromises()
 expect(wrapper.findAll('article button')).toHaveLength(0)
 wrapper.unmount()
})

test('stale single-item mark-read failure never rolls back a newer category unread count', async () => {
 await setLocale('en')
 let rejectMark!: (reason: unknown) => void
 const staleMark = new Promise<boolean>((_, reject) => { rejectMark = reject })
 vi.mocked(markNotificationRead).mockReturnValueOnce(staleMark)
 unreadRefresh.mockResolvedValue(undefined)
 const wrapper = mount(NotificationsPage, { props: { layout: {} as any, props: { notifications: [item()], total: 1, unreadCount: 3, pagination: { page: 1, nextPage: 0, hasNext: false, nextUrl: '' } } }, global: { plugins: [i18n] } })
 expect(wrapper.text()).toContain('3 unread')
 await wrapper.findAll('article').find(article => article.text().includes('Campus'))!.find('a').trigger('click')
 // Switch to Mentions, which loads a fresh category with a smaller unread count.
 fetchNotifications.mockResolvedValueOnce({ items: [item('mention')], nextCursor: 0, hasNext: false, unreadCount: 1 })
 await wrapper.findAll('button').find(button => button.text() === 'Mentions')!.trigger('click')
 await flushPromises()
 expect(wrapper.text()).toContain('1 unread')
 // Now the category A request fails; the stale rollback must not resurrect A's count.
 rejectMark(new Error('rejected'))
 await flushPromises()
 expect(wrapper.text()).toContain('1 unread')
 expect(wrapper.text()).not.toContain('3 unread')
 expect(unreadRefresh).toHaveBeenCalledWith(true)
 wrapper.unmount()
})

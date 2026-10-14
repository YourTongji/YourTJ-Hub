// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n, setLocale } from '../src/runtime/i18n'
import { notificationText } from '../src/runtime/notification-text'
import type { NotificationPayload } from '@gooseforum/client'
const { fetchNotifications } = vi.hoisted(() => ({fetchNotifications:vi.fn()}))
vi.mock('../src/runtime/api',()=>({fetchNotifications,markAllNotificationsRead:vi.fn(),markNotificationRead:vi.fn()}))
vi.mock('../src/runtime/unread-status',()=>({useUnreadStatus:()=>({clearNotifications:vi.fn(),setNotifications:vi.fn()})}))
import NotificationsPage from '../src/site/pages/NotificationsPage.vue'
import { markAllNotificationsRead } from '../src/runtime/api'
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

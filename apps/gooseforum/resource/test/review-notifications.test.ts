// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import NotificationsPage from '../src/site/pages/NotificationsPage.vue'
import { i18n } from '../src/runtime/i18n'
vi.mock('../src/runtime/unread-status', () => ({ useUnreadStatus: () => ({ clearNotifications: vi.fn() }) }))
afterEach(() => vi.clearAllMocks())
for (const [state, subject] of [['Pending', '没有标题的待审正文'], ['Approved', '没有标题的通过正文'], ['Rejected', '首******尾']]) {
 test(`${state} uses reviewed subject even when the live topic title is empty`, () => {
  const wrapper = mount(NotificationsPage, { props: { layout: {} as any, props: {
   notifications: [{ id: 1, eventType: `review_${state.toLowerCase()}`, isRead: true, createdAt: '', title: '', content: '', actor: { id: 0, username: '' }, topic: { id: 42, title: '', url: '/p/42' }, payload: { actorId: 0, templateKey: `notifications.templates.review${state}`, topicTitle: subject } }],
   total: 1, unreadCount: 0, pagination: { hasNext: false },
  } as any }, global: { plugins: [i18n] } })
  const link = wrapper.get('article a')
  expect(link.text()).toBe(subject)
  expect(link.attributes('href')).toBe(state === 'Rejected' ? '/settings?tab=content' : '/p/42')
  wrapper.unmount()
 })
}
for (const profileUrl of ['/settings?tab=content', 'https://example.com/redirect', '/admin', '/settings?tab=content&extra=1']) {
 test(`pending appeal notification only accepts the exact content-management destination: ${profileUrl}`, () => {
  const wrapper = mount(NotificationsPage, { props: { layout: {} as any, props: {
   notifications: [{ id: 2, eventType: 'review_pending', isRead: true, createdAt: '', title: '', content: '', actor: { id: 0, username: '' }, topic: { id: 42, title: '', url: '/p/42' }, payload: { actorId: 0, templateKey: 'notifications.templates.reviewPending', topicTitle: 'Redacted candidate', metadata: { profileUrl } } }],
   total: 1, unreadCount: 0, pagination: { hasNext: false },
  } as any }, global: { plugins: [i18n] } })
  expect(wrapper.get('article a').attributes('href')).toBe(profileUrl === '/settings?tab=content' ? '/settings?tab=content' : '/p/42')
  wrapper.unmount()
 })
}
for (const [template, eventType, title, href] of [
 ['appealSubmitted', 'review_pending', 'Appeal submitted — awaiting a moderator', '/settings?tab=content'],
 ['appealApproved', 'review_approved', 'Appeal approved — your post is public', '/p/42'],
 ['appealRejected', 'review_rejected', 'Appeal not approved', '/settings?tab=content'],
 ['appealStale', 'review_rejected', 'Appeal closed', '/settings?tab=content'],
]) {
 test(`${template} notification has its own wording and destination`, () => {
  i18n.global.locale.value = 'en'
  const wrapper = mount(NotificationsPage, { props: { layout: {} as any, props: {
   notifications: [{ id: 3, eventType, isRead: false, createdAt: '', title: '', content: '', actor: { id: 0, username: '' }, topic: { id: 42, title: '', url: '/p/42' }, payload: { actorId: 0, templateKey: `notifications.templates.${template}`, topicTitle: 'S******t' } }],
   total: 1, unreadCount: 1, pagination: { hasNext: false },
  } as any }, global: { plugins: [i18n] } })
  const article = wrapper.get('article')
  expect(article.text()).toContain(title)
  expect(article.text()).not.toContain('Your post wasn’t approved')
  expect(article.get('a').attributes('href')).toBe(href)
  wrapper.unmount()
 })
}

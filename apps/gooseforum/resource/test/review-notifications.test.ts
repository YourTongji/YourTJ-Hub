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

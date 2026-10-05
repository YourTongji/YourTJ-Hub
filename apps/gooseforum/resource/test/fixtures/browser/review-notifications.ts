import { createApp, h } from 'vue'
import NotificationsPage from '../../../src/site/pages/NotificationsPage.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
import type { LayoutPayload, NotificationPayload } from '@gooseforum/client'
await setLocale('zh')
document.documentElement.dataset.theme = new URLSearchParams(location.search).get('theme') || 'gf-light'
const notifications = ['Pending', 'Approved', 'Rejected'].map((state, index) => ({
  id: index + 1, eventType: `review_${state.toLowerCase()}`, isRead: false, createdAt: '2026-10-04T08:00:00Z', title: '', content: '', actor: { id: 0, username: '' },
  topic: { id: 42, title: state === 'Rejected' ? '校******论' : '校园生活中的一次讨论', url: state === 'Rejected' ? '/settings?tab=content' : '/p/post/42/1' },
  payload: { templateKey: `notifications.templates.review${state}`, actorId: 0, topicId: 42, postNo: 1, topicTitle: state === 'Rejected' ? '校******论' : '校园生活中的一次讨论', templateParams: {}, metadata: { followerName: '' } },
})) as NotificationPayload[]
createApp({ render: () => h('main', { class: 'mx-auto max-w-3xl bg-base-100 p-4 text-base-content' }, h(NotificationsPage, { layout: {} as LayoutPayload, props: { total: 3, unreadCount: 3, notifications, pagination: { page: 1, hasNext: false, nextPage: 0, nextUrl: '' } } })) }).use(i18n).mount('#app')

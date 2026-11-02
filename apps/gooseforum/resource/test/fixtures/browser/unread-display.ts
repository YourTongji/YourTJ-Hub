import { h } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import type { LayoutPayload, MessagesPageProps, NotificationsPageProps } from '@gooseforum/client'
import { i18n, setLocale, type Locale } from '../../../src/runtime/i18n'
import AppShell from '../../../src/site/components/AppShell.vue'
import UnreadDisplaySettings from '../../../src/site/components/UnreadDisplaySettings.vue'
import NotificationsPage from '../../../src/site/pages/NotificationsPage.vue'
import MessagesPage from '../../../src/site/pages/MessagesPage.vue'
import { createBrowserApp } from './create-app'
import '../../../src/styles/resource.css'

const params = new URLSearchParams(location.search)
const language = params.get('lang') || 'en'
document.documentElement.dataset.theme = params.get('theme') === 'dark' ? 'gf-dark' : 'gf-light'
await setLocale(language as Locale)
const initial = (await (await fetch('/api/forum/unread-status')).json()).result
const layout = {
  site: { name: 'yourtj', description: '', logo: '', favicon: '', brandType: 'text', brandText: 'yourtj', brandImage: '' },
  viewer: { id: 1024, username: 'unread-tester', email: '', avatarUrl: '/assets/test/fixtures/browser/anonymous-admin-avatar-1.svg', isAuthenticated: true, canAccessAdmin: false, canManageAgents: false, isModerator: false, requiresEmailVerification: false, adminPermissions: [] },
  sidebar: { main: [], resources: [], groups: [], categories: [], activeKey: 'notifications' },
  footer: { links: [], primary: [] }, unread: initial,
  theme: { enabled: false, current: 'gf-light', themeColor: '#fbfdff' }, umamiEnabled: false,
} as LayoutPayload
const notices = {
  notifications: [1, 2, 3].map(id => ({ id, eventType: 'comment', isRead: false, createdAt: '2026-10-01T12:00:00Z', title: 'Reply', content: 'Fixture notification', actor: { id: 9, username: 'Peer' }, payload: { actorId: 9, actorName: 'Peer' } })),
  total: 3, unreadCount: 3, pagination: { page: 1, nextPage: 0, hasNext: false, nextUrl: '' },
} as NotificationsPageProps
const conversations = {
  conversations: ['Read peer', 'Unread peer'].map((name, index) => ({ id: index + 1, peerId: index + 8, peerUsername: name, peerAvatar: '', convId: index + 10, lastMsg: 'Hello', lastMsgTime: '', unreadCount: index === 0 ? 0 : 12, peerUrl: `/u/${index + 8}` })),
} as MessagesPageProps
const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:pathMatch(.*)*', component: { render: () => null } }] })
await router.push('/settings?tab=notifications')
await router.isReady()
createBrowserApp({ render: () => h(AppShell, { layout }, { default: () => h('main', { class: 'mx-auto max-w-3xl space-y-8 p-4' }, [
  h(UnreadDisplaySettings), h(NotificationsPage, { layout, props: notices }),
  h('section', { class: 'h-[540px]', id: 'fixture-messages' }, h(MessagesPage, { layout, props: conversations })),
]) }) }).use(i18n).use(router).mount('#app')
document.documentElement.dataset.switchLabel = i18n.global.t('notifications.unreadStyleNumbers')
document.documentElement.dataset.ready = 'true'

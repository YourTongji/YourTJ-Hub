import { createApp, h } from 'vue'
import type { LayoutPayload, MessagesPageProps } from '@gooseforum/client'
import MessagesPage from '../../../src/site/pages/MessagesPage.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'

await setLocale('en')
document.documentElement.dataset.theme = new URLSearchParams(location.search).get('theme') || 'gf-light'
const layout = { viewer: { id: 1, username: 'Alice', avatarUrl: '' } } as LayoutPayload
const props: MessagesPageProps = {
  conversations: [{ id: 1, convId: 1, peerId: 2, peerUsername: 'Bob', peerAvatar: '', lastMsg: '', lastMsgTime: '', unreadCount: 0, peerUrl: '/u/2' }],
  suggestedUsers: [],
}
createApp({ render: () => h(MessagesPage, { layout, props }) }).use(i18n).mount('#app')

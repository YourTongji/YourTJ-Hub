import { h } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import UnreadDisplaySettings from '../../../src/site/components/UnreadDisplaySettings.vue'
import { useUnreadStatus } from '../../../src/runtime/unread-status'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
const params = new URLSearchParams(location.search)
await setLocale(params.get('locale') === 'de' ? 'de' : 'zh')
document.documentElement.dataset.theme = params.get('theme') || 'gf-light'
useUnreadStatus().startPolling({ notifications: true, messages: false, notificationCount: 8, badgeStyle: 'number' }, 42)
createApp({ render: () => h('main', { class: 'mx-auto max-w-xl bg-base-100 p-4 text-base-content' }, h(UnreadDisplaySettings)) }).use(i18n).mount('#app')

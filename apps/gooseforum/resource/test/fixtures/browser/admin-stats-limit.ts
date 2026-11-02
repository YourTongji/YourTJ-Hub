import { h } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import { createMemoryHistory, createRouter } from 'vue-router'
import AdminSettingsPage from '../../../src/admin/pages/AdminSettingsPage.vue'
import { adminText } from '../../../src/admin/runtime/i18n-text'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/admin/styles/admin.css'
const query = new URLSearchParams(location.search)
await setLocale((query.get('lang') || 'zh') as 'zh' | 'en' | 'ja' | 'de')
document.documentElement.dataset.theme = query.get('dark') === 'true' ? 'gf-dark' : 'gf-light'
document.documentElement.classList.toggle('dark', query.get('dark') === 'true')
const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
await router.push('/')
createApp({ render: () => h(AdminSettingsPage, { kind: 'rate-limit', payload: { layout: { site: { name: 'YourTJ' }, sidebar: { categories: [] }, viewer: { username: 'admin', avatarUrl: '' } } } as never }) }).use(i18n).use(router).mount('#app')
document.querySelector<HTMLElement>('#app')!.dataset.saveLabel = adminText('k004f')
document.querySelector<HTMLElement>('#app')!.dataset.limitHint = i18n.global.t('statsAdmin.rateLimitExemptionHint')

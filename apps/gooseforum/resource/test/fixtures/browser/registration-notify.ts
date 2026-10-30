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
const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }, { path: '/admin/settings/ai-moderation', component: { template: '<div />' } }] })
await router.push('/')
createApp({ render: () => h(AdminSettingsPage, { kind: query.get('kind') === 'security' ? 'security' : 'http-notify', payload: { layout: { site: { name: 'YourTJ' }, sidebar: { categories: [] }, viewer: { username: 'admin', avatarUrl: '' } } } as never }) }).use(i18n).use(router).mount('#app')
const element = document.querySelector<HTMLElement>('#app')!
for (const [name, key] of Object.entries({ saveLabel: 'k004f', signupLabel: 'k00cn', reportLabel: 'k00x1', verificationHint: 'k0091', feishuHint: 'k00xb' })) element.dataset[name] = adminText(key)

element.dataset.ready = 'true'

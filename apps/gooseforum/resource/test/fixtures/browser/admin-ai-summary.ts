import { h } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import { createMemoryHistory, createRouter } from 'vue-router'
import AdminSettingsPage from '../../../src/admin/pages/AdminSettingsPage.vue'
import { adminText } from '../../../src/admin/runtime/i18n-text'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/admin/styles/admin.css'
const query = new URLSearchParams(location.search)
await setLocale('zh')
document.documentElement.dataset.theme = query.get('theme') || 'gf-light'
document.documentElement.classList.toggle('dark', query.get('theme') === 'gf-dark')
const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
await router.push('/')
createApp({ render: () => h(AdminSettingsPage, { payload: { layout: {} } as never, kind: 'ai-summary' }) }).use(i18n).use(router).mount('#app')
// Reproduce the real app's locale change after the initial selector mount.
await setLocale((query.get('lang') || 'zh') as 'zh' | 'en' | 'ja' | 'de')
const app = document.querySelector<HTMLElement>('#app')!
app.dataset.refreshLabel = adminText('k00pd')
app.dataset.ready = 'true'

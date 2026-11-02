import { h } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import { createMemoryHistory, createRouter } from 'vue-router'
import AdminSettingsPage from '../../../src/admin/pages/AdminSettingsPage.vue'
import FeedStatisticsPage from '../../../src/admin/pages/FeedStatisticsPage.vue'
import WikiManage from '../../../src/admin/pages/WikiManage.vue'
import SponsorsManagementPage from '../../../src/admin/pages/management/SponsorsManagementPage.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/admin/styles/admin.css'

const query = new URLSearchParams(location.search)
await setLocale((query.get('lang') || 'de') as 'zh' | 'en' | 'ja' | 'de')
document.documentElement.dataset.theme = query.get('theme') || 'gf-light'
document.documentElement.classList.toggle('dark', query.get('theme') === 'gf-dark')
const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
await router.push('/')
const kind = query.get('kind') || 'security'
const pages = { feed: FeedStatisticsPage, wiki: WikiManage, sponsors: SponsorsManagementPage }
const page = pages[kind as keyof typeof pages] || AdminSettingsPage
createApp({ render: () => h(page as typeof AdminSettingsPage, { payload: { layout: {} } as never, kind: kind as 'security' }) }).use(i18n).use(router).mount('#app')
document.querySelector<HTMLElement>('#app')!.dataset.ready = 'true'

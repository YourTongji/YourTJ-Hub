import { createApp, h } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import HomePage from '../../../src/site/pages/HomePage.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import type { HomeProps, LayoutPayload } from '@gooseforum/client'
import '../../../src/styles/resource.css'

await setLocale('zh')

const query = new URLSearchParams(location.search)
document.documentElement.dataset.theme = query.get('theme') || 'gf-light'
const props: HomeProps = {
  sort: 'latest',
  tabs: [{ key: 'latest', url: '/', active: true }],
  topics: [],
  pagination: { page: 1, nextPage: 0, hasNext: false, nextUrl: '' },
  announcement: {
    enabled: true,
    html: '',
    items: [{ id: 'preview', title: '', html: '<p>欢迎来到 YourTJHub！</p>' }],
  },
}
const layout = {
  viewer: { id: 0, isAuthenticated: false, requiresEmailVerification: false },
} as LayoutPayload
const router = createRouter({
  history: createMemoryHistory(),
  routes: [{ path: '/', component: { render: () => null } }],
})
await router.push('/')
await router.isReady()

createApp({ render: () => h(HomePage, { layout, props, pageUrl: '/' }) })
  .use(i18n)
  .use(router)
  .mount('#app')
document.documentElement.dataset.ready = 'true'

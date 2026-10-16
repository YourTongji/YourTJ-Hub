import { h } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import { createMemoryHistory, createRouter } from 'vue-router'
import HomePage from '../../../src/site/pages/HomePage.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
import type { HomeProps, LayoutPayload } from '@gooseforum/client'

// 首页公告轮播高度夹具（issue #1102）：三条行数差异明显的公告（一条一行、
// 一条多段、一条两行），由 test/home-announcement-rotation.browser.mjs 在真实
// Chromium 里量几何，确认轮播/圆点切换时公告面板高度与下方内容位置不变。
await setLocale('zh')

const longHtml = [
  '<p>第二条公告的第一段说明文字。</p>',
  '<p>第二条公告的第二段说明文字。</p>',
  '<p>第二条公告的第三段带一个<a href="https://example.com/announcement-link">说明链接</a>。</p>',
].join('')

const announcement: HomeProps['announcement'] = {
  enabled: true,
  html: '',
  publishedAt: new Date().toISOString(),
  items: [
    { id: 'ann-short', title: '短公告', html: '<p>一行短公告</p>' },
    { id: 'ann-long', title: '长公告', html: longHtml },
    { id: 'ann-medium', title: '', html: '<p>中等公告第一行</p><p>中等公告第二行</p>' },
  ],
}

const props: HomeProps = {
  sort: 'latest',
  tabs: [{ key: 'latest', url: '/', active: true }],
  topics: [],
  pagination: { page: 1, nextPage: 0, hasNext: false, nextUrl: '' },
  announcement,
}

const layout = {
  viewer: {
    id: 1,
    username: 'tester',
    email: 'tester@example.com',
    avatarUrl: '',
    isAuthenticated: true,
    canAccessAdmin: false,
    isModerator: false,
    requiresEmailVerification: false,
    adminPermissions: [],
  },
} as unknown as LayoutPayload

// HomePage 依赖 useRouter()（导航与会话恢复），与既有 home-announcement 夹具一致提供内存路由。
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

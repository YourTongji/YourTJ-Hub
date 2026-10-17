// issue #1164：首页「列表 / 卡片」视图切换的 INP 基线夹具。
// 渲染真实 HomePage（latest 排序，20 条话题，含图文卡片），供 feed-switch.browser.mjs
// 量测切换时的处理耗时与 DOM 复用情况。
import { defineComponent, h, KeepAlive, shallowRef } from 'vue'
import HomePage from '../../../src/site/pages/HomePage.vue'
import { installNavigation, type PreparedPage } from '../../../src/runtime/router'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import type { HomeProps, LayoutPayload, TopicPayload } from '@gooseforum/client'
import { createBrowserApp } from './create-app'
import '../../../src/styles/resource.css'

await setLocale('zh')
document.documentElement.dataset.theme = 'gf-light'
const query = new URLSearchParams(location.search)
const count = Number(query.get('count') ?? 20)

const art = '/assets/test/fixtures/browser/topic-feed-art.svg'
const withImages = query.get('images') !== '0'
const seed = (id: number): TopicPayload => ({
  id,
  title: `话题标题 ${id}：关于校园生活的一些讨论`,
  description: '这是一段描述文本，用于模拟真实话题的摘要内容，帮助读者判断是否要打开这篇帖子。'.repeat(2),
  url: `/p/post/${id}`,
  author: { id: 1000 + id, username: `作者 ${id}`, avatarUrl: art },
  participants: [
    { id: 1000 + id, username: `作者 ${id}`, avatarUrl: art },
    { id: 2000 + id, username: `回复者 ${id}`, avatarUrl: art },
  ],
  categories: [{ id: 7, name: '校园生活', url: '/c/7', color: '#3b82f6' }],
  replyCount: 12, likeCount: 5, viewCount: 640,
  activityText: '刚刚', lastUpdateTime: '2026-10-07T00:00:00Z',
  pinWeight: 0, processStatus: 0, contentType: 0, liked: false, bookmarked: false,
  images: !withImages ? [] : id % 3 === 0 ? [art, art, art] : id % 3 === 1 ? [art] : [],
  feedTrace: 'trace-fixture', feedPosition: id - 1,
})

const props: HomeProps = {
  sort: 'latest',
  tabs: [{ key: 'latest', active: true, url: '/?sort=latest' }, { key: 'hot', active: false, url: '/?sort=hot' }],
  topics: Array.from({ length: count }, (_, i) => seed(i + 1)),
  pagination: { page: 1, nextPage: 2, hasNext: false, nextUrl: '' },
  announcement: { enabled: false, html: '', items: [] },
}
const layout = { viewer: { id: 12, isAuthenticated: true, requiresEmailVerification: false } } as LayoutPayload
const initial: PreparedPage = { component: HomePage, payload: { component: 'home.index', props, layout, url: '/?sort=latest', version: '1.0' } }
const current = shallowRef(initial)
window.history.replaceState(null, '', '/?sort=latest')
const root = defineComponent({ setup: () => () => h('main', [
  h(KeepAlive, {}, () => current.value.payload.component === 'home.index'
    ? h(HomePage, { key: 'home.index', props: current.value.payload.props as HomeProps, layout, pageUrl: current.value.payload.url }) : null),
]) })
const router = installNavigation(initial, root, (page) => { current.value = page })
const app = createBrowserApp(root).use(i18n).use(router)
await router.isReady()
performance.mark('feed-mount-start')
app.mount('#app')
performance.mark('feed-mount-end')
performance.measure('feed-mount', 'feed-mount-start', 'feed-mount-end')
document.documentElement.dataset.ready = 'true'

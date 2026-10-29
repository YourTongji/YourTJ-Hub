import { defineComponent, h, KeepAlive, shallowRef } from 'vue'
import HomePage from '../../../src/site/pages/HomePage.vue'
import CategoryPage from '../../../src/site/pages/CategoryPage.vue'
import { installNavigation, type PreparedPage } from '../../../src/runtime/router'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import type { HomeProps, LayoutPayload, TopicPayload } from '@gooseforum/client'
import { createBrowserApp } from './create-app'
import '../../../src/styles/resource.css'

await setLocale('zh')
document.documentElement.dataset.theme = 'gf-light'

const art = '/assets/test/fixtures/browser/topic-feed-art.svg'
export const topic = (id: number, title = `返回测试话题 ${id}`): TopicPayload => ({
  id,
  title,
  description: '这是用于验证详情返回后分页列表与滚动位置的测试话题。'.repeat(2),
  url: `/p/post/${id}`,
  author: { id: 1000 + id, username: `作者 ${id}`, avatarUrl: art },
  participants: [],
  categories: [{ id: 7, name: '校园生活', url: '/c/7', color: '#3b82f6' }],
  replyCount: 3,
  likeCount: 0,
  viewCount: 12,
  activityText: '刚刚',
  lastUpdateTime: '2026-10-07T00:00:00Z',
  pinWeight: 0,
  processStatus: 0,
  contentType: 0,
  liked: false,
  bookmarked: false,
  feedTrace: `trace-${id}`,
  feedPosition: id - 1,
})

const layout = {
  viewer: { id: 12, isAuthenticated: true, requiresEmailVerification: false },
} as LayoutPayload
const props: HomeProps = {
  sort: 'latest',
  actualSort: 'latest',
  tabs: [
    { key: 'latest', active: true, url: '/?sort=latest' },
    { key: 'hot', active: false, url: '/?sort=hot' },
  ],
  topics: Array.from({ length: 20 }, (_, index) => topic(index + 1)),
  pagination: { page: 1, nextPage: 2, hasNext: true, nextUrl: '/?sort=latest&cursor=next-20' },
  announcement: { enabled: false, html: '', items: [] },
}
const initial: PreparedPage = {
  component: HomePage,
  payload: { component: 'home.index', props, layout, url: '/?sort=latest', version: '1.0' },
}
const current = shallowRef(initial)
window.history.replaceState(null, '', '/?sort=latest')

// Keep the real HomePage and router lifecycle, while rendering a small detail
// surface so the test controls only the remote page payload.
const root = defineComponent({
  setup: () => () => h('main', [
    h(KeepAlive, {}, () => {
      if (current.value.payload.component === 'home.index') return h(HomePage, {
          key: 'home.index',
          props: current.value.payload.props as HomeProps,
          layout: current.value.payload.layout,
          pageUrl: current.value.payload.url,
        })
      if (current.value.payload.component === 'category.index') return h(CategoryPage, {
        key: 'category.index',
        props: current.value.payload.props as import('@gooseforum/client').CategoryPageProps,
        layout: current.value.payload.layout,
        pageUrl: current.value.payload.url,
      })
      return null
    }),
    current.value.payload.component === 'topic.detail'
      ? h('article', { 'data-testid': 'detail' }, [h('h1', `详情 ${current.value.payload.url}`)])
      : null,
  ]),
})

const router = installNavigation(initial, root, (page) => { current.value = page })
const app = createBrowserApp(root).use(i18n).use(router)
await router.isReady()
app.mount('#app')
document.documentElement.dataset.ready = 'true'

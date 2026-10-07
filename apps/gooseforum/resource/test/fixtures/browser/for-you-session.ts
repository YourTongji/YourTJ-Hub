import { createApp, defineComponent, h, KeepAlive, shallowRef } from 'vue'
import HomePage from '../../../src/site/pages/HomePage.vue'
import { installNavigation, type PreparedPage } from '../../../src/runtime/router'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import type { HomeProps, LayoutPayload, TopicPayload } from '@gooseforum/client'
import '../../../src/styles/resource.css'

await setLocale('zh')
document.documentElement.dataset.theme = 'gf-light'
const query = new URLSearchParams(location.search)
const empty = query.get('empty') === '1'
const seed = (id: number): TopicPayload => ({ id, title: `推荐帖子 ${id}`, description: '校园里的新讨论，读者可以继续原来的浏览位置。'.repeat(3),
  url: `/p/post/${id}`, author: { id: 1000 + id, username: `作者 ${id}`, avatarUrl: '/assets/test/fixtures/browser/topic-feed-art.svg' }, participants: [], categories: [],
  replyCount: 3, likeCount: 0, viewCount: 12, activityText: '刚刚', lastUpdateTime: '2026-10-07T00:00:00Z',
  pinWeight: 0, processStatus: 0, contentType: 0, liked: false, bookmarked: false, feedTrace: 'trace-original', feedPosition: (id - 1) % 20 })
const props: HomeProps = { sort: 'for_you', actualSort: 'for_you', snapshotId: 'original',
  tabs: [{ key: 'for_you', active: true, url: '/?sort=for_you' }, { key: 'latest', active: false, url: '/?sort=latest' }],
  topics: empty ? [] : Array.from({ length: 20 }, (_, i) => seed(i + 1)),
  seenProofs: empty ? [] : [{ token: 'proof-1', topicIds: Array.from({ length: 20 }, (_, i) => i + 1), issuedAt: Date.now(), expiresAt: Date.now() + 1_800_000 }],
  pagination: { page: 1, nextPage: 2, hasNext: !empty, nextUrl: empty ? '' : '/?sort=for_you&cursor=next-20' },
  announcement: { enabled: false, html: '', items: [] } }
const layout = { viewer: { id: 12, isAuthenticated: true, requiresEmailVerification: false } } as LayoutPayload
const initial: PreparedPage = { component: HomePage, payload: { component: 'home.index', props, layout, url: '/?sort=for_you', version: '1.0' } }
const current = shallowRef(initial)
window.history.replaceState(null, '', '/?sort=for_you')
const root = defineComponent({ setup: () => () => h('main', [
  h(KeepAlive, {}, () => current.value.payload.component === 'home.index'
    ? h(HomePage, { key: 'home.index', props: current.value.payload.props as HomeProps, layout, pageUrl: current.value.payload.url }) : null),
  current.value.payload.component === 'topic.detail' ? h('section', { 'data-testid': 'detail', style: 'height:1200px' }, [
    h('h1', '详情'), ...['like', 'bookmark', 'comment'].map((action) => h('button', {
      'data-action': action, onClick: () => fetch(`/api/forum/test-${action}`, { method: 'POST' }),
    }, action)),
  ]) : null,
]) })
const router = installNavigation(initial, root, (page) => { current.value = page })
const app = createApp(root).use(i18n).use(router)
await router.isReady()
app.mount('#app')
document.documentElement.dataset.ready = 'true'

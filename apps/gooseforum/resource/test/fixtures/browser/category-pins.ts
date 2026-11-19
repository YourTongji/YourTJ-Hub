import { createBrowserApp } from './create-app'
import CategoryPage from '../../../src/site/pages/CategoryPage.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import type { CategoryPageProps, LayoutPayload, TopicPayload } from '@gooseforum/client'
import '../../../src/styles/resource.css'

const query = new URLSearchParams(location.search)
await setLocale('zh')
document.documentElement.dataset.theme = query.get('theme') || 'gf-light'
const sort = query.get('sort') || 'latest'
const topic = (id: number, title: string, pinWeight = 0): TopicPayload => ({
  id, title, pinWeight, url: `/p/post/${id}`, description: '此内容为本地验收演示数据。',
  author: { id: 1, username: '演示同学', avatarUrl: '' }, participants: [], categories: [],
  replyCount: 3, viewCount: 28, activityText: '刚刚', lastUpdateTime: '', processStatus: 0,
})
const pins = [topic(1, '社区使用指南与发帖须知（演示）', 2), topic(2, '校园生活 · 新生常见问题与实用信息汇总（演示）', 1)]
const regular = [topic(3, '图书馆里的秋日午后（演示）'), topic(4, '一起分享校园里的新发现（演示）')]
const props: CategoryPageProps = {
  category: { id: 7, name: '校园生活', description: '本地分类置顶验收 · 演示数据', icon: '', color: '#3b82f6', url: '/c/7' },
  sort,
  tabs: ['latest', 'new', 'hot', 'popular'].map(key => ({ key, active: key === sort, url: `?sort=${key}` })),
  topics: query.get('state') === 'none' ? regular : query.get('state') === 'only' ? pins : [...pins, ...regular],
  pagination: { page: 1, nextPage: 0, hasNext: false, nextUrl: '' },
}
createBrowserApp(CategoryPage, { props, layout: {} as LayoutPayload, pageUrl: '/c/7' }).use(i18n).mount('#app')
document.documentElement.dataset.ready = 'true'

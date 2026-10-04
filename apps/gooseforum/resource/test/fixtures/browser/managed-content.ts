import { createApp, h, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import QuickPublishModal from '../../../src/site/components/QuickPublishModal.vue'
import type { LayoutPayload } from '@gooseforum/client'
import ManagedContent from '../../../src/site/components/ManagedContent.vue'
import { useContentUpdates } from '../../../src/runtime/content-updates'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
import type { MyContentItem } from '../../../src/runtime/api'
await setLocale('zh')
document.documentElement.dataset.theme = new URLSearchParams(location.search).get('theme') || 'gf-light'
const params = new URLSearchParams(location.search)
const layout = { viewer: { isAuthenticated: true, id: 1, username: 'viewer' }, sidebar: { categories: [{ id: 101, label: '校园生活', color: '#10b981', url: '/c/101' }], activeKey: '' }, posting: { maxTitleLength: 100 }, theme: { current: params.get('theme') || 'gf-light' } } as unknown as LayoutPayload
const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
const initial: MyContentItem = {
  id: 42, contentType: params.has('topic') ? 'topic' : 'post', title: '校园生活中的一次讨论', excerpt: '我的最新修改',
  createdAt: '2026-10-04T08:00:00Z', content: '**完整正文**\n\n被拒的修改仍可编辑，原来公开的版本继续保留。',
  processStatus: 1, revisionId: 105, reviewReason: '请修改正文后重新提交。', hasPublishedVersion: true, images: [],
}
createApp({
  setup() {
    const item = ref(initial)
    useContentUpdates(async () => { item.value = await (await fetch('/fixture/content')).json() })
    return () => h('main', { class: 'mx-auto max-w-xl space-y-4 bg-base-100 p-6 text-base-content' }, [
      h(QuickPublishModal, { layout }),
      h('h1', { class: 'text-xl font-semibold' }, '内容管理'),
      h('h2', { class: 'font-medium' }, item.value.title),
      h(ManagedContent, { item: item.value, onSaved: () => { item.value = { ...item.value, processStatus: 2, reviewReason: '' } } }),
    ])
  },
}).use(router).use(i18n).mount('#app')

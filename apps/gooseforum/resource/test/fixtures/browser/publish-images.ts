import { createApp, h } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import PublishPage from '../../../src/site/pages/PublishPage.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
await setLocale('zh')
document.documentElement.dataset.theme = new URLSearchParams(location.search).get('theme') || 'gf-light'
const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
await router.push('/')
await router.isReady()
createApp({
  render: () => h(PublishPage, {
    layout: { viewer: { id: 7, isAuthenticated: true }, posting: {} } as any,
    props: { topicId: 17, isEditing: true, categories: [{ id: 1, name: '校园生活', color: '#059669' }], topic: {
      title: '修改后重新提交', content: '修改正文和图片后，可以重新提交审核。', categoryIds: [1], contentType: new URLSearchParams(location.search).get('type') === '2' ? 2 : 3,
      images: ['/assets/test/fixtures/browser/topic-feed-art.svg'],
    } } as any,
  }),
}).use(i18n).use(router).mount('#app')

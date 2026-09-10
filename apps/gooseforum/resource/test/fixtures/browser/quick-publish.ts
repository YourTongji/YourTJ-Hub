import { createApp, h } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import QuickPublishModal from '../../../src/site/components/QuickPublishModal.vue'
import { useQuickPublish } from '../../../src/site/composables/useQuickPublish'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'
import type { LayoutPayload } from '@gooseforum/client'
await setLocale('zh')
const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
const layout = { viewer: { isAuthenticated: true, id: 1, username: 'viewer' }, sidebar: { categories: [{ id: 101, label: '学习', color: '#10b981', url: '/c/101' }], activeKey: '' }, posting: { maxTitleLength: 100 }, theme: { current: 'gf-light' } } as unknown as LayoutPayload
createApp({ render: () => h(QuickPublishModal, { layout }) }).use(router).use(i18n).mount('#app')
useQuickPublish().openQuickPublish(2)

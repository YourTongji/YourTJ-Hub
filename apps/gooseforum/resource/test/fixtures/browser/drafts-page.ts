import { h } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import DraftsPage from '../../../src/site/pages/DraftsPage.vue'
import GlobalFlash from '../../../src/site/components/GlobalFlash.vue'
import { hydrateFlashMessages } from '../../../src/runtime/flash-message'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'

const params = new URLSearchParams(location.search)
await setLocale(params.get('lang') ?? 'zh')
document.documentElement.dataset.theme = params.get('theme') || 'gf-light'
const count = params.get('count') === '100' ? 100 : 1

createApp({
  render: () => h('div', [h(DraftsPage, {
    layout: {} as any,
    props: {
      drafts: Array.from({ length: count }, (_, i) => ({
        id: 42 + i,
        title: i === 0 ? 'A saved draft' : `Draft ${i}`,
        description: 'Draft body preview',
        editUrl: `/publish?id=${42 + i}`,
        replyCount: 0,
        viewCount: 0,
        processStatus: 0,
        updatedAt: '2026-10-16T00:00:00Z',
        createdAt: '2026-10-15T00:00:00Z',
        categories: [],
      })),
      total: count,
      pagination: { page: 1, nextPage: 0, hasNext: false, nextUrl: '' },
    } as any,
  }), h(GlobalFlash)]),
}).use(i18n).mount('#app')
hydrateFlashMessages()

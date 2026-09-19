import { createApp, h } from 'vue'
import CampusPage from '../../../src/site/pages/CampusPage.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import type { LayoutPayload } from '@gooseforum/client'
import '../../../src/styles/resource.css'

await setLocale((new URLSearchParams(location.search).get('lang') || 'zh') as 'zh' | 'en' | 'ja' | 'de')
createApp({ render: () => h(CampusPage, { layout: { viewer: { isAuthenticated: true, id: 1 } } as LayoutPayload, props: {} }) })
  .use(i18n)
  .mount('#app')
document.documentElement.dataset.ready = 'true'

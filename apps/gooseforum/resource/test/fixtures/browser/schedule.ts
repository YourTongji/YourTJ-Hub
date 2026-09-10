import { createApp, h } from 'vue'
import SchedulePage from '../../../src/site/pages/SchedulePage.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import type { LayoutPayload } from '@gooseforum/client'
import '../../../src/styles/resource.css'

// Render the production page and CSS; API responses are supplied by the browser test.
await setLocale((new URLSearchParams(location.search).get('lang') || 'zh') as 'zh' | 'en' | 'ja' | 'de')
createApp({ render: () => h(SchedulePage, { layout: {} as LayoutPayload, props: {} }) })
  .use(i18n)
  .mount('#app')
document.documentElement.dataset.ready = 'true'

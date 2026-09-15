import { createApp, h } from 'vue'
import SearchIndexesPage from '../../../src/admin/pages/management/SearchIndexesPage.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/admin/styles/admin.css'

await setLocale((new URLSearchParams(location.search).get('lang') || 'zh') as 'zh' | 'en' | 'ja' | 'de')
createApp({ render: () => h('div', { style: 'max-width:1440px;margin:auto;padding:80px 16px 16px' }, [h(SearchIndexesPage)]) }).use(i18n).mount('#app')
document.documentElement.dataset.ready = 'true'

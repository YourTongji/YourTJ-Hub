import { createApp, h } from 'vue'
import StatusPage from '../../../src/site/pages/StatusPage.vue'
import { i18n, setLocale, type Locale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'

const query = new URLSearchParams(location.search)
await setLocale((query.get('lang') || 'zh') as Locale)
document.documentElement.dataset.theme = query.get('theme') || 'gf-light'
createApp({ render: () => h(StatusPage) }).use(i18n).mount('#app')

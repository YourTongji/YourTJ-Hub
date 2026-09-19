import { createApp, h } from 'vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
await setLocale((new URLSearchParams(location.search).get('lang') || 'zh') as 'zh' | 'en' | 'ja' | 'de')
import CampusCalendarRulesPage from '../../../src/admin/pages/CampusCalendarRulesPage.vue'
import '../../../src/admin/styles/admin.css'
createApp({ render: () => h('div', { class: 'p-4' }, h(CampusCalendarRulesPage)) }).use(i18n).mount('#app')

import { createApp, h } from 'vue'
import CourseMaterializePanel from '../../../src/admin/components/CourseMaterializePanel.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/admin/styles/admin.css'

await setLocale((new URLSearchParams(location.search).get('lang') || 'zh') as 'zh' | 'en' | 'ja' | 'de')
createApp({ render: () => h('main', { style: 'max-width:768px;margin:24px auto;padding:12px' }, [h(CourseMaterializePanel, {
  calendars: [{ calendarId: 122, calendarName: '2026-2027学年第1学期', status: 'completed', rowsWritten: 4647, totalPages: 24, lastCommittedPage: 24, errorMsg: '' }],
})]) }).use(i18n).mount('#app')
document.documentElement.dataset.ready = 'true'

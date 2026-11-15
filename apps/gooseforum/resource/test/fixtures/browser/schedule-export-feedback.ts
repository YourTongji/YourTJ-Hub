import { h } from 'vue'
import { createBrowserApp } from './create-app'
import ScheduleExportDialog from '../../../src/site/components/schedule/ScheduleExportDialog.vue'
import GlobalFlash from '../../../src/site/components/GlobalFlash.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'

await setLocale('zh')
createBrowserApp({ render: () => h('div', [
  h(ScheduleExportDialog, { open: true, cellCourses: [], cellSpans: [], occupiedGrid: [], activeCalendar: null, activePlanName: '测试方案', weekViewLabel: '', exportSummaryText: '' }),
  h(GlobalFlash),
]) }).use(i18n).mount('#app')
document.documentElement.dataset.ready = 'true'

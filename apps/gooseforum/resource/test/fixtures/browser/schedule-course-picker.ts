import { h, ref } from 'vue'
import { createBrowserApp } from './create-app'
import ScheduleCoursePicker from '../../../src/site/components/schedule/ScheduleCoursePicker.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import '../../../src/styles/resource.css'

await setLocale('zh')
document.documentElement.dataset.theme = 'gf-light'
const open = ref(true)
createBrowserApp({ render: () => h(ScheduleCoursePicker, { open: open.value, onClose: () => { open.value = false } }) })
  .use(i18n)
  .mount('#app')
document.documentElement.dataset.ready = 'true'

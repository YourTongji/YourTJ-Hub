import { createApp, h } from 'vue'
import CampusCalendarRulesPage from '../../../src/admin/pages/CampusCalendarRulesPage.vue'
import '../../../src/admin/styles/admin.css'
createApp({ render: () => h('div', { class: 'p-4' }, h(CampusCalendarRulesPage)) }).mount('#app')

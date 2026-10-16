import { h } from 'vue'
import AgentContentCorrectionDialog from '../../../src/admin/pages/management/AgentContentCorrectionDialog.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import type { AdminAgent } from '../../../src/admin/types'
import '../../../src/admin/styles/admin.css'
import { createBrowserApp } from './create-app'
await setLocale('en')
createBrowserApp({ render: () => h(AgentContentCorrectionDialog, { agent: { agentId: 9, nickname: 'Campus Bot' } as AdminAgent, categories: [{ id: 1, label: 'Campus' }, { id: 2, label: 'Study' }, { id: 3, label: 'Life' }, { id: 4, label: 'Help' }] }) }).use(i18n).mount('#app')

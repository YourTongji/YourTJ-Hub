import { createApp, h } from 'vue'
import AgentCommentPolicyPage from '../../../src/admin/pages/management/AgentCommentPolicyPage.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import type { AdminPayload, ManageHomeProps } from '../../../src/admin/types'
import '../../../src/admin/styles/admin.css'
await setLocale('en')
const payload = { component: 'manage-home', props: {}, meta: { title: 'admin' }, layout: {}, url: '/admin/agent-comment-policy', version: 'test' } as unknown as AdminPayload<ManageHomeProps>
createApp({ render: () => h('main', { style: 'padding:64px 16px 16px;min-width:0' }, [h(AgentCommentPolicyPage, { payload })]) }).use(i18n).mount('#app')

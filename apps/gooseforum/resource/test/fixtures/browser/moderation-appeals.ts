import { h } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import ManagedContent from '../../../src/site/components/ManagedContent.vue'
import ModerationAppealQueue from '../../../src/site/components/ModerationAppealQueue.vue'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import { setTheme } from '../../../src/runtime/site-theme'
import '../../../src/styles/resource.css'
const params = new URLSearchParams(location.search)
await setLocale(params.get('locale') === 'de' ? 'de' : 'en')
setTheme(params.get('theme') === 'gf-dark' ? 'gf-dark' : 'gf-light')
createApp({ render: () => h('main', { class: 'mx-auto max-w-3xl space-y-6 p-4 text-base-content' }, [
  h('section', { 'data-test': 'author' }, [h('h1', { class: 'text-xl font-semibold' }, 'Content management'), h(ManagedContent, { item: { id: 7, contentType: 'post', revisionId: 70, processStatus: 1, title: 'Candidate', content: 'Rejected edit', images: [], hasPublishedVersion: true, reviewReason: 'Prior reason', createdAt: '2026-10-10T00:00:00Z', canAppeal: true, appeal: null } })]),
  h('section', { 'data-test': 'moderator' }, [h('h2', { class: 'text-xl font-semibold' }, 'Moderation appeals'), h(ModerationAppealQueue)]),
]) }).use(i18n).mount('#app')

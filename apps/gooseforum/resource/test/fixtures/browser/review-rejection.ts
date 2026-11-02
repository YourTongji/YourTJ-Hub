import { h } from 'vue'
import { createBrowserApp as createApp } from './create-app'
import { i18n, setLocale } from '../../../src/runtime/i18n'
import { setTheme } from '../../../src/runtime/site-theme'
import { adminText } from '../../../src/admin/runtime/i18n-text'
import { layout } from '../profile'
const params = new URLSearchParams(location.search)
const mode = params.get('mode') || 'moderator'
if (params.get('scale') === '2') document.documentElement.style.fontSize = '200%'
await setLocale((params.get('lang') || 'en') as 'zh' | 'en' | 'ja' | 'de')
setTheme(params.get('theme') === 'dark' ? 'gf-dark' : 'gf-light')
if (mode === 'admin') await import('../../../src/admin/styles/admin.css')
else await import('../../../src/styles/resource.css')
const ReviewQueueManagementPage = mode === 'admin' ? (await import('../../../src/admin/pages/management/ReviewQueueManagementPage.vue')).default : undefined
const ModerationReviewQueue = mode === 'moderator' ? (await import('../../../src/site/components/ModerationReviewQueue.vue')).default : undefined
const ModerationActionPage = mode === 'quick' ? (await import('../../../src/site/pages/ModerationActionPage.vue')).default : undefined
const ManagedContent = mode === 'owner' ? (await import('../../../src/site/components/ManagedContent.vue')).default : undefined
const t = i18n.global.t
const reason = 'Please remove the private address. <script>window.reasonExecuted=true</script>'
Object.assign(window, { reviewLabels: { reject: t('moderation.review.reject'), adminReject: adminText('k00gk'), confirm: t('moderation.review.confirmReject'), quickConfirm: t('moderationAction.actions.reject'), cancel: t('common.cancel') } })
const body = mode === 'admin'
  ? h(ReviewQueueManagementPage!, { payload: { component: 'manage-home', props: {}, layout, meta: { title: 'Review' }, url: '/admin/review-queue', version: 'fixture' } })
  : mode === 'quick'
    ? h(ModerationActionPage!, { layout, props: {} as never })
    : mode === 'owner'
      ? h(ManagedContent!, { item: { id: 7, contentType: 'post', revisionId: 19, processStatus: 1, title: 'Candidate', content: 'Private candidate', reviewReason: reason, createdAt: '', canAppeal: true, appeal: null } })
      : h(ModerationReviewQueue!)
createApp({ render: () => h('main', { class: 'mx-auto min-w-0 max-w-3xl p-4' }, [body]) }).use(i18n).mount('#app')

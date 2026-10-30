<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Ban, ExternalLink, Loader2, ShieldCheck } from '@lucide/vue'
import { executeModerationApprovalAction, previewModerationApprovalAction } from '@/runtime/api'
import { formatDateTime } from '@/runtime/format'
import RejectionReasonFields from '@/site/components/RejectionReasonFields.vue'
import { emptyRejectionDraft, rejectionReason, rejectionError } from '@/runtime/review-rejection'
import PageHeader from '@/site/components/PageHeader.vue'
import type { LayoutPayload, ModerationActionPageProps, ModerationApprovalActionView } from '@gooseforum/client'

// 快捷审批确认页（issue #1049）：通知卡片按钮只打开本页，不直接执行。页面先 preview
// 展示目标与当前状态，版主显式确认后才 execute；两者都由服务端按当前登录账号复核权限。
defineProps<{
  layout: LayoutPayload
  props: ModerationActionPageProps
}>()

const { t } = useI18n()
const token = ref('')
const view = ref<ModerationApprovalActionView | null>(null)
const loading = ref(true)
const executing = ref(false)
const error = ref('')

const reason = ref('')
const rejectDraft = ref(emptyRejectionDraft())
const contentReject = computed(() => ['review.topic', 'review.post'].includes(view.value?.subject ?? '') && view.value?.action === 'reject')
const reasonError = ref('')
const appeal = computed(() => view.value?.subject === 'appeal')
const destructive = computed(() => ['ban', 'hide', 'reject'].includes(view.value?.action ?? ''))
// 申诉的通过/驳回针对申诉本身，文案与待审内容区分；驳回理由会给作者看，必须填写。
const actionLabel = computed(() => view.value?.action ? t(`moderationAction.${appeal.value ? 'appealActions' : 'actions'}.${view.value.action}`) : '')
const actionHint = computed(() => view.value?.action ? t(`moderationAction.${appeal.value ? 'appealActionHints' : 'actionHints'}.${view.value.action}`) : '')
const reasonRequired = computed(() => appeal.value && view.value?.action === 'reject')
const stateText = computed(() => {
  if (!view.value) return ''
  if (appeal.value && view.value.state === 'changed') return t('moderationAction.appealChanged')
  return t(`moderationAction.states.${view.value.state}`)
})
const targetTypeKey = computed(() => {
  switch (view.value?.targetType) {
    case 'chat_message':
      return 'chatMessage'
    case 'course_review':
      return 'courseReview'
    default:
      return view.value?.targetType ?? ''
  }
})
const stateTone = computed(() => {
  switch (view.value?.state) {
    case 'ready':
      return 'border-primary/30 bg-primary/10'
    case 'done':
      return 'border-success/25 bg-success/10'
    default:
      return 'border-warning/25 bg-warning/10'
  }
})

onMounted(async () => {
  token.value = new URLSearchParams(window.location.search).get('token') ?? ''
  if (!token.value) {
    view.value = { state: 'invalid', workbenchUrl: '/moderation' }
    loading.value = false
    return
  }
  await refreshPreview()
})

async function refreshPreview() {
  if (executing.value) return
  loading.value = true
  error.value = ''
  try {
    view.value = await previewModerationApprovalAction(token.value)
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('common.loadFailed')
  } finally {
    loading.value = false
  }
}

async function confirmAction() {
  if (executing.value || view.value?.state !== 'ready') return
  const explanation = contentReject.value ? rejectionReason(t, rejectDraft.value) : appeal.value ? reason.value.trim() : ''
  if (contentReject.value) {
    reasonError.value = rejectionError(t, explanation)
    if (reasonError.value) { document.getElementById('moderation-content-rejection-preset')?.focus(); return }
  }
  if (reasonRequired.value && !reason.value.trim()) {
    reasonError.value = t('moderationAction.appealReasonRequired')
    document.getElementById('moderation-action-reason')?.focus()
    return
  }
  executing.value = true
  error.value = ''
  reasonError.value = ''
  try {
    view.value = await executeModerationApprovalAction(token.value, explanation)
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('api.moderationActionFailed')
  } finally {
    executing.value = false
  }
}
</script>

<template>
  <main class="mx-auto min-w-0 max-w-2xl pb-8">
    <PageHeader :title="t('moderationAction.title')" :description="t('moderationAction.description')" compact />

    <section class="gf-card space-y-4 p-5">
      <div v-if="loading" class="flex items-center gap-2 text-sm text-base-content/60">
        <Loader2 class="h-4 w-4 animate-spin" />
        {{ t('moderationAction.loading') }}
      </div>

      <template v-else>
        <p v-if="error" class="rounded-lg border border-error/25 bg-error/10 px-3 py-2 text-sm text-error" role="alert">{{ error }}</p>

        <template v-if="view">
          <div class="rounded-lg border px-3 py-2 text-sm" :class="stateTone" role="status">
            {{ stateText }}
          </div>

          <dl v-if="view.action" class="grid gap-3 text-sm sm:grid-cols-[7rem_minmax(0,1fr)]">
            <dt class="text-base-content/60">{{ t('moderationAction.actionLabel') }}</dt>
            <dd>
              <div class="font-medium">{{ actionLabel }}</div>
              <div class="text-base-content/60">{{ actionHint }}</div>
            </dd>
            <template v-if="targetTypeKey">
              <dt class="text-base-content/60">{{ t('moderationAction.targetLabel') }}</dt>
              <dd>
                <div class="text-base-content/60">{{ t(`moderationAction.targetTypes.${targetTypeKey}`) }}</div>
                <div v-if="view.title" class="font-medium break-words">{{ view.title }}</div>
                <p v-if="view.excerpt" class="mt-1 whitespace-pre-line break-words text-base-content/80">{{ view.excerpt }}</p>
                <div v-if="view.anonymous" class="mt-1 text-xs text-base-content/60">{{ t('moderationAction.anonymous') }}</div>
              </dd>
            </template>
            <template v-if="view.originalReason">
              <dt class="text-base-content/60">{{ t('moderationAction.originalReasonLabel') }}</dt>
              <dd class="whitespace-pre-wrap break-words">{{ view.originalReason }}</dd>
            </template>
            <template v-if="view.appealReason">
              <dt class="text-base-content/60">{{ t('moderationAction.appealReasonLabel') }}</dt>
              <dd class="whitespace-pre-wrap break-words rounded-field bg-info/10 px-3 py-2">{{ view.appealReason }}</dd>
            </template>
            <template v-if="view.expiresAt && view.state === 'ready'">
              <dt class="text-base-content/60">{{ t('moderationAction.expiresAt') }}</dt>
              <dd>{{ formatDateTime(view.expiresAt) }}</dd>
            </template>
          </dl>

          <RejectionReasonFields v-if="contentReject && (view.state === 'ready' || view.state === 'failed')" v-model="rejectDraft" id-prefix="moderation-content-rejection" :disabled="executing || view.state !== 'ready'" :error="reasonError" />
          <div v-if="appeal && (view.state === 'ready' || view.state === 'failed')" class="space-y-1.5">
            <label for="moderation-action-reason" class="text-sm font-medium">
              {{ t(reasonRequired ? 'moderationAction.appealRejectReasonLabel' : 'moderationAction.appealApproveReasonLabel') }}
            </label>
            <textarea
              id="moderation-action-reason"
              v-model="reason"
              :disabled="view.state !== 'ready'"
              rows="3"
              maxlength="512"
              class="gf-textarea w-full text-base sm:text-sm"
              :placeholder="t('moderationAction.appealReasonPlaceholder')"
              :aria-invalid="reasonError ? 'true' : undefined"
              aria-describedby="moderation-action-reason-help"
              @input="reasonError = ''"
            />
            <div id="moderation-action-reason-help" class="flex items-start justify-between gap-3 text-xs">
              <span :class="reasonError ? 'text-error' : 'text-base-content/55'">{{ reasonError || t('moderationAction.appealReasonHint') }}</span>
              <span class="shrink-0 tabular-nums text-base-content/45">{{ reason.length }}/512</span>
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-2 pt-2">
            <button v-if="view.state === 'failed'" data-testid="moderation-recheck" type="button" class="gf-button gf-button-md h-auto max-w-full whitespace-normal py-2 text-center [overflow-wrap:anywhere] gf-button-secondary" @click="refreshPreview">{{ t('common.retry') }}</button>
            <button
              v-if="view.state === 'ready'"
              type="button"
              class="gf-button gf-button-md h-auto max-w-full whitespace-normal py-2 text-center [overflow-wrap:anywhere]"
              :class="destructive ? 'gf-button-danger' : 'gf-button-primary'"
              :disabled="executing"
              @click="confirmAction"
            >
              <Loader2 v-if="executing" class="h-4 w-4 animate-spin" />
              <component :is="destructive ? Ban : ShieldCheck" v-else class="h-4 w-4" />
              {{ executing ? t('moderationAction.executing') : actionLabel }}
            </button>
            <a v-if="view.targetUrl" :href="view.targetUrl" class="gf-button gf-button-md h-auto max-w-full whitespace-normal py-2 text-center [overflow-wrap:anywhere] gf-button-secondary" target="_blank" rel="noopener noreferrer">
              <ExternalLink class="h-4 w-4" />
              {{ t('moderationAction.viewTarget') }}
            </a>
            <a :href="view.workbenchUrl || '/moderation'" class="gf-button gf-button-md h-auto max-w-full whitespace-normal py-2 text-center [overflow-wrap:anywhere] gf-button-ghost">
              {{ t('moderationAction.openWorkbench') }}
            </a>
          </div>
        </template>
        <!-- 预览失败（未登录、网络错误等）时仍给出去处，从通知跳进来的版主不会困在本页 -->
        <div v-else class="flex flex-wrap items-center gap-2 pt-2">
          <a href="/moderation" class="gf-button gf-button-md h-auto max-w-full whitespace-normal py-2 text-center [overflow-wrap:anywhere] gf-button-ghost">
            {{ t('moderationAction.openWorkbench') }}
          </a>
        </div>
      </template>
    </section>
  </main>
</template>

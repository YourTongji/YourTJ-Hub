<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Ban, ExternalLink, Loader2, ShieldCheck } from '@lucide/vue'
import { executeModerationApprovalAction, previewModerationApprovalAction } from '@/runtime/api'
import { formatDateTime } from '@/runtime/format'
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

const destructive = computed(() => ['ban', 'hide', 'reject'].includes(view.value?.action ?? ''))
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
  try {
    view.value = await previewModerationApprovalAction(token.value)
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('common.loadFailed')
  } finally {
    loading.value = false
  }
})

async function confirmAction() {
  if (executing.value || view.value?.state !== 'ready') return
  executing.value = true
  error.value = ''
  try {
    view.value = await executeModerationApprovalAction(token.value)
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
            {{ t(`moderationAction.states.${view.state}`) }}
          </div>

          <dl v-if="view.action" class="grid gap-3 text-sm sm:grid-cols-[7rem_minmax(0,1fr)]">
            <dt class="text-base-content/60">{{ t('moderationAction.actionLabel') }}</dt>
            <dd>
              <div class="font-medium">{{ t(`moderationAction.actions.${view.action}`) }}</div>
              <div class="text-base-content/60">{{ t(`moderationAction.actionHints.${view.action}`) }}</div>
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
            <template v-if="view.expiresAt && view.state === 'ready'">
              <dt class="text-base-content/60">{{ t('moderationAction.expiresAt') }}</dt>
              <dd>{{ formatDateTime(view.expiresAt) }}</dd>
            </template>
          </dl>

          <div class="flex flex-wrap items-center gap-2 pt-2">
            <button
              v-if="view.state === 'ready'"
              type="button"
              class="gf-button gf-button-md"
              :class="destructive ? 'gf-button-danger' : 'gf-button-primary'"
              :disabled="executing"
              @click="confirmAction"
            >
              <Loader2 v-if="executing" class="h-4 w-4 animate-spin" />
              <component :is="destructive ? Ban : ShieldCheck" v-else class="h-4 w-4" />
              {{ executing ? t('moderationAction.executing') : t(`moderationAction.actions.${view.action}`) }}
            </button>
            <a v-if="view.targetUrl" :href="view.targetUrl" class="gf-button gf-button-md gf-button-secondary" target="_blank" rel="noopener noreferrer">
              <ExternalLink class="h-4 w-4" />
              {{ t('moderationAction.viewTarget') }}
            </a>
            <a :href="view.workbenchUrl || '/moderation'" class="gf-button gf-button-md gf-button-ghost">
              {{ t('moderationAction.openWorkbench') }}
            </a>
          </div>
        </template>
      </template>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { CheckCircle2, CircleSlash, Clock3, Loader2, XCircle } from '@lucide/vue'
import { ApiResponseError, submitContentAppeal, type MyContentItem, type OwnerAppeal } from '@/runtime/api'
import { formatDateTime } from '@/runtime/format'

// 内容管理里被拒条目的申诉区（#1114）：展示最新版本的申诉进度，并在可申诉时内联展开表单。
// 是否可申诉、已有申诉都来自列表接口（canAppeal / appeal），不再逐条请求历史。
const MAX = 512
const props = defineProps<{ item: MyContentItem, open: boolean }>()
const emit = defineEmits<{ 'update:open': [boolean], saved: [] }>()
const { t } = useI18n()
const reason = ref('')
const busy = ref(false)
const error = ref('')
const announce = ref('')
// 提交成功到列表刷新回来之间，先按“复核中”展示，避免状态闪回。
const submitted = ref<OwnerAppeal | null>(null)
const field = ref<HTMLTextAreaElement | null>(null)
const appeal = computed(() => props.item.appeal ?? submitted.value)
const length = computed(() => [...reason.value].length)
const formOpen = computed(() => props.open && !appeal.value && !!props.item.canAppeal)
const tone = computed(() => ({
  pending: { icon: Clock3, color: 'text-warning' },
  approved: { icon: CheckCircle2, color: 'text-success' },
  rejected: { icon: XCircle, color: 'text-error' },
  stale: { icon: CircleSlash, color: 'text-base-content/55' },
}[appeal.value?.status ?? 'pending']))

watch(() => [props.item.id, props.item.revisionId], () => {
  submitted.value = null
  reason.value = ''
  error.value = ''
})
watch(formOpen, async (value) => {
  if (!value) return
  await nextTick()
  field.value?.focus()
})

function close() {
  if (busy.value) return
  error.value = ''
  emit('update:open', false)
}

async function submit() {
  const explanation = reason.value.trim()
  if (busy.value || !explanation || length.value > MAX) return
  const { contentType, id, revisionId } = props.item
  busy.value = true
  error.value = ''
  try {
    await submitContentAppeal(contentType, id, revisionId, explanation)
    submitted.value = { status: 'pending', reason: explanation, reviewReason: '', createdAt: new Date().toISOString(), reviewedAt: null }
    reason.value = ''
    announce.value = t('appeals.submitted')
    emit('update:open', false)
    emit('saved')
  } catch (err) {
    // 业务失败给出服务端的具体原因；网络错误统一提示重试，理由保留在输入框里。
    error.value = err instanceof ApiResponseError ? err.message : t('appeals.submitFailed')
    if (err instanceof ApiResponseError && err.messageCode === 'content.appeal.duplicate') emit('saved')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="contents">
    <span class="sr-only" role="status" aria-live="polite">{{ announce }}</span>

    <div v-if="appeal" class="mt-3 flex gap-2.5 rounded-field bg-base-100 p-3 shadow-[0_0_0_1px_var(--gf-color-line)]" data-test="appeal-state">
      <component :is="tone.icon" class="mt-0.5 h-4 w-4 shrink-0" :class="tone.color" aria-hidden="true" />
      <div class="min-w-0 flex-1 space-y-1.5">
        <p class="text-sm font-semibold leading-5 text-base-content">{{ t(`appeals.state.${appeal.status}.title`) }}</p>
        <p class="text-[13px] leading-5 text-base-content/65 [text-wrap:pretty]">{{ t(`appeals.state.${appeal.status}.body`) }}</p>
        <div v-if="appeal.status === 'rejected'" class="rounded-field bg-base-200/70 px-2.5 py-2">
          <p class="text-xs font-medium text-base-content/55">{{ t('appeals.reviewerNote') }}</p>
          <p class="mt-0.5 whitespace-pre-wrap break-words text-[13px] leading-5 text-base-content">{{ appeal.reviewReason || t('appeals.noReviewerNote') }}</p>
        </div>
        <details class="group text-[13px] leading-5">
          <summary class="inline-flex cursor-pointer list-none items-center gap-1 rounded-sm text-xs font-medium text-base-content/55 hover:text-base-content focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary [&::-webkit-details-marker]:hidden">
            {{ t('appeals.yourReason') }}
            <span class="inline-block transition-transform duration-150 ease-out group-open:rotate-90 motion-reduce:transition-none" aria-hidden="true">›</span>
          </summary>
          <p class="mt-1 whitespace-pre-wrap break-words text-base-content/75">{{ appeal.reason }}</p>
        </details>
        <p class="text-xs tabular-nums text-base-content/45">
          {{ t('appeals.submittedAt', { time: formatDateTime(appeal.createdAt) }) }}<template v-if="appeal.reviewedAt"> · {{ t('appeals.decidedAt', { time: formatDateTime(appeal.reviewedAt) }) }}</template>
        </p>
      </div>
    </div>

    <!-- 高度用 grid 0fr↔1fr 过渡；收起动画结束后卸载表单。 -->
    <Transition
      enter-active-class="transition-[grid-template-rows,opacity] duration-200 ease-out motion-reduce:transition-none"
      enter-from-class="grid-rows-[0fr] opacity-0"
      enter-to-class="grid-rows-[1fr]"
      leave-active-class="transition-[grid-template-rows,opacity] duration-150 ease-out motion-reduce:transition-none"
      leave-from-class="grid-rows-[1fr]"
      leave-to-class="grid-rows-[0fr] opacity-0"
    >
    <div v-if="formOpen" class="grid">
      <div class="min-h-0 overflow-hidden">
        <form class="mt-3 rounded-field bg-base-100 p-3 shadow-[0_0_0_1px_var(--gf-color-line)]" data-test="appeal-form" @submit.prevent="submit" @keydown.esc.prevent="close">
          <p class="text-sm font-semibold text-base-content">{{ t('appeals.formTitle') }}</p>
          <p class="mt-1 max-w-prose text-[13px] leading-5 text-base-content/60 [text-wrap:pretty]">{{ t('appeals.formIntro') }}</p>
          <label :for="`appeal-${item.contentType}-${item.id}`" class="mt-3 block text-xs font-semibold text-base-content/70">{{ t('appeals.reasonLabel') }}</label>
          <textarea
            :id="`appeal-${item.contentType}-${item.id}`"
            ref="field"
            v-model="reason"
            rows="3"
            :maxlength="MAX"
            :disabled="busy"
            :placeholder="t('appeals.reasonPlaceholder')"
            :aria-invalid="!!error"
            :aria-describedby="`appeal-help-${item.contentType}-${item.id}`"
            class="gf-input mt-1 block min-h-24 w-full resize-y py-2 text-base leading-6 sm:text-sm"
          />
          <div :id="`appeal-help-${item.contentType}-${item.id}`" class="mt-1.5 flex items-start justify-between gap-3 text-xs text-base-content/50">
            <span>{{ t('appeals.oncePerVersion') }}</span>
            <span class="shrink-0 tabular-nums" :class="length > MAX - 32 ? 'text-warning' : ''">{{ length }}/{{ MAX }}</span>
          </div>
          <p v-if="error" role="alert" class="gf-status-message gf-status-message-error mt-2">{{ error }}</p>
          <div class="mt-3 flex flex-wrap justify-end gap-2">
            <button type="button" class="gf-button gf-button-sm gf-button-ghost" :disabled="busy" @click="close">{{ t('appeals.cancel') }}</button>
            <button type="submit" class="gf-button gf-button-sm gf-button-primary active:scale-[0.96]" :disabled="busy || !reason.trim()">
              <Loader2 v-if="busy" class="h-4 w-4 animate-spin" aria-hidden="true" />
              {{ busy ? t('appeals.submitting') : t('appeals.submit') }}
            </button>
          </div>
        </form>
      </div>
    </div>
    </Transition>
  </div>
</template>

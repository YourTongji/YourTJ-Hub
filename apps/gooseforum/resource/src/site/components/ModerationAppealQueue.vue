<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, CheckCircle2, CircleSlash, ExternalLink, Loader2, RotateCcw, Scale, X, XCircle } from '@lucide/vue'
import { ApiResponseError, fetchModerationAppeals, moderationAppealAction, type AppealItem, type AppealStatus } from '@/runtime/api'
import { formatDateTime } from '@/runtime/format'
import MarkdownImageViewer from '@/components/MarkdownImageViewer.vue'
import EmptyState from '@/site/components/EmptyState.vue'

// 版主工作台「申诉」队列（#1114）：左右对照原审核说明与作者理由，通过即公开被申诉的那个版本，
// 驳回须写说明。作者身份不下发；处理结果在卡片原位反馈，再从待处理列表移除。
const MAX = 512
const PAGE_SIZE = 20
const emit = defineEmits<{ changed: [], count: [number] }>()
const { t } = useI18n()
const statuses: AppealStatus[] = ['pending', 'approved', 'rejected', 'stale']
const status = ref<AppealStatus>('pending')
const items = ref<AppealItem[]>([])
const page = ref(0)
const total = ref(0)
const pendingTotal = ref<number | null>(null)
const loading = ref(false)
const error = ref('')
const notice = ref('')
const busyIds = ref<number[]>([])
const reasons = ref<Record<number, string>>({})
const fieldErrors = ref<Record<number, string>>({})
const expanded = ref<number[]>([])
const imageViewer = ref<InstanceType<typeof MarkdownImageViewer> | null>(null)
const hasMore = computed(() => page.value * PAGE_SIZE < total.value)
let request = 0
let noticeTimer = 0

const outcome = {
  approved: { icon: CheckCircle2, color: 'text-success', badge: 'gf-badge-success' },
  rejected: { icon: XCircle, color: 'text-error', badge: 'gf-badge-error' },
  stale: { icon: CircleSlash, color: 'text-base-content/55', badge: 'gf-badge-muted' },
  pending: { icon: Scale, color: 'text-warning', badge: 'gf-badge-warning' },
} as const

function topicURL(item: AppealItem) {
  return item.kind === 'post' ? `/p/post/${item.topicId}/${item.postNo}` : `/p/post/${item.topicId}`
}

function setPendingTotal(value: number) {
  pendingTotal.value = value
  emit('count', value)
}

async function load(reset = false) {
  const version = ++request
  const selected = status.value
  loading.value = true
  error.value = ''
  try {
    const pages = reset ? Array.from({ length: Math.max(page.value, 1) }, (_, index) => index + 1) : [page.value + 1]
    const results = await Promise.all(pages.map(value => fetchModerationAppeals(selected, value, PAGE_SIZE)))
    if (version !== request) return
    const result = results[results.length - 1]!
    const seen = new Set(reset ? [] : items.value.map(item => item.id))
    const incoming = results.flatMap(value => value.items).filter(item => !seen.has(item.id) && seen.add(item.id))
    items.value = [...(reset ? [] : items.value), ...incoming]
    page.value = result.page
    total.value = result.total
    if (selected === 'pending') setPendingTotal(result.total)
  } catch (err) {
    if (version === request) error.value = err instanceof ApiResponseError ? err.message : t('appeals.queue.loadFailed')
  } finally {
    if (version === request) loading.value = false
  }
}

function switchStatus(value: AppealStatus) {
  if (value === status.value) return
  status.value = value
  notice.value = ''
  items.value = []
  page.value = 0
  total.value = 0
  void load(true)
}

function showNotice(message: string) {
  notice.value = message
  window.clearTimeout(noticeTimer)
  noticeTimer = window.setTimeout(() => { notice.value = '' }, 5000)
}

function removeItem(id: number) {
  items.value = items.value.filter(item => item.id !== id)
  total.value = Math.max(0, total.value - 1)
  if (status.value === 'pending') setPendingTotal(total.value)
  delete reasons.value[id]
  delete fieldErrors.value[id]
}

function toggleExpanded(id: number) {
  expanded.value = expanded.value.includes(id) ? expanded.value.filter(value => value !== id) : [...expanded.value, id]
}

function preview(item: AppealItem, index: number, event: MouseEvent) {
  (event.currentTarget as HTMLButtonElement).focus()
  imageViewer.value?.open(item.images.map(src => ({ src, alt: '' })), index)
}

async function review(item: AppealItem, approve: boolean) {
  if (busyIds.value.includes(item.id)) return
  const reason = (reasons.value[item.id] ?? '').trim()
  if (!approve && !reason) {
    // 驳回理由是作者唯一能看到的解释：缺失时就地提示并聚焦输入框，而不是禁用按钮让人猜。
    fieldErrors.value = { ...fieldErrors.value, [item.id]: t('appeals.queue.rejectNeedsReason') }
    await nextTick()
    document.getElementById(`appeal-decision-${item.id}`)?.focus()
    return
  }
  busyIds.value = [...busyIds.value, item.id]
  fieldErrors.value = { ...fieldErrors.value, [item.id]: '' }
  error.value = ''
  try {
    await moderationAppealAction(item.id, approve, reason)
    removeItem(item.id)
    showNotice(approve ? t('appeals.queue.approvedToast') : t('appeals.queue.rejectedToast'))
    emit('changed')
  } catch (err) {
    const code = err instanceof ApiResponseError ? err.messageCode : ''
    const message = err instanceof ApiResponseError ? err.message : t('appeals.queue.actionFailed')
    if (code === 'moderation.appeal.stale' || code === 'moderation.appeal.processed' || code === 'moderation.appeal.notFound') {
      // 目标已变或别人已处理：不是你的错，移出列表并说明原因，说明草稿无需保留。
      removeItem(item.id)
      showNotice(message)
      emit('changed')
    } else {
      fieldErrors.value = { ...fieldErrors.value, [item.id]: message }
    }
  } finally {
    busyIds.value = busyIds.value.filter(id => id !== item.id)
  }
}

onMounted(() => { void load(true) })
onUnmounted(() => {
  request++
  window.clearTimeout(noticeTimer)
})
</script>

<template>
  <section class="space-y-3" data-test="moderation-appeal-queue">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="inline-flex flex-wrap gap-1 rounded-field bg-base-200/70 p-1" role="group" :aria-label="t('moderation.managementTabs.appeals')">
        <button
          v-for="value in statuses"
          :key="value"
          type="button"
          class="gf-tab h-7 gap-1.5 px-2.5"
          :aria-pressed="status === value"
          :class="status === value ? 'bg-base-100 text-base-content shadow-[0_1px_2px_oklch(0_0_0/0.08),0_0_0_1px_var(--gf-color-line)]' : 'text-base-content/55 hover:text-base-content'"
          @click="switchStatus(value)"
        >
          {{ t(`appeals.queue.tabs.${value}`) }}
          <span v-if="value === 'pending' && pendingTotal" class="min-w-5 rounded-full bg-warning/15 px-1.5 text-center text-xs font-semibold tabular-nums text-warning">{{ pendingTotal }}</span>
        </button>
      </div>
      <button type="button" class="gf-button gf-button-sm gf-button-ghost" :disabled="loading" @click="load(true)">
        <RotateCcw class="h-4 w-4" :class="loading ? 'motion-safe:animate-spin' : ''" aria-hidden="true" />{{ t('appeals.queue.refresh') }}
      </button>
    </div>

    <p class="sr-only" role="status" aria-live="polite">{{ notice }}</p>
    <Transition enter-from-class="opacity-0 -translate-y-1" leave-to-class="opacity-0 -translate-y-1" enter-active-class="transition duration-200 ease-out motion-reduce:transition-none" leave-active-class="transition duration-150 ease-out motion-reduce:transition-none">
      <p v-if="notice" class="flex items-center gap-2 rounded-field bg-base-100 px-3 py-2 text-sm text-base-content shadow-[0_0_0_1px_var(--gf-color-line)]" aria-hidden="true">
        <Check class="h-4 w-4 shrink-0 text-success" aria-hidden="true" />{{ notice }}
      </p>
    </Transition>
    <p v-if="error" role="alert" class="gf-status-message gf-status-message-error">{{ error }}</p>

    <TransitionGroup tag="div" class="space-y-3" leave-active-class="transition duration-200 ease-out motion-reduce:transition-none" leave-to-class="opacity-0 translate-y-1">
      <article
        v-for="item in items"
        :key="item.id"
        class="gf-card overflow-hidden"
        :data-test="`appeal-card-${item.id}`"
      >
        <header class="flex flex-wrap items-center gap-x-2 gap-y-1 px-4 pt-3.5">
          <span class="gf-badge gf-badge-muted">{{ item.kind === 'post' ? t('appeals.queue.reply', { postNo: item.postNo }) : t('appeals.queue.topic') }}</span>
          <a :href="topicURL(item)" target="_blank" rel="noopener" class="group inline-flex min-w-0 max-w-full items-center gap-1 text-[15px] font-semibold leading-6 text-base-content hover:text-primary" :title="t('appeals.queue.viewTopic')">
            <span class="truncate">{{ item.title || t('moderation.review.untitled') }}</span>
            <ExternalLink class="h-3.5 w-3.5 shrink-0 opacity-0 transition-opacity group-hover:opacity-60 group-focus-visible:opacity-60" aria-hidden="true" />
          </a>
          <span v-if="item.status !== 'pending'" class="gf-badge" :class="outcome[item.status].badge">
            <component :is="outcome[item.status].icon" class="h-3.5 w-3.5" aria-hidden="true" />{{ t(`appeals.queue.tabs.${item.status}`) }}
          </span>
          <time class="ms-auto text-xs tabular-nums text-base-content/50">{{ t('appeals.submittedAt', { time: formatDateTime(item.createdAt) }) }}</time>
        </header>

        <div class="space-y-4 px-4 pb-4 pt-3">
          <section>
            <h3 class="text-xs font-medium text-base-content/55">{{ t('appeals.queue.rejectedContent') }}</h3>
            <div class="mt-1.5 rounded-field bg-base-200/60 px-3 py-2.5">
              <p class="whitespace-pre-wrap break-words text-sm leading-6 text-base-content/85" :class="expanded.includes(item.id) ? '' : 'line-clamp-4'">{{ item.content || t('moderation.review.noExcerpt') }}</p>
              <button v-if="item.content.length > 160 || item.content.split('\n').length > 4" type="button" class="mt-1 text-xs font-medium text-primary hover:underline" :aria-expanded="expanded.includes(item.id)" @click="toggleExpanded(item.id)">
                {{ expanded.includes(item.id) ? t('appeals.queue.collapse') : t('appeals.queue.expand') }}
              </button>
              <div v-if="item.images.length" class="mt-2 flex flex-wrap gap-1.5">
                <button v-for="(url, index) in item.images" :key="index" type="button" :aria-label="`${t('common.preview')} ${index + 1}`" class="rounded-field focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary" @click="preview(item, index, $event)">
                  <img :src="url" alt="" loading="lazy" class="gf-image-outline h-14 w-14 rounded-field object-cover">
                </button>
              </div>
            </div>
          </section>

          <div class="grid gap-3 sm:grid-cols-2">
            <section class="rounded-field px-3 py-2.5 shadow-[0_0_0_1px_var(--gf-color-line)]">
              <h3 class="text-xs font-medium text-base-content/55">{{ t('appeals.queue.originalReason') }}</h3>
              <p class="mt-1 whitespace-pre-wrap break-words text-sm leading-6" :class="item.originalReviewReason ? 'text-base-content' : 'text-base-content/50'">{{ item.originalReviewReason || t('appeals.queue.noOriginalReason') }}</p>
            </section>
            <section class="rounded-field bg-info/10 px-3 py-2.5">
              <h3 class="text-xs font-medium text-primary/80">{{ t('appeals.queue.authorReason') }}</h3>
              <p class="mt-1 whitespace-pre-wrap break-words text-sm leading-6 text-base-content">{{ item.reason }}</p>
            </section>
          </div>

          <section v-if="item.status !== 'pending' && (item.reviewReason || item.reviewedAt)" class="flex gap-2.5 text-sm">
            <component :is="outcome[item.status].icon" class="mt-1 h-4 w-4 shrink-0" :class="outcome[item.status].color" aria-hidden="true" />
            <div class="min-w-0">
              <template v-if="item.reviewReason">
                <p class="text-xs font-medium text-base-content/55">{{ t('appeals.queue.decisionLabel') }}</p>
                <p class="whitespace-pre-wrap break-words leading-6 text-base-content">{{ item.reviewReason }}</p>
              </template>
              <p v-if="item.reviewedAt" class="text-xs tabular-nums text-base-content/50">{{ t('appeals.decidedAt', { time: formatDateTime(item.reviewedAt) }) }}</p>
            </div>
          </section>

          <form v-if="item.status === 'pending'" class="space-y-2" @submit.prevent>
            <div class="flex items-baseline justify-between gap-3">
              <label :for="`appeal-decision-${item.id}`" class="text-xs font-semibold text-base-content/70">{{ t('appeals.queue.decisionLabel') }}</label>
              <span class="text-xs tabular-nums text-base-content/45">{{ [...(reasons[item.id] ?? '')].length }}/{{ MAX }}</span>
            </div>
            <textarea
              :id="`appeal-decision-${item.id}`"
              v-model="reasons[item.id]"
              rows="2"
              :maxlength="MAX"
              :disabled="busyIds.includes(item.id)"
              :placeholder="t('appeals.queue.decisionPlaceholder')"
              :aria-invalid="!!fieldErrors[item.id]"
              :aria-describedby="`appeal-decision-help-${item.id}`"
              class="gf-input block min-h-20 w-full resize-y py-2 text-base leading-6 sm:text-sm"
              @input="fieldErrors[item.id] = ''"
            />
            <p :id="`appeal-decision-help-${item.id}`" class="text-xs leading-5" :class="fieldErrors[item.id] ? 'font-medium text-error' : 'text-base-content/50'" :role="fieldErrors[item.id] ? 'alert' : undefined">
              {{ fieldErrors[item.id] || t('appeals.queue.decisionHint') }}
            </p>
            <div class="flex flex-wrap justify-end gap-2 pt-1">
              <button type="button" class="gf-button gf-button-sm gf-button-secondary active:scale-[0.96]" data-test="appeal-reject" :disabled="busyIds.includes(item.id)" @click="review(item, false)">
                <X class="h-4 w-4" aria-hidden="true" />{{ t('appeals.queue.reject') }}
              </button>
              <button type="button" class="gf-button gf-button-sm gf-button-primary active:scale-[0.96]" data-test="appeal-approve" :disabled="busyIds.includes(item.id)" @click="review(item, true)">
                <Loader2 v-if="busyIds.includes(item.id)" class="h-4 w-4 animate-spin" aria-hidden="true" /><Check v-else class="h-4 w-4" aria-hidden="true" />{{ t('appeals.queue.approve') }}
              </button>
            </div>
          </form>
        </div>
      </article>
    </TransitionGroup>

    <div v-if="!items.length && !loading" class="gf-card">
      <EmptyState :icon="status === 'pending' ? Check : outcome[status].icon" :title="t(`appeals.queue.empty.${status}.title`)" :description="t(`appeals.queue.empty.${status}.body`)" />
    </div>
    <div v-if="loading && !items.length" class="gf-card"><EmptyState loading :title="t('common.loadingShort')" /></div>
    <div v-else-if="hasMore" class="text-center">
      <button type="button" class="gf-button gf-button-sm gf-button-ghost" :disabled="loading" @click="load()">
        <Loader2 v-if="loading" class="h-4 w-4 animate-spin" aria-hidden="true" />{{ t('appeals.queue.loadMore') }}
      </button>
    </div>
    <p class="max-w-prose text-xs leading-5 text-base-content/55 [text-wrap:pretty]">{{ t('appeals.queue.hint') }}</p>
    <MarkdownImageViewer ref="imageViewer" />
  </section>
</template>

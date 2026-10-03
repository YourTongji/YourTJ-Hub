<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, Loader2, ShieldAlert, X } from '@lucide/vue'
import { userDisplayName } from '@/runtime/private-notes'
import { fetchModerationReviewQueue, moderationReviewAction } from '@/runtime/api'
import { formatDateTime } from '@/runtime/format'
import { reasonText } from '@/admin/utils/aiModerationReasons'
import EmptyState from '@/site/components/EmptyState.vue'
import type { ReviewQueueItem } from '@/admin/types'

// 前台版主工作台的待审队列（issue #975）：版主不必进入管理后台即可审核
// 管辖分类内的待审话题与回复；通过后内容与图片公开并通知作者，拒绝后保持不公开。
const emit = defineEmits<{ changed: [] }>()
const { t } = useI18n()

type Kind = 'topic' | 'post'
const kind = ref<Kind>('topic')
const items = ref<ReviewQueueItem[]>([])
const page = ref(1)
const total = ref(0)
const loading = ref(false)
const error = ref('')
const busyIds = ref<number[]>([])
const confirmRejectId = ref(0)
const hasMore = computed(() => items.value.length < total.value)
let confirmTimer = 0

async function load(reset = false) {
  if (loading.value) return
  loading.value = true
  error.value = ''
  try {
    const result = await fetchModerationReviewQueue(kind.value, reset ? 1 : page.value + 1, 20)
    const seen = new Set(reset ? [] : items.value.map(item => item.id))
    items.value = reset ? result.items : [...items.value, ...result.items.filter(item => !seen.has(item.id))]
    page.value = result.page
    total.value = result.total
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('moderation.review.loadFailed')
  } finally {
    loading.value = false
  }
}

function switchKind(next: Kind) {
  if (kind.value === next) return
  kind.value = next
  items.value = []
  total.value = 0
  void load(true)
}

function targetURL(item: ReviewQueueItem) {
  return item.postNo ? `/p/post/${item.topicId}/${item.postNo}` : `/p/post/${item.id}`
}

async function review(item: ReviewQueueItem, approve: boolean) {
  if (busyIds.value.includes(item.id)) return
  // 拒绝是不可撤回的结论：第一次点击进入确认态，再次点击才执行。
  if (!approve && confirmRejectId.value !== item.id) {
    confirmRejectId.value = item.id
    window.clearTimeout(confirmTimer)
    confirmTimer = window.setTimeout(() => { confirmRejectId.value = 0 }, 4000)
    return
  }
  confirmRejectId.value = 0
  busyIds.value = [...busyIds.value, item.id]
  error.value = ''
  try {
    await moderationReviewAction(kind.value, item.id, approve)
    items.value = items.value.filter(entry => entry.id !== item.id)
    total.value = Math.max(total.value - 1, 0)
    emit('changed')
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('moderation.review.actionFailed')
  } finally {
    busyIds.value = busyIds.value.filter(id => id !== item.id)
  }
}

onMounted(() => { void load(true) })
</script>

<template>
  <section class="space-y-3" data-test="moderation-review-queue">
    <p v-if="error" class="rounded border border-error/25 bg-error/10 px-3 py-2 text-sm text-error">{{ error }}</p>
    <div class="gf-card overflow-hidden">
      <div class="flex items-center justify-between gap-2 border-b border-line bg-base-200/60 p-2">
        <div class="flex items-center gap-1">
          <button
            v-for="value in ['topic', 'post'] as const"
            :key="value"
            type="button"
            class="gf-tab"
            :class="kind === value ? 'bg-base-100 text-base-content shadow-sm ring-1 ring-line' : 'text-base-content/55 hover:bg-base-100/70 hover:text-base-content'"
            @click="switchKind(value)"
          >
            {{ t(`moderation.review.kinds.${value}`) }}
          </button>
        </div>
        <span class="px-1 text-xs text-base-content/55">{{ t('moderation.review.count', { count: total }) }}</span>
      </div>

      <div v-if="items.length" class="divide-y divide-line">
        <article v-for="item in items" :key="item.id" class="grid grid-cols-[28px_minmax(0,1fr)] gap-3 px-3 py-2.5 lg:grid-cols-[28px_minmax(0,1fr)_auto] lg:items-start">
          <div class="flex h-7 w-7 shrink-0 items-center justify-center rounded bg-base-200 text-warning">
            <ShieldAlert class="h-4 w-4" />
          </div>
          <div class="min-w-0 space-y-1">
            <div class="flex min-w-0 items-center gap-1.5 text-[15px] leading-5">
              <span v-if="item.postNo" class="shrink-0 text-base-content/45">#{{ item.postNo }}</span>
              <a :href="targetURL(item)" target="_blank" rel="noopener" class="min-w-0 truncate font-medium text-primary/90 hover:text-primary">{{ item.title || t('moderation.review.untitled') }}</a>
              <span v-if="item.aiChecking" class="inline-flex shrink-0 items-center gap-1 rounded bg-base-200 px-1.5 py-0.5 text-xs font-medium text-base-content/65" :title="t('aiModerationAdmin.queueCheckingHint')">
                <Loader2 class="h-3 w-3 motion-safe:animate-spin" />{{ t('aiModerationAdmin.queueChecking') }}
              </span>
              <span v-else-if="item.aiReview" class="shrink-0 rounded bg-warning/15 px-1.5 py-0.5 text-xs font-medium text-warning">{{ t('aiModerationAdmin.queueBadge') }}</span>
            </div>
            <p v-if="item.aiChecking" class="text-xs leading-5 text-base-content/55">{{ t('aiModerationAdmin.queueCheckingHint') }}</p>
            <p class="line-clamp-2 text-[13px] leading-5 text-base-content/60">{{ item.excerpt || t('moderation.review.noExcerpt') }}</p>
            <ul v-if="item.aiReview?.reasons?.length" class="space-y-0.5 text-xs leading-5 text-base-content/55">
              <li v-for="(reason, index) in item.aiReview.reasons" :key="index">{{ reasonText(t, reason) }}</li>
            </ul>
            <div v-if="item.images?.length" class="flex flex-wrap gap-1.5 pt-0.5">
              <a v-for="url in item.images" :key="url" :href="url" target="_blank" rel="noopener">
                <img :src="url" alt="" loading="lazy" class="h-14 w-14 rounded-field border border-line object-cover">
              </a>
            </div>
            <div class="flex flex-wrap items-center gap-x-2.5 text-xs text-base-content/50">
              <a :href="`/u/${item.userId}`" class="font-medium text-base-content/65 hover:text-primary">{{ userDisplayName(item.userId, item.username || `#${item.userId}`, item.nickname) }}</a>
              <time>{{ formatDateTime(item.createdAt) }}</time>
            </div>
          </div>
          <div class="col-span-2 flex justify-end gap-1.5 lg:col-span-1">
            <button type="button" class="gf-button gf-button-sm gf-button-primary" :disabled="busyIds.includes(item.id)" @click="review(item, true)">
              <Loader2 v-if="busyIds.includes(item.id)" class="h-4 w-4 animate-spin" /><Check v-else class="h-4 w-4" />{{ t('moderation.review.approve') }}
            </button>
            <button
              type="button"
              class="gf-button gf-button-sm"
              :class="confirmRejectId === item.id ? 'gf-button-danger' : 'gf-button-secondary'"
              :disabled="busyIds.includes(item.id)"
              @click="review(item, false)"
            >
              <X class="h-4 w-4" />{{ confirmRejectId === item.id ? t('moderation.review.confirmReject') : t('moderation.review.reject') }}
            </button>
          </div>
        </article>
      </div>
      <EmptyState v-else-if="!loading" :icon="Check" :title="t('moderation.review.emptyTitle')" :description="t('moderation.review.emptyDescription')" />
      <div v-if="loading" class="flex justify-center py-6 text-base-content/55"><Loader2 class="h-5 w-5 animate-spin" /></div>
      <div v-else-if="hasMore" class="border-t border-line p-2 text-center">
        <button type="button" class="gf-button gf-button-sm gf-button-ghost" @click="load()">{{ t('moderation.review.loadMore') }}</button>
      </div>
    </div>
    <p class="text-xs leading-5 text-base-content/55">{{ t('moderation.review.hint') }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, Loader2, RotateCcw } from '@lucide/vue'
import { fetchModerationAppeals, moderationAppealAction, type AppealItem, type AppealStatus } from '@/runtime/api'
import { formatDateTime } from '@/runtime/format'
import MarkdownImageViewer from '@/components/MarkdownImageViewer.vue'
import EmptyState from '@/site/components/EmptyState.vue'
const emit = defineEmits<{ changed: [] }>()
const { t } = useI18n()
const statuses: AppealStatus[] = ['pending', 'approved', 'rejected', 'stale']
const status = ref<AppealStatus>('pending')
const items = ref<AppealItem[]>([])
const page = ref(0)
const total = ref(0)
const loading = ref(false)
const error = ref('')
const busyIds = ref<number[]>([])
const reasons = ref<Record<number, string>>({})
const imageViewer = ref<InstanceType<typeof MarkdownImageViewer> | null>(null)
const hasMore = computed(() => page.value * 20 < total.value)
let request = 0
async function load(reset = false) {
  const version = ++request
  const selectedStatus = status.value
  loading.value = true
  error.value = ''
  try {
    const pages = reset ? Array.from({ length: Math.max(page.value, 1) }, (_, index) => index + 1) : [page.value + 1]
    const results = await Promise.all(pages.map(value => fetchModerationAppeals(selectedStatus, value, 20)))
    if (version !== request) return
    const result = results[results.length - 1]!
    const incoming = results.flatMap(value => value.items)
    const seen = new Set(reset ? [] : items.value.map(item => item.id))
    items.value = [...(reset ? [] : items.value), ...incoming.filter(item => { if (seen.has(item.id)) return false; seen.add(item.id); return true })]
    page.value = result.page
    total.value = result.total
  } catch (err) {
    if (version === request) error.value = err instanceof Error ? err.message : t('appeals.loadFailed')
  } finally { if (version === request) loading.value = false }
}
function switchStatus(value: AppealStatus) {
  if (value === status.value) return
  status.value = value
  items.value = []
  page.value = 0
  total.value = 0
  void load(true)
}
function preview(item: AppealItem, index: number, event: MouseEvent) {
  (event.currentTarget as HTMLButtonElement).focus()
  imageViewer.value?.open(item.images.map(src => ({ src, alt: '' })), index)
}
async function review(item: AppealItem, approve: boolean) {
  if (busyIds.value.includes(item.id)) return
  const reason = (reasons.value[item.id] ?? '').trim()
  if (!approve && !reason) { error.value = t('appeals.reviewReasonRequired'); return }
  busyIds.value.push(item.id)
  error.value = ''
  try {
    await moderationAppealAction(item.id, approve, reason)
    emit('changed')
    await load(true)
    delete reasons.value[item.id]
  } catch (err) {
    // A concurrent decision or newer revision must refresh the queue, while preserving the explanation.
    await load(true)
    error.value = err instanceof Error ? err.message : t('appeals.actionFailed')
  } finally { busyIds.value = busyIds.value.filter(id => id !== item.id) }
}
onMounted(() => { void load(true) })
onUnmounted(() => { request++ })
</script>
<template>
  <section class="space-y-3" data-test="moderation-appeal-queue">
    <p v-if="error" role="alert" class="rounded border border-error/25 bg-error/10 p-3 text-sm text-error">{{ error }}</p>
    <div class="gf-card overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-2 border-b border-line bg-base-200/60 p-2">
        <div class="flex flex-wrap gap-1">
          <button v-for="value in statuses" :key="value" type="button" class="gf-tab" :aria-pressed="status === value" :class="status === value ? 'bg-base-100 text-base-content shadow-sm ring-1 ring-line' : 'text-base-content/55'" @click="switchStatus(value)">{{ t(`appeals.status.${value}`) }}</button>
        </div>
        <button type="button" class="gf-button gf-button-sm gf-button-secondary" :disabled="loading" @click="load(true)"><RotateCcw class="h-4 w-4" />{{ t('appeals.refresh') }}</button>
      </div>
      <div class="divide-y divide-line">
        <article v-for="item in items" :key="item.id" class="space-y-2 p-3">
          <div class="flex flex-wrap items-center gap-2 text-sm">
            <span>{{ t(`moderation.review.kinds.${item.kind}`) }}<span v-if="item.kind === 'post'"> #{{ item.postNo }}</span></span>
            <span class="min-w-0 break-words font-semibold">{{ item.title || t('moderation.review.untitled') }}</span>
            <time class="text-xs text-base-content/55">{{ formatDateTime(item.createdAt) }}</time>
          </div>
          <p class="whitespace-pre-wrap break-words text-sm text-base-content/70">{{ item.content }}</p>
          <div v-if="item.images.length" class="flex flex-wrap gap-2">
            <button v-for="(url, index) in item.images" :key="index" type="button" :aria-label="`${t('common.preview')} ${index + 1}`" class="rounded focus-visible:outline-2 focus-visible:outline-primary" @click="preview(item, index, $event)"><img :src="url" alt="" loading="lazy" class="h-14 w-14 rounded border border-line object-cover" /></button>
          </div>
          <p class="whitespace-pre-wrap break-words text-sm"><strong>{{ t('appeals.reason') }}:</strong> {{ item.reason }}</p>
          <p v-if="item.reviewReason" class="whitespace-pre-wrap break-words text-sm text-base-content/70"><strong>{{ t('appeals.reviewReason') }}:</strong> {{ item.reviewReason }}</p>
          <div v-if="item.status === 'pending'" class="space-y-2">
            <label class="block text-sm"><span>{{ t('appeals.reviewReason') }}</span><textarea v-model="reasons[item.id]" maxlength="512" :disabled="busyIds.includes(item.id)" class="gf-input mt-1 min-h-20 w-full" /></label>
            <div class="flex justify-end gap-2">
              <button type="button" class="gf-button gf-button-sm gf-button-primary" :disabled="busyIds.includes(item.id)" @click="review(item, true)">{{ t('moderation.review.approve') }}</button>
              <button type="button" class="gf-button gf-button-sm gf-button-danger" :disabled="busyIds.includes(item.id) || !reasons[item.id]?.trim()" @click="review(item, false)">{{ t('moderation.review.reject') }}</button>
            </div>
          </div>
        </article>
      </div>
      <EmptyState v-if="!items.length && !loading" :icon="Check" :title="t('appeals.empty')" />
      <div v-if="loading" class="flex justify-center p-6" role="status" :aria-label="t('common.loading')"><Loader2 class="h-5 w-5 animate-spin" /></div>
      <div v-else-if="hasMore" class="border-t border-line p-2 text-center"><button type="button" class="gf-button gf-button-sm gf-button-ghost" @click="load()">{{ t('moderation.review.loadMore') }}</button></div>
    </div>
    <MarkdownImageViewer ref="imageViewer" />
  </section>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { fetchContentAppeals, submitContentAppeal, type AppealItem, type MyContentItem } from '@/runtime/api'
import { formatDateTime } from '@/runtime/format'
const props = defineProps<{ item: MyContentItem }>()
const emit = defineEmits<{ saved: [] }>()
const { t } = useI18n()
const open = ref(false)
const loaded = ref(false)
const loading = ref(false)
const busy = ref(false)
const error = ref('')
const reason = ref('')
const items = ref<AppealItem[]>([])
const current = computed(() => items.value.find(item => item.revisionId === props.item.revisionId))
let request = 0
async function load() {
  const version = ++request
  const { contentType, id } = props.item
  loading.value = true
  loaded.value = false
  error.value = ''
  try {
    const result = await fetchContentAppeals(contentType, id)
    if (version !== request) return
    items.value = result.items
    loaded.value = true
  } catch (err) {
    if (version === request) error.value = err instanceof Error ? err.message : t('appeals.loadFailed')
  } finally { if (version === request) loading.value = false }
}
watch(() => props.item, (next, previous) => {
  if (next.id !== previous.id || next.contentType !== previous.contentType || next.revisionId !== previous.revisionId) {
    request++
    loaded.value = false
    loading.value = false
    items.value = []
    reason.value = ''
  }
  if (open.value && !busy.value) void load()
})
onUnmounted(() => { request++ })
function toggle() {
  open.value = !open.value
  if (open.value) void load()
}
async function submit() {
  if (busy.value || !loaded.value || current.value || props.item.processStatus !== 1) return
  const explanation = reason.value.trim()
  if (!explanation) { error.value = t('appeals.reasonRequired'); return }
  const { contentType, id, revisionId } = props.item
  busy.value = true
  error.value = ''
  try {
    await submitContentAppeal(contentType, id, revisionId, explanation)
    reason.value = ''
    await load()
    emit('saved')
  } catch (err) { error.value = err instanceof Error ? err.message : t('appeals.actionFailed') }
  finally { busy.value = false }
}
</script>
<template>
  <span class="block space-y-2">
    <button type="button" class="text-xs text-primary" :aria-expanded="open" :disabled="busy" @click="toggle">{{ t('appeals.title') }}</button>
    <span v-if="open" class="block space-y-2 rounded border border-line p-3">
      <span class="block text-xs text-base-content/60">{{ t('appeals.hint') }}</span>
      <span v-if="error" role="alert" class="block text-sm text-error">{{ error }}</span>
      <span v-if="loading" role="status" class="block text-sm">{{ t('common.loading') }}</span>
      <button v-if="!loading && !loaded" type="button" class="gf-button gf-button-sm gf-button-secondary" @click="load">{{ t('common.retry') }}</button>
      <form v-if="loaded && !current && item.processStatus === 1" class="space-y-2" @submit.prevent="submit">
        <label class="block text-sm">
          <span>{{ t('appeals.reason') }}</span>
          <textarea v-model="reason" required maxlength="512" :disabled="busy" class="gf-input mt-1 min-h-24 w-full" />
        </label>
        <button type="submit" class="gf-button gf-button-sm gf-button-primary" :disabled="busy || !reason.trim()">{{ t('appeals.submit') }}</button>
      </form>
      <span v-for="appeal in items" :key="appeal.id" class="block space-y-1 border-t border-line pt-2 text-sm">
        <span class="block font-medium">{{ t(`appeals.status.${appeal.status}`) }} · {{ formatDateTime(appeal.createdAt) }}</span>
        <span class="block whitespace-pre-wrap break-words">{{ appeal.reason }}</span>
        <span v-if="appeal.reviewReason" class="block whitespace-pre-wrap break-words text-base-content/70">{{ t('appeals.reviewReason') }}: {{ appeal.reviewReason }}</span>
      </span>
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { renderMarkdownPreview } from '@/runtime/markdown'
import { useI18n } from 'vue-i18n'
import { updatePost, type MyContentItem } from '@/runtime/api'
const props = defineProps<{ item: MyContentItem }>()
const emit = defineEmits<{ saved: [] }>()
const { t } = useI18n()
const open = ref(false)
const editing = ref(false)
const body = ref('')
const busy = ref(false)
const error = ref('')
const rendered = computed(() => renderMarkdownPreview(props.item.content))
function edit() { body.value = props.item.content; editing.value = true }
async function save() {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try { await updatePost(props.item.id, body.value); editing.value = false; emit('saved') }
  catch (e) { error.value = e instanceof Error ? e.message : String(e) }
  finally { busy.value = false }
}
</script>
<template>
  <span class="block space-y-2" @click.stop>
    <span v-if="item.processStatus === 2" class="text-xs text-warning">{{ t('topic.pendingReviewBadge') }}</span>
    <span v-if="item.processStatus === 1" class="text-xs text-error">{{ t('contentReview.blocked') }}</span>
    <span v-if="item.reviewReason" class="block text-sm text-base-content/70">{{ item.reviewReason }}</span>
    <span v-if="item.hasPublishedVersion && item.processStatus !== 0" class="block text-xs text-base-content/60">{{ t('contentReview.live') }}</span>
    <button type="button" class="mr-3 text-xs text-primary" @click.stop="open = !open">{{ t('contentReview.view') }}</button>
    <a v-if="item.contentType === 'topic'" :href="`/publish?id=${item.id}`" class="text-xs text-primary" @click.stop>{{ t('contentReview.retry') }}</a>
    <button v-else type="button" class="text-xs text-primary" @click.stop="edit">{{ t('contentReview.retry') }}</button>
    <span v-if="open" class="block break-words text-sm"><span class="markdown-body block" v-html="rendered" /><img v-for="url in item.images" :key="url" :src="url" class="my-2 max-h-64 rounded" alt="" /></span>
    <span v-if="editing" class="block space-y-2">
      <textarea v-model="body" :aria-label="t('contentReview.retry')" rows="6" class="w-full rounded border border-line bg-base-100 p-2 text-sm" />
      <span v-if="error" class="block text-sm text-error">{{ error }}</span>
      <button type="button" class="text-sm text-primary" :disabled="busy" @click.stop="save">{{ t('contentReview.retry') }}</button>
    </span>
  </span>
</template>

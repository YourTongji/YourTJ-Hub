<script setup lang="ts">
import { computed, defineAsyncComponent, ref } from 'vue'
import { renderMarkdownPreview } from '@/runtime/markdown'
import { useI18n } from 'vue-i18n'
import { updatePost, type MyContentItem } from '@/runtime/api'
import { fetchPage } from '@/runtime/router'
import { useQuickPublish } from '@/site/composables/useQuickPublish'
import type { PublishPageProps } from '@gooseforum/client'
const PostComposer = defineAsyncComponent(() => import('@/site/components/PostComposer.vue'))
const props = defineProps<{ item: MyContentItem }>()
const emit = defineEmits<{ saved: [] }>()
const { t } = useI18n()
const open = ref(false)
const editing = ref(false)
const body = ref('')
const hasReplyDraft = ref(false)
const busy = ref(false)
const error = ref('')
const rendered = computed(() => renderMarkdownPreview(props.item.content))
async function edit() {
  if (busy.value) return
  if (props.item.contentType !== 'topic') {
    if (!hasReplyDraft.value) body.value = props.item.content
    hasReplyDraft.value = true
    editing.value = true
    return
  }
  busy.value = true
  error.value = ''
  try {
    const url = new URL(`/publish?id=${props.item.id}`, window.location.origin)
    const page = await fetchPage(url)
    if (page.component !== 'publish.index') throw new Error(t('api.operationFailed'))
    const topic = (page.props as PublishPageProps).topic
    if (topic && (topic.contentType === 1 || topic.contentType === 2)) {
      useQuickPublish().openQuickPublishEdit({ topicId: props.item.id, contentType: topic.contentType, title: topic.title, content: topic.content, categoryIds: topic.categoryIds, images: topic.images })
    } else {
      window.location.assign(url.href)
    }
  } catch (e) { error.value = e instanceof Error ? e.message : String(e) }
  finally { busy.value = false }
}
async function save() {
  if (busy.value) return
  if (!body.value.trim()) { error.value = t('topic.replyRequired'); return }
  busy.value = true
  error.value = ''
  try { await updatePost(props.item.id, body.value.trim()); editing.value = false; hasReplyDraft.value = false; emit('saved') }
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
    <button type="button" class="text-xs text-primary" :disabled="busy" @click.stop="edit">{{ t('contentReview.retry') }}</button>
    <span v-if="error && !editing" class="block text-sm text-error" role="alert">{{ error }}</span>
    <span v-if="open" class="block break-words text-sm"><span class="markdown-body block" v-html="rendered" /><img v-for="url in item.images" :key="url" :src="url" class="my-2 max-h-64 rounded" alt="" /></span>
  </span>
  <PostComposer
    v-if="hasReplyDraft"
    v-model="body"
    v-model:open="editing"
    mode="edit"
    :authenticated="true"
    :submitting="busy"
    :error-message="error"
    success-message=""
    @submit="save"
    @clear-validation="error = ''"
    @image-error="error = $event"
  />
</template>

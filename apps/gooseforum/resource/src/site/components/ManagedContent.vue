<script setup lang="ts">
import { computed, defineAsyncComponent, ref } from 'vue'
import { Clock, Eye, PenLine, Scale, ShieldAlert } from '@lucide/vue'
import ContentAppeal from '@/site/components/ContentAppeal.vue'
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
const appealOpen = ref(false)
const confirmEdit = ref(false)
const rendered = computed(() => renderMarkdownPreview(props.item.content))
// 修改重提会让复核中的申诉失效，先在原地说清楚再继续（#1114）。
const appealPending = computed(() => props.item.appeal?.status === 'pending')
function requestEdit() {
  if (appealPending.value && !confirmEdit.value) { confirmEdit.value = true; return }
  confirmEdit.value = false
  void edit()
}
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
  <div class="mt-2" @click.stop>
    <div v-if="item.processStatus !== 0" class="rounded-box bg-base-200/55 p-3" :data-test="`content-status-${item.processStatus}`">
      <div class="flex flex-wrap items-center gap-1.5">
        <span v-if="item.processStatus === 1" class="gf-badge gf-badge-error"><ShieldAlert class="h-3.5 w-3.5" aria-hidden="true" />{{ t('contentReview.blocked') }}</span>
        <span v-else class="gf-badge gf-badge-warning"><Clock class="h-3.5 w-3.5" aria-hidden="true" />{{ t('topic.pendingReviewBadge') }}</span>
      </div>
      <div v-if="item.processStatus === 1 && item.reviewReason" class="mt-2">
        <p class="text-xs font-medium text-base-content/55">{{ t('contentReview.reasonLabel') }}</p>
        <p class="mt-0.5 whitespace-pre-wrap break-words text-sm leading-6 text-base-content">{{ item.reviewReason }}</p>
      </div>
      <p v-if="item.hasPublishedVersion" class="mt-2 text-[13px] leading-5 text-base-content/60">{{ t('contentReview.live') }}</p>
      <ContentAppeal v-if="item.processStatus === 1" v-model:open="appealOpen" :item="item" @saved="emit('saved')" />
      <div v-if="confirmEdit" class="mt-3 flex flex-wrap items-center gap-x-3 gap-y-2 rounded-field bg-warning/10 px-3 py-2 text-[13px] leading-5 text-base-content" role="alert">
        <span class="min-w-0 flex-1">{{ t('appeals.editWarning') }}</span>
        <span class="flex gap-2">
          <button type="button" class="gf-button gf-button-xs gf-button-ghost" @click="confirmEdit = false">{{ t('appeals.cancel') }}</button>
          <button type="button" class="gf-button gf-button-xs gf-button-secondary" @click="requestEdit">{{ t('appeals.editAnyway') }}</button>
        </span>
      </div>
      <div class="mt-3 flex flex-wrap gap-2">
        <button type="button" class="gf-button gf-button-xs gf-button-secondary" :aria-expanded="open" @click="open = !open"><Eye class="h-3.5 w-3.5" aria-hidden="true" />{{ t('contentReview.view') }}</button>
        <button type="button" class="gf-button gf-button-xs gf-button-secondary" :disabled="busy" @click="requestEdit"><PenLine class="h-3.5 w-3.5" aria-hidden="true" />{{ t('contentReview.retry') }}</button>
        <button v-if="item.processStatus === 1 && item.canAppeal && !item.appeal" type="button" class="gf-button gf-button-xs" :class="appealOpen ? 'gf-button-neutral' : 'gf-button-secondary'" :aria-expanded="appealOpen" data-test="appeal-toggle" @click="appealOpen = !appealOpen"><Scale class="h-3.5 w-3.5" aria-hidden="true" />{{ t('appeals.action') }}</button>
      </div>
    </div>
    <div v-else class="flex flex-wrap gap-3">
      <button type="button" class="text-xs font-medium text-primary hover:underline" :aria-expanded="open" @click="open = !open">{{ t('contentReview.view') }}</button>
      <button type="button" class="text-xs font-medium text-primary hover:underline" :disabled="busy" @click="requestEdit">{{ t('contentReview.edit') }}</button>
    </div>
    <p v-if="error && !editing" class="gf-status-message gf-status-message-error mt-2" role="alert">{{ error }}</p>
    <div v-if="open" class="mt-3 break-words rounded-field bg-base-100 p-3 text-sm shadow-[0_0_0_1px_var(--gf-color-line)]"><div class="markdown-body" v-html="rendered" /><img v-for="url in item.images" :key="url" :src="url" class="gf-image-outline my-2 max-h-64 rounded-field" alt="" /></div>
  </div>
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

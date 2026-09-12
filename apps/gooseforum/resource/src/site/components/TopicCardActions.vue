<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Bookmark, Eye, Heart, MessageSquare } from '@lucide/vue'
import { formatNumber } from '@/runtime/format'
import type { TopicPayload } from '@gooseforum/client'
import { createTopicCardInteraction, type TopicCardInteraction } from '@/site/utils/topic-card-interactions'

const props = defineProps<{ topic: TopicPayload; interaction?: TopicCardInteraction }>()
const { t } = useI18n()
// Standalone cards own their state; feed cards receive the list's longer-lived owner.
const local = props.interaction ? undefined : createTopicCardInteraction(props.topic, t)
const interaction = computed(() => props.interaction ?? local!)
const state = computed(() => interaction.value.state)
watch(() => [props.topic, props.topic.liked, props.topic.bookmarked, props.topic.likeCount], () => local?.sync(props.topic))
onBeforeUnmount(() => local?.dispose())
const commentUrl = computed(() => {
  const url = new URL(props.topic.url, window.location.href)
  url.searchParams.set('reply', '1')
  return `${url.pathname}${url.search}${url.hash}`
})
</script>

<template>
  <div class="flex flex-wrap items-center gap-0.5 text-xs text-base-content/55">
    <a
      :href="commentUrl"
      class="inline-flex h-7 items-center gap-1.5 rounded-md px-2 transition-colors hover:bg-base-200 hover:text-base-content"
      :title="t('topic.reply')"
    >
      <MessageSquare class="h-4 w-4" />
      <span class="tabular-nums">{{ formatNumber(topic.replyCount) }}</span>
    </a>
    <span
      class="inline-flex h-7 items-center gap-1.5 rounded-md px-2"
      :title="t('topicList.columns.views')"
    >
      <Eye class="h-4 w-4" />
      <span class="tabular-nums">{{ formatNumber(topic.viewCount) }}</span>
    </span>
    <button
      type="button"
      class="inline-flex h-7 items-center gap-1.5 rounded-md px-2 transition-colors hover:bg-base-200 hover:text-base-content disabled:cursor-default disabled:opacity-60"
      :class="state.liked ? 'text-error' : ''"
      :disabled="state.actingLike"
      :aria-pressed="state.liked === true"
      :title="t('topic.like')"
      @click="interaction.toggle(false)"
    >
      <Heart class="h-4 w-4" :class="state.liked ? 'fill-current' : ''" />
      <span class="tabular-nums">{{ formatNumber(state.likeCount) }}</span>
    </button>
    <button
      type="button"
      class="inline-flex h-7 items-center gap-1.5 rounded-md px-2 transition-colors hover:bg-base-200 hover:text-base-content disabled:cursor-default disabled:opacity-60"
      :class="state.bookmarked ? 'text-primary' : ''"
      :disabled="state.actingBookmark"
      :aria-pressed="state.bookmarked === true"
      :title="state.bookmarked ? t('topic.bookmarked') : t('topic.bookmark')"
      @click="interaction.toggle(true)"
    >
      <Bookmark class="h-4 w-4" :class="state.bookmarked ? 'fill-current' : ''" />
    </button>
    <p v-if="state.error" role="alert" class="w-full px-2 pt-1 text-error">{{ state.error }}</p>
  </div>
</template>

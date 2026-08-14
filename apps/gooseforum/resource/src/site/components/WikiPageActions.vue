<script setup lang="ts">
import { ref, watch } from 'vue'
import { Bell, Bookmark, Heart, PencilLine } from '@lucide/vue'
import { bookmarkTopic, likeTopic, watchTopic } from '@/runtime/api'
import { formatNumber } from '@/runtime/format'
import { useFlashMessages } from '@/runtime/flash-message'
import type { WikiPageDetailPayload } from '@gooseforum/client'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  page: WikiPageDetailPayload
  canEdit: boolean
}>()

const emit = defineEmits<{
  edit: []
  'interaction-change': [state: { likeCount: number; isLiked: boolean; isBookmarked: boolean; isWatched: boolean }]
}>()

const { t } = useI18n()
const { push: pushFlash } = useFlashMessages()
const likeCount = ref(props.page.likeCount)
const isLiked = ref(props.page.liked)
const isBookmarked = ref(props.page.bookmarked)
const isWatched = ref(props.page.watched)
const actingLike = ref(false)
const actingBookmark = ref(false)
const actingWatch = ref(false)

watch(
  () => props.page,
  (next) => {
    likeCount.value = next.likeCount
    isLiked.value = next.liked
    isBookmarked.value = next.bookmarked
    isWatched.value = next.watched
  },
  { immediate: true },
)

async function toggleLike() {
  if (actingLike.value) return

  const nextLiked = !isLiked.value
  const previousLiked = isLiked.value
  const previousCount = likeCount.value
  actingLike.value = true
  isLiked.value = nextLiked
  likeCount.value = Math.max(0, likeCount.value + (nextLiked ? 1 : -1))
  emitInteraction()
  try {
    await likeTopic(props.page.topicId, nextLiked ? 1 : 2)
  } catch (error) {
    isLiked.value = previousLiked
    likeCount.value = previousCount
    pushFlash(error instanceof Error ? error.message : t('api.likeFailed'), 'error')
  } finally {
    actingLike.value = false
  }
}

async function toggleBookmark() {
  if (actingBookmark.value) return

  const nextBookmarked = !isBookmarked.value
  const previousBookmarked = isBookmarked.value
  actingBookmark.value = true
  isBookmarked.value = nextBookmarked
  emitInteraction()
  try {
    await bookmarkTopic(props.page.topicId, nextBookmarked ? 1 : 2)
    pushFlash(nextBookmarked ? t('topic.bookmarkAdded') : t('topic.bookmarkRemoved'))
  } catch (error) {
    isBookmarked.value = previousBookmarked
    pushFlash(error instanceof Error ? error.message : t('api.bookmarkFailed'), 'error')
  } finally {
    actingBookmark.value = false
  }
}

async function toggleWatch() {
  if (actingWatch.value) return

  const nextWatched = !isWatched.value
  const previousWatched = isWatched.value
  actingWatch.value = true
  isWatched.value = nextWatched
  emitInteraction()
  try {
    await watchTopic(props.page.topicId, nextWatched ? 1 : 2)
    pushFlash(nextWatched ? t('topic.watchAdded') : t('topic.watchRemoved'))
  } catch (error) {
    isWatched.value = previousWatched
    pushFlash(error instanceof Error ? error.message : t('api.watchFailed'), 'error')
  } finally {
    actingWatch.value = false
  }
}

function emitInteraction() {
  emit('interaction-change', {
    likeCount: likeCount.value,
    isLiked: isLiked.value,
    isBookmarked: isBookmarked.value,
    isWatched: isWatched.value,
  })
}
</script>

<template>
  <div class="flex flex-wrap items-center gap-2">
    <button
      v-if="canEdit"
      type="button"
      class="gf-button gf-button-sm"
      @click="emit('edit')"
    >
      <PencilLine class="h-4 w-4" />
      {{ t('wiki.editPage') }}
    </button>
    <button
      type="button"
      class="gf-button gf-button-sm px-2.5"
      :class="isLiked ? 'bg-error/10 text-error hover:bg-error/10' : 'text-base-content/55 hover:bg-base-200 hover:text-base-content'"
      :disabled="actingLike"
      @click="toggleLike"
    >
      <Heart class="h-4 w-4" :fill="isLiked ? 'currentColor' : 'none'" />
      {{ likeCount ? formatNumber(likeCount) : t('topic.like') }}
    </button>
    <button
      type="button"
      class="gf-button gf-button-sm px-2.5"
      :class="isBookmarked ? 'bg-info/10 text-primary hover:bg-info/10' : 'text-base-content/55 hover:bg-base-200 hover:text-base-content'"
      :disabled="actingBookmark"
      @click="toggleBookmark"
    >
      <Bookmark class="h-4 w-4" :fill="isBookmarked ? 'currentColor' : 'none'" />
      {{ isBookmarked ? t('topic.bookmarked') : t('topic.bookmark') }}
    </button>
    <button
      type="button"
      class="gf-button gf-button-sm px-2.5"
      :class="isWatched ? 'bg-success/10 text-success hover:bg-success/15' : 'text-base-content/55 hover:bg-base-200 hover:text-base-content'"
      :disabled="actingWatch"
      @click="toggleWatch"
    >
      <Bell class="h-4 w-4" :fill="isWatched ? 'currentColor' : 'none'" />
      {{ isWatched ? t('topic.watched') : t('topic.watch') }}
    </button>
  </div>
</template>

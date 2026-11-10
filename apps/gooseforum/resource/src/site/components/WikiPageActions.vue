<script setup lang="ts">
import { Bell, Bookmark, ExternalLink, Heart, History, PencilLine } from '@lucide/vue'
import { formatNumber } from '@/runtime/format'
import type { WikiInteractionPending, WikiInteractionState } from '@/site/utils/wiki-page-interactions'
import type { WikiPageDetailPayload } from '@gooseforum/client'
import { useI18n } from 'vue-i18n'

defineProps<{
  page: WikiPageDetailPayload
  canEdit: boolean
  interactions: WikiInteractionState
  pending: WikiInteractionPending
}>()

const emit = defineEmits<{
  like: []
  bookmark: []
  watch: []
}>()
const { t } = useI18n()
</script>
<template>
  <div class="flex flex-wrap items-center gap-2">
    <!-- GitHub SSOT：编辑/历史走仓库外链（fork + PR），站内无编辑。 -->
    <a
      v-if="canEdit && page.editUrl"
      :href="page.editUrl"
      target="_blank"
      rel="noopener noreferrer"
      class="gf-button gf-button-sm"
    >
      <PencilLine class="h-4 w-4" />
      {{ t('wiki.editPage') }}
      <ExternalLink class="h-3.5 w-3.5 opacity-60" />
    </a>
    <a
      v-if="page.historyUrl"
      :href="page.historyUrl"
      target="_blank"
      rel="noopener noreferrer"
      class="gf-button gf-button-sm gf-button-muted"
    >
      <History class="h-4 w-4" />
      {{ t('wiki.historyPage') }}
      <ExternalLink class="h-3.5 w-3.5 opacity-60" />
    </a>
    <button
      type="button"
      class="gf-button gf-button-sm px-2.5"
      :class="interactions.isLiked ? 'bg-error/10 text-error hover:bg-error/10' : 'text-base-content/55 hover:bg-base-200 hover:text-base-content'"
      :disabled="pending.like"
      @click="emit('like')"
    >
      <Heart class="h-4 w-4" :fill="interactions.isLiked ? 'currentColor' : 'none'" />
      {{ interactions.likeCount ? formatNumber(interactions.likeCount) : t('topic.like') }}
    </button>
    <button
      type="button"
      class="gf-button gf-button-sm px-2.5"
      :class="interactions.isBookmarked ? 'bg-info/10 text-primary hover:bg-info/10' : 'text-base-content/55 hover:bg-base-200 hover:text-base-content'"
      :disabled="pending.bookmark"
      @click="emit('bookmark')"
    >
      <Bookmark class="h-4 w-4" :fill="interactions.isBookmarked ? 'currentColor' : 'none'" />
      {{ interactions.isBookmarked ? t('topic.bookmarked') : t('topic.bookmark') }}
    </button>
    <button
      type="button"
      class="gf-button gf-button-sm px-2.5"
      :class="interactions.isWatched ? 'bg-success/10 text-success hover:bg-success/15' : 'text-base-content/55 hover:bg-base-200 hover:text-base-content'"
      :disabled="pending.watch"
      @click="emit('watch')"
    >
      <Bell class="h-4 w-4" :fill="interactions.isWatched ? 'currentColor' : 'none'" />
      {{ interactions.isWatched ? t('topic.watched') : t('topic.watch') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { BookOpen, Eye, HelpCircle, MessageSquare, Pin, Sparkles } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { formatNumber, timeAgo } from '@/runtime/format'
import UserAvatar from '@/site/components/UserAvatar.vue'
import type { TopicPayload } from '@gooseforum/client'

withDefaults(defineProps<{
  topic: TopicPayload
  compact?: boolean
  showCategories?: boolean
  showHot?: boolean
  showPinned?: boolean
}>(), {
  compact: false,
  showCategories: true,
  showHot: true,
  showPinned: true,
})

const { t } = useI18n()
</script>

<template>
  <div class="min-w-0">
    <div class="flex items-center gap-2.5">
      <UserAvatar
        :src="topic.author.avatarUrl"
        :alt="topic.author.username"
        :badge="topic.author.wornBadge"
        class="h-8 w-8 shrink-0 rounded-full ring-2 ring-primary/20"
      />
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5">
          <span class="max-w-full truncate text-sm font-semibold leading-5 text-base-content">
            {{ topic.author.nickname || topic.author.username }}
          </span>
          <span class="text-xs leading-5 text-base-content/55">
            {{ timeAgo(topic.lastUpdateTime) }}
          </span>
        </div>
        <div class="mt-1 flex min-w-0 flex-wrap items-center gap-1.5">
          <!-- Content type badge -->
          <span v-if="topic.contentType === 1" class="inline-flex h-5 items-center gap-1 rounded-full bg-success/15 px-1.5 text-[11px] font-semibold text-success">
            <HelpCircle class="h-3 w-3" />
            <span>{{ t('publish.contentTypes.question') }}</span>
          </span>
          <span v-else-if="topic.contentType === 2" class="inline-flex h-5 items-center gap-1 rounded-full bg-purple-500/15 px-1.5 text-[11px] font-semibold text-purple-600 dark:text-purple-400">
            <Sparkles class="h-3 w-3" />
            <span>{{ t('publish.contentTypes.thought') }}</span>
          </span>
          <span v-else-if="topic.contentType === 3" class="inline-flex h-5 items-center gap-1 rounded-full bg-amber-500/15 px-1.5 text-[11px] font-semibold text-amber-600 dark:text-amber-400">
            <BookOpen class="h-3 w-3" />
            <span>{{ t('publish.contentTypes.article') }}</span>
          </span>
          <a
            v-for="category in showCategories ? topic.categories : []"
            :key="category.id"
            :href="category.url"
            class="gf-topic-chip"
          >
            <span class="h-1.5 w-1.5 rounded-full" :style="{ backgroundColor: category.color }" />
            {{ category.name }}
          </a>
          <span
            v-if="showHot && topic.viewCount > 500"
            class="inline-flex h-5 items-center gap-1 rounded-full bg-warning/10 px-1.5 text-[11px] font-semibold text-warning"
          >
            <Sparkles class="h-3 w-3" /> hot
          </span>
        </div>
      </div>
      <Pin
        v-if="showPinned && topic.pinWeight > 0"
        class="mt-0.5 h-4 w-4 shrink-0 rotate-45 text-error"
        :aria-label="t('topicList.pinned')"
      />
    </div>

    <h3
      class="mt-3 line-clamp-2 font-semibold text-base-content transition-colors group-hover:text-primary"
      :class="compact ? 'text-[15px] leading-6' : 'text-base leading-7'"
    >
      {{ topic.title }}
      <span
        v-if="topic.unseen"
        class="ml-1.5 inline-block h-2 w-2 shrink-0 rounded-full bg-primary align-middle"
        aria-hidden="true"
      />
    </h3>
    <p
      v-if="topic.description"
      class="mt-1.5 line-clamp-2 text-sm leading-6 text-base-content/55"
    >
      {{ topic.description }}
    </p>

    <slot name="media" />

    <div class="mt-3 flex items-center gap-1 border-t border-line/70 pt-2.5 text-xs text-base-content/55">
      <span
        class="inline-flex h-7 items-center gap-1.5 rounded-md px-2"
        :title="t('topicList.columns.replies') + ': ' + formatNumber(topic.replyCount)"
      >
        <MessageSquare class="h-4 w-4" />
        <span class="tabular-nums">{{ formatNumber(topic.replyCount) }}</span>
      </span>
      <span class="inline-flex h-7 items-center gap-1.5 rounded-md px-2" :title="t('topicList.columns.views')">
        <Eye class="h-4 w-4" />
        <span class="tabular-nums">{{ formatNumber(topic.viewCount) }}</span>
      </span>
      <span class="ml-auto inline-flex h-7 items-center rounded-md px-2 tabular-nums">
        {{ timeAgo(topic.lastUpdateTime) }}
      </span>
    </div>
  </div>
</template>

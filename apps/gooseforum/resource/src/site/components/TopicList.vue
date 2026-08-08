<script setup lang="ts">
import { useSlots } from 'vue'
import { useI18n } from 'vue-i18n'
import TopicFeedPreview from '@/site/components/TopicFeedPreview.vue'
import TopicRow from '@/site/components/TopicRow.vue'
import type { TopicPayload } from '@gooseforum/client'

withDefaults(defineProps<{
  topics: TopicPayload[]
  home?: boolean
  showCategories?: boolean
  showHot?: boolean
  showPinned?: boolean
  feedMode?: 'table' | 'card'
}>(), {
  home: false,
  showCategories: true,
  showHot: true,
  showPinned: false,
  feedMode: 'table',
})

const { t } = useI18n()
const slots = useSlots()
</script>

<template>
  <template v-if="feedMode === 'table'">
    <div class="gf-topic-list-header">
      <div>{{ t('topicList.columns.topic') }}</div>
      <div class="text-center">{{ t('topicList.columns.users') }}</div>
      <div class="text-center">{{ t('topicList.columns.replies') }}</div>
      <div class="text-center">{{ t('topicList.columns.views') }}</div>
      <div class="text-right">
        <slot name="activity-header">
          {{ t('topicList.columns.activity') }}
        </slot>
      </div>
    </div>

    <div class="relative bg-base-100">
      <TopicRow
        v-for="topic in topics"
        :key="topic.id"
        :topic="topic"
        :home="home"
        :show-categories="showCategories"
        :show-hot="showHot"
        :show-pinned="showPinned"
      >
        <template v-if="slots.activity" #activity="{ topic: rowTopic }">
          <slot name="activity" :topic="rowTopic" />
        </template>
        <template v-if="slots['mobile-action']" #mobile-action="{ topic: rowTopic }">
          <slot name="mobile-action" :topic="rowTopic" />
        </template>
      </TopicRow>
    </div>
  </template>

  <template v-else>
    <div class="relative space-y-4 p-4">
      <div
        v-for="topic in topics"
        :key="topic.id"
        class="gf-card group overflow-hidden"
      >
        <a :href="topic.url" class="block outline-none focus-visible:ring-2 focus-visible:ring-primary/50">
          <TopicFeedPreview :topic="topic" compact />
        </a>
      </div>
    </div>
  </template>
  <slot name="empty" />
</template>

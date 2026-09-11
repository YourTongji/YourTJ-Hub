<script setup lang="ts">
import { useSlots } from 'vue'
import { useI18n } from 'vue-i18n'
import type { TopicPayload } from '@gooseforum/client'
import TopicCardActions from '@/site/components/TopicCardActions.vue'
import TopicFeedPreview from '@/site/components/TopicFeedPreview.vue'
import TopicRow from '@/site/components/TopicRow.vue'
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

// 成功互动回写共享 topic 对象：card↔table 切换或重挂载时新实例读到的即最新状态。
function applyInteraction(
  topic: TopicPayload,
  patch: { liked?: boolean; bookmarked?: boolean; likeCount?: number },
) {
  Object.assign(topic, patch)
}
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
        class="gf-card group relative overflow-hidden [&_a.gf-topic-chip]:pointer-events-auto [&_a.gf-topic-chip]:relative [&_a.gf-topic-chip]:z-10"
      >
        <TopicFeedPreview :topic="topic" compact :show-stats="false" class="pointer-events-none" />
        <!-- stretched-link：整卡可点，互动按钮浮于遮罩之上，避免嵌套交互元素 -->
        <a
          :href="topic.url"
          class="absolute inset-0 outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
          :aria-label="topic.title"
        />
        <TopicCardActions
          :topic="topic"
          class="relative px-3.5 pb-3"
          @interacted="(patch) => applyInteraction(topic, patch)"
        />
      </div>
    </div>
  </template>
  <slot name="empty" />
</template>

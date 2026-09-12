<script setup lang="ts">
import { onBeforeUnmount, shallowReactive, useSlots, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { TopicPayload } from '@gooseforum/client'
import { createTopicCardInteraction, type TopicCardInteraction } from '@/site/utils/topic-card-interactions'
import TopicCardActions from '@/site/components/TopicCardActions.vue'
import TopicFeedPreview from '@/site/components/TopicFeedPreview.vue'
import TopicRow from '@/site/components/TopicRow.vue'
const props = withDefaults(defineProps<{
  topics: TopicPayload[]
  viewerId?: number
  home?: boolean
  showCategories?: boolean
  showHot?: boolean
  showPinned?: boolean
  feedMode?: 'table' | 'card'
}>(), {
  viewerId: 0,
  home: false,
  showCategories: true,
  showHot: true,
  showPinned: false,
  feedMode: 'table',
})

const { t } = useI18n()
const slots = useSlots()

const interactions = shallowReactive(new Map<number, TopicCardInteraction>())
function clearInteractions() {
  for (const interaction of interactions.values()) interaction.dispose()
  interactions.clear()
}
watch(() => props.viewerId, clearInteractions, { flush: 'sync' })
watch(() => props.topics.map(topic => [topic, topic.liked, topic.bookmarked, topic.likeCount]), () => {
  const ids = new Set(props.topics.map(topic => topic.id))
  for (const [id, interaction] of interactions) {
    if (!ids.has(id)) { interaction.dispose(); interactions.delete(id) }
  }
  for (const topic of props.topics) {
    const interaction = interactions.get(topic.id)
    if (interaction) interaction.sync(topic)
    else interactions.set(topic.id, createTopicCardInteraction(topic, t))
  }
}, { immediate: true })
// Recreate owners even when a viewer switch retains the same topic array.
watch(() => props.viewerId, () => {
  for (const topic of props.topics) {
    if (!interactions.has(topic.id)) interactions.set(topic.id, createTopicCardInteraction(topic, t))
  }
})
onBeforeUnmount(clearInteractions)
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
        <TopicFeedPreview :topic="topic" :show-categories="showCategories" :show-hot="showHot" :show-pinned="showPinned" compact :show-stats="false" class="pointer-events-none" />
        <!-- stretched-link：整卡可点，互动按钮浮于遮罩之上，避免嵌套交互元素 -->
        <a
          :href="topic.url"
          class="absolute inset-0 outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
          :aria-label="topic.title"
        />
        <TopicCardActions
          :topic="topic"
          class="relative px-3.5 pb-3"
          :interaction="interactions.get(topic.id)"
        />
      </div>
    </div>
  </template>
  <slot name="empty" />
</template>

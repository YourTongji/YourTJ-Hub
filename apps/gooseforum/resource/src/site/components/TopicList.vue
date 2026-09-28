<script setup lang="ts">
import { computed, onBeforeUnmount, ref, shallowReactive, useId, useSlots, watch } from 'vue'
import { ChevronDown } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import type { TopicPayload } from '@gooseforum/client'
import { createTopicCardInteraction, type TopicCardInteraction } from '@/site/utils/topic-card-interactions'
import TopicCardActions from '@/site/components/TopicCardActions.vue'
import TopicFeedPreview from '@/site/components/TopicFeedPreview.vue'
import TopicRow from '@/site/components/TopicRow.vue'
import { topicDisplayLabel } from '@/runtime/topic-description'
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
const pinnedTopicsId = `gf-pinned-topics-${useId()}`
const pinnedTopicsExpanded = ref(false)
const pinnedTopics = computed(() => props.home && props.showPinned ? props.topics.filter(topic => topic.pinWeight > 0) : [])
const visiblePinnedTopics = computed(() => pinnedTopicsExpanded.value ? pinnedTopics.value.slice(1) : [])
const regularTopics = computed(() => pinnedTopics.value.length
  ? props.topics.filter(topic => topic.pinWeight <= 0)
  : props.topics)

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
  <section
    v-if="pinnedTopics.length"
    :aria-label="t('topicList.pinnedTopics')"
    class="mx-4 mt-3 overflow-hidden rounded-xl border border-primary/15 bg-primary/[0.025]"
    :class="feedMode === 'table' ? 'mb-3' : ''"
  >
    <!-- 所有行共用一套内缩：行距边缘由列表层 [&>li] 统一定义，行内 px-1 是 hover 高亮的内衬；改任一处需两处同动，否则首行与展开行错位 -->
    <ul :id="pinnedTopicsId" class="[&>li]:mx-2 sm:[&>li]:mx-3">
      <li>
        <div class="flex min-h-11 items-center gap-2 px-1">
          <span class="shrink-0 rounded bg-primary/10 px-1.5 py-0.5 text-[11px] font-medium text-primary">{{ t('topicList.pinned') }}</span>
          <a
            :href="pinnedTopics[0]!.url"
            :title="topicDisplayLabel(pinnedTopics[0]!.id, pinnedTopics[0]!.title, pinnedTopics[0]!.description)"
            class="flex min-h-11 min-w-0 flex-1 items-center rounded text-sm font-medium text-base-content/90 outline-none hover:text-primary focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary"
          >
            <span class="truncate">{{ topicDisplayLabel(pinnedTopics[0]!.id, pinnedTopics[0]!.title, pinnedTopics[0]!.description) }}</span>
          </a>
          <button
            v-if="pinnedTopics.length > 1"
            type="button"
            class="ml-auto flex min-h-11 shrink-0 items-center gap-1 whitespace-nowrap rounded-md px-1 text-xs tabular-nums text-base-content/60 outline-none transition-colors hover:text-primary focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary motion-reduce:transition-none"
            :aria-expanded="pinnedTopicsExpanded"
            :aria-controls="pinnedTopicsId"
            :aria-label="`${pinnedTopicsExpanded ? t('topicList.collapsePinnedTopics') : t('topicList.expandPinnedTopics')} · ${t('topicList.pinnedTopicsCount', { count: pinnedTopics.length })}`"
            @click="pinnedTopicsExpanded = !pinnedTopicsExpanded"
          >
            {{ t('topicList.pinnedTopicsCount', { count: pinnedTopics.length }) }}
            <ChevronDown class="size-3.5 transition-transform duration-200 motion-reduce:transition-none" :class="{ 'rotate-180': pinnedTopicsExpanded }" aria-hidden="true" />
          </button>
          <span v-else class="ml-auto shrink-0 whitespace-nowrap text-xs tabular-nums text-base-content/60">{{ t('topicList.pinnedTopicsCount', { count: pinnedTopics.length }) }}</span>
        </div>
      </li>
      <TransitionGroup name="pinned-topic">
        <li v-for="topic in visiblePinnedTopics" :key="topic.id" class="grid grid-rows-[1fr]">
          <div class="min-h-0 overflow-hidden">
            <a
              :href="topic.url"
              :title="topicDisplayLabel(topic.id, topic.title, topic.description)"
              class="group flex min-h-11 items-center gap-2.5 rounded-md border-t border-primary/5 px-1 py-2 text-sm outline-none transition-colors hover:bg-primary/5 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary motion-reduce:transition-none"
            >
              <span class="shrink-0 rounded bg-primary/10 px-1.5 py-0.5 text-[11px] font-medium text-primary">{{ t('topicList.pinned') }}</span>
              <span class="min-w-0 flex-1 line-clamp-2 break-words font-medium leading-5 text-base-content/90 group-hover:text-primary">{{ topicDisplayLabel(topic.id, topic.title, topic.description) }}</span>
            </a>
          </div>
        </li>
      </TransitionGroup>
    </ul>
  </section>
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
        v-for="topic in regularTopics"
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
        v-for="topic in regularTopics"
        :key="topic.id"
        class="gf-card group relative overflow-hidden [&_a.gf-topic-chip]:pointer-events-auto [&_a.gf-topic-chip]:relative [&_a.gf-topic-chip]:z-10"
      >
        <TopicFeedPreview :topic="topic" :show-categories="showCategories" :show-hot="showHot" :show-pinned="showPinned" compact :show-stats="false" class="pointer-events-none" />
        <!-- stretched-link：整卡可点，互动按钮浮于遮罩之上，避免嵌套交互元素 -->
        <a
          :href="topic.url"
          class="absolute inset-0 outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
          :aria-label="topicDisplayLabel(topic.id, topic.title, topic.description)"
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

<style scoped>
.pinned-topic-enter-active,
.pinned-topic-leave-active {
  transition: grid-template-rows 200ms ease, opacity 200ms ease;
}

.pinned-topic-enter-from,
.pinned-topic-leave-to {
  grid-template-rows: 0fr;
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .pinned-topic-enter-active,
  .pinned-topic-leave-active {
    transition: none;
  }
}
</style>

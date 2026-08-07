<script setup lang="ts">
import { computed } from 'vue'
import TopicFeedMeta from '@/site/components/TopicFeedMeta.vue'
import type { TopicPayload } from '@gooseforum/client'

const props = withDefaults(defineProps<{
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

const images = computed(() => {
  const all = (props.topic.images ?? []).filter(Boolean)
  // 双图只展示第一张（统一左文右图布局），三图保留网格
  return all.length <= 2 ? all.slice(0, 1) : all.slice(0, 3)
})
const imageCount = computed(() => images.value.length)
const hasImages = computed(() => imageCount.value > 0)
const singleImage = computed(() => imageCount.value === 1)

const gridClass = computed(() => 'grid-cols-3')

const multiImageClass = computed(() => {
  if (props.compact) return 'h-32 rounded-sm sm:h-56'
  return 'h-32 rounded-md sm:h-56'
})

// 单图：左侧文字 + 右侧竖幅图（约 4:3，不被裁扁）
const singleImageClass = computed(() => {
  if (props.compact) return 'h-24 w-28 rounded-md'
  return 'h-40 w-48 rounded-lg sm:h-48 sm:w-60'
})
</script>

<template>
  <div :class="compact ? 'p-3.5' : 'p-4 sm:p-5'">
    <div v-if="singleImage" class="flex gap-3 sm:gap-4">
      <TopicFeedMeta
        :topic="topic"
        :compact="compact"
        :show-categories="showCategories"
        :show-hot="showHot"
        :show-pinned="showPinned"
        class="min-w-0 flex-1"
      />
      <img
        :src="images[0]"
        :alt="topic.title"
        loading="lazy"
        decoding="async"
        class="shrink-0 self-center object-cover ring-1 ring-black/5 dark:ring-white/10"
        :class="singleImageClass"
      />
    </div>

    <template v-else>
      <TopicFeedMeta
        :topic="topic"
        :compact="compact"
        :show-categories="showCategories"
        :show-hot="showHot"
        :show-pinned="showPinned"
      />
      <div v-if="hasImages" class="mt-3 grid gap-1.5" :class="gridClass">
        <img
          v-for="(url, index) in images"
          :key="`${topic.id}-${url}`"
          :src="url"
          :alt="topic.title"
          loading="lazy"
          decoding="async"
          class="min-h-0 w-full object-cover ring-1 ring-black/5 dark:ring-white/10"
          :class="multiImageClass"
        />
      </div>
    </template>
  </div>
</template>

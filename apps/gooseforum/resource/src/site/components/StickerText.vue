<script setup lang="ts">
import { computed } from 'vue'
import { renderableStickerSegments } from '@/site/utils/sticker-token'

// 列表/卡片预览内联渲染 [:sticker:name:]：已解析的渲染为图片，
// 未知/停用 token 丢弃（issue #1304）。属性类场景改用 stripStickerTokens。
const props = withDefaults(defineProps<{
  content: string
  stickerUrls?: Map<string, string>
  // 对齐所在行行高：leading-5 → h-5 w-5，leading-6 → h-6 w-6
  size?: string
}>(), {
  stickerUrls: () => new Map<string, string>(),
  size: 'h-5 w-5',
})

const segments = computed(() => renderableStickerSegments(props.content, props.stickerUrls))
</script>

<template>
  <template v-for="(segment, index) in segments" :key="index">
    <img
      v-if="segment.type === 'sticker'"
      :src="segment.url"
      :alt="`[:sticker:${segment.name}:]`"
      class="inline-block max-w-full object-contain align-middle"
      :class="size"
      loading="lazy"
    /><template v-else>{{ segment.text }}</template>
  </template>
</template>

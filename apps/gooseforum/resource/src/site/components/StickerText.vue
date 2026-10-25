<script setup lang="ts">
import { computed } from 'vue'
import { renderableStickerSegments } from '@/site/utils/sticker-token'

// 列表/卡片预览内联渲染 [:sticker:name:]：已解析的渲染为图片，
// 未知/停用 token 丢弃（issue #1304）；文本段走 TwemojiText 渲染表情。
// 属性类场景改用 stripStickerTokens。
const props = withDefaults(defineProps<{
  content: string
  stickerUrls?: Map<string, string>
  // 默认与帖内正文一致（2.5em，居中于文字行）；调用方可覆盖尺寸类
  size?: string
}>(), {
  stickerUrls: () => new Map<string, string>(),
  size: 'h-[2.5em] w-auto max-w-[5em] align-[-0.87em]',
})

const segments = computed(() => renderableStickerSegments(props.content, props.stickerUrls))
</script>

<template>
  <template v-for="(segment, index) in segments" :key="index">
    <img
      v-if="segment.type === 'sticker'"
      :src="segment.url"
      :alt="`[${segment.name}]`"
      class="inline-block object-contain"
      :class="size"
      loading="lazy"
    /><template v-else><TwemojiText :text="segment.text" /></template>
  </template>
</template>

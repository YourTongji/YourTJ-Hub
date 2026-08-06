<script setup lang="ts">
import { computed } from 'vue'
import { badgeIconURL, badgeTooltip } from '@/site/utils/badge-style'
import type { UserBadgePayload } from '@gooseforum/client'

defineOptions({ inheritAttrs: false })

const props = withDefaults(defineProps<{
  src: string
  alt: string
  size?: 'medium' | 'large'
  badge?: UserBadgePayload | null
  imgClass?: string
}>(), {
  size: 'medium',
  badge: null,
  imgClass: '',
})

const resolvedSrc = computed(() => {
  if (props.size === 'large') return props.src
  return avatarVariantUrl(props.src)
})

function avatarVariantUrl(src: string): string {
  try {
    const url = new URL(src, window.location.origin)
    const staticMatch = url.pathname.match(/^(\/static\/pic\/(?:(?:[1-9]|1[0-2])|default-avatar))\.webp$/)
    if (staticMatch) {
      url.pathname = `${staticMatch[1]}_medium.webp`
      return formatAvatarUrl(src, url)
    }

    const match = url.pathname.match(/^(.*\/)avatar(\.[^/.]+)$/)
    if (!match) return src

    url.pathname = `${match[1]}avatar_medium${match[2]}`
    return formatAvatarUrl(src, url)
  } catch {
    return src
  }
}

function formatAvatarUrl(src: string, url: URL): string {
  if (!src.startsWith('http://') && !src.startsWith('https://')) {
    return `${url.pathname}${url.search}${url.hash}`
  }
  return url.toString()
}
</script>

<template>
  <span v-if="badge" v-bind="$attrs" class="group/avatar relative inline-block shrink-0">
    <img :src="resolvedSrc" :alt="alt" width="96" height="96" decoding="async" class="h-full w-full object-cover" :class="imgClass">
    <span
      class="absolute -bottom-1 -right-1 z-10 flex h-[32%] min-h-5 w-[32%] min-w-5 items-center justify-center rounded-full bg-base-100 p-[3px] shadow-md shadow-black/10 ring-2 ring-line transition-transform duration-150 ease-out hover:scale-110"
      :title="badgeTooltip(badge)"
    >
      <img :src="badgeIconURL(badge)" :alt="badge.name" class="h-full w-full object-contain" />
    </span>
  </span>
  <img v-else v-bind="$attrs" :src="resolvedSrc" :alt="alt" width="96" height="96" decoding="async">
</template>

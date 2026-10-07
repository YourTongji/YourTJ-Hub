<script setup lang="ts">
import { computed } from 'vue'
import { badgeClass, badgeIconURL, badgeTooltip, isSystemBadgeArtwork } from '@/site/utils/badge-style'
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

const systemArtwork = computed(() => !!props.badge && isSystemBadgeArtwork(props.badge))

const resolvedSrc = computed(() => {
  if (props.size === 'large') return props.src
  return avatarVariantUrl(props.src)
})

function avatarVariantUrl(src: string): string {
  try {
    const url = new URL(src, window.location.origin)
    // SVG avatars scale without raster derivatives; persona URLs expose only this route.
    if (/\.svg$/i.test(url.pathname)) return src
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
      :class="['absolute -bottom-1.5 -right-1.5 z-10 flex h-[30%] min-h-3.5 w-[30%] min-w-3.5 items-center justify-center rounded-full shadow-md shadow-black/10 ring-2 transition-transform duration-150 ease-out hover:scale-110', badgeClass(badge.color, badge.level), { 'p-[2px]': !systemArtwork }]"
      :title="badgeTooltip(badge)"
    >
      <!-- 内置图形自带边距，图标区按角标比例缩放，各头像尺寸下占比一致；自定义图形保持 2px 内边距 -->
      <img :src="badgeIconURL(badge)" :alt="badge.name" :class="['object-contain', systemArtwork ? 'h-[88%] w-[88%]' : 'h-full w-full']" />
    </span>
  </span>
  <img v-else v-bind="$attrs" :src="resolvedSrc" :alt="alt" width="96" height="96" decoding="async">
</template>

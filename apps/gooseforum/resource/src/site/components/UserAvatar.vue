<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { badgeClass, badgeIconURL, badgeTooltip, isSystemBadgeArtwork } from '@/site/utils/badge-style'
import { decorationAssetScale, decorationVariables, isSafeDecorationAssetUrl } from '@/runtime/decoration-style'
import type { DecorationStyle, UserBadgePayload } from '@gooseforum/client'

defineOptions({ inheritAttrs: false })

const props = withDefaults(defineProps<{
  src: string
  alt: string
  size?: 'medium' | 'large'
  frame?: DecorationStyle | null
  badge?: UserBadgePayload | null
  imgClass?: string
}>(), {
  size: 'medium',
  badge: null,
  imgClass: '',
})

const frameVariables = computed(() => decorationVariables(props.frame?.kind === 'avatarFrame' ? props.frame : null))
const frameLayout = computed(() => {
  const percent = Math.round(decorationAssetScale(props.frame) * 10000) / 100
  return { ...frameVariables.value, '--avatar-frame-size': `${percent}%`, '--avatar-frame-offset': `${(100 - percent) / 2}%` }
})
const hasFrame = computed(() => Object.keys(frameVariables.value).length > 0)

const frameImageFailed = ref(false)
const frameAsset = computed(() => hasFrame.value && isSafeDecorationAssetUrl(props.frame?.assetUrl) ? props.frame.assetUrl : '')
watch(frameAsset, () => { frameImageFailed.value = false })

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
  <span v-if="badge || hasFrame" v-bind="$attrs" class="group/avatar relative inline-block shrink-0" :style="frameLayout">
    <!-- 图片框与颜色框互斥：有合法图片时只画图片，加载中/失败都不回退配色边框（#1246） -->
    <span v-if="hasFrame && !frameAsset" class="gf-avatar-frame" aria-hidden="true" />
    <img :src="resolvedSrc" :alt="alt" width="96" height="96" decoding="async" class="h-full w-full rounded-[inherit] object-cover" :class="imgClass">
    <img v-if="frameAsset && !frameImageFailed" :src="frameAsset" alt="" aria-hidden="true" class="gf-avatar-frame-image" decoding="async" @error="frameImageFailed = true" />
    <span
      v-if="badge"
      :class="['absolute -bottom-1.5 -right-1.5 z-10 flex h-[30%] min-h-3.5 w-[30%] min-w-3.5 items-center justify-center rounded-full shadow-md shadow-black/10 ring-2 transition-transform duration-150 ease-out hover:scale-110', badgeClass(badge.color, badge.level), { 'p-[2px]': !systemArtwork }]"
      :title="badgeTooltip(badge)"
    >
      <!-- 内置图形自带边距，图标区按角标比例缩放，各头像尺寸下占比一致；自定义图形保持 2px 内边距 -->
      <img :src="badgeIconURL(badge)" :alt="badge.name" :class="['object-contain', systemArtwork ? 'h-[88%] w-[88%]' : 'h-full w-full']" />
    </span>
  </span>
  <img v-else v-bind="$attrs" :src="resolvedSrc" :alt="alt" width="96" height="96" decoding="async">
</template>

<style scoped>
.gf-avatar-frame-image {
  position: absolute;
  width: var(--avatar-frame-size, 125%);
  height: var(--avatar-frame-size, 125%);
  max-width: none;
  left: var(--avatar-frame-offset, -12.5%);
  top: var(--avatar-frame-offset, -12.5%);
  z-index: 1;
  object-fit: contain;
  pointer-events: none;
}
.gf-avatar-frame {
  position: absolute;
  /* 环厚按头像直径的 7.5% 等比缩放（40px 头像正好 3px）：
     固定 3px 会在列表小头像上显得过粗（20px 头像=15%）、在大头像上细到看不见（108px=2.8%）。
     padding 百分比按包含块宽度解析，配合 mask 的 padding-box ⊕ border-box 只留环带。 */
  inset: -7.5%;
  padding: 7.5%;
  border-radius: inherit;
  background: var(--decoration-light) border-box;
  mask: linear-gradient(#fff 0 0) padding-box, linear-gradient(#fff 0 0);
  mask-composite: exclude;
  pointer-events: none;
}
:global([data-theme="gf-dark"] .gf-avatar-frame) { background-image: var(--decoration-dark); }
@media (forced-colors: active) {
  /* 强制色彩下图片框仍显示图片；纯色框退化为固定 3px 系统色描边（高对比度下不追求等比） */
  .gf-avatar-frame { background: none; border: 3px solid CanvasText; padding: 0; mask: none; }
}
</style>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { LinkPreview } from '@gooseforum/client'
import { safeUrl } from '@/runtime/safe-url'

const props = defineProps<{
  preview: LinkPreview
}>()

const { t } = useI18n()
const imageFailed = ref(false)
const faviconFailed = ref(false)
const href = computed(() => safeUrl(props.preview.url, 'site-link'))
const imageUrl = computed(() => safeUrl(props.preview.imageUrl, 'image'))
const faviconUrl = computed(() => safeUrl(props.preview.faviconUrl, 'image'))
const hasCover = computed(() => Boolean(imageUrl.value) && !imageFailed.value)

/**
 * 校园网卡片由服务端按部署配置本地渲染，配置没给名字时 title / description 为空：
 * 兜底文案必须由客户端按语言出，服务端不留任何中文（issue #729）。
 */
const titleText = computed(() => props.preview.title?.trim()
  || (props.preview.campus ? t('linkPreview.campusFallbackTitle') : ''))
const descriptionText = computed(() => props.preview.description?.trim()
  || (props.preview.campus ? t('linkPreview.campusFallbackDescription') : ''))

/**
 * 导轨高度按封面自身比例走，而不是钉死成接近正方形。
 *
 * 实测：GitHub 的 OG 图是 1200×600（2:1），而原先的导轨是 168×134.7（1.247），
 * 配 `object-fit: cover` 会横向裁掉 37.6%，用户看到的就是「封面没显示全」。
 * 现在加载完成后按真实比例定高（钳制在 1.2–2.4：下限避免方图把卡片撑高，
 * 上限避免超宽图缩成一条），配合 contain 从结构上保证永不裁切。
 * 1.91:1 是 OG 最常见的比例，加载完成前先拿它占位，避免先按原始比例撑高一帧。
 */
const DEFAULT_COVER_ASPECT = 1.91
const MIN_COVER_ASPECT = 1.2
const MAX_COVER_ASPECT = 2.4
const coverAspect = ref(DEFAULT_COVER_ASPECT)
const coverImageElement = ref<HTMLImageElement | null>(null)

/**
 * 从已解码的图片同步封面比例。除了 `@load`，也在封面出现后主动查一次：
 * 图片命中缓存时 load 可能早于监听器挂上，那时永远不会收到事件。
 */
function syncCoverAspect() {
  const image = coverImageElement.value
  if (!image?.naturalWidth || !image.naturalHeight) return
  const ratio = image.naturalWidth / image.naturalHeight
  coverAspect.value = Math.min(MAX_COVER_ASPECT, Math.max(MIN_COVER_ASPECT, ratio))
}

watch(hasCover, async () => {
  await nextTick()
  syncCoverAspect()
})

/** 来源名优先站点名，缺失时回退 host（与后端 SiteName 的 fallback 一致）。 */
const sourceLabel = computed(() => props.preview.siteName?.trim() || props.preview.displayHost?.trim() || '')

/**
 * #733 要求离开前能看到真实 host。只有当它与来源行是同一个字符串时（后端
 * SiteName 回退成 host 的站点）才省略，避免同一行信息打印两遍。
 */
const hostLabel = computed(() => {
  const host = props.preview.displayHost?.trim() ?? ''
  return host.toLowerCase() === sourceLabel.value.toLowerCase() ? '' : host
})
</script>

<template>
  <a
    v-if="href"
    :href="href"
    :data-no-external-guard="preview.kind === 'internal' ? '' : undefined"
    class="gf-link-preview-card not-prose relative my-3 block min-w-0 overflow-hidden rounded-box border border-line/80 bg-base-100 text-left text-base-content no-underline transition-[color,background-color,border-color,transform] duration-150 ease-out hover:border-primary/40 hover:bg-base-300/40 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary active:scale-[0.96] motion-reduce:transition-none"
  >
    <!--
      右侧内边距按“有没有封面”切换，给绝对定位的封面让位：<640px 是
      14 + 56 + 12 = 82px，≥640px 是 168 + 16 = 184px。
    -->
    <span
      class="gf-link-preview-card__body block min-w-0 py-3.5 pl-3.5 sm:py-4 sm:pl-4"
      :class="hasCover ? 'pr-[5.125rem] sm:pr-[11.5rem]' : 'pr-3.5 sm:pr-4'"
    >
      <span class="gf-link-preview-card__source flex min-w-0 items-center gap-2 text-xs text-base-content/60">
        <img
          v-if="faviconUrl && !faviconFailed"
          :src="faviconUrl"
          alt=""
          class="gf-link-preview-card__favicon h-4 w-4 shrink-0"
          loading="lazy"
          referrerpolicy="no-referrer"
          @error="faviconFailed = true"
        >
        <span class="truncate">{{ sourceLabel }}</span>
      </span>
      <strong class="gf-link-preview-card__title mt-1 line-clamp-2 text-sm font-semibold leading-5 text-base-content">
        {{ titleText }}
      </strong>
      <!--
        描述按可用宽度分三级：<480px 不渲染（一行中文描述的信息量接近零，
        却要占掉小卡化最缺的垂直空间），480–639px 一行，≥640px 两行。
        可见性必须放在外层 span：Tailwind 的 hidden/block 与 line-clamp-* 都会
        写 display，挂在同一个元素上按源序互相覆盖，line-clamp 立即失效
        （线上表现：描述完整展开成 26 行，卡片高 692px）。
      -->
      <span v-if="descriptionText" class="gf-link-preview-card__desc mt-3 hidden min-[480px]:block">
        <span class="gf-link-preview-card__desc-text line-clamp-1 text-[13px] leading-5 text-base-content/65 min-[640px]:line-clamp-2">
          {{ descriptionText }}
        </span>
      </span>
      <span v-if="hostLabel" class="gf-link-preview-card__host mt-3 block truncate text-[11px] text-base-content/50">
        {{ hostLabel }}
      </span>
    </span>
    <!--
      封面刻意不参与卡片高度计算：竖版封面若留在文档流里，会按自身比例把整张
      卡片撑高（实测 1:3 源图把卡片从 164px 撑到 362px）。外层 span 绝对定位，
      高度由 aspect-ratio 给出（来自图片自身比例，见 coverAspect），卡片高度就
      只由文字决定；max-h-full 兜住「封面比文字还高」的情况，overflow-hidden
      因此永远不会切到图片。
      <640px：右上角 56px 方形缩略图。这里保留 cover 裁切是刻意的：56px 宽下
      完整显示一张 2:1 封面只有约 28px 高，细缝不可辨认，而居中裁切恰好框住
      站点截图的中部（标题/主视觉），作为「这条链接有配图」的识别信号更有效。
      ≥640px：右侧 168px 缩略图，按图片自身比例定高并垂直居中，contain 保证
      完整显示（原先钉死成 168×134.7 的贴边导轨会把 2:1 封面横向裁掉 37.6%）。
    -->
    <span
      v-if="hasCover"
      class="gf-link-preview-card__cover absolute right-3.5 top-3.5 h-14 w-14 overflow-hidden rounded-md outline outline-1 outline-black/10 sm:right-4 sm:top-1/2 sm:h-auto sm:max-h-full sm:w-[168px] sm:-translate-y-1/2 dark:outline-white/10"
      :style="{ aspectRatio: String(coverAspect) }"
    >
      <img
        ref="coverImageElement"
        :src="imageUrl"
        alt=""
        class="gf-link-preview-card__cover-image h-full w-full"
        loading="lazy"
        referrerpolicy="no-referrer"
        @load="syncCoverAspect"
        @error="imageFailed = true"
      >
    </span>
  </a>
</template>

<style scoped>
/*
 * prose.css 的 `.gf-prose img`（my-2.5 / border / rounded-lg / object-contain）
 * 优先级是 (0,1,1)，高于 Tailwind 工具类 (0,1,0)。它会把封面的 object-cover
 * 盖成 letterbox，并给 16px 的 favicon 加上外边距与边框，把来源行从 16px 顶到
 * 28px。卡片自身带 not-prose，该规则本就不该作用于它；这里用 scoped 属性选择
 * 器 (0,2,0) 就地收回，不改动全局 prose 规则以免影响正文图片。
 */
.gf-link-preview-card__favicon {
  margin: 0;
  border: 0;
  border-radius: 0.25rem;
  object-fit: contain;
}

.gf-link-preview-card__cover-image {
  margin: 0;
  border: 0;
  border-radius: 0;
  /* <640px：方形缩略图靠居中裁切取站点截图的中部，作为「有配图」的识别信号。 */
  object-fit: cover;
}

/* ≥640px：缩略图按图片自身比例定高（见 coverAspect），contain 从结构上
   保证任何比例的封面都完整显示，不会被裁掉两端。 */
@media (min-width: 640px) {
  .gf-link-preview-card__cover-image {
    object-fit: contain;
  }
}
</style>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import logoMotionUrl from '@/site/assets/logo-motion-loader.svg?url'

/**
 * 品牌 logo 循环加载动画：以 <img> 引用循环版 logo_motion SVG
 * （site/assets/logo-motion-loader.svg，由 logo_motion_package/exports/logo_motion.svg
 * 派生：入场编排只播一次并定格组装完成姿态，外层包裹叠加无限循环的轻微呼吸缩放——
 * 品牌标识在整个等待期间保持完整可见且有生命感）。
 *
 * 选型说明：动画 SVG 相比 WebM/MP4/GIF 透明、体积小（~26KB）、任意尺寸清晰、
 * 无解码开销；且样式隔离在图片文档内，不会泄漏到页面其它 SVG。
 * prefers-reduced-motion 时资产内部自行停帧为静态 logo（可访问性契约）。
 *
 * 适用页面/区块级加载态（列表加载、编辑器初始化、内容面板等）；
 * 按钮内联等高频小尺寸场景仍用 Loader2 spinner（动效克制 + logo 细节在
 * <24px 下不可读），请勿在用户头像照片等第三方内容之上叠加品牌动画。
 */
const props = withDefaults(
  defineProps<{
    /** 渲染边长（px）；logo 细节较多，建议 ≥24 */
    size?: number
    /** 屏幕阅读器文案；留空复用通用「加载中」 */
    label?: string
  }>(),
  { size: 24, label: '' },
)

const { t } = useI18n()
const accessibleLabel = computed(() => props.label || t('common.loadingShort'))
</script>

<template>
  <span
    class="logo-loader"
    role="status"
    :aria-label="accessibleLabel"
    :style="{ width: `${props.size}px`, height: `${props.size}px` }"
  >
    <img :src="logoMotionUrl" alt="" class="logo-loader__img" draggable="false" decoding="async" />
  </span>
</template>

<style scoped>
.logo-loader {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.logo-loader__img {
  width: 100%;
  height: 100%;
  user-select: none;
}
</style>

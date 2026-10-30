<!--
  横向滚动条带（表情面板页签、分类 chips）：参考 Telegram Web / bilibili 的表情面板，
  - 竖向滚轮在条带上转为横向滚动（到头后交回页面滚动，不吞掉滚轮）；
  - 还有隐藏内容的一侧淡出边缘，提示可以继续滚动；
  - 鼠标悬停时两侧出现翻页按钮，每次翻过约 3/4 个可视宽度；
  - 当前项（aria-pressed="true"）变化时自动滚入视野。
  触屏照常原生横滑；键盘用户 Tab 逐项聚焦时浏览器自行滚动，翻页按钮不进 Tab 序列。
-->
<script setup lang="ts">
import { ChevronLeft, ChevronRight } from '@lucide/vue'
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  /** 当前项标识：变化后把 aria-pressed="true" 的子项滚入视野 */
  active?: unknown
}>()

const EDGE_EPSILON = 1
const LINE_HEIGHT_PX = 16

const track = ref<HTMLElement | null>(null)
const canBack = ref(false)
const canForward = ref(false)
let resizeObserver: ResizeObserver | null = null
let mutationObserver: MutationObserver | null = null

function isRtl(element: HTMLElement) {
  return getComputedStyle(element).direction === 'rtl'
}

/** 起始侧（LTR 左、RTL 右）已滚动距离；RTL 下 scrollLeft 为 0 到负值 */
function scrolledFromStart(element: HTMLElement) {
  return Math.abs(element.scrollLeft)
}

function syncEdges() {
  const element = track.value
  if (!element) return
  const max = element.scrollWidth - element.clientWidth
  const scrolled = scrolledFromStart(element)
  canBack.value = scrolled > EDGE_EPSILON
  canForward.value = max - scrolled > EDGE_EPSILON
}

function page(direction: 1 | -1) {
  const element = track.value
  if (!element) return
  const distance = Math.max(element.clientWidth * 0.75, 48) * direction * (isRtl(element) ? -1 : 1)
  element.scrollBy({ left: distance, behavior: 'smooth' })
}

function onWheel(event: WheelEvent) {
  const element = track.value
  if (!element || event.ctrlKey) return
  // 触控板横向手势本身就是横向滚动，交给浏览器
  if (Math.abs(event.deltaX) >= Math.abs(event.deltaY)) return
  const max = element.scrollWidth - element.clientWidth
  if (max <= EDGE_EPSILON) return
  const delta = event.deltaMode === WheelEvent.DOM_DELTA_LINE ? event.deltaY * LINE_HEIGHT_PX : event.deltaY
  const forward = delta > 0
  const scrolled = scrolledFromStart(element)
  // 已到这一侧尽头：放行，让外层（面板网格/页面）继续滚动
  if (forward ? max - scrolled <= EDGE_EPSILON : scrolled <= EDGE_EPSILON) return
  event.preventDefault()
  element.scrollLeft += isRtl(element) ? -delta : delta
}

/** 当前项滚入视野：只动条带自身的 scrollLeft，不让外层页面跟着竖向滚动 */
function revealActive(behavior: ScrollBehavior = 'smooth') {
  const element = track.value
  const item = element?.querySelector<HTMLElement>('[aria-pressed="true"]')
  if (!element || !item) return
  const trackBox = element.getBoundingClientRect()
  const itemBox = item.getBoundingClientRect()
  const inset = 24
  if (itemBox.left < trackBox.left + inset) {
    element.scrollBy({ left: itemBox.left - trackBox.left - inset, behavior })
  } else if (itemBox.right > trackBox.right - inset) {
    element.scrollBy({ left: itemBox.right - trackBox.right + inset, behavior })
  }
}

watch(() => props.active, () => {
  void nextTick(() => revealActive())
})

onMounted(() => {
  const element = track.value
  if (!element) return
  element.addEventListener('wheel', onWheel, { passive: false })
  resizeObserver = new ResizeObserver(syncEdges)
  resizeObserver.observe(element)
  // 子项增删（订阅/移除表情包、分类加载完成）改变 scrollWidth，但不一定改变容器尺寸
  mutationObserver = new MutationObserver(syncEdges)
  mutationObserver.observe(element, { childList: true, subtree: true })
  syncEdges()
  revealActive('auto')
})

onBeforeUnmount(() => {
  track.value?.removeEventListener('wheel', onWheel)
  resizeObserver?.disconnect()
  mutationObserver?.disconnect()
})
</script>

<template>
  <div class="gf-scroll-strip" :data-back="canBack || undefined" :data-forward="canForward || undefined">
    <div ref="track" v-bind="$attrs" class="gf-scroll-strip-track" @scroll.passive="syncEdges">
      <slot />
    </div>
    <button
      v-if="canBack"
      type="button"
      class="gf-scroll-strip-arrow gf-scroll-strip-arrow-back"
      tabindex="-1"
      aria-hidden="true"
      @pointerdown.prevent
      @click="page(-1)"
    >
      <ChevronLeft class="gf-scroll-strip-icon" />
    </button>
    <button
      v-if="canForward"
      type="button"
      class="gf-scroll-strip-arrow gf-scroll-strip-arrow-forward"
      tabindex="-1"
      aria-hidden="true"
      @pointerdown.prevent
      @click="page(1)"
    >
      <ChevronRight class="gf-scroll-strip-icon" />
    </button>
  </div>
</template>

<style>
.gf-scroll-strip {
  --gf-scroll-strip-fade: 24px;
  position: relative;
  min-width: 0;
}

.gf-scroll-strip-track {
  display: flex;
  overflow-x: auto;
  overscroll-behavior-x: contain;
  scrollbar-width: none;
  /* 有隐藏内容的一侧淡出：两侧各自独立，滚到头的一侧恢复实边 */
  --gf-scroll-strip-start: 0px;
  --gf-scroll-strip-end: 0px;
  -webkit-mask-image: linear-gradient(to right, transparent 0, #000 var(--gf-scroll-strip-start), #000 calc(100% - var(--gf-scroll-strip-end)), transparent 100%);
  mask-image: linear-gradient(to right, transparent 0, #000 var(--gf-scroll-strip-start), #000 calc(100% - var(--gf-scroll-strip-end)), transparent 100%);
}

.gf-scroll-strip-track::-webkit-scrollbar {
  display: none;
}

.gf-scroll-strip[data-back] > .gf-scroll-strip-track {
  --gf-scroll-strip-start: var(--gf-scroll-strip-fade);
}

.gf-scroll-strip[data-forward] > .gf-scroll-strip-track {
  --gf-scroll-strip-end: var(--gf-scroll-strip-fade);
}

/* RTL 的起始侧在右：渐变方向翻转，start/end 变量保持逻辑含义 */
[dir='rtl'] .gf-scroll-strip-track {
  -webkit-mask-image: linear-gradient(to left, transparent 0, #000 var(--gf-scroll-strip-start), #000 calc(100% - var(--gf-scroll-strip-end)), transparent 100%);
  mask-image: linear-gradient(to left, transparent 0, #000 var(--gf-scroll-strip-start), #000 calc(100% - var(--gf-scroll-strip-end)), transparent 100%);
}

/* 翻页按钮只给能悬停的指针设备：触屏直接横滑，不额外占位 */
.gf-scroll-strip-arrow {
  position: absolute;
  top: 50%;
  display: none;
  align-items: center;
  justify-content: center;
  width: 1.5rem;
  height: 1.5rem;
  border-radius: 999px;
  background: var(--gf-color-base-100);
  color: var(--gf-color-base-content);
  box-shadow:
    0 0 0 1px color-mix(in oklch, var(--gf-color-base-content) 10%, transparent),
    0 2px 6px color-mix(in oklch, var(--gf-color-base-content) 14%, transparent);
  opacity: 0;
  translate: 0 -50%;
  transition: opacity 0.15s ease-out, scale 0.15s ease-out;
}

.gf-scroll-strip-arrow-back {
  inset-inline-start: 2px;
}

.gf-scroll-strip-arrow-forward {
  inset-inline-end: 2px;
}

[dir='rtl'] .gf-scroll-strip-icon {
  scale: -1 1;
}

.gf-scroll-strip-icon {
  width: 0.875rem;
  height: 0.875rem;
}

@media (hover: hover) and (pointer: fine) {
  .gf-scroll-strip-arrow {
    display: inline-flex;
  }

  .gf-scroll-strip:hover > .gf-scroll-strip-arrow {
    opacity: 1;
  }

  .gf-scroll-strip-arrow:active {
    scale: 0.96;
  }
}

@media (prefers-reduced-motion: reduce) {
  .gf-scroll-strip-arrow {
    transition: none;
  }
}
</style>

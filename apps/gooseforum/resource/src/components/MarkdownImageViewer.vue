<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { ChevronLeft, ChevronRight, Maximize2, Minimize2, X } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { FocusScope } from 'reka-ui'
import { decideHorizontalSwipe, IDENTITY_ZOOM, zoomViewAt, clampZoomView, type ZoomView } from '@/runtime/image-gestures'

export interface MarkdownPreviewImage {
  src: string
  alt: string
}

const { t } = useI18n()
const images = ref<MarkdownPreviewImage[]>([])
const currentIndex = ref(0)
const view = ref<ZoomView>(IDENTITY_ZOOM)
const isZoomed = computed(() => view.value.scale > 1)
/** 拖动 / 双指缩放进行中：跟手，不走过渡 */
const gesturing = ref(false)
const stage = ref<HTMLElement | null>(null)
const picture = ref<HTMLImageElement | null>(null)
const imageTransform = computed(() => `translate3d(${view.value.x}px, ${view.value.y}px, 0) scale(${view.value.scale})`)

const viewerOpen = computed(() => images.value.length > 0)
const currentImage = computed(() => images.value[currentIndex.value])
const hasMultipleImages = computed(() => images.value.length > 1)

let lastBodyOverflow = ''
let bodyScrollLocked = false

let touchStartX = 0
let touchStartY = 0
let touchStartTime = 0
/** 本次触摸出现过双指：结束时不再判定为翻页滑动 */
let multiTouch = false

const pointers = new Map<number, { x: number; y: number }>()
let gestureStart: { view: ZoomView; x: number; y: number; distance: number } | null = null
/** 指针移动超过阈值后，紧随的 click 不再切换缩放 */
let suppressClick = false
const DRAG_SLOP_PX = 4

function open(nextImages: MarkdownPreviewImage[], index: number) {
  const normalizedImages = nextImages.filter((image) => image.src)
  if (!normalizedImages.length) return
  suppressClick = false

  images.value = normalizedImages
  currentIndex.value = Math.max(0, Math.min(index, normalizedImages.length - 1))
  view.value = IDENTITY_ZOOM
  lockBodyScroll()
  void nextTick(() => {
    if (viewerOpen.value) window.addEventListener('keydown', handleKeydown, true)
  })
}

function close() {
  if (!viewerOpen.value) return
  images.value = []
  currentIndex.value = 0
  view.value = IDENTITY_ZOOM
  pointers.clear()
  gestureStart = null
  gesturing.value = false
  suppressClick = false
  window.removeEventListener('keydown', handleKeydown, true)
  unlockBodyScroll()
}

function showPrevious() {
  if (!hasMultipleImages.value) return
  currentIndex.value = currentIndex.value <= 0 ? images.value.length - 1 : currentIndex.value - 1
  view.value = IDENTITY_ZOOM
}

function showNext() {
  if (!hasMultipleImages.value) return
  currentIndex.value = currentIndex.value >= images.value.length - 1 ? 0 : currentIndex.value + 1
  view.value = IDENTITY_ZOOM
}

function goToImage(index: number) {
  if (index >= 0 && index < images.value.length) {
    currentIndex.value = index
    view.value = IDENTITY_ZOOM
  }
}

function sizes() {
  const stageEl = stage.value
  const img = picture.value
  if (!stageEl || !img) return null
  const rect = stageEl.getBoundingClientRect()
  return {
    content: { width: img.offsetWidth, height: img.offsetHeight },
    stage: { width: rect.width, height: rect.height },
    center: { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 },
  }
}

function zoomTo(scale: number, clientX?: number, clientY?: number, from = view.value) {
  const measured = sizes()
  if (!measured) return
  const point = clientX === undefined || clientY === undefined
    ? { x: 0, y: 0 }
    : { x: clientX - measured.center.x, y: clientY - measured.center.y }
  view.value = zoomViewAt(from, scale, point, measured.content, measured.stage)
}

/** 单击 / 双击 / 按 0：适应屏幕 ↔ 2 倍（以点击处为中心） */
function toggleZoom(event?: MouseEvent) {
  if (suppressClick) {
    suppressClick = false
    return
  }
  if (isZoomed.value) view.value = IDENTITY_ZOOM
  else zoomTo(2, event?.clientX, event?.clientY)
}

function handleWheel(event: WheelEvent) {
  const delta = event.deltaMode === 1 ? event.deltaY * 16 : event.deltaY
  zoomTo(view.value.scale * Math.exp(-delta * 0.0015), event.clientX, event.clientY)
}

function handlePointerDown(event: PointerEvent) {
  if (event.pointerType === 'mouse' && event.button !== 0) return
  // 新的一次按下：上次拖动/双指缩放留下的标记作废（触屏结束后不一定有 click 来消费它）
  if (pointers.size === 0) suppressClick = false
  pointers.set(event.pointerId, { x: event.clientX, y: event.clientY })
  if (pointers.size === 2) multiTouch = true
  if (!isZoomed.value && pointers.size < 2) return
  ;(event.currentTarget as HTMLElement).setPointerCapture?.(event.pointerId)
  startGesture()
}

function startGesture() {
  const points = [...pointers.values()]
  const [a, b] = points
  if (!a) return
  gestureStart = {
    view: view.value,
    x: b ? (a.x + b.x) / 2 : a.x,
    y: b ? (a.y + b.y) / 2 : a.y,
    distance: b ? Math.hypot(a.x - b.x, a.y - b.y) : 0,
  }
}

function handlePointerMove(event: PointerEvent) {
  if (!pointers.has(event.pointerId)) return
  pointers.set(event.pointerId, { x: event.clientX, y: event.clientY })
  const start = gestureStart
  const measured = sizes()
  if (!start || !measured) return
  const [a, b] = [...pointers.values()]
  const x = b ? (a.x + b.x) / 2 : a.x
  const y = b ? (a.y + b.y) / 2 : a.y
  if (Math.hypot(x - start.x, y - start.y) > DRAG_SLOP_PX) suppressClick = true
  gesturing.value = true
  let next = start.view
  if (b && start.distance > 0) {
    suppressClick = true
    const point = { x: start.x - measured.center.x, y: start.y - measured.center.y }
    next = zoomViewAt(start.view, start.view.scale * (Math.hypot(a.x - b.x, a.y - b.y) / start.distance), point, measured.content, measured.stage)
  }
  view.value = clampZoomView({ ...next, x: next.x + x - start.x, y: next.y + y - start.y }, measured.content, measured.stage)
}

function handlePointerUp(event: PointerEvent) {
  if (!pointers.delete(event.pointerId)) return
  gesturing.value = false
  // 双指松开一指：余下那根手指接着拖动
  if (pointers.size && isZoomed.value) startGesture()
  else gestureStart = null
}

function handleKeydown(event: KeyboardEvent) {
  if (!viewerOpen.value) return
  if (event.key === 'Escape') {
    // 捕获阶段拦下：外层模态（reka Dialog 在 window 冒泡阶段监听）不随之关闭
    event.preventDefault()
    event.stopPropagation()
    close()
    return
  }
  if (event.key === 'ArrowLeft') {
    event.preventDefault()
    showPrevious()
    return
  }
  if (event.key === 'ArrowRight') {
    event.preventDefault()
    showNext()
    return
  }
  if (event.key === '0') {
    event.preventDefault()
    toggleZoom()
    return
  }
  if (event.key === '+' || event.key === '=' || event.key === '-') {
    event.preventDefault()
    zoomTo(view.value.scale * (event.key === '-' ? 0.8 : 1.25))
  }
}

function handleTouchStart(e: TouchEvent) {
  if (e.touches.length === 1) multiTouch = false
  if (isZoomed.value || !e.touches[0]) return
  touchStartX = e.touches[0].clientX
  touchStartY = e.touches[0].clientY
  touchStartTime = Date.now()
}

function handleTouchEnd(e: TouchEvent) {
  if (isZoomed.value || multiTouch || !e.changedTouches[0]) return
  const direction = decideHorizontalSwipe(touchStartX, touchStartY, touchStartTime, e.changedTouches[0])
  if (direction === 'left') showNext()
  if (direction === 'right') showPrevious()
}

function lockBodyScroll() {
  if (typeof document === 'undefined') return
  if (document.body.style.overflow === 'hidden') return
  lastBodyOverflow = document.body.style.overflow
  document.body.style.overflow = 'hidden'
  bodyScrollLocked = true
}

function unlockBodyScroll() {
  if (typeof document === 'undefined') return
  if (!bodyScrollLocked) return
  document.body.style.overflow = lastBodyOverflow
  lastBodyOverflow = ''
  bodyScrollLocked = false
}

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown, true)
  unlockBodyScroll()
})

defineExpose({
  open,
  close,
})
</script>

<template>
  <!--
    始终挂到 body：放进带 transform 的对话框里，fixed 会以对话框为包含块并被其 overflow 裁切。
    data-dismissable-layer 让外层 reka 模态把这里的点按视为层内（后出现的层），不会因此关闭；
    模态给 body 设了 pointer-events: none，这里显式恢复。焦点由嵌套的 FocusScope 接管。
  -->
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <FocusScope
        v-if="viewerOpen && currentImage"
        trapped
        loop
        class="fixed inset-0 z-[150] flex flex-col bg-black/92 backdrop-blur-md select-none animate-in fade-in-0 duration-200"
        role="dialog"
        aria-modal="true"
        data-dismissable-layer
        data-test="image-viewer"
        style="pointer-events: auto"
        :aria-label="currentImage.alt || t('common.preview')"
        @touchstart="handleTouchStart"
        @touchend="handleTouchEnd"
      >
        <!-- 顶栏控制栏 -->
        <div class="relative flex h-14 w-full items-center justify-between px-4 sm:px-6 z-20 shrink-0">
          <!-- 页码计数器 -->
          <div
            v-if="hasMultipleImages"
            class="font-mono text-sm font-semibold tracking-wider text-white/80"
          >
            {{ currentIndex + 1 }} / {{ images.length }}
          </div>
          <div v-else class="text-sm font-medium text-white/70 truncate max-w-[60vw]">
            {{ currentImage.alt }}
          </div>

          <!-- 右侧操作区：原图缩放 + 关闭按钮 -->
          <div class="flex items-center gap-2">
            <span
              v-if="isZoomed"
              class="min-w-[3.5ch] text-end text-xs font-medium tabular-nums text-white/70"
            >{{ Math.round(view.scale * 100) }}%</span>
            <button
              type="button"
              class="flex h-9 w-9 items-center justify-center rounded-full bg-white/10 text-white/90 hover:bg-white/20 hover:text-white active:scale-[0.96] transition-[background-color,color,scale] duration-150 cursor-pointer"
              :aria-label="isZoomed ? t('common.preview') : t('image.originalSize')"
              :title="isZoomed ? t('common.preview') : t('image.originalSize')"
              @click.stop="toggleZoom()"
            >
              <Minimize2 v-if="isZoomed" class="h-4.5 w-4.5" />
              <Maximize2 v-else class="h-4.5 w-4.5" />
            </button>

            <button
              type="button"
              class="flex h-9 w-9 items-center justify-center rounded-full bg-white/10 text-white hover:bg-white/20 active:scale-[0.96] transition-[background-color,color,scale] duration-150 cursor-pointer"
              :aria-label="t('common.close')"
              :title="t('common.close')"
              @click="close"
            >
              <X class="h-5 w-5" />
            </button>
          </div>
        </div>

        <!-- 主画廊视图内容区 -->
        <!-- 舞台：滚轮 / 双指缩放（以指针处为中心），放大后拖动平移，单击在适应屏幕与 2 倍间切换 -->
        <div
          ref="stage"
          class="relative flex-1 flex items-center justify-center p-2 sm:p-6 overflow-hidden min-h-0 touch-none"
          @click="close"
          @wheel.prevent="handleWheel"
        >
          <img
            ref="picture"
            :key="currentImage.src"
            :src="currentImage.src"
            :alt="currentImage.alt || `Image ${currentIndex + 1}`"
            draggable="false"
            class="max-h-full max-w-full object-contain select-none will-change-transform"
            :class="[
              gesturing ? 'cursor-grabbing' : isZoomed ? 'cursor-grab' : 'cursor-zoom-in',
              gesturing ? '' : 'transition-transform duration-200 ease-[cubic-bezier(0.2,0,0,1)] motion-reduce:transition-none',
            ]"
            :style="{ transform: imageTransform }"
            @click.stop="toggleZoom"
            @pointerdown="handlePointerDown"
            @pointermove="handlePointerMove"
            @pointerup="handlePointerUp"
            @pointercancel="handlePointerUp"
          />

          <!-- 悬浮左翻页按钮 -->
          <button
            v-if="hasMultipleImages && currentIndex > 0"
            type="button"
            class="absolute left-4 top-1/2 -translate-y-1/2 z-20 flex h-10 w-10 sm:h-12 sm:w-12 items-center justify-center rounded-full bg-black/50 text-white/90 hover:bg-black/75 hover:text-white active:scale-[0.96] transition-[background-color,color,scale] duration-150 shadow-md cursor-pointer"
            :aria-label="t('common.previousPage')"
            :title="t('common.previousPage')"
            @click.stop="showPrevious"
          >
            <ChevronLeft class="h-6 w-6 sm:h-7 sm:w-7" />
          </button>

          <!-- 悬浮右翻页按钮 -->
          <button
            v-if="hasMultipleImages && currentIndex < images.length - 1"
            type="button"
            class="absolute right-4 top-1/2 -translate-y-1/2 z-20 flex h-10 w-10 sm:h-12 sm:w-12 items-center justify-center rounded-full bg-black/50 text-white/90 hover:bg-black/75 hover:text-white active:scale-[0.96] transition-[background-color,color,scale] duration-150 shadow-md cursor-pointer"
            :aria-label="t('common.nextPage')"
            :title="t('common.nextPage')"
            @click.stop="showNext"
          >
            <ChevronRight class="h-6 w-6 sm:h-7 sm:w-7" />
          </button>
        </div>

        <!-- 底栏缩略图序列（多图时渲染，支持快捷点选切图） -->
        <div
          v-if="hasMultipleImages"
          class="h-16 shrink-0 flex items-center justify-center gap-2 px-4 overflow-x-auto z-20 bg-black/40 backdrop-blur-xs"
        >
          <button
            v-for="(img, idx) in images"
            :key="idx"
            type="button"
            class="h-10 w-10 shrink-0 overflow-hidden rounded-lg border-2 transition-[border-color,opacity,scale] duration-150 cursor-pointer"
            :class="idx === currentIndex ? 'border-primary scale-105 shadow-md' : 'border-transparent opacity-50 hover:opacity-80'"
            :aria-label="`${t('common.preview')} ${idx + 1} / ${images.length}`"
            :aria-pressed="idx === currentIndex"
            @click.stop="goToImage(idx)"
          >
            <img :src="img.src" :alt="img.alt" loading="lazy" decoding="async" class="h-full w-full object-cover" />
          </button>
        </div>
      </FocusScope>
    </Transition>
  </Teleport>
</template>

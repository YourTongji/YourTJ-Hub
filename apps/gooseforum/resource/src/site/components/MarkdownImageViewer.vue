<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref } from 'vue'
import { ChevronLeft, ChevronRight, Maximize2, Minimize2, X } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { decideSwipe } from '@/runtime/swipe'

interface MarkdownPreviewImage {
  src: string
  alt: string
}

const { t } = useI18n()
const images = ref<MarkdownPreviewImage[]>([])
const currentIndex = ref(0)
const actualSize = ref(false)
const actualImageWidth = ref<number | null>(null)
const viewerOpen = computed(() => images.value.length > 0)
const currentImage = computed(() => images.value[currentIndex.value])
const hasMultipleImages = computed(() => images.value.length > 1)
let lastBodyOverflow = ''
let bodyScrollLocked = false

const ACTUAL_IMAGE_SCALE = 2

/** 最近一次图片切换方向，用于给滑动过渡动画选择 enter/leave 位移 */
const slideDirection = ref<'next' | 'prev'>('next')

/** 触摸滑动手势状态（仅移动端生效） */
const SWIPE_LOCK_MS = 200
let touchStartX = 0
let touchStartY = 0
let touchTracking = false
let lastSwipeAt = 0
/** 滑动后抑制紧随其后的 click，避免把滑动手势误判为"点击空白关闭" */
let suppressNextClick = false

function open(nextImages: MarkdownPreviewImage[], index: number) {
  const normalizedImages = nextImages.filter((image) => image.src)
  if (!normalizedImages.length) return

  images.value = normalizedImages
  currentIndex.value = Math.max(0, Math.min(index, normalizedImages.length - 1))
  actualSize.value = false
  actualImageWidth.value = null
  lockBodyScroll()
  void nextTick(() => {
    window.addEventListener('keydown', handleKeydown)
  })
}

function close() {
  if (!viewerOpen.value) return
  images.value = []
  currentIndex.value = 0
  actualSize.value = false
  actualImageWidth.value = null
  window.removeEventListener('keydown', handleKeydown)
  unlockBodyScroll()
}

function showPrevious() {
  if (!hasMultipleImages.value) return
  slideDirection.value = 'prev'
  currentIndex.value = currentIndex.value <= 0 ? images.value.length - 1 : currentIndex.value - 1
  actualSize.value = false
  actualImageWidth.value = null
}

function showNext() {
  if (!hasMultipleImages.value) return
  slideDirection.value = 'next'
  currentIndex.value = currentIndex.value >= images.value.length - 1 ? 0 : currentIndex.value + 1
  actualSize.value = false
  actualImageWidth.value = null
}

function toggleActualSize() {
  if (!actualSize.value) {
    updateActualImageWidth()
  }
  actualSize.value = !actualSize.value
}

function updateActualImageWidth(imageElement?: HTMLImageElement) {
  const naturalWidth = imageElement?.naturalWidth
  if (!naturalWidth) return
  actualImageWidth.value = naturalWidth * ACTUAL_IMAGE_SCALE
}

function handleImageLoad(event: Event) {
  const imageElement = event.currentTarget
  if (!(imageElement instanceof HTMLImageElement)) return
  updateActualImageWidth(imageElement)
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
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
    toggleActualSize()
  }
}

function handleTouchStart(event: TouchEvent) {
  const touch = event.touches[0]
  if (!touch) return
  touchStartX = touch.clientX
  touchStartY = touch.clientY
  touchTracking = true
  suppressNextClick = false
}

function handleTouchEnd(event: TouchEvent) {
  const touch = event.changedTouches[0]
  if (!touch || !touchTracking) return

  const swipeDecision = decideSwipe(touchStartX, touchStartY, touch.clientX, touch.clientY)
  touchStartX = 0
  touchStartY = 0
  touchTracking = false
  if (swipeDecision.direction === 'none') return

  // 即使短时间内被节流，也要吞掉浏览器可能合成的 click。
  suppressNextClick = true
  const now = Date.now()
  if (now - lastSwipeAt < SWIPE_LOCK_MS) return
  lastSwipeAt = now

  if (swipeDecision.direction === 'left') showNext()
  else showPrevious()
}

function handleTouchCancel() {
  touchStartX = 0
  touchStartY = 0
  touchTracking = false
  suppressNextClick = false
}

/** 点击 stage 留白（图片之外的空白）时关闭；滑动后忽略紧随的 click */
function handleStageClick() {
  if (suppressNextClick) {
    suppressNextClick = false
    return
  }
  close()
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
  window.removeEventListener('keydown', handleKeydown)
  unlockBodyScroll()
})

defineExpose({
  open,
  close,
})
</script>

<template>
  <Teleport to="body">
    <Transition name="gf-modal">
      <div
        v-if="viewerOpen && currentImage"
        class="gf-markdown-image-viewer fixed inset-0 z-[120] flex items-center justify-center px-3 py-4 backdrop-blur-sm sm:px-6"
        role="dialog"
        aria-modal="true"
        :aria-label="currentImage.alt || t('common.preview')"
        @click.self="close"
      >
        <div
          v-if="hasMultipleImages"
          class="gf-markdown-image-viewer-count absolute left-3 top-3 z-10 rounded-full px-3 py-2 text-xs font-semibold tabular-nums sm:left-5 sm:top-5"
        >
          {{ currentIndex + 1 }} / {{ images.length }}
        </div>

        <div class="absolute right-3 top-3 z-10 flex items-center gap-2 sm:right-5 sm:top-5">
          <button
            type="button"
            class="gf-markdown-image-viewer-button inline-flex h-10 w-10 items-center justify-center rounded-full transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            :aria-label="actualSize ? t('common.preview') : t('image.originalSize')"
            :title="actualSize ? t('common.preview') : t('image.originalSize')"
            @click.stop="toggleActualSize"
          >
            <Minimize2 v-if="actualSize" class="h-4 w-4" />
            <Maximize2 v-else class="h-4 w-4" />
            <span class="sr-only">{{ actualSize ? t('common.preview') : t('image.originalSize') }}</span>
          </button>

          <button
            type="button"
            class="gf-markdown-image-viewer-button inline-flex h-10 w-10 items-center justify-center rounded-full transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            :aria-label="t('common.close')"
            :title="t('common.close')"
            @click="close"
          >
            <X class="h-4 w-4" />
            <span class="sr-only">{{ t('common.close') }}</span>
          </button>
        </div>

        <button
          v-if="hasMultipleImages"
          type="button"
          class="gf-markdown-image-viewer-button absolute left-3 top-1/2 z-10 inline-flex h-10 w-10 -translate-y-1/2 items-center justify-center rounded-full transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary sm:left-5"
          :aria-label="t('common.previousPage')"
          :title="t('common.previousPage')"
          @click.stop="showPrevious"
        >
          <ChevronLeft class="h-5 w-5" />
          <span class="sr-only">{{ t('common.previousPage') }}</span>
        </button>

        <div
          class="gf-markdown-image-viewer-stage grid h-full w-full place-items-center overflow-hidden p-0"
          @touchstart.passive="handleTouchStart"
          @touchend.passive="handleTouchEnd"
          @touchcancel.passive="handleTouchCancel"
          @click.self="handleStageClick"
        >
          <Transition :name="slideDirection === 'next' ? 'gf-image-slide-next' : 'gf-image-slide-prev'" mode="out-in">
            <img
              :key="currentImage.src"
              :src="currentImage.src"
              :alt="currentImage.alt"
              class="gf-markdown-image-viewer-image rounded-md object-contain"
              :class="actualSize ? 'gf-markdown-image-viewer-image--actual cursor-zoom-out' : 'cursor-zoom-in'"
              decoding="async"
              :style="actualSize && actualImageWidth ? { width: `${actualImageWidth}px` } : undefined"
              @load="handleImageLoad"
              @click.stop="toggleActualSize"
            >
          </Transition>
        </div>

        <button
          v-if="hasMultipleImages"
          type="button"
          class="gf-markdown-image-viewer-button absolute right-3 top-1/2 z-10 inline-flex h-10 w-10 -translate-y-1/2 items-center justify-center rounded-full transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary sm:right-5"
          :aria-label="t('common.nextPage')"
          :title="t('common.nextPage')"
          @click.stop="showNext"
        >
          <ChevronRight class="h-5 w-5" />
          <span class="sr-only">{{ t('common.nextPage') }}</span>
        </button>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.gf-markdown-image-viewer {
  --gf-image-viewer-backdrop: color-mix(in oklch, var(--gf-color-base-content) 62%, transparent);
  --gf-image-viewer-backdrop-glow: color-mix(in oklch, var(--gf-color-base-100) 16%, transparent);
  background:
    radial-gradient(circle at top, var(--gf-image-viewer-backdrop-glow), transparent 42%),
    var(--gf-image-viewer-backdrop);
}

:global([data-theme="gf-dark"]) .gf-markdown-image-viewer {
  --gf-image-viewer-backdrop: color-mix(in oklch, var(--gf-color-base-200) 82%, transparent);
  --gf-image-viewer-backdrop-glow: color-mix(in oklch, var(--gf-color-base-content) 8%, transparent);
}

.gf-markdown-image-viewer-button {
  border: var(--gf-border) solid color-mix(in oklch, var(--gf-color-line) 76%, transparent);
  background: color-mix(in oklch, var(--gf-color-base-100) 86%, transparent);
  color: color-mix(in oklch, var(--gf-color-base-content) 78%, transparent);
  box-shadow:
    0 16px 36px -26px color-mix(in oklch, var(--gf-color-neutral) 70%, transparent),
    0 2px 10px -8px color-mix(in oklch, var(--gf-color-neutral) 55%, transparent);
}

.gf-markdown-image-viewer-button:hover {
  background: color-mix(in oklch, var(--gf-color-base-200) 90%, var(--gf-color-base-100));
  color: var(--gf-color-base-content);
}

.gf-markdown-image-viewer-count {
  border: var(--gf-border) solid color-mix(in oklch, var(--gf-color-line) 70%, transparent);
  background: color-mix(in oklch, var(--gf-color-base-100) 82%, transparent);
  color: color-mix(in oklch, var(--gf-color-base-content) 72%, transparent);
  box-shadow: 0 12px 30px -24px color-mix(in oklch, var(--gf-color-neutral) 65%, transparent);
  backdrop-filter: blur(8px);
}

.gf-markdown-image-viewer-image {
  display: block;
  width: auto;
  height: auto;
  max-width: calc(100vw - 1.5rem);
  max-height: calc(100dvh - 6rem);
  box-shadow:
    0 28px 80px -36px color-mix(in oklch, var(--gf-color-neutral) 82%, transparent),
    0 10px 32px -24px color-mix(in oklch, var(--gf-color-neutral) 68%, transparent);
}

/* 放大态由脚本按原图宽度设置目标值，CSS 只负责限制视口，避免图片撑满灯箱。 */
.gf-markdown-image-viewer-image--actual {
  max-width: calc(100vw - 1.5rem);
  max-height: calc(100dvh - 3rem);
}

@media (min-width: 640px) {
  .gf-markdown-image-viewer-image {
    max-width: calc(100vw - 7rem);
  }

  .gf-markdown-image-viewer-image--actual {
    max-width: min(calc(100vw - 7rem), 64rem);
  }
}

.gf-markdown-image-viewer-stage {
  /* 纵向滑动交给页面，横向滑动由灯箱切图逻辑处理。 */
  touch-action: pan-y;
}

/* 图片切换的滑动过渡：方向由 slideDirection 决定，进入滑入、离开滑出 */
.gf-markdown-image-viewer-image {
  transition:
    transform 0.22s cubic-bezier(0.22, 1, 0.36, 1),
    opacity 0.22s cubic-bezier(0.22, 1, 0.36, 1);
}

.gf-image-slide-next-enter-from,
.gf-image-slide-prev-leave-to {
  transform: translateX(10%);
  opacity: 0;
}

.gf-image-slide-prev-enter-from,
.gf-image-slide-next-leave-to {
  transform: translateX(-10%);
  opacity: 0;
}

.gf-image-slide-next-enter-to,
.gf-image-slide-prev-enter-to,
.gf-image-slide-next-leave-from,
.gf-image-slide-prev-leave-from {
  transform: translateX(0);
  opacity: 1;
}

@media (prefers-reduced-motion: reduce) {
  .gf-markdown-image-viewer-image {
    transition: none;
  }
}
</style>

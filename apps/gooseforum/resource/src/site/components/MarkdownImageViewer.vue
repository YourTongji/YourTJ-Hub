<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { ChevronLeft, ChevronRight, Maximize2, Minimize2, X } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { decideHorizontalSwipe } from '@/runtime/image-gestures'

export interface MarkdownPreviewImage {
  src: string
  alt: string
}

const { t } = useI18n()
const images = ref<MarkdownPreviewImage[]>([])
const currentIndex = ref(0)
const isZoomed = ref(false)

const viewerOpen = computed(() => images.value.length > 0)
const currentImage = computed(() => images.value[currentIndex.value])
const hasMultipleImages = computed(() => images.value.length > 1)

let lastBodyOverflow = ''
let bodyScrollLocked = false

let touchStartX = 0
let touchStartY = 0
let touchStartTime = 0

function open(nextImages: MarkdownPreviewImage[], index: number) {
  const normalizedImages = nextImages.filter((image) => image.src)
  if (!normalizedImages.length) return

  images.value = normalizedImages
  currentIndex.value = Math.max(0, Math.min(index, normalizedImages.length - 1))
  isZoomed.value = false
  lockBodyScroll()
  void nextTick(() => {
    window.addEventListener('keydown', handleKeydown)
  })
}

function close() {
  if (!viewerOpen.value) return
  images.value = []
  currentIndex.value = 0
  isZoomed.value = false
  window.removeEventListener('keydown', handleKeydown)
  unlockBodyScroll()
}

function showPrevious() {
  if (!hasMultipleImages.value) return
  currentIndex.value = currentIndex.value <= 0 ? images.value.length - 1 : currentIndex.value - 1
  isZoomed.value = false
}

function showNext() {
  if (!hasMultipleImages.value) return
  currentIndex.value = currentIndex.value >= images.value.length - 1 ? 0 : currentIndex.value + 1
  isZoomed.value = false
}

function goToImage(index: number) {
  if (index >= 0 && index < images.value.length) {
    currentIndex.value = index
    isZoomed.value = false
  }
}

function toggleZoom() {
  isZoomed.value = !isZoomed.value
}

function handleKeydown(event: KeyboardEvent) {
  if (!viewerOpen.value) return
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
    toggleZoom()
  }
}

function handleTouchStart(e: TouchEvent) {
  if (isZoomed.value || !e.touches[0]) return
  touchStartX = e.touches[0].clientX
  touchStartY = e.touches[0].clientY
  touchStartTime = Date.now()
}

function handleTouchEnd(e: TouchEvent) {
  if (isZoomed.value || !e.changedTouches[0]) return
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
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="viewerOpen && currentImage"
        class="fixed inset-0 z-[150] flex flex-col bg-black/92 backdrop-blur-md select-none animate-in fade-in-0 duration-200"
        role="dialog"
        aria-modal="true"
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
            <button
              type="button"
              class="flex h-9 w-9 items-center justify-center rounded-full bg-white/10 text-white/90 hover:bg-white/20 hover:text-white active:scale-95 transition-all cursor-pointer"
              :aria-label="isZoomed ? t('common.preview') : t('image.originalSize')"
              :title="isZoomed ? t('common.preview') : t('image.originalSize')"
              @click.stop="toggleZoom"
            >
              <Minimize2 v-if="isZoomed" class="h-4.5 w-4.5" />
              <Maximize2 v-else class="h-4.5 w-4.5" />
            </button>

            <button
              type="button"
              class="flex h-9 w-9 items-center justify-center rounded-full bg-white/10 text-white hover:bg-white/20 active:scale-95 transition-all cursor-pointer"
              :aria-label="t('common.close')"
              :title="t('common.close')"
              @click="close"
            >
              <X class="h-5 w-5" />
            </button>
          </div>
        </div>

        <!-- 主画廊视图内容区 -->
        <div
          class="relative flex-1 flex items-center justify-center p-2 sm:p-6 overflow-hidden min-h-0"
          @click="close"
        >
          <img
            :key="currentImage.src"
            :src="currentImage.src"
            :alt="currentImage.alt || `Image ${currentIndex + 1}`"
            class="max-h-full max-w-full object-contain transition-transform duration-200 ease-out select-none"
            :class="isZoomed ? 'scale-150 cursor-zoom-out' : 'cursor-zoom-in'"
            @click.stop="toggleZoom"
          />

          <!-- 悬浮左翻页按钮 -->
          <button
            v-if="hasMultipleImages && currentIndex > 0"
            type="button"
            class="absolute left-4 top-1/2 -translate-y-1/2 z-20 flex h-10 w-10 sm:h-12 sm:w-12 items-center justify-center rounded-full bg-black/50 text-white/90 hover:bg-black/75 hover:text-white active:scale-90 transition-all shadow-md cursor-pointer"
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
            class="absolute right-4 top-1/2 -translate-y-1/2 z-20 flex h-10 w-10 sm:h-12 sm:w-12 items-center justify-center rounded-full bg-black/50 text-white/90 hover:bg-black/75 hover:text-white active:scale-90 transition-all shadow-md cursor-pointer"
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
            class="h-10 w-10 shrink-0 overflow-hidden rounded-lg border-2 transition-all duration-150 cursor-pointer"
            :class="idx === currentIndex ? 'border-primary scale-105 shadow-md' : 'border-transparent opacity-50 hover:opacity-80'"
            @click.stop="goToImage(idx)"
          >
            <img :src="img.src" :alt="img.alt" loading="lazy" decoding="async" class="h-full w-full object-cover" />
          </button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ChevronLeft, ChevronRight, X, ZoomIn } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { decideHorizontalSwipe, SLIDE_SWIPE_THRESHOLD_PX } from '@/runtime/image-gestures'

const { t } = useI18n()
const props = defineProps<{
  images: string[]
  title?: string
}>()

const currentIndex = ref(0)
const lightboxOpen = ref(false)
const lightboxIndex = ref(0)
const isZoomed = ref(false)
const galleryContainer = ref<HTMLElement | null>(null)

let touchStartX = 0
let touchStartY = 0
let touchStartTime = 0

const currentImage = computed(() => props.images[currentIndex.value] || '')
const currentLightboxImage = computed(() => props.images[lightboxIndex.value] || '')

function nextSlide() {
  if (currentIndex.value < props.images.length - 1) {
    currentIndex.value++
  }
}

function prevSlide() {
  if (currentIndex.value > 0) {
    currentIndex.value--
  }
}

function goToSlide(index: number) {
  if (index >= 0 && index < props.images.length) {
    currentIndex.value = index
  }
}

function openLightbox(index: number) {
  lightboxIndex.value = index
  isZoomed.value = false
  lightboxOpen.value = true
}

function closeLightbox() {
  lightboxOpen.value = false
  isZoomed.value = false
}

function nextLightbox() {
  if (lightboxIndex.value < props.images.length - 1) {
    lightboxIndex.value++
    isZoomed.value = false
  }
}

function prevLightbox() {
  if (lightboxIndex.value > 0) {
    lightboxIndex.value--
    isZoomed.value = false
  }
}

function toggleZoom() {
  isZoomed.value = !isZoomed.value
}

// 触摸手势滑动支持（移动端左右滑动翻页）
function handleTouchStart(e: TouchEvent) {
  if (!e.touches[0]) return
  touchStartX = e.touches[0].clientX
  touchStartY = e.touches[0].clientY
  touchStartTime = Date.now()
}

function handleTouchEnd(e: TouchEvent) {
  if (!e.changedTouches[0]) return
  const direction = decideHorizontalSwipe(touchStartX, touchStartY, touchStartTime, e.changedTouches[0], SLIDE_SWIPE_THRESHOLD_PX)
  if (direction === 'left') nextSlide()
  if (direction === 'right') prevSlide()
}

function handleLightboxTouchStart(e: TouchEvent) {
  if (isZoomed.value || !e.touches[0]) return
  touchStartX = e.touches[0].clientX
  touchStartY = e.touches[0].clientY
  touchStartTime = Date.now()
}

function handleLightboxTouchEnd(e: TouchEvent) {
  if (isZoomed.value || !e.changedTouches[0]) return
  const direction = decideHorizontalSwipe(touchStartX, touchStartY, touchStartTime, e.changedTouches[0])
  if (direction === 'left') nextLightbox()
  if (direction === 'right') prevLightbox()
}

function handleKeyDown(e: KeyboardEvent) {
  if (lightboxOpen.value) {
    if (e.key === 'Escape') closeLightbox()
    if (e.key === 'ArrowLeft') prevLightbox()
    if (e.key === 'ArrowRight') nextLightbox()
    return
  }

  // 焦点在轮播图容器或其子元素时响应键盘方向键
  if (galleryContainer.value && galleryContainer.value.contains(document.activeElement)) {
    if (e.key === 'ArrowLeft') {
      e.preventDefault()
      prevSlide()
    } else if (e.key === 'ArrowRight') {
      e.preventDefault()
      nextSlide()
    }
  }
}

watch(lightboxOpen, (open) => {
  if (typeof document !== 'undefined') {
    if (open) {
      document.body.style.overflow = 'hidden'
    } else {
      document.body.style.overflow = ''
    }
  }
})

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeyDown)
  if (typeof document !== 'undefined') {
    document.body.style.overflow = ''
  }
})
</script>

<template>
  <div v-if="images && images.length > 0" class="w-full mb-4 sm:mb-5">
    <!-- 主轮播视窗容器（自适应多种图片尺寸与高宽比） -->
    <div
      ref="galleryContainer"
      tabindex="0"
      class="group relative w-full h-[300px] xs:h-[340px] sm:h-[400px] md:h-[460px] max-h-[60vh] overflow-hidden rounded-2xl sm:rounded-3xl border border-line/70 bg-base-200/30 shadow-xs outline-none focus-visible:ring-2 focus-visible:ring-primary/40 select-none transition-shadow duration-200"
      @touchstart="handleTouchStart"
      @touchend="handleTouchEnd"
    >
      <!-- 双层背景渲染：当前图片的氛围环境毛玻璃铺底，消除生硬灰边/黑边 -->
      <div class="absolute inset-0 overflow-hidden pointer-events-none">
        <img
          :src="currentImage"
          aria-hidden="true"
          class="h-full w-full object-cover blur-2xl opacity-20 dark:opacity-25 scale-110 transform-gpu transition-all duration-500 ease-out"
        />
        <div class="absolute inset-0 bg-base-100/35 dark:bg-black/25 backdrop-blur-xs" />
      </div>

      <!-- 顶层主图：完整以 object-contain 呈现，100% 不裁切任何长图与超宽图 -->
      <div class="relative h-full w-full flex items-center justify-center p-2 sm:p-3 z-10">
        <img
          :key="currentImage"
          :src="currentImage"
          :alt="title ? `${title} - ${currentIndex + 1}` : `Image ${currentIndex + 1}`"
          :fetchpriority="currentIndex === 0 ? 'high' : 'low'"
          :loading="currentIndex === 0 ? 'eager' : 'lazy'"
          class="h-full w-full max-h-full max-w-full object-contain rounded-xl sm:rounded-2xl drop-shadow-sm cursor-zoom-in transition-all duration-300 ease-out animate-in fade-in-50 zoom-in-95"
          @click="openLightbox(currentIndex)"
        />
      </div>

      <!-- 左上角：放大查看全屏大图控件 -->
      <button
        type="button"
        class="absolute top-3 left-3 z-20 flex h-8 w-8 sm:h-9 sm:w-9 items-center justify-center rounded-full bg-black/40 text-white/90 backdrop-blur-md hover:bg-black/65 hover:text-white active:scale-95 transition-all shadow-xs cursor-pointer"
        :aria-label="t('publish.gallery.zoomIn')"
        @click.stop="openLightbox(currentIndex)"
      >
        <ZoomIn class="h-4 w-4 sm:h-4.5 sm:w-4.5" />
      </button>

      <!-- 右上角：1/N 页码徽章指示器 -->
      <div
        v-if="images.length > 1"
        class="absolute top-3 right-3 z-20 inline-flex items-center rounded-full bg-black/45 px-2.5 py-1 text-[11px] sm:text-xs font-mono font-semibold tracking-wider text-white backdrop-blur-md shadow-xs select-none"
      >
        {{ currentIndex + 1 }}/{{ images.length }}
      </div>

      <button
        v-if="images.length > 1"
        type="button"
        class="absolute left-3 top-1/2 -translate-y-1/2 z-20 flex h-8 w-8 sm:h-9 sm:w-9 items-center justify-center rounded-full bg-black/40 text-white/90 backdrop-blur-md hover:bg-black/65 hover:text-white active:scale-90 transition-all opacity-80 sm:opacity-0 sm:group-hover:opacity-100 disabled:opacity-0 disabled:pointer-events-none shadow-sm cursor-pointer"
        :aria-label="t('common.previousPage')"
        :title="t('common.previousPage')"
        :disabled="currentIndex === 0"
        @click.stop="prevSlide"
      >
        <ChevronLeft class="h-5 w-5" />
      </button>

      <button
        v-if="images.length > 1"
        type="button"
        class="absolute right-3 top-1/2 -translate-y-1/2 z-20 flex h-8 w-8 sm:h-9 sm:w-9 items-center justify-center rounded-full bg-black/40 text-white/90 backdrop-blur-md hover:bg-black/65 hover:text-white active:scale-90 transition-all opacity-80 sm:opacity-0 sm:group-hover:opacity-100 disabled:opacity-0 disabled:pointer-events-none shadow-sm cursor-pointer"
        :aria-label="t('common.nextPage')"
        :title="t('common.nextPage')"
        :disabled="currentIndex === images.length - 1"
        @click.stop="nextSlide"
      >
        <ChevronRight class="h-5 w-5" />
      </button>

      <!-- 底部居中药丸指示条 -->
      <div v-if="images.length > 1" class="absolute bottom-3 inset-x-0 z-20 flex items-center justify-center pointer-events-none">
        <div class="inline-flex items-center gap-1.5 rounded-full bg-black/35 backdrop-blur-md px-2.5 py-1 pointer-events-auto shadow-xs">
          <button
            v-for="(_, idx) in images"
            :key="idx"
            type="button"
            class="h-1.5 rounded-full transition-all duration-200 cursor-pointer"
            :class="idx === currentIndex ? 'w-5 bg-white shadow-xs' : 'w-1.5 bg-white/40 hover:bg-white/70'"
            :aria-label="t('publish.gallery.goToImage', { index: idx + 1 })"
            @click.stop="goToSlide(idx)"
          />
        </div>
      </div>
    </div>

    <!-- 全屏沉浸式画廊模态窗（Lightbox 原图预览） -->
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
          v-if="lightboxOpen"
          class="fixed inset-0 z-[150] flex flex-col bg-black/92 backdrop-blur-md select-none animate-in fade-in-0 duration-200"
          @touchstart="handleLightboxTouchStart"
          @touchend="handleLightboxTouchEnd"
        >
          <!-- 顶栏控制栏 -->
          <div class="relative flex h-14 w-full items-center justify-between px-4 sm:px-6 z-20 shrink-0">
            <!-- 当前页码 -->
            <div class="font-mono text-sm font-semibold tracking-wider text-white/80">
              {{ lightboxIndex + 1 }} / {{ images.length }}
            </div>

            <button
              type="button"
              class="flex h-9 w-9 items-center justify-center rounded-full bg-white/10 text-white hover:bg-white/20 active:scale-95 transition-all cursor-pointer"
              :aria-label="t('common.close')"
              :title="t('common.close')"
              @click="closeLightbox"
            >
              <X class="h-5 w-5" />
            </button>
          </div>

          <!-- 主画廊内容区 -->
          <div
            class="relative flex-1 flex items-center justify-center p-2 sm:p-6 overflow-hidden min-h-0"
            @click="closeLightbox"
          >
            <img
              :key="currentLightboxImage"
              :src="currentLightboxImage"
              :alt="title ? `${title} - ${lightboxIndex + 1}` : `Image ${lightboxIndex + 1}`"
              class="max-h-full max-w-full object-contain transition-transform duration-200 ease-out"
              :class="isZoomed ? 'scale-150 cursor-zoom-out' : 'cursor-zoom-in'"
              @click.stop="toggleZoom"
            />

            <button
              v-if="images.length > 1 && lightboxIndex > 0"
              type="button"
              class="absolute left-4 top-1/2 -translate-y-1/2 z-20 flex h-10 w-10 sm:h-12 sm:w-12 items-center justify-center rounded-full bg-black/50 text-white/90 hover:bg-black/75 hover:text-white active:scale-90 transition-all shadow-md cursor-pointer"
              :aria-label="t('common.previousPage')"
              :title="t('common.previousPage')"
              @click.stop="prevLightbox"
            >
              <ChevronLeft class="h-6 w-6 sm:h-7 sm:w-7" />
            </button>
            <button
              v-if="images.length > 1 && lightboxIndex < images.length - 1"
              type="button"
              class="absolute right-4 top-1/2 -translate-y-1/2 z-20 flex h-10 w-10 sm:h-12 sm:w-12 items-center justify-center rounded-full bg-black/50 text-white/90 hover:bg-black/75 hover:text-white active:scale-90 transition-all shadow-md cursor-pointer"
              :aria-label="t('common.nextPage')"
              :title="t('common.nextPage')"
              @click.stop="nextLightbox"
            >
              <ChevronRight class="h-6 w-6 sm:h-7 sm:w-7" />
            </button>
          </div>

          <!-- 底栏缩略图序列（多图时方便快捷点选） -->
          <div
            v-if="images.length > 1"
            class="h-16 shrink-0 flex items-center justify-center gap-2 px-4 overflow-x-auto z-20 bg-black/40 backdrop-blur-xs"
          >
            <button
              v-for="(img, idx) in images"
              :key="idx"
              type="button"
              class="h-10 w-10 shrink-0 overflow-hidden rounded-lg border-2 transition-all duration-150 cursor-pointer"
              :class="idx === lightboxIndex ? 'border-primary scale-105' : 'border-transparent opacity-50 hover:opacity-80'"
              :aria-label="t('publish.gallery.goToImage', { index: idx + 1 })"
              @click="lightboxIndex = idx"
            >
              <img :src="img" :alt="t('publish.gallery.goToImage', { index: idx + 1 })" loading="lazy" decoding="async" class="h-full w-full object-cover" />
            </button>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

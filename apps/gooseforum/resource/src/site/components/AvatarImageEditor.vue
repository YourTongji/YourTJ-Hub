<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { Image } from '@lucide/vue'
import { useI18n } from 'vue-i18n'

// 知乎式头像编辑器：
// - 居中 1:1 方形预览即最终裁切结果（所见即所得）
// - 拖拽平移 + 滑条/滚轮缩放（cover-fit 为最小，不露白）
// - 输出方图画布（默认 300×300）
const props = withDefaults(
  defineProps<{
    imageUrl: string
    /** 预览舞台边长（px），默认 256 贴近知乎弹窗比例 */
    stageSize?: number
    /** 导出边长（px） */
    outputSize?: number
    saving?: boolean
  }>(),
  {
    stageSize: 256,
    outputSize: 300,
    saving: false,
  },
)

const emit = defineEmits<{
  save: [canvas: HTMLCanvasElement]
  cancel: []
}>()

const { t } = useI18n()
const stageRef = ref<HTMLElement | null>(null)
const imageRef = ref<HTMLImageElement | null>(null)
const transform = reactive({ x: 0, y: 0, scale: 1 })
let baseScale = 1
let dragState: { startX: number; startY: number; originX: number; originY: number } | null = null
let resizeObserver: ResizeObserver | null = null

const imageStyle = computed(() => ({
  transform: `translate(${transform.x}px, ${transform.y}px) scale(${transform.scale})`,
}))

// 滑条：cover-fit 为 1x 最小；最大 3x
const sliderMin = 1
const sliderMax = 3
const sliderValue = computed(() => {
  if (baseScale <= 0) return 1
  return Math.round((transform.scale / baseScale) * 100) / 100
})

function onSliderInput(event: Event) {
  const ratio = Number((event.target as HTMLInputElement).value)
  if (!Number.isFinite(ratio) || baseScale <= 0) return
  zoomToRatio(ratio)
}

function zoomToRatio(ratio: number) {
  const stage = stageRef.value
  const previousScale = transform.scale
  const nextScale = baseScale * ratio
  if (!stage) {
    transform.scale = nextScale
    clampTransform()
    return
  }
  const centerX = stage.clientWidth / 2
  const centerY = stage.clientHeight / 2
  transform.scale = nextScale
  if (previousScale > 0) {
    transform.x = centerX - ((centerX - transform.x) * nextScale) / previousScale
    transform.y = centerY - ((centerY - transform.y) * nextScale) / previousScale
  }
  clampTransform()
}

function resetTransform() {
  const stage = stageRef.value
  const image = imageRef.value
  if (!stage || !image) return
  const stageWidth = stage.clientWidth
  const stageHeight = stage.clientHeight
  if (stageWidth <= 0 || stageHeight <= 0 || !image.naturalWidth) return
  // cover-fit：铺满方形舞台，不露白
  baseScale = Math.max(stageWidth / image.naturalWidth, stageHeight / image.naturalHeight)
  transform.scale = baseScale
  transform.x = (stageWidth - image.naturalWidth * transform.scale) / 2
  transform.y = (stageHeight - image.naturalHeight * transform.scale) / 2
}

function clampTransform() {
  const stage = stageRef.value
  const image = imageRef.value
  if (!stage || !image) return
  const scaledWidth = image.naturalWidth * transform.scale
  const scaledHeight = image.naturalHeight * transform.scale
  if (scaledWidth < stage.clientWidth || scaledHeight < stage.clientHeight) {
    resetTransform()
    return
  }
  transform.x = Math.min(0, Math.max(stage.clientWidth - scaledWidth, transform.x))
  transform.y = Math.min(0, Math.max(stage.clientHeight - scaledHeight, transform.y))
}

function startDrag(event: PointerEvent) {
  if (event.button !== 0) return
  dragState = {
    startX: event.clientX,
    startY: event.clientY,
    originX: transform.x,
    originY: transform.y,
  }
  try {
    ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
  } catch {
    // 合成事件可能没有活动指针
  }
}

function onDrag(event: PointerEvent) {
  if (!dragState) return
  transform.x = dragState.originX + event.clientX - dragState.startX
  transform.y = dragState.originY + event.clientY - dragState.startY
  clampTransform()
}

function endDrag() {
  dragState = null
}

function zoomBy(factor: number) {
  if (baseScale <= 0) return
  const previousScale = transform.scale
  const nextScale = Math.min(baseScale * sliderMax, Math.max(baseScale * sliderMin, previousScale * factor))
  if (nextScale === previousScale) return
  zoomToRatio(nextScale / baseScale)
}

function onWheel(event: WheelEvent) {
  event.preventDefault()
  zoomBy(event.deltaY < 0 ? 1.08 : 1 / 1.08)
}

function zoomIn() {
  zoomBy(1.15)
}

function zoomOut() {
  zoomBy(1 / 1.15)
}

// 保存：把方形舞台当前视图映射到 outputSize×outputSize 画布
function handleSave() {
  const stage = stageRef.value
  const image = imageRef.value
  if (!stage || !image || !image.naturalWidth) return
  const outputSize = props.outputSize
  const canvas = document.createElement('canvas')
  canvas.width = outputSize
  canvas.height = outputSize
  const context = canvas.getContext('2d')
  if (!context) return
  context.imageSmoothingEnabled = true
  context.imageSmoothingQuality = 'high'

  const sourceX = -transform.x / transform.scale
  const sourceY = -transform.y / transform.scale
  const sourceWidth = stage.clientWidth / transform.scale
  const sourceHeight = stage.clientHeight / transform.scale
  context.drawImage(image, sourceX, sourceY, sourceWidth, sourceHeight, 0, 0, outputSize, outputSize)
  emit('save', canvas)
}

function fitImageOnce() {
  const image = imageRef.value
  if (!image) return
  if (image.complete && image.naturalWidth > 0) {
    resetTransform()
  } else {
    image.addEventListener('load', resetTransform, { once: true })
  }
}

function bindResizeObserver() {
  resizeObserver?.disconnect()
  if (!stageRef.value || typeof ResizeObserver === 'undefined') return
  resizeObserver = new ResizeObserver(() => {
    if (imageRef.value?.naturalWidth) {
      clampTransform()
      if (transform.scale < baseScale) resetTransform()
    }
  })
  resizeObserver.observe(stageRef.value)
}

onMounted(async () => {
  await nextTick()
  fitImageOnce()
  bindResizeObserver()
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  resizeObserver = null
})

watch(
  () => props.imageUrl,
  async () => {
    await nextTick()
    fitImageOnce()
  },
)

defineExpose({ resetTransform, save: handleSave })
</script>

<template>
  <div class="avatar-editor">
    <!-- 1:1 预览：所见即所得的最终方图裁切 -->
    <div class="avatar-editor-stage-wrap">
      <div
        ref="stageRef"
        class="avatar-editor-stage"
        :style="{ width: `${stageSize}px`, height: `${stageSize}px` }"
        @wheel="onWheel"
      >
        <img
          ref="imageRef"
          :src="imageUrl"
          :style="imageStyle"
          :alt="t('settings.avatar.cropAlt')"
          class="avatar-editor-image"
          draggable="false"
          @pointerdown="startDrag"
          @pointermove="onDrag"
          @pointerup="endDrag"
          @pointercancel="endDrag"
        />
      </div>
    </div>

    <!-- 缩放滑条：两端用不同尺寸 Image 图标（小图=缩小 / 大图=放大） -->
    <div class="avatar-editor-slider">
      <button
        type="button"
        class="avatar-editor-slider-icon"
        :aria-label="t('settings.avatar.zoomOut')"
        :title="t('settings.avatar.zoomOut')"
        @click="zoomOut"
      >
        <Image class="avatar-editor-slider-glyph avatar-editor-slider-glyph--sm" :stroke-width="2.25" aria-hidden="true" />
      </button>
      <input
        type="range"
        class="avatar-editor-slider-input"
        :min="sliderMin"
        :max="sliderMax"
        step="0.01"
        :value="sliderValue"
        :aria-label="t('settings.avatar.zoomSlider')"
        :aria-valuetext="`${Math.round(sliderValue * 100)}%`"
        @input="onSliderInput"
      />
      <button
        type="button"
        class="avatar-editor-slider-icon"
        :aria-label="t('settings.avatar.zoomIn')"
        :title="t('settings.avatar.zoomIn')"
        @click="zoomIn"
      >
        <Image class="avatar-editor-slider-glyph avatar-editor-slider-glyph--lg" :stroke-width="2.25" aria-hidden="true" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.avatar-editor {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  gap: 1.25rem;
}

/* 知乎预览区外圈浅色衬底，让方图更突出 */
.avatar-editor-stage-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  padding: 1.25rem 0.75rem;
  border-radius: 0.5rem;
  background: color-mix(in oklab, var(--gf-color-base-200) 42%, var(--gf-color-base-100));
}

.avatar-editor-stage {
  position: relative;
  flex: 0 0 auto;
  overflow: hidden;
  border-radius: 0.125rem;
  /* 内框：图片 cover 铺满，不露底 */
  background: var(--gf-color-base-100);
  touch-action: none;
  cursor: grab;
}

.avatar-editor-stage:active {
  cursor: grabbing;
}

.avatar-editor-image {
  position: absolute;
  top: 0;
  left: 0;
  transform-origin: 0 0;
  max-width: none;
  user-select: none;
  -webkit-user-drag: none;
}

.avatar-editor-slider {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.625rem;
  width: 100%;
  max-width: 16rem;
}

.avatar-editor-slider-icon {
  display: inline-flex;
  height: 1.75rem;
  width: 1.75rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  color: var(--gf-color-base-content);
  opacity: 0.88;
  transition: background-color 0.15s, opacity 0.15s, color 0.15s;
  outline: none;
}

.avatar-editor-slider-icon:hover {
  background: color-mix(in oklab, var(--gf-color-base-300) 88%, transparent);
  opacity: 1;
}

.avatar-editor-slider-icon:focus-visible {
  outline: 2px solid color-mix(in oklab, var(--gf-color-primary) 55%, transparent);
  outline-offset: 1px;
}

.avatar-editor-slider-glyph {
  display: block;
  color: inherit;
}

.avatar-editor-slider-glyph--sm {
  width: 0.8125rem;
  height: 0.8125rem;
}

.avatar-editor-slider-glyph--lg {
  width: 1.0625rem;
  height: 1.0625rem;
}

.avatar-editor-slider-input {
  flex: 1 1 auto;
  min-width: 0;
  height: 1.25rem;
  accent-color: var(--gf-color-primary);
  cursor: pointer;
}
</style>

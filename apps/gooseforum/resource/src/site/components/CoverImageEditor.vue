<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { Minus, Plus } from '@lucide/vue'
import { useI18n } from 'vue-i18n'

// 知乎式封面编辑器：
// - 预览区即桌面端最终效果（所见即所得）
// - 拖拽平移 + 滑条/滚轮缩放
// - 预览区左右遮罩标识「移动端会被裁切」的区域
// - floatMode：预览铺满父容器，操作条从预览区底部向下延伸（盖在头像上层，不挤占预览高度）
const props = withDefaults(
  defineProps<{
    imageUrl: string
    aspectRatio?: number
    mobileRatio?: number
    showMobilePreview?: boolean
    floatMode?: boolean
    saving?: boolean
  }>(),
  {
    aspectRatio: 5,
    mobileRatio: 3,
    showMobilePreview: true,
    floatMode: false,
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
const maskSidePx = ref(0)
let baseScale = 1
let dragState: { startX: number; startY: number; originX: number; originY: number } | null = null
let resizeObserver: ResizeObserver | null = null

const aspect = computed(() => props.aspectRatio || 5)
const mobileAspect = computed(() => props.mobileRatio || 3)

const imageStyle = computed(() => ({
  transform: `translate(${transform.x}px, ${transform.y}px) scale(${transform.scale})`,
}))

// 滑条：cover-fit 为最小（1x），不可再缩小露出空白；最大 3x
const sliderMin = 1
const sliderMax = 3
const sliderValue = computed(() => {
  if (baseScale <= 0) return 1
  return Math.round((transform.scale / baseScale) * 100) / 100
})

function onSliderInput(event: Event) {
  const ratio = Number((event.target as HTMLInputElement).value)
  if (!Number.isFinite(ratio) || baseScale <= 0) return
  const stage = stageRef.value
  if (!stage) {
    transform.scale = baseScale * ratio
    clampTransform()
    return
  }
  // 以编辑区中心为锚点，把滑条值映射到绝对缩放
  const previousScale = transform.scale
  const nextScale = baseScale * ratio
  const centerX = stage.clientWidth / 2
  const centerY = stage.clientHeight / 2
  transform.scale = nextScale
  if (previousScale > 0) {
    transform.x = centerX - ((centerX - transform.x) * nextScale) / previousScale
    transform.y = centerY - ((centerY - transform.y) * nextScale) / previousScale
  }
  clampTransform()
}

// 根据舞台尺寸计算移动端裁切遮罩（左右暗带 = 移动端会裁掉的区域）
function updateMobileMask() {
  const stage = stageRef.value
  if (!stage) {
    maskSidePx.value = 0
    return
  }
  const stageWidth = stage.clientWidth
  const stageHeight = stage.clientHeight
  if (stageWidth <= 0 || stageHeight <= 0) {
    maskSidePx.value = 0
    return
  }
  // 移动端安全区：同高、比例 mobileAspect → 宽度 = height * mobileAspect
  const mobileSafeWidth = stageHeight * mobileAspect.value
  maskSidePx.value = Math.max(0, (stageWidth - mobileSafeWidth) / 2)
}

function resetTransform() {
  const stage = stageRef.value
  const image = imageRef.value
  if (!stage || !image) return
  const stageWidth = stage.clientWidth
  const stageHeight = stage.clientHeight
  if (stageWidth <= 0 || stageHeight <= 0 || !image.naturalWidth) return
  baseScale = Math.max(stageWidth / image.naturalWidth, stageHeight / image.naturalHeight)
  transform.scale = baseScale
  transform.x = (stageWidth - image.naturalWidth * transform.scale) / 2
  transform.y = (stageHeight - image.naturalHeight * transform.scale) / 2
  updateMobileMask()
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
  const stage = stageRef.value
  const image = imageRef.value
  if (!stage || !image || baseScale <= 0) return
  const previousScale = transform.scale
  const nextScale = Math.min(baseScale * sliderMax, Math.max(baseScale * sliderMin, previousScale * factor))
  if (nextScale === previousScale) return
  const centerX = stage.clientWidth / 2
  const centerY = stage.clientHeight / 2
  transform.scale = nextScale
  transform.x = centerX - ((centerX - transform.x) * nextScale) / previousScale
  transform.y = centerY - ((centerY - transform.y) * nextScale) / previousScale
  clampTransform()
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

// 保存：把编辑区当前视图映射到输出画布。
// 舞台比例与目标不一致时，按目标比例居中裁切（移动端遮罩提示的含义）。
function handleSave() {
  const stage = stageRef.value
  const image = imageRef.value
  if (!stage || !image || !image.naturalWidth) return
  const outputWidth = 1600
  const outputHeight = Math.round(1600 / aspect.value)
  const canvas = document.createElement('canvas')
  canvas.width = outputWidth
  canvas.height = outputHeight
  const context = canvas.getContext('2d')
  if (!context) return
  context.imageSmoothingEnabled = true
  context.imageSmoothingQuality = 'high'

  let sourceX = -transform.x / transform.scale
  let sourceY = -transform.y / transform.scale
  let sourceWidth = stage.clientWidth / transform.scale
  let sourceHeight = stage.clientHeight / transform.scale
  const targetRatio = aspect.value
  if (sourceWidth / sourceHeight > targetRatio) {
    const croppedWidth = sourceHeight * targetRatio
    sourceX += (sourceWidth - croppedWidth) / 2
    sourceWidth = croppedWidth
  } else if (sourceWidth / sourceHeight < targetRatio) {
    const croppedHeight = sourceWidth / targetRatio
    sourceY += (sourceHeight - croppedHeight) / 2
    sourceHeight = croppedHeight
  }
  context.drawImage(image, sourceX, sourceY, sourceWidth, sourceHeight, 0, 0, outputWidth, outputHeight)
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
    updateMobileMask()
    // 尺寸变化时重新 cover 适配，避免露出空白
    if (imageRef.value?.naturalWidth) {
      clampTransform()
      // 若缩放过小导致 reset，保持一致
      if (transform.scale < baseScale) resetTransform()
    }
  })
  resizeObserver.observe(stageRef.value)
}

onMounted(async () => {
  await nextTick()
  fitImageOnce()
  bindResizeObserver()
  updateMobileMask()
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
    updateMobileMask()
  },
)

defineExpose({ resetTransform, save: handleSave })
</script>

<template>
  <div class="cover-editor" :class="{ 'cover-editor--float': floatMode }">
    <!-- 预览区：桌面端完整高度，不被操作条挤占 -->
    <div
      ref="stageRef"
      class="cover-editor-stage"
      :class="{ 'cover-editor-stage--float': floatMode }"
      :style="floatMode ? undefined : { aspectRatio: `${aspect} / 1` }"
      @wheel="onWheel"
    >
      <img
        ref="imageRef"
        :src="imageUrl"
        :style="imageStyle"
        :alt="t('settings.cover.cropAlt')"
        class="cover-editor-image"
        draggable="false"
        @pointerdown="startDrag"
        @pointermove="onDrag"
        @pointerup="endDrag"
        @pointercancel="endDrag"
      />

      <!-- 移动端裁切遮罩：暗带 = 移动端会被裁切的区域，中间亮区 = 移动端保留 -->
      <div class="cover-editor-mask" aria-hidden="true">
        <div class="cover-editor-mask-side" :style="{ width: `${maskSidePx}px` }" />
        <div class="cover-editor-mask-center" />
        <div class="cover-editor-mask-side" :style="{ width: `${maskSidePx}px` }" />
      </div>
    </div>

    <!-- 浮层模式：操作条从预览区底部向下延伸，盖在头像上层；含标题 / 滑条 / 按钮 -->
    <div v-if="floatMode" class="cover-editor-action">
      <div class="cover-editor-action-left">
        <p class="cover-editor-action-title">{{ t('settings.cover.editTitle') }}</p>
        <p class="cover-editor-action-hint">{{ t('settings.cover.mobileCropHint') }}</p>
      </div>

      <div class="cover-editor-slider cover-editor-slider--action">
        <button
          type="button"
          class="cover-editor-slider-icon"
          :aria-label="t('settings.cover.zoomOut')"
          :title="t('settings.cover.zoomOut')"
          @click="zoomOut"
        >
          <Minus class="cover-editor-slider-glyph" :stroke-width="2.25" aria-hidden="true" />
        </button>
        <input
          type="range"
          class="cover-editor-slider-input"
          :min="sliderMin"
          :max="sliderMax"
          step="0.01"
          :value="sliderValue"
          :aria-label="t('settings.cover.zoomSlider')"
          :aria-valuetext="`${Math.round(sliderValue * 100)}%`"
          @input="onSliderInput"
        />
        <button
          type="button"
          class="cover-editor-slider-icon"
          :aria-label="t('settings.cover.zoomIn')"
          :title="t('settings.cover.zoomIn')"
          @click="zoomIn"
        >
          <Plus class="cover-editor-slider-glyph" :stroke-width="2.25" aria-hidden="true" />
        </button>
      </div>

      <div class="cover-editor-action-right">
        <slot name="actions">
          <button
            type="button"
            class="gf-button gf-button-md gf-button-muted font-medium"
            :disabled="saving"
            @click="emit('cancel')"
          >
            {{ t('common.cancel') }}
          </button>
          <button
            type="button"
            class="gf-button gf-button-md gf-button-primary min-w-24 font-semibold disabled:cursor-wait"
            :disabled="saving"
            :aria-busy="saving"
            @click="handleSave"
          >
            {{ saving ? t('settings.cover.uploading') : t('common.save') }}
          </button>
        </slot>
      </div>
    </div>

    <!-- 非浮层（设置页模态）：滑条在预览下方流式排列 -->
    <div v-else class="cover-editor-slider cover-editor-slider--inline">
      <button
        type="button"
        class="cover-editor-slider-icon"
        :aria-label="t('settings.cover.zoomOut')"
        :title="t('settings.cover.zoomOut')"
        @click="zoomOut"
      >
        <Minus class="cover-editor-slider-glyph" :stroke-width="2.25" aria-hidden="true" />
      </button>
      <input
        type="range"
        class="cover-editor-slider-input"
        :min="sliderMin"
        :max="sliderMax"
        step="0.01"
        :value="sliderValue"
        :aria-label="t('settings.cover.zoomSlider')"
        :aria-valuetext="`${Math.round(sliderValue * 100)}%`"
        @input="onSliderInput"
      />
      <button
        type="button"
        class="cover-editor-slider-icon"
        :aria-label="t('settings.cover.zoomIn')"
        :title="t('settings.cover.zoomIn')"
        @click="zoomIn"
      >
        <Plus class="cover-editor-slider-glyph" :stroke-width="2.25" aria-hidden="true" />
      </button>
    </div>

    <div v-if="showMobilePreview && !floatMode" class="mt-3 flex items-start gap-3">
      <div class="cover-editor-mobile shrink-0" :style="{ aspectRatio: `${mobileAspect} / 1` }">
        <img :src="imageUrl" :style="imageStyle" :alt="t('settings.cover.previewMobile')" class="cover-editor-image" draggable="false" />
      </div>
      <p class="pt-0.5 text-xs leading-5 text-base-content/55">
        {{ t('settings.cover.mobileCropHint') }}
      </p>
    </div>
  </div>
</template>

<style scoped>
.cover-editor {
  position: relative;
  width: 100%;
}

.cover-editor--float {
  height: 100%;
}

.cover-editor-stage {
  position: relative;
  width: 100%;
  overflow: hidden;
  border: var(--gf-border) solid var(--gf-color-line);
  border-radius: var(--gf-radius-field);
  background: var(--gf-color-base-200);
  touch-action: none;
  cursor: grab;
}

.cover-editor-stage:active {
  cursor: grabbing;
}

/* 浮层：预览铺满父容器（完整封面高度），无边框 */
.cover-editor-stage--float {
  height: 100%;
  border: 0;
  border-radius: 0;
  background: var(--gf-color-base-300);
}

.cover-editor-image {
  position: absolute;
  top: 0;
  left: 0;
  transform-origin: 0 0;
  max-width: none;
  user-select: none;
  -webkit-user-drag: none;
}

/* 移动端裁切遮罩：左右暗带 + 中间透明安全区 */
.cover-editor-mask {
  pointer-events: none;
  position: absolute;
  inset: 0;
  z-index: 2;
  display: flex;
  align-items: stretch;
}

.cover-editor-mask-side {
  flex: 0 0 auto;
  background: rgb(0 0 0 / 0.42);
}

.cover-editor-mask-center {
  flex: 1 1 auto;
  min-width: 0;
  box-shadow: inset 0 0 0 1px rgb(255 255 255 / 0.18);
}

.cover-editor-mobile {
  position: relative;
  width: 9rem;
  overflow: hidden;
  border: var(--gf-border) solid var(--gf-color-line);
  border-radius: var(--gf-radius-field);
  background: var(--gf-color-base-200);
  touch-action: none;
}

/* 操作条：从预览区底部向下延伸（top: 100%），盖在头像上层 */
.cover-editor-action {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  z-index: 30;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  min-height: 3.5rem;
  padding: 0.625rem 1rem;
  border-top: 1px solid var(--gf-color-line);
  background: color-mix(in oklab, var(--gf-color-base-100) 92%, transparent);
  backdrop-filter: blur(10px);
  box-shadow: 0 8px 24px rgb(0 0 0 / 0.08);
}

.cover-editor-action-left {
  position: relative;
  z-index: 2;
  min-width: 0;
  flex: 1 1 12rem;
  max-width: calc(50% - 8.5rem);
  padding-right: 0.75rem;
}

.cover-editor-action-title {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--gf-color-base-content);
  line-height: 1.25;
}

.cover-editor-action-hint {
  margin: 0.125rem 0 0;
  font-size: 0.75rem;
  color: color-mix(in oklab, var(--gf-color-base-content) 55%, transparent);
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cover-editor-action-right {
  position: relative;
  z-index: 2;
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 0.5rem;
  margin-left: auto;
}

.cover-editor-slider {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.375rem;
}

.cover-editor-slider--inline {
  margin-top: 0.75rem;
  width: 100%;
}

/* 操作条内：相对操作条绝对居中，标题/按钮宽度不影响视觉中心 */
.cover-editor-slider--action {
  position: absolute;
  left: 50%;
  top: 50%;
  z-index: 1;
  transform: translate(-50%, -50%);
  flex: 0 0 auto;
  padding: 0.2rem 0.45rem;
  border-radius: 9999px;
  border: 1px solid color-mix(in oklab, var(--gf-color-line) 95%, var(--gf-color-base-content) 5%);
  background: color-mix(in oklab, var(--gf-color-base-100) 88%, var(--gf-color-base-content) 4%);
  box-shadow: 0 1px 2px rgb(0 0 0 / 0.04);
}

.cover-editor-slider-icon {
  display: inline-flex;
  height: 1.75rem;
  width: 1.75rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  /* 浅色主题需足够对比度：不用 58% 淡化，改用 content 实色 */
  color: var(--gf-color-base-content);
  opacity: 0.92;
  transition: background-color 0.15s, opacity 0.15s, color 0.15s;
  outline: none;
}

.cover-editor-slider-icon:hover {
  background: color-mix(in oklab, var(--gf-color-base-300) 88%, transparent);
  opacity: 1;
  color: var(--gf-color-base-content);
}

.cover-editor-slider-icon:focus-visible {
  outline: 2px solid color-mix(in oklab, var(--gf-color-primary) 55%, transparent);
  outline-offset: 1px;
}

/* Lucide Minus/Plus：与全站工具栏同族；略加粗保证浅色可读 */
.cover-editor-slider-glyph {
  display: block;
  width: 0.9375rem;
  height: 0.9375rem;
  stroke-width: 2.25;
}

.cover-editor-slider-input {
  width: 7.5rem;
  height: 1.25rem;
  accent-color: var(--gf-color-primary);
  cursor: pointer;
  vertical-align: middle;
}

@media (min-width: 640px) {
  .cover-editor-slider-input {
    width: 11rem;
  }

  .cover-editor-action {
    padding: 0.75rem 1.25rem;
    min-height: 4rem;
  }
}

/* 窄屏：滑条顶部居中；右侧按钮换行 */
@media (max-width: 639px) {
  .cover-editor-action {
    flex-wrap: wrap;
    row-gap: 0.5rem;
    justify-content: flex-end;
    padding-top: 2.75rem;
  }

  .cover-editor-action-left {
    display: none;
  }

  .cover-editor-slider--action {
    top: 0.7rem;
    transform: translate(-50%, 0);
  }

  .cover-editor-action-right {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>

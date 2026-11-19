<script setup lang="ts">
/**
 * 编辑区图片工具（MADR 0131）：单击图片后出现选中框与浮动工具条。
 * - 宽度：分段切换「原始 / 33% / 50% / 75% / 100%」，或拖动左右手柄，松手吸附到最近的预设；
 * - 替代文字：就地编辑（读屏与加载失败时显示）；
 * - 预览：打开大图查看（编辑区内的全部图片可左右切换）；删除。
 * 宽度写进图片地址的 `#w=` 片段，发布后与 App 按同一规则排版。
 */
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { Maximize2, Trash2 } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { placeFloating, placeOverlay, rectVisibleIn, type FloatingRect } from '@/runtime/floating-position'
import { IMAGE_MIN_RENDERED_WIDTH, snapImageWidth, type ImageWidth } from '@/runtime/image-width'

const props = defineProps<{
  host: HTMLElement | null
  image: HTMLImageElement | null
  boundary: FloatingRect | null
  width: ImageWidth | null
  /** 地址带有其他片段时不能写入宽度 */
  resizable: boolean
  alt: string
}>()
const emit = defineEmits<{
  width: [width: ImageWidth | null]
  alt: [alt: string]
  preview: []
  remove: []
}>()

const { t } = useI18n()
const frame = ref<HTMLElement | null>(null)
const panel = ref<HTMLElement | null>(null)
const segments = ref<HTMLElement | null>(null)
const altInput = ref<HTMLInputElement | null>(null)
const visible = ref(false)
const editingAlt = ref(false)
const altDraft = ref('')
const dragBadge = ref<ImageWidth | null>(null)
const indicator = ref<{ left: number; width: number } | null>(null)

const SIZE_OPTIONS: Array<ImageWidth | null> = [null, 33, 50, 75, 100]
const shownWidth = computed(() => dragBadge.value ?? props.width)

interface DragState {
  pointerId: number
  edge: 'left' | 'right'
  startX: number
  startWidth: number
  container: number
  inRow: boolean
}
let drag: DragState | null = null

function anchorRect(): FloatingRect | null {
  const img = props.image
  if (!img?.isConnected) return null
  const rect = img.getBoundingClientRect()
  return { top: rect.top, left: rect.left, right: rect.right, bottom: rect.bottom }
}

function syncIndicator() {
  const container = segments.value
  const active = container?.querySelector<HTMLElement>('[aria-pressed="true"]')
  indicator.value = active ? { left: active.offsetLeft, width: active.offsetWidth } : null
}

/** 图片只露出一部分时，选中框裁到可见范围（不压住吸顶工具栏、顶栏或编辑区外的内容）：未被裁的一侧
 *  留出外扩描边的空间；手柄与宽度提示移到露出部分的中间 / 顶部 */
function clipFrame(el: HTMLElement, rect: FloatingRect, boundary: FloatingRect) {
  const RING = 12
  const cut = [boundary.top - rect.top, rect.right - boundary.right, rect.bottom - boundary.bottom, boundary.left - rect.left]
  el.style.clipPath = cut.some((value) => value > 0)
    ? `inset(${cut.map((value) => (value > 0 ? `${value}px` : `-${RING}px`)).join(' ')})`
    : ''
  const top = Math.max(rect.top, boundary.top)
  const bottom = Math.min(rect.bottom, boundary.bottom)
  el.style.setProperty('--gf-frame-visible-top', `${top - rect.top}px`)
  el.style.setProperty('--gf-frame-visible-middle', `${(top + bottom) / 2 - rect.top}px`)
}

function reposition() {
  const rect = anchorRect()
  visible.value = Boolean(rect && props.boundary && rectVisibleIn(rect, props.boundary))
  if (!rect || !props.boundary || !visible.value) return
  if (frame.value) {
    placeOverlay(frame.value, rect)
    clipFrame(frame.value, rect, props.boundary)
  }
  if (panel.value) placeFloating(panel.value, rect, props.boundary, { side: 'top', gap: 10 })
}

watch(
  () => [props.image, props.width, props.boundary] as const,
  ([image], previous) => {
    if (image !== previous?.[0]) editingAlt.value = false
    void nextTick(() => {
      reposition()
      syncIndicator()
    })
  },
  { immediate: true },
)

watch(shownWidth, () => void nextTick(syncIndicator))

// 宽度过渡、图片加载完成、窗口变化都会改变图片尺寸：选中框与工具条随之贴合
let resizeObserver: ResizeObserver | null = null
watch(
  () => props.image,
  (image) => {
    resizeObserver?.disconnect()
    resizeObserver = null
    if (!image || typeof ResizeObserver === 'undefined') return
    resizeObserver = new ResizeObserver(() => reposition())
    resizeObserver.observe(image)
  },
  { immediate: true },
)

function containerWidthOf(img: HTMLImageElement): { width: number; inRow: boolean } {
  const block = img.parentElement?.closest<HTMLElement>('p, li, blockquote, td, th') ?? img.parentElement
  const style = block ? getComputedStyle(block) : null
  const padding = style ? parseFloat(style.paddingLeft) + parseFloat(style.paddingRight) : 0
  return { width: Math.max(1, (block?.clientWidth ?? img.offsetWidth) - padding), inRow: style?.display === 'flex' }
}

function startResize(event: PointerEvent, edge: 'left' | 'right') {
  const img = props.image
  if (!img || !props.resizable || event.button !== 0) return
  event.preventDefault()
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
  const { width, inRow } = containerWidthOf(img)
  drag = { pointerId: event.pointerId, edge, startX: event.clientX, startWidth: img.getBoundingClientRect().width, container: width, inRow }
  img.style.transition = 'none'
  img.style.maxHeight = 'none'
}

function moveResize(event: PointerEvent) {
  const img = props.image
  if (!drag || !img || event.pointerId !== drag.pointerId) return
  const delta = (event.clientX - drag.startX) * (drag.edge === 'right' ? 1 : -1)
  // 单张图居中，两侧同时伸缩；并排的图从左缘起算
  const next = drag.startWidth + (drag.inRow ? delta : delta * 2)
  const min = Math.min(IMAGE_MIN_RENDERED_WIDTH, drag.container)
  const width = Math.min(Math.max(next, min), drag.container)
  img.style.width = `${width}px`
  dragBadge.value = snapImageWidth(width / drag.container)
  reposition()
}

function endResize(event: PointerEvent) {
  const img = props.image
  if (!drag || event.pointerId !== drag.pointerId) return
  drag = null
  const snapped = dragBadge.value
  dragBadge.value = null
  if (img) {
    img.style.removeProperty('width')
    img.style.removeProperty('max-height')
    // 下一帧恢复过渡：从拖拽宽度平滑吸附到预设
    requestAnimationFrame(() => img.style.removeProperty('transition'))
  }
  if (snapped !== null && snapped !== props.width) emit('width', snapped)
  else void nextTick(reposition)
}

function startAltEdit() {
  altDraft.value = props.alt
  editingAlt.value = true
  void nextTick(() => {
    reposition()
    altInput.value?.focus()
    altInput.value?.select()
  })
}

function saveAlt() {
  if (!editingAlt.value) return
  editingAlt.value = false
  if (altDraft.value.trim() !== props.alt) emit('alt', altDraft.value.trim())
  void nextTick(reposition)
}

function cancelAlt() {
  editingAlt.value = false
  void nextTick(reposition)
}

function sizeLabel(option: ImageWidth | null) {
  return option ? `${option}%` : t('editor.floating.imageAuto')
}

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  if (drag && props.image) {
    props.image.style.removeProperty('width')
    props.image.style.removeProperty('max-height')
    props.image.style.removeProperty('transition')
  }
})

defineExpose({
  reposition,
  editingAlt,
  contains: (node: Node | null) => Boolean(node && (panel.value?.contains(node) || frame.value?.contains(node))),
})
</script>

<template>
  <Teleport v-if="host" :to="host">
    <div v-if="image" v-show="visible" ref="frame" class="gf-editor-image-frame" :class="{ 'is-dragging': dragBadge !== null }" data-test="editor-image-frame">
      <template v-if="resizable && !editingAlt">
        <span
          v-for="edge in (['left', 'right'] as const)"
          :key="edge"
          class="gf-editor-image-handle"
          :class="`gf-editor-image-handle--${edge}`"
          role="presentation"
          :title="t('editor.floating.imageResize')"
          :data-edge="edge"
          @pointerdown="startResize($event, edge)"
          @pointermove="moveResize"
          @pointerup="endResize"
          @pointercancel="endResize"
        />
      </template>
      <Transition name="gf-editor-float">
        <span v-if="dragBadge !== null" class="gf-editor-image-badge tabular-nums" aria-live="polite">{{ dragBadge }}%</span>
      </Transition>
    </div>
    <Transition name="gf-editor-float">
      <div
        v-if="image"
        v-show="visible"
        ref="panel"
        role="toolbar"
        :aria-label="t('editor.floating.imageToolbar')"
        class="gf-editor-float gf-floating-surface"
        data-test="editor-image-toolbar"
      >
        <form v-if="editingAlt" class="flex items-center gap-1.5 px-1" @submit.prevent="saveAlt">
          <input
            ref="altInput"
            v-model="altDraft"
            type="text"
            class="gf-editor-float-input"
            :placeholder="t('editor.floating.imageAltPlaceholder')"
            :aria-label="t('editor.floating.imageAlt')"
            maxlength="200"
            @keydown.esc.prevent.stop="cancelAlt"
            @blur="saveAlt"
          />
          <button type="submit" class="gf-editor-float-button gf-editor-float-button--text gf-editor-float-button--primary" @pointerdown.prevent>
            {{ t('editor.floating.imageAltSave') }}
          </button>
        </form>
        <template v-else>
          <div
            ref="segments"
            class="gf-editor-float-segments"
            role="group"
            :aria-label="t('editor.floating.imageWidth')"
            @pointerdown.prevent
          >
            <span v-if="indicator" class="gf-editor-float-segment-indicator" :style="{ transform: `translateX(${indicator.left}px)`, width: `${indicator.width}px` }" aria-hidden="true" />
            <button
              v-for="option in SIZE_OPTIONS"
              :key="option ?? 'auto'"
              type="button"
              class="gf-editor-float-segment tabular-nums"
              :aria-pressed="shownWidth === option"
              :disabled="!resizable"
              :title="option ? t('editor.floating.imageWidthValue', { percent: option }) : t('editor.floating.imageAutoHint')"
              @click="emit('width', option)"
            >
              {{ sizeLabel(option) }}
            </button>
          </div>
          <span class="gf-editor-float-divider" aria-hidden="true" />
          <button
            type="button"
            class="gf-editor-float-button gf-editor-float-button--text"
            :class="{ 'is-active': alt }"
            :aria-label="t('editor.floating.imageAlt')"
            :title="alt ? t('editor.floating.imageAltCurrent', { alt }) : t('editor.floating.imageAlt')"
            @pointerdown.prevent
            @click="startAltEdit"
          >
            <span class="text-[0.6875rem] font-semibold tracking-wide">ALT</span>
          </button>
          <button type="button" class="gf-editor-float-button" :aria-label="t('editor.floating.imagePreview')" :title="t('editor.floating.imagePreview')" @pointerdown.prevent @click="emit('preview')">
            <Maximize2 class="h-4 w-4" />
          </button>
          <span class="gf-editor-float-divider" aria-hidden="true" />
          <button type="button" class="gf-editor-float-button gf-editor-float-button--danger" :aria-label="t('editor.floating.imageRemove')" :title="t('editor.floating.imageRemove')" @pointerdown.prevent @click="emit('remove')">
            <Trash2 class="h-4 w-4" />
          </button>
        </template>
      </div>
    </Transition>
  </Teleport>
</template>

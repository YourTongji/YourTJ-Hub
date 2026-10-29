<script setup lang="ts">
import { ArrowLeft, Loader2, Plus, RotateCcw, Store } from '@lucide/vue'
import type { StickerItem, StickerMarketPack } from '@gooseforum/client'
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useStickerLibrary } from '@/site/composables/useStickerLibrary'
import { useStickerMarket } from '@/site/composables/useStickerMarket'

/**
 * 表情包选择面板（MADR 0030；市场见 issue #1429）：官方贴纸、已添加的市场
 * 表情包与市场浏览共用一个浮层，点选后向宿主 emit select(name)。数据走
 * useStickerLibrary / useStickerMarket 全站共享缓存，打开时按需拉取。
 * 打开时监听文档级 pointerdown 点击面板外即 emit close；vditor 工具栏内的
 * 点击交给宿主 toggle 处理（否则关了又立刻被 click 重开）。
 *
 * 预览：鼠标悬停片刻、键盘聚焦或触屏长按时显示放大气泡。长按后滑动手指
 * 切换预览目标，松手或取消只关闭气泡，不插入贴纸；单击照常插入。
 */
const props = defineProps<{
  open: boolean
  /** 面板相对锚点的展开方向：below=锚点下方（顶部工具栏），above=锚点上方（底部工具栏） */
  placement?: 'below' | 'above'
}>()

const emit = defineEmits<{
  select: [name: string]
  close: []
}>()

type PickerView =
  | { kind: 'official' }
  | { kind: 'pack', pack: StickerMarketPack }
  | { kind: 'market' }
  | { kind: 'marketPack', pack: StickerMarketPack }

const HOVER_DELAY_MS = 300
const LONG_PRESS_MS = 350
const MOVE_TOLERANCE_PX = 8

const { t } = useI18n()
const { stickers, ensureStickers, loading, failed } = useStickerLibrary()
const market = useStickerMarket()
const { myPacks, topics, marketPacks, marketTopic, marketTotal, marketLoading, marketFailed, actionError } = market
const panelRef = ref<HTMLElement | null>(null)
const gridRef = ref<HTMLElement | null>(null)
const view = ref<PickerView>({ kind: 'official' })
const packItems = ref<StickerItem[]>([])
const packLoading = ref(false)
const packFailed = ref(false)
const pendingPackId = ref<number | null>(null)
const preview = ref<StickerItem | null>(null)
let outsidePointerDown: ((event: PointerEvent) => void) | null = null
let hoverTimer: ReturnType<typeof setTimeout> | undefined
let pressTimer: ReturnType<typeof setTimeout> | undefined
let pressOrigin: { x: number, y: number } | null = null
let pressPreviewing = false
let suppressClick = false

const gridStickers = computed(() => (view.value.kind === 'official' ? stickers.value : packItems.value))
const gridLoading = computed(() => (view.value.kind === 'official' ? loading.value : packLoading.value))
const gridFailed = computed(() => (view.value.kind === 'official' ? failed.value : packFailed.value))
const viewPack = computed(() => (view.value.kind === 'pack' || view.value.kind === 'marketPack' ? view.value.pack : null))
const hasMoreMarket = computed(() => marketPacks.value.length < marketTotal.value)

watch(() => props.open, (open) => {
  if (!open) {
    detachOutsideClose()
    hidePreview()
    return
  }
  void ensureStickers()
  void market.ensureMyPacks()
  void nextTick(() => {
    if (!props.open || outsidePointerDown) return
    outsidePointerDown = (event) => {
      const target = event.target
      if (!(target instanceof Node)) return
      if (panelRef.value?.contains(target)) return
      if (target instanceof Element && target.closest('.vditor')) return
      emit('close')
    }
    document.addEventListener('pointerdown', outsidePointerDown, true)
  })
}, { immediate: true })

// A pack removed elsewhere (another editor, another device) leaves its tab.
watch(myPacks, (packs) => {
  if (view.value.kind === 'pack' && !packs.some((pack) => pack.id === viewPack.value?.id)) {
    view.value = { kind: 'official' }
  }
})

function detachOutsideClose() {
  if (outsidePointerDown) document.removeEventListener('pointerdown', outsidePointerDown, true)
  outsidePointerDown = null
}

onBeforeUnmount(() => {
  detachOutsideClose()
  hidePreview()
})

async function showPack(pack: StickerMarketPack, kind: 'pack' | 'marketPack') {
  hidePreview()
  view.value = { kind, pack }
  packItems.value = []
  packFailed.value = false
  packLoading.value = true
  try {
    const items = await market.loadPackStickers(pack.id)
    if (viewPack.value?.id === pack.id) packItems.value = items
  } catch {
    if (viewPack.value?.id === pack.id) packFailed.value = true
  } finally {
    if (viewPack.value?.id === pack.id) packLoading.value = false
  }
}

function retryGrid() {
  if (viewPack.value && view.value.kind !== 'official') {
    void showPack(viewPack.value, view.value.kind === 'pack' ? 'pack' : 'marketPack')
    return
  }
  void ensureStickers()
}

function showOfficial() {
  hidePreview()
  view.value = { kind: 'official' }
}

function showMarket() {
  hidePreview()
  view.value = { kind: 'market' }
  void market.loadMarket()
}

function selectTopic(topic: string) {
  void market.loadMarket(topic)
}

async function togglePack(pack: StickerMarketPack) {
  pendingPackId.value = pack.id
  try {
    if (market.isSubscribed(pack.id)) {
      await market.unsubscribe(pack)
    } else {
      await market.subscribe(pack)
    }
  } finally {
    pendingPackId.value = null
  }
}

function selectSticker(sticker: StickerItem) {
  if (suppressClick) {
    suppressClick = false
    return
  }
  emit('select', sticker.name)
}

function stickerLabel(sticker: StickerItem) {
  return sticker.displayName || sticker.name
}

function hidePreview() {
  clearTimeout(hoverTimer)
  clearTimeout(pressTimer)
  pressOrigin = null
  pressPreviewing = false
  preview.value = null
}

function stickerAtPoint(x: number, y: number) {
  const cell = document.elementFromPoint(x, y)?.closest<HTMLElement>('[data-sticker-index]')
  if (!cell || !gridRef.value?.contains(cell)) return null
  return gridStickers.value[Number(cell.dataset.stickerIndex)] ?? null
}

function onCellPointerEnter(sticker: StickerItem, event: PointerEvent) {
  if (event.pointerType !== 'mouse') return
  clearTimeout(hoverTimer)
  if (preview.value) {
    preview.value = sticker
    return
  }
  hoverTimer = setTimeout(() => {
    preview.value = sticker
  }, HOVER_DELAY_MS)
}

function onCellPointerDown(sticker: StickerItem, event: PointerEvent) {
  suppressClick = false
  if (event.pointerType === 'mouse') return
  clearTimeout(pressTimer)
  pressOrigin = { x: event.clientX, y: event.clientY }
  pressTimer = setTimeout(() => {
    pressPreviewing = true
    suppressClick = true
    preview.value = sticker
    navigator.vibrate?.(10)
  }, LONG_PRESS_MS)
}

function onGridPointerMove(event: PointerEvent) {
  if (event.pointerType === 'mouse') return
  if (pressPreviewing) {
    const sticker = stickerAtPoint(event.clientX, event.clientY)
    if (sticker && sticker !== preview.value) {
      preview.value = sticker
      navigator.vibrate?.(5)
    }
    return
  }
  if (pressOrigin && Math.hypot(event.clientX - pressOrigin.x, event.clientY - pressOrigin.y) > MOVE_TOLERANCE_PX) {
    clearTimeout(pressTimer)
    pressOrigin = null
  }
}

function onGridPointerEnd(event: PointerEvent) {
  if (event.pointerType === 'mouse') return
  hidePreview()
}

function onGridPointerLeave(event: PointerEvent) {
  if (event.pointerType === 'mouse') hidePreview()
}

// While a long-press preview is open, the finger moves the preview, not the page.
function onGridTouchMove(event: TouchEvent) {
  if (pressPreviewing && event.cancelable) event.preventDefault()
}

function onCellFocus(sticker: StickerItem, event: FocusEvent) {
  if ((event.target as HTMLElement).matches(':focus-visible')) preview.value = sticker
}

function onPanelKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  if (preview.value) {
    hidePreview()
    return
  }
  emit('close')
}

const gridEl = (el: unknown) => {
  const next = el instanceof HTMLElement ? el : null
  if (gridRef.value === next) return
  gridRef.value?.removeEventListener('touchmove', onGridTouchMove)
  gridRef.value = next
  next?.addEventListener('touchmove', onGridTouchMove, { passive: false })
}
</script>

<template>
  <div
    v-if="open"
    ref="panelRef"
    class="gf-menu-surface gf-sticker-picker absolute left-2 z-30"
    :class="props.placement === 'above' ? 'bottom-14' : 'top-12'"
    role="dialog"
    :aria-label="t('stickers.pickerTitle')"
    @keydown="onPanelKeydown"
  >
    <div class="gf-sticker-tabs" role="toolbar" :aria-label="t('stickers.pickerTitle')">
      <button
        type="button"
        class="gf-sticker-tab gf-sticker-tab-text"
        :aria-pressed="view.kind === 'official'"
        @click="showOfficial"
      >
        {{ t('stickers.tabOfficial') }}
      </button>
      <button
        v-for="pack in myPacks"
        :key="pack.id"
        type="button"
        class="gf-sticker-tab"
        :aria-pressed="view.kind === 'pack' && viewPack?.id === pack.id"
        :aria-label="pack.name"
        :title="pack.name"
        @click="showPack(pack, 'pack')"
      >
        <img v-if="pack.coverUrl" :src="pack.coverUrl" alt="" loading="lazy" class="gf-sticker-tab-img" />
        <span v-else class="truncate text-xs">{{ pack.name.slice(0, 2) }}</span>
      </button>
      <button
        type="button"
        class="gf-sticker-tab"
        :aria-pressed="view.kind === 'market' || view.kind === 'marketPack'"
        :aria-label="t('stickers.openMarket')"
        :title="t('stickers.openMarket')"
        @click="showMarket"
      >
        <Store class="h-4 w-4" />
      </button>
    </div>

    <div class="gf-sticker-body">
      <template v-if="view.kind === 'market'">
        <div v-if="topics.length > 0" class="gf-sticker-topics" role="toolbar" :aria-label="t('stickers.marketTopics')">
          <button type="button" class="gf-sticker-chip" :aria-pressed="marketTopic === ''" @click="selectTopic('')">
            {{ t('stickers.allTopics') }}
          </button>
          <button
            v-for="topic in topics"
            :key="topic.id"
            type="button"
            class="gf-sticker-chip"
            :aria-pressed="marketTopic === topic.id"
            @click="selectTopic(topic.id)"
          >
            {{ topic.label }}
          </button>
        </div>
        <ul class="flex flex-col gap-1">
          <li v-for="pack in marketPacks" :key="pack.id" class="gf-sticker-pack-row">
            <button type="button" class="gf-sticker-pack-open" @click="showPack(pack, 'marketPack')">
              <img v-if="pack.coverUrl" :src="pack.coverUrl" alt="" loading="lazy" class="gf-sticker-pack-cover" />
              <span v-else class="gf-sticker-pack-cover" aria-hidden="true" />
              <span class="min-w-0 text-start">
                <span class="block truncate text-sm font-medium">{{ pack.name }}</span>
                <span class="block text-xs text-base-content/60">{{ t('stickers.packCount', { count: pack.stickerCount }, pack.stickerCount) }}</span>
              </span>
            </button>
            <button
              type="button"
              class="gf-sticker-pack-action"
              :class="market.isSubscribed(pack.id) ? 'gf-sticker-pack-action-done' : ''"
              :disabled="pendingPackId === pack.id"
              :aria-label="market.isSubscribed(pack.id) ? t('stickers.removePackLabel', { name: pack.name }) : t('stickers.addPackLabel', { name: pack.name })"
              @click="togglePack(pack)"
            >
              <Plus v-if="!market.isSubscribed(pack.id)" class="h-3.5 w-3.5" />
              {{ market.isSubscribed(pack.id) ? t('stickers.removePack') : t('stickers.addPack') }}
            </button>
          </li>
        </ul>
        <div v-if="marketLoading" class="flex items-center justify-center gap-2 px-2 py-4 text-sm text-base-content/55">
          <Loader2 class="h-4 w-4 animate-spin" />
          <span>{{ t('common.loadingShort') }}</span>
        </div>
        <div v-else-if="marketFailed" class="flex items-center justify-center gap-2 px-2 py-4 text-sm text-error">
          <span>{{ t('api.stickersLoadFailed') }}</span>
          <button type="button" class="gf-icon-button h-7 w-7" :aria-label="t('common.retry')" @click="market.loadMarket(marketTopic)">
            <RotateCcw class="h-3.5 w-3.5" />
          </button>
        </div>
        <p v-else-if="marketPacks.length === 0" class="px-2 py-4 text-center text-sm text-base-content/55">
          {{ t('stickers.marketEmpty') }}
        </p>
        <button v-else-if="hasMoreMarket" type="button" class="gf-sticker-more" @click="market.loadMarket(marketTopic, true)">
          {{ t('common.loadMore') }}
        </button>
      </template>

      <template v-else>
        <div v-if="view.kind === 'marketPack' && viewPack" class="gf-sticker-pack-header">
          <button type="button" class="gf-icon-button h-7 w-7 shrink-0" :aria-label="t('common.back')" @click="showMarket">
            <ArrowLeft class="h-4 w-4" />
          </button>
          <span class="min-w-0 flex-1 truncate text-sm font-medium">{{ viewPack.name }}</span>
          <button
            type="button"
            class="gf-sticker-pack-action"
            :class="market.isSubscribed(viewPack.id) ? 'gf-sticker-pack-action-done' : ''"
            :disabled="pendingPackId === viewPack.id"
            :aria-label="market.isSubscribed(viewPack.id) ? t('stickers.removePackLabel', { name: viewPack.name }) : t('stickers.addPackLabel', { name: viewPack.name })"
            @click="togglePack(viewPack)"
          >
            <Plus v-if="!market.isSubscribed(viewPack.id)" class="h-3.5 w-3.5" />
            {{ market.isSubscribed(viewPack.id) ? t('stickers.removePack') : t('stickers.addPack') }}
          </button>
        </div>
        <div
          :ref="gridEl"
          class="gf-sticker-picker-grid"
          @pointermove="onGridPointerMove"
          @pointerup="onGridPointerEnd"
          @pointercancel="onGridPointerEnd"
          @pointerleave="onGridPointerLeave"
        >
          <button
            v-for="(sticker, index) in gridStickers"
            :key="sticker.name"
            type="button"
            class="gf-sticker-cell"
            :data-sticker-index="index"
            :title="stickerLabel(sticker)"
            :aria-label="stickerLabel(sticker)"
            @click="selectSticker(sticker)"
            @pointerenter="onCellPointerEnter(sticker, $event)"
            @pointerdown="onCellPointerDown(sticker, $event)"
            @focus="onCellFocus(sticker, $event)"
            @blur="hidePreview"
            @contextmenu.prevent
          >
            <img :src="sticker.url" :alt="stickerLabel(sticker)" loading="lazy" draggable="false" class="gf-sticker-img" />
          </button>
        </div>
        <div v-if="gridLoading" class="flex items-center justify-center gap-2 px-2 py-4 text-sm text-base-content/55">
          <Loader2 class="h-4 w-4 animate-spin" />
          <span>{{ t('common.loadingShort') }}</span>
        </div>
        <div v-else-if="gridFailed" class="flex items-center justify-center gap-2 px-2 py-4 text-sm text-error">
          <span>{{ t('api.stickersLoadFailed') }}</span>
          <button type="button" class="gf-icon-button h-7 w-7" :aria-label="t('common.retry')" @click="retryGrid">
            <RotateCcw class="h-3.5 w-3.5" />
          </button>
        </div>
        <div v-else-if="gridStickers.length === 0" class="px-2 py-4 text-center text-sm text-base-content/55">
          {{ t('stickers.empty') }}
        </div>
        <a v-if="view.kind === 'official'" href="https://github.com/YourTongji/YourTJ-Hub/blob/dev/apps/gooseforum/app/console/stickerpresets/preset_stickers/NOTICE.md" target="_blank" rel="noopener noreferrer" class="mt-2 block border-t border-base-content/10 pt-2 text-center text-xs text-base-content/60 underline">
          {{ t('stickers.sources') }}
        </a>
      </template>
      <p v-if="actionError" role="alert" class="mt-2 px-1 text-xs text-error">{{ actionError }}</p>
    </div>

    <Transition name="gf-sticker-preview">
      <div v-if="preview" class="gf-floating-surface gf-sticker-preview" aria-hidden="true">
        <img :src="preview.url" alt="" class="gf-sticker-preview-img" />
        <span class="gf-sticker-preview-label">{{ stickerLabel(preview) }}</span>
      </div>
    </Transition>
  </div>
</template>

<style>
/* 面板定宽；页签固定在顶部，内容区纵向滚动，不撑爆宿主容器 */
.gf-sticker-picker {
  display: flex;
  flex-direction: column;
  width: 17rem;
  max-width: calc(100vw - 2rem);
  max-height: min(288px, 44vh);
  padding: 0;
  overflow: hidden;
}

.gf-sticker-tabs {
  display: flex;
  flex-shrink: 0;
  gap: 4px;
  padding: 6px 8px;
  overflow-x: auto;
  scrollbar-width: none;
  border-bottom: var(--gf-border) solid var(--gf-color-line);
}

.gf-sticker-tab {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  min-width: 2rem;
  height: 2rem;
  padding: 0 4px;
  border-radius: var(--gf-radius-field);
  color: color-mix(in oklab, var(--gf-color-base-content) 70%, transparent);
  transition: background-color 0.15s ease-out, color 0.15s ease-out;
}

.gf-sticker-tab-text {
  padding: 0 10px;
  font-size: 0.8125rem;
}

.gf-sticker-tab:hover {
  background: var(--gf-color-base-200);
}

.gf-sticker-tab[aria-pressed='true'] {
  background: var(--gf-color-base-200);
  color: var(--gf-color-base-content);
  font-weight: 600;
}

.gf-sticker-tab-img {
  width: 1.5rem;
  height: 1.5rem;
  object-fit: contain;
}

.gf-sticker-body {
  min-height: 0;
  flex: 1;
  padding: 8px;
  overflow-y: auto;
  overscroll-behavior: contain;
}

.gf-sticker-picker-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 4px;
}

.gf-sticker-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  aspect-ratio: 1 / 1;
  padding: 4px;
  border-radius: var(--gf-radius-field);
  transition: background-color 0.15s ease-out, transform 0.15s ease-out;
  -webkit-touch-callout: none;
  user-select: none;
}

.gf-sticker-cell:hover {
  background: var(--gf-color-base-200);
  transform: scale(1.06);
}

.gf-sticker-img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  pointer-events: none;
}

.gf-sticker-topics {
  display: flex;
  gap: 4px;
  margin-bottom: 8px;
  overflow-x: auto;
  scrollbar-width: none;
}

.gf-sticker-chip {
  flex-shrink: 0;
  padding: 2px 10px;
  border-radius: 999px;
  font-size: 0.75rem;
  background: var(--gf-color-base-200);
  color: color-mix(in oklab, var(--gf-color-base-content) 75%, transparent);
  transition: background-color 0.15s ease-out, color 0.15s ease-out;
}

.gf-sticker-chip[aria-pressed='true'] {
  background: var(--gf-color-base-content);
  color: var(--gf-color-base-100);
}

.gf-sticker-pack-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.gf-sticker-pack-open {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: center;
  gap: 8px;
  padding: 4px;
  border-radius: var(--gf-radius-field);
  transition: background-color 0.15s ease-out;
}

.gf-sticker-pack-open:hover {
  background: var(--gf-color-base-200);
}

.gf-sticker-pack-cover {
  width: 2.25rem;
  height: 2.25rem;
  flex-shrink: 0;
  border-radius: calc(var(--gf-radius-field) - 4px);
  background: var(--gf-color-base-200);
  object-fit: contain;
}

.gf-sticker-pack-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}

.gf-sticker-pack-action {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  gap: 4px;
  height: 1.75rem;
  padding: 0 10px;
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 500;
  background: var(--gf-color-primary);
  color: var(--gf-color-primary-content);
  transition: background-color 0.15s ease-out, color 0.15s ease-out, scale 0.15s ease-out;
}

.gf-sticker-pack-action:active:not(:disabled) {
  scale: 0.96;
}

.gf-sticker-pack-action:disabled {
  opacity: 0.6;
}

.gf-sticker-pack-action-done {
  background: var(--gf-color-base-200);
  color: var(--gf-color-base-content);
}

.gf-sticker-more {
  display: block;
  width: 100%;
  margin-top: 8px;
  padding: 6px;
  border-radius: var(--gf-radius-field);
  font-size: 0.8125rem;
  color: color-mix(in oklab, var(--gf-color-base-content) 70%, transparent);
}

.gf-sticker-more:hover {
  background: var(--gf-color-base-200);
}

/* 预览气泡浮在内容区上方居中；不拦截指针，手指或鼠标仍落在网格上 */
.gf-sticker-preview {
  position: absolute;
  top: 2.75rem;
  left: 50%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  max-width: calc(100% - 1rem);
  padding: 10px;
  translate: -50% 0;
  pointer-events: none;
}

.gf-sticker-preview-img {
  width: 7.5rem;
  height: 7.5rem;
  object-fit: contain;
}

.gf-sticker-preview-label {
  max-width: 100%;
  overflow: hidden;
  font-size: 0.75rem;
  color: color-mix(in oklab, var(--gf-color-base-content) 70%, transparent);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gf-sticker-preview-enter-active,
.gf-sticker-preview-leave-active {
  transition: opacity 0.12s ease-out, scale 0.12s ease-out;
}

.gf-sticker-preview-enter-from,
.gf-sticker-preview-leave-to {
  opacity: 0;
  scale: 0.96;
}

@media (prefers-reduced-motion: reduce) {
  .gf-sticker-preview-enter-active,
  .gf-sticker-preview-leave-active {
    transition: opacity 0.12s ease-out;
  }

  .gf-sticker-preview-enter-from,
  .gf-sticker-preview-leave-to {
    scale: 1;
  }
}
</style>

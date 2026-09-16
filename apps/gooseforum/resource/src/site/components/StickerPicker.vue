<script setup lang="ts">
import { Loader2, RotateCcw } from '@lucide/vue'
import type { StickerItem } from '@gooseforum/client'
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useStickerLibrary } from '@/site/composables/useStickerLibrary'

/**
 * 表情包选择面板（MADR 0030）：网格展示启用贴纸图预览，点选后向宿主
 * emit select(name)。数据走 useStickerLibrary 全站共享缓存，打开时按需拉取。
 * GIF 动图直接 <img loading="lazy"> 展示；样式对齐 MessagesPage 自建 emoji
 * 面板（gf-menu-surface 浮层 + daisyUI token，深浅色自动跟随）。
 * 打开时监听文档级 pointerdown 点击面板外即 emit close；vditor 工具栏内的
 * 点击交给宿主 toggle 处理（否则关了又立刻被 click 重开）。
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

const { t } = useI18n()
const { stickers, ensureStickers, loading, failed } = useStickerLibrary()
const panelRef = ref<HTMLElement | null>(null)
let outsidePointerDown: ((event: PointerEvent) => void) | null = null

watch(() => props.open, (open) => {
  if (!open) {
    detachOutsideClose()
    return
  }
  void ensureStickers()
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

function detachOutsideClose() {
  if (outsidePointerDown) document.removeEventListener('pointerdown', outsidePointerDown, true)
  outsidePointerDown = null
}

onBeforeUnmount(detachOutsideClose)

function selectSticker(sticker: StickerItem) {
  emit('select', sticker.name)
}
</script>

<template>
  <div
    v-if="open"
    ref="panelRef"
    class="gf-menu-surface gf-sticker-picker absolute left-2 z-30 p-2"
    :class="props.placement === 'above' ? 'bottom-14' : 'top-12'"
    role="dialog"
    :aria-label="t('stickers.pickerTitle')"
  >
    <div class="gf-sticker-picker-grid">
      <button
        v-for="sticker in stickers"
        :key="sticker.name"
        type="button"
        class="gf-sticker-cell"
        :title="sticker.name"
        :aria-label="sticker.name"
        @click="selectSticker(sticker)"
      >
        <img :src="sticker.url" :alt="sticker.name" loading="lazy" class="gf-sticker-img" />
      </button>
    </div>
    <div v-if="loading" class="flex items-center justify-center gap-2 px-2 py-4 text-sm text-base-content/55">
      <Loader2 class="h-4 w-4 animate-spin" />
      <span>{{ t('common.loadingShort') }}</span>
    </div>
    <div v-else-if="failed" class="flex items-center justify-center gap-2 px-2 py-4 text-sm text-error">
      <span>{{ t('api.stickersLoadFailed') }}</span>
      <button type="button" class="gf-icon-button h-7 w-7" :aria-label="t('common.retry')" @click="ensureStickers()">
        <RotateCcw class="h-3.5 w-3.5" />
      </button>
    </div>
    <div v-else-if="stickers.length === 0" class="px-2 py-4 text-center text-sm text-base-content/55">
      {{ t('stickers.empty') }}
    </div>
    <a href="https://github.com/YourTongji/YourTJ-Hub/blob/dev/apps/gooseforum/app/console/stickerpresets/preset_stickers/NOTICE.md" target="_blank" rel="noopener noreferrer" class="mt-2 block border-t border-base-content/10 pt-2 text-center text-xs text-base-content/60 underline">
      {{ t('stickers.sources') }}
    </a>
  </div>
</template>

<style>
/* 面板定宽 + 内部纵向滚动（约 30 张 gif，不撑爆宿主容器） */
.gf-sticker-picker {
  width: 17rem;
  max-width: calc(100vw - 2rem);
  max-height: min(240px, 38vh);
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
  transition: background-color 0.15s ease, transform 0.15s ease;
}

.gf-sticker-cell:hover {
  background: var(--color-base-200);
  transform: scale(1.06);
}

.gf-sticker-img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}
</style>

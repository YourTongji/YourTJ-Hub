<!--
  手动输入 `[:sticker:` 时的贴纸候选浮层（MADR 0106 补充）。纯展示 listbox：
  会话、筛选与键盘编排见 useStickerSuggest.ts；浮层贴近光标、挂到 body 上，
  不受编辑器容器 overflow 裁剪。视觉与 @mention 候选面板同一套浮动语言。
-->
<script setup lang="ts">
import { ref, watch, type CSSProperties } from 'vue'
import { useI18n } from 'vue-i18n'
import { STICKER_SUGGEST_LISTBOX_ID, STICKER_SUGGEST_OPTION_ID_PREFIX } from '@/site/composables/useStickerSuggest'
import type { StickerSuggestion } from '@/site/utils/sticker-token'

const props = defineProps<{
  open: boolean
  query: string
  candidates: StickerSuggestion[]
  activeIndex: number
  loading: boolean
  panelStyle: CSSProperties
}>()

const emit = defineEmits<{
  select: [item: StickerSuggestion]
}>()

const { t } = useI18n()
const panelRef = ref<HTMLElement | null>(null)

watch(
  () => props.activeIndex,
  () => {
    const list = panelRef.value
    const item = list?.querySelector<HTMLElement>('[data-active="true"]')
    if (!list || !item) return
    if (item.offsetTop < list.scrollTop) list.scrollTop = item.offsetTop
    else if (item.offsetTop + item.offsetHeight > list.scrollTop + list.clientHeight) {
      list.scrollTop = item.offsetTop + item.offsetHeight - list.clientHeight
    }
  },
)

// 点选不让编辑器失焦：光标留在原处，替换才能落在片段上
function keepEditorFocus(event: PointerEvent) {
  event.preventDefault()
}
</script>

<template>
  <Teleport to="body">
    <Transition name="gf-sticker-suggest">
      <div
        v-if="open"
        :id="STICKER_SUGGEST_LISTBOX_ID"
        ref="panelRef"
        role="listbox"
        :aria-label="t('stickers.suggest.listbox')"
        class="gf-sticker-suggest"
        :style="panelStyle"
        @pointerdown="keepEditorFocus"
      >
        <template v-if="candidates.length">
          <div
            v-for="(item, index) in candidates"
            :id="`${STICKER_SUGGEST_OPTION_ID_PREFIX}${index}`"
            :key="item.name"
            role="option"
            :aria-selected="index === activeIndex"
            :aria-label="item.label"
            class="gf-sticker-suggest-option"
            :class="{ 'is-active': index === activeIndex }"
            :data-active="index === activeIndex || undefined"
            @click="emit('select', item)"
          >
            <img :src="item.url" alt="" loading="lazy" draggable="false" class="gf-sticker-suggest-img" />
            <span class="gf-sticker-suggest-label">{{ item.label }}</span>
          </div>
        </template>
        <div v-else class="gf-sticker-suggest-status">
          {{ loading ? t('common.loadingShort') : t('stickers.suggest.noResults') }}
        </div>
      </div>
    </Transition>
  </Teleport>
  <div aria-live="polite" class="sr-only">
    {{ open ? (candidates.length ? t('stickers.suggest.resultCount', { count: candidates.length }, candidates.length) : t('stickers.suggest.noResults')) : '' }}
  </div>
</template>

<style>
.gf-sticker-suggest {
  position: fixed;
  /* 挂在 body 上：须高于回复浮层/快捷发布弹窗（100）与 Vditor 全屏（90），与页内弹出菜单同级 */
  z-index: 300;
  max-height: calc(6 * 44px + 8px);
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 4px;
  border: 1px solid color-mix(in oklch, var(--gf-color-base-content) 14%, transparent);
  border-radius: 12px;
  background: var(--color-base-100, #fff);
  box-shadow: 0 8px 24px 0 rgba(0, 0, 0, 0.12);
  scrollbar-width: thin;
}

.gf-sticker-suggest-option {
  position: relative;
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  padding: 4px 10px 4px 8px;
  border-radius: 8px;
  cursor: pointer;
  -webkit-user-select: none;
  user-select: none;
}

.gf-sticker-suggest-option:hover {
  background: color-mix(in oklch, var(--gf-color-base-content) 6%, transparent);
}

/* active 不只靠颜色：前缘 3px 强调条 + aria-selected + 背景 */
.gf-sticker-suggest-option.is-active {
  background: color-mix(in oklch, var(--gf-color-primary) 12%, transparent);
}

.gf-sticker-suggest-option.is-active::before {
  content: '';
  position: absolute;
  inset-inline-start: 0;
  top: 10px;
  bottom: 10px;
  width: 3px;
  border-radius: 999px;
  background: var(--gf-color-primary);
}

.gf-sticker-suggest-img {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  object-fit: contain;
  pointer-events: none;
}

.gf-sticker-suggest-label {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  font-size: 14px;
  color: var(--gf-color-base-content);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gf-sticker-suggest-status {
  padding: 10px 12px;
  font-size: 13px;
  color: color-mix(in oklch, var(--gf-color-base-content) 55%, transparent);
}

.gf-sticker-suggest-enter-active {
  transition:
    opacity 0.15s cubic-bezier(0.2, 0, 0, 1),
    transform 0.15s cubic-bezier(0.2, 0, 0, 1);
}

.gf-sticker-suggest-enter-from {
  opacity: 0;
  transform: translateY(-4px);
}

@media (prefers-reduced-motion: reduce) {
  .gf-sticker-suggest-enter-active {
    transition: none;
  }

  .gf-sticker-suggest-enter-from {
    opacity: 1;
    transform: none;
  }
}
</style>

/**
 * 编辑器内手动输入 `[:sticker:` 时的贴纸候选会话（MADR 0106 补充）。
 *
 * 光标前识别到未闭合片段即打开候选，按名字/显示名筛选「我的 → 官方 → 已添加的表情包」；
 * 选中后用完整 token 整段替换片段，编辑器随即把它显示为贴纸图。片段补齐为完整
 * token 时会话自然关闭（闭合的 token 不再是片段）。DOM 采集与替换由宿主编辑器桥接。
 */
import { computed, ref, watch, type CSSProperties } from 'vue'
import type { StickerItem } from '@gooseforum/client'
import { isStickerUsable, useMyStickers } from '@/site/composables/useMyStickers'
import { rememberStickerUrls } from '@/site/composables/useResolvedStickers'
import { useStickerLibrary } from '@/site/composables/useStickerLibrary'
import { useStickerMarket } from '@/site/composables/useStickerMarket'
import {
  extractStickerQuery,
  rankStickerSuggestions,
  type StickerSuggestion,
} from '@/site/utils/sticker-token'

export const STICKER_SUGGEST_LISTBOX_ID = 'gf-sticker-suggest-listbox'
export const STICKER_SUGGEST_OPTION_ID_PREFIX = 'gf-sticker-suggest-option-'

const PANEL_WIDTH = 288
const ROW_HEIGHT = 44
const MAX_VISIBLE_ROWS = 6
const GAP = 6
const VIEWPORT_MARGIN = 8

export interface StickerSuggestBridge {
  getMentionContext(): { prefix: string, rect: DOMRect | null, inCode: boolean, element: HTMLElement | null } | null
  /** 以光标为终点向前删除 length 个字符并插入 token（不补空格） */
  replaceStickerQuery(length: number, token: string): boolean
}

export function useStickerSuggest(options: {
  bridge: () => StickerSuggestBridge | null
  label: (item: StickerItem) => string
}) {
  const library = useStickerLibrary()
  const mine = useMyStickers()
  const market = useStickerMarket()
  const packItems = ref<StickerItem[]>([])

  const open = ref(false)
  const query = ref('')
  const queryLength = ref(0)
  const activeIndex = ref(0)
  const rect = ref<DOMRect | null>(null)
  const editorElement = ref<HTMLElement | null>(null)
  /** Escape 关闭后，同一片段不再自动重开；片段一变（继续输入/删除）即恢复 */
  let dismissedPrefix: string | null = null
  let sourcesRequested = false

  const sources = computed<StickerSuggestion[]>(() => {
    const items = [...mine.usable.value, ...library.stickers.value, ...packItems.value.filter(isStickerUsable)]
    return items.map((item) => ({ name: item.name, url: item.url, label: options.label(item) }))
  })
  const candidates = computed(() => (open.value ? rankStickerSuggestions(sources.value, query.value) : []))
  const loading = computed(() => library.loading.value || (mine.loading.value && !mine.ready.value))

  function requestSources() {
    if (sourcesRequested) return
    sourcesRequested = true
    void library.ensureStickers()
    void mine.load()
    void market.ensureMyPacks().then(async () => {
      const lists = await Promise.allSettled(market.myPacks.value.map((pack) => market.loadPackStickers(pack.id)))
      packItems.value = lists.flatMap((list) => (list.status === 'fulfilled' ? list.value : []))
    })
  }

  function refresh() {
    const ctx = options.bridge()?.getMentionContext()
    const token = ctx && !ctx.inCode ? extractStickerQuery(ctx.prefix) : null
    if (!ctx || !token) {
      dismissedPrefix = null
      close()
      return
    }
    if (dismissedPrefix === ctx.prefix) return
    dismissedPrefix = null
    requestSources()
    if (!open.value || query.value !== token.query) activeIndex.value = 0
    open.value = true
    query.value = token.query
    queryLength.value = token.length
    rect.value = ctx.rect
    editorElement.value = ctx.element
  }

  function close() {
    open.value = false
    query.value = ''
    queryLength.value = 0
    activeIndex.value = 0
    rect.value = null
  }

  function dismiss() {
    dismissedPrefix = options.bridge()?.getMentionContext()?.prefix ?? null
    close()
  }

  function select(item: StickerSuggestion) {
    if (!open.value) return
    const length = queryLength.value
    close()
    rememberStickerUrls([item])
    options.bridge()?.replaceStickerQuery(length, `[:sticker:${item.name}:]`)
  }

  /** 会话打开时接管上下/回车/Tab/Esc；返回 true 表示已处理（宿主阻止默认行为与冒泡） */
  function handleKeydown(event: KeyboardEvent): boolean {
    if (!open.value || event.isComposing) return false
    const count = candidates.value.length
    switch (event.key) {
      case 'ArrowDown':
      case 'ArrowUp':
        if (!count) return false
        activeIndex.value = (activeIndex.value + (event.key === 'ArrowDown' ? 1 : -1) + count) % count
        return true
      case 'Enter':
      case 'Tab':
        if (!count || event.shiftKey) return false
        select(candidates.value[activeIndex.value])
        return true
      case 'Escape':
        dismiss()
        return true
      default:
        return false
    }
  }

  /** 贴近光标的浮层：默认在光标下方，空间不足时翻到上方，水平方向不越出视口 */
  const panelStyle = computed<CSSProperties>(() => {
    const caret = rect.value
    if (!caret || typeof window === 'undefined') return { visibility: 'hidden' }
    const width = Math.min(PANEL_WIDTH, window.innerWidth - VIEWPORT_MARGIN * 2)
    const rows = Math.min(Math.max(candidates.value.length, 1), MAX_VISIBLE_ROWS)
    const height = rows * ROW_HEIGHT + 8
    const left = Math.min(Math.max(caret.left, VIEWPORT_MARGIN), window.innerWidth - width - VIEWPORT_MARGIN)
    const below = window.innerHeight - caret.bottom
    const top = below < height + GAP && caret.top > height + GAP
      ? caret.top - height - GAP
      : caret.bottom + GAP
    return { left: `${left}px`, top: `${top}px`, width: `${width}px` }
  })

  // 编辑元素即 combobox 触发器（与 mention 会话同一 a11y 约定）
  watch([open, activeIndex, candidates, editorElement], ([isOpen], [, , , previous]) => {
    const element = editorElement.value
    if (previous && previous !== element) clearAria(previous)
    if (!element) return
    if (!isOpen) {
      clearAria(element)
      return
    }
    element.setAttribute('aria-expanded', 'true')
    element.setAttribute('aria-controls', STICKER_SUGGEST_LISTBOX_ID)
    const candidate = candidates.value[activeIndex.value]
    if (candidate) element.setAttribute('aria-activedescendant', `${STICKER_SUGGEST_OPTION_ID_PREFIX}${activeIndex.value}`)
    else element.removeAttribute('aria-activedescendant')
  })

  function clearAria(element: HTMLElement) {
    if (element.getAttribute('aria-controls') !== STICKER_SUGGEST_LISTBOX_ID) return
    for (const name of ['aria-expanded', 'aria-controls', 'aria-activedescendant']) element.removeAttribute(name)
  }

  return { open, query, candidates, activeIndex, loading, panelStyle, refresh, close, select, handleKeydown }
}

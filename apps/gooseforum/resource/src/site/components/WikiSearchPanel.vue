<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ArrowDown, ArrowUp, CornerDownLeft, Search, X } from '@lucide/vue'
import { useDialogAccessibility } from '@/site/composables/useDialogAccessibility'
import { useI18n } from 'vue-i18n'
import {
  closePanel,
  highlightQuery,
  openPanel,
  panelOpen,
  saveWikiJumpState,
  searchWikiPages,
  type WikiSearchItem,
  type WikiSearchResponse,
} from '@/runtime/use-wiki-search'

const { t } = useI18n()

// ---- 面板开关（模块级共享）+ 可达性 ----
const { panelRef } = useDialogAccessibility(panelOpen, {
  onClose: () => closePanel(),
})

// ---- 搜索状态机：idle / loading / results / empty / unavailable ----
const query = ref('')
const results = ref<WikiSearchItem[]>([])
const total = ref(0)
const status = ref<'idle' | 'loading' | 'results' | 'empty' | 'unavailable'>('idle')
const activeIndex = ref(-1)
const inputRef = ref<HTMLInputElement | null>(null)
const listRef = ref<HTMLDivElement | null>(null)

let debounceTimer: number | undefined
let abortController: AbortController | null = null

const hasQuery = computed(() => query.value.trim().length > 0)
const activeItem = computed(() => (activeIndex.value >= 0 ? results.value[activeIndex.value] : null))
const activeId = computed(() => (activeItem.value ? `wiki-search-option-${activeIndex.value}` : undefined))

// 按命名空间分组展示（对齐 WikiSidebar 分组），组内保持 score 降序（后端已排序）。
// 每项携带全局索引 globalIndex（面板内唯一 id / aria-activedescendant 用）。
const groupedResults = computed(() => {
  const groups: Array<{ namespace: string; items: Array<WikiSearchItem & { globalIndex: number }> }> = []
  const byNamespace = new Map<string, WikiSearchItem[]>()
  for (const item of results.value) {
    const key = item.namespace || '—'
    if (!byNamespace.has(key)) byNamespace.set(key, [])
    byNamespace.get(key)!.push(item)
  }
  let cursor = 0
  for (const [namespace, items] of byNamespace) {
    groups.push({
      namespace,
      items: items.map((item) => ({ ...item, globalIndex: cursor++ })),
    })
  }
  return groups
})

function runSearch(raw: string) {
  window.clearTimeout(debounceTimer)
  abortController?.abort()
  const trimmed = raw.trim()
  if (!trimmed) {
    status.value = 'idle'
    results.value = []
    total.value = 0
    activeIndex.value = -1
    return
  }
  status.value = 'loading'
  debounceTimer = window.setTimeout(async () => {
    abortController = new AbortController()
    try {
      const resp: WikiSearchResponse = await searchWikiPages(trimmed, abortController.signal)
      if (query.value.trim() !== resp.query) return // 竞态：输入已变化
      if (resp.searchUnavailable) {
        status.value = 'unavailable'
        results.value = []
        total.value = 0
        activeIndex.value = -1
        return
      }
      results.value = resp.items
      total.value = resp.total
      status.value = resp.items.length ? 'results' : 'empty'
      activeIndex.value = resp.items.length ? 0 : -1
    } catch (err) {
      if ((err as Error).name === 'AbortError') return
      status.value = 'unavailable'
      results.value = []
      activeIndex.value = -1
    }
  }, 120)
}

watch(query, (value) => runSearch(value))

// ---- 键盘导航：↑↓ 循环选择 / Enter 跳转 / Esc 由 useDialogAccessibility 处理 ----
function handleInputKeydown(event: KeyboardEvent) {
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    if (results.value.length === 0) return
    activeIndex.value = (activeIndex.value + 1) % results.value.length
    scrollActiveIntoView()
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    if (results.value.length === 0) return
    activeIndex.value = (activeIndex.value - 1 + results.value.length) % results.value.length
    scrollActiveIntoView()
  } else if (event.key === 'Enter') {
    if (activeItem.value) {
      event.preventDefault()
      jumpTo(activeItem.value)
    }
  }
}

function scrollActiveIntoView() {
  listRef.value
    ?.querySelector<HTMLElement>(`[id="${activeId.value}"]`)
    ?.scrollIntoView({ block: 'nearest' })
}

/** 跳转到目标页：缓存连续定位上下文，导航到 <path>#<firstAnchor>。 */
function jumpTo(item: WikiSearchItem) {
  const anchor = item.anchors[0] ?? ''
  if (item.anchors.length) {
    saveWikiJumpState({ query: query.value, anchors: item.anchors })
  }
  closePanel()
  window.location.assign(`/wiki/${encodeWikiPath(item.path)}${anchor ? `#${anchor}` : ''}`)
}

function encodeWikiPath(path: string): string {
  return path.split('/').map((segment) => encodeURIComponent(segment)).join('/')
}

// ---- 键盘呼出：wiki 模式下 / 与 Ctrl/Cmd+K（面板组件唯一绑定，避免多入口重复） ----
function handleGlobalKeydown(event: KeyboardEvent) {
  const target = event.target as HTMLElement | null
  const typing = target?.matches('input, textarea, select, [contenteditable]')
  if (event.key === 'Escape') return // 交给 useDialogAccessibility
  const meta = event.ctrlKey || event.metaKey
  if ((event.key.toLowerCase() === 'k' && meta) || event.key === '/') {
    if (typing) return // 输入中不劫持
    event.preventDefault()
    if (panelOpen.value) {
      closePanel()
    } else {
      openWithFocus()
    }
  }
}

function openWithFocus() {
  openPanel()
}

// 面板打开时把焦点交给输入框（useDialogAccessibility 已处理移入，这里补充清空选择态）。
watch(panelOpen, (open) => {
  if (!open) {
    query.value = ''
    results.value = []
    total.value = 0
    status.value = 'idle'
    activeIndex.value = -1
    window.clearTimeout(debounceTimer)
    abortController?.abort()
  } else {
    requestAnimationFrame(() => {
      if (panelOpen.value) inputRef.value?.focus()
    })
  }
})

onMounted(() => document.addEventListener('keydown', handleGlobalKeydown))
onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleGlobalKeydown)
  window.clearTimeout(debounceTimer)
  abortController?.abort()
  // 离开 wiki 模式时关闭共享面板，避免重新进入时残留弹出。
  if (panelOpen.value) closePanel()
})

// 供模板使用
const copy = {
  placeholder: t('wikiSearch.placeholder'),
  emptyTitle: t('wikiSearch.emptyTitle'),
  emptyHint: t('wikiSearch.emptyHint'),
  loading: t('wikiSearch.loading'),
  unavailable: t('wikiSearch.unavailable'),
  titleHit: t('wikiSearch.titleHit'),
  bodyHit: t('wikiSearch.bodyHit'),
  resultCount: (count: number) => t('wikiSearch.resultCount', { count }),
  hint: t('wikiSearch.hint'),
  close: t('wikiSearch.close'),
}
const highlight = highlightQuery
</script>

<template>
  <Teleport to="body">
    <div
      v-if="panelOpen"
      ref="panelRef"
      class="fixed inset-0 z-[80] flex items-center justify-center px-3"
      role="dialog"
      aria-modal="true"
      :aria-label="t('wikiSearch.panelLabel')"
    >
      <button
        class="absolute inset-0 -z-10 bg-neutral/30 backdrop-blur-[2px]"
        type="button"
        tabindex="-1"
        :aria-label="copy.close"
        @click="closePanel"
      />
      <div class="gf-card w-full max-w-xl overflow-hidden shadow-lg max-h-[85vh] bg-base-100/85 backdrop-blur-xl">
        <div class="flex h-12 items-center gap-2 border-b border-line px-4">
          <Search class="h-5 w-5 shrink-0 text-icon-muted" aria-hidden="true" />
          <input
            ref="inputRef"
            v-model="query"
            type="search"
            role="combobox"
            :aria-expanded="status === 'results'"
            :aria-controls="'wiki-search-listbox'"
            :aria-activedescendant="activeId"
            aria-autocomplete="list"
            autocomplete="off"
            spellcheck="false"
            class="h-12 min-w-0 flex-1 bg-transparent text-base text-base-content outline-none placeholder:text-base-content/45"
            :placeholder="copy.placeholder"
            @keydown="handleInputKeydown"
          />
          <span v-if="hasQuery" class="hidden shrink-0 items-center gap-1 rounded bg-base-200 px-1.5 py-0.5 text-[11px] font-semibold text-base-content/55 sm:inline-flex">
            <ArrowDown class="h-3 w-3" aria-hidden="true" />
            <ArrowUp class="h-3 w-3" aria-hidden="true" />
          </span>
          <button
            type="button"
            class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-icon-muted transition-colors hover:bg-base-300 hover:text-base-content"
            :aria-label="copy.close"
            @click="closePanel"
          >
            <X class="h-5 w-5" />
          </button>
        </div>

        <div class="max-h-[55vh] overflow-y-auto">
          <div v-if="status === 'loading'" class="flex items-center gap-2 px-4 py-4 text-sm text-base-content/55">
            <span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-line border-t-primary" />
            {{ copy.loading }}
          </div>

          <div v-else-if="status === 'unavailable'" class="px-4 py-6 text-center text-sm text-base-content/55">
            {{ copy.unavailable }}
          </div>

          <div v-else-if="status === 'empty'" class="px-4 py-6 text-center">
            <p class="text-sm font-medium text-base-content">{{ copy.emptyTitle }}</p>
            <p class="mt-1 text-xs text-base-content/55">{{ copy.emptyHint }}</p>
          </div>

          <div
            v-else-if="status === 'results'"
            id="wiki-search-listbox"
            ref="listRef"
            role="listbox"
            :aria-label="t('wikiSearch.resultsLabel')"
            class="py-1"
          >
            <div v-for="group in groupedResults" :key="group.namespace">
              <div class="flex items-center gap-2 px-4 pt-2.5 pb-1">
                <span class="text-[11px] font-bold uppercase tracking-wide text-base-content/55">{{ group.namespace }}</span>
                <span class="h-px flex-1 bg-line/70" />
              </div>
              <button
                v-for="(item) in group.items"
                :id="`wiki-search-option-${item.globalIndex}`"
                :key="`${item.path}-${item.anchors.join(',')}`"
                type="button"
                role="option"
                :aria-selected="item.globalIndex === activeIndex"
                class="block w-full px-4 py-2.5 text-left transition-colors"
                :class="item.globalIndex === activeIndex ? 'bg-info/10' : 'hover:bg-base-200/60'"
                @mouseenter="activeIndex = item.globalIndex"
                @click="jumpTo(item)"
              >
                <span class="flex min-w-0 items-center gap-2">
                  <span class="min-w-0 flex-1 truncate text-sm font-semibold text-base-content" v-html="highlight(item.title, query)" />
                  <span
                    class="shrink-0 rounded px-1.5 py-0.5 text-[10px] font-semibold"
                    :class="item.hitType === 'title' ? 'bg-primary/10 text-primary' : 'bg-base-200 text-base-content/55'"
                  >
                    {{ item.hitType === 'title' ? copy.titleHit : copy.bodyHit }}
                  </span>
                </span>
                <span class="mt-0.5 block truncate text-xs text-base-content/45">{{ item.namespace }} › {{ item.path }}</span>
                <span v-if="item.heading" class="mt-0.5 block truncate text-xs text-base-content/60">§ {{ item.heading }}</span>
                <span class="mt-1 block text-[13px] leading-5 text-base-content/70" v-html="item.snippet" />
              </button>
            </div>
          </div>

          <div v-else class="px-4 py-4 text-sm text-base-content/45">
            {{ t('wikiSearch.idleHint') }}
          </div>
        </div>

        <div class="flex items-center justify-between border-t border-line bg-base-200/40 px-4 py-2 text-[11px] text-base-content/45">
          <span v-if="status === 'results'">{{ copy.resultCount(total) }}</span>
          <span v-else>{{ copy.hint }}</span>
          <span class="inline-flex items-center gap-1">
            <CornerDownLeft class="h-3 w-3" aria-hidden="true" />
            {{ t('wikiSearch.enterToJump') }}
          </span>
        </div>
      </div>
    </div>
  </Teleport>
</template>

import { readonly, ref } from 'vue'

/**
 * Wiki 局内搜索面板的模块级共享状态。
 * 桌面侧栏与移动抽屉各有入口按钮，但面板是全局唯一（Teleport 到 body），
 * 故用模块级 ref 共享 open 状态，两个入口 + 键盘呼出都能控制同一面板。
 */
const panelOpen = ref(false)
export { panelOpen }

export function openPanel() {
  panelOpen.value = true
}

export function closePanel() {
  panelOpen.value = false
}

export function togglePanel() {
  panelOpen.value = !panelOpen.value
}

export function useWikiSearchPanel() {
  return {
    panelOpen: readonly(panelOpen),
    openPanel,
    closePanel,
    togglePanel,
  }
}

/** wiki 站内搜索结果条目（与后端 WikiSearchJSONResp.items 对齐）。 */
export interface WikiSearchItem {
  namespace: string
  path: string
  title: string
  titleHit: boolean
  heading?: string
  anchors: string[]
  snippet: string
  score: number
  hitType: 'title' | 'body'
}

export interface WikiSearchResponse {
  query: string
  total: number
  items: WikiSearchItem[]
  searchUnavailable?: boolean
}

/** 调用 /api/wiki/search（JSON API，公开只读）。 */
export async function searchWikiPages(query: string, signal?: AbortSignal): Promise<WikiSearchResponse> {
  const params = new URLSearchParams({ q: query, limit: '12' })
  const response = await fetch(`/api/wiki/search?${params.toString()}`, {
    headers: { Accept: 'application/json' },
    signal,
  })
  const data = (await response.json().catch(() => undefined)) as { result?: WikiSearchResponse } | undefined
  const result = data?.result ?? { query, total: 0, items: [] as WikiSearchItem[] }
  return result
}

/**
 * 去掉 snippet 中的高亮标记，得到纯文本（供 aria-label 等可访问性描述）。
 */
export function stripHighlightMarkup(text: string): string {
  return text.replace(/<\/?mark>/g, '').trim()
}

const htmlEntityPattern = /^&(?:#\d+|#x[\da-f]+|[a-z][\da-z]+);$/i

/**
 * Escapes untrusted search text while preserving entities already encoded by
 * the renderer and the highlight tags emitted by Meilisearch.
 */
function escapeSearchText(text: string): string {
  return text.replace(/[&<>"']/g, (character) => {
    if (character === '&') return '&amp;'
    if (character === '<') return '&lt;'
    if (character === '>') return '&gt;'
    if (character === '"') return '&quot;'
    return '&#39;'
  })
}

function escapeSearchTextPreservingEntities(text: string): string {
  return text.replace(/&(?:#\d+|#x[\da-f]+|[a-z][\da-z]+);|[<>"'&]/gi, (token) => {
    if (htmlEntityPattern.test(token)) return token
    return escapeSearchText(token)
  })
}

/**
 * Allows only bare <mark> tags in server-provided snippets. All other markup
 * is escaped, including tag-shaped text authored in Wiki Markdown.
 */
export function sanitizeHighlightMarkup(markup: string): string {
  const markTagPattern = /<\/?mark>/gi
  let safeMarkup = ''
  let lastIndex = 0
  for (const match of markup.matchAll(markTagPattern)) {
    const tagIndex = match.index ?? 0
    safeMarkup += escapeSearchTextPreservingEntities(markup.slice(lastIndex, tagIndex))
    safeMarkup += match[0].toLowerCase()
    lastIndex = tagIndex + match[0].length
  }
  return safeMarkup + escapeSearchTextPreservingEntities(markup.slice(lastIndex))
}

/**
 * 复制命中高亮的 <mark> 语义到纯文本：把 query 词在文本中出现的位置包裹 <mark>，
 * 供标题/面包屑等非 _formatted 字段的展示高亮，同时先转义不可信文本。
 */
export function highlightQuery(text: string, query: string): string {
  const needle = query.trim()
  if (!needle || !text) return escapeSearchTextPreservingEntities(text)
  const escaped = needle.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const matcher = new RegExp(escaped, 'gi')
  let highlighted = ''
  let lastIndex = 0
  for (const match of text.matchAll(matcher)) {
    const matchIndex = match.index ?? 0
    highlighted += escapeSearchTextPreservingEntities(text.slice(lastIndex, matchIndex))
    highlighted += `<mark>${escapeSearchText(match[0])}</mark>`
    lastIndex = matchIndex + match[0].length
  }
  return highlighted + escapeSearchTextPreservingEntities(text.slice(lastIndex))
}

/** 面板跳转前缓存当前搜索上下文，供目标页连续定位（Enter/⌘G）与首个命中高亮。 */
export interface WikiJumpState {
  query: string
  anchors: string[]
}

const JUMP_KEY = 'wiki-search-jump'

export function saveWikiJumpState(state: WikiJumpState) {
  try {
    sessionStorage.setItem(JUMP_KEY, JSON.stringify(state))
  } catch {
    // sessionStorage 不可用时忽略（连续定位降级为首个锚点定位）
  }
}

export function consumeWikiJumpState(): WikiJumpState | null {
  try {
    const raw = sessionStorage.getItem(JUMP_KEY)
    if (!raw) return null
    sessionStorage.removeItem(JUMP_KEY)
    const parsed = JSON.parse(raw) as WikiJumpState
    return parsed && Array.isArray(parsed.anchors) ? parsed : null
  } catch {
    return null
  }
}

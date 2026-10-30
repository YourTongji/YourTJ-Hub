/**
 * 表情包 token（[:sticker:name:]）纯文本分段解析（MADR 0030）。
 *
 * 私信消息是纯文本存储（服务端只渲染帖子 markdown 链路），聊天气泡在
 * 客户端做安全分段渲染：识别到的启用表情包渲染为内联图片，未知/停用
 * token 保持原文。正则与规则对齐服务端 markdown2html.stickerTokenRe
 * （name ≤64、无空白/冒号/方括号）。
 */

const STICKER_TOKEN_RE = /\[:sticker:([^\s:[\]]{1,64}):]/g

export function stickerNamesInContent(contents: readonly string[]): string[] {
  return [...new Set(contents.flatMap((content) =>
    [...content.matchAll(STICKER_TOKEN_RE)].map((match) => match[1]),
  ))]
}

export type MessageSegment =
  | { type: 'text'; text: string }
  | { type: 'sticker'; name: string; url: string }

/**
 * 把消息文本切分为文本/贴纸段序列。
 * 未识别（不在 urlByName 中）或停用的 token 保持原始文本；
 * 无 token 时返回单文本段（内容与输入一致）。
 */
export function parseStickerSegments(content: string, urlByName: Map<string, string>): MessageSegment[] {
  if (!content.includes('[:sticker:')) {
    return [{ type: 'text', text: content }]
  }
  const segments: MessageSegment[] = []
  let cursor = 0
  for (const match of content.matchAll(STICKER_TOKEN_RE)) {
    const start = match.index ?? 0
    const url = urlByName.get(match[1])
    if (!url || start < cursor) continue
    if (start > cursor) {
      segments.push({ type: 'text', text: content.slice(cursor, start) })
    }
    segments.push({ type: 'sticker', name: match[1], url })
    cursor = start + match[0].length
  }
  if (cursor < content.length) {
    segments.push({ type: 'text', text: content.slice(cursor) })
  }
  return segments
}

/** 纯文本场景（title/aria-label 等属性）用的 token 剥离。 */
export function stripStickerTokens(content: string): string {
  return content.replace(STICKER_TOKEN_RE, '')
}

/**
 * 列表/卡片预览的渲染段：未知或停用 token 直接丢弃，不暴露原文
 * （issue #1304）。文本段剔除残留 token，无内容的段不保留。
 */
export function renderableStickerSegments(content: string, urlByName: Map<string, string>): MessageSegment[] {
  const segments: MessageSegment[] = []
  for (const segment of parseStickerSegments(content, urlByName)) {
    if (segment.type === 'sticker') {
      segments.push(segment)
      continue
    }
    const text = stripStickerTokens(segment.text)
    if (text) segments.push({ type: 'text', text })
  }
  return segments
}

/**
 * 小字视图（列表行、置顶条）的纯文本贴纸文案：可解析的贴纸显示 [名字]，
 * 未知/停用/个人贴纸省略。卡片与预览仍用 renderableStickerSegments 渲染图片。
 */
export function stickerLabelText(content: string, urlByName: Map<string, string>): string {
  const parts = renderableStickerSegments(content, urlByName).map((segment) =>
    segment.type === 'sticker' ? `[${segment.name}]` : segment.text,
  )
  return parts.join('').replace(/\s+/g, ' ').trim()
}

/** 预览是否还有可见内容：全部 token 都未解析时不应留下空行。 */
export function hasRenderableStickerText(content: string, urlByName: Map<string, string>): boolean {
  return renderableStickerSegments(content, urlByName)
    .some((segment) => segment.type === 'sticker' || segment.text.trim() !== '')
}

/**
 * 会话列表预览用的可读标签：不暴露随机 token 名；未知/停用 token
 * 同样显示统一文案（预览层不做启停区分）。
 */
export function stickerPreviewLabel(content: string, stickerLabel?: string): string {
  return content.replace(STICKER_TOKEN_RE, (_, name: string) => `[${stickerLabel ?? name}]`)
}

/** 贴纸 token 在纯文本内容里的区间（[start, end)）。 */
type StickerDeleteRange = { start: number; end: number }

export type StickerDeleteDirection = 'backward' | 'forward'

const STICKER_TOKEN_PREFIX = '[:sticker:'
const STICKER_NAME_MAX = 64

function isStickerNameChar(char: string | undefined): boolean {
  return char !== undefined && !/[\s:[\]]/.test(char)
}

/**
 * content 中所有贴纸 token 与未闭合片段的区间。片段 = `[:sticker:` + 最多 64 个
 * 名字字符且不以 `:]` 收尾（粘贴/撤销/逐字输入留下的 `[:sticker:na`）。
 * 前缀后紧跟空白/`:`/`[`/`]` 说明没有名字在输入中，不算片段。
 */
function stickerAtomSpans(content: string): StickerDeleteRange[] {
  const spans: StickerDeleteRange[] = []
  for (
    let start = content.indexOf(STICKER_TOKEN_PREFIX);
    start !== -1;
    start = content.indexOf(STICKER_TOKEN_PREFIX, start + 1)
  ) {
    const nameStart = start + STICKER_TOKEN_PREFIX.length
    let end = nameStart
    while (end - nameStart < STICKER_NAME_MAX && isStickerNameChar(content[end])) end++
    // 名字超过 64 字符：不是合法 token 也不是片段，交回默认删除
    if (isStickerNameChar(content[end])) continue
    const hasName = end > nameStart
    if (hasName && content.startsWith(':]', end)) {
      spans.push({ start, end: end + 2 })
      continue
    }
    // 名字后紧跟未闭合的 `:` 也算片段的一部分，否则 `[:sticker:huaji:` 会残留 `[:sticker:huaji`
    if (hasName && content[end] === ':') {
      spans.push({ start, end: end + 1 })
      continue
    }
    if (hasName || end === content.length) spans.push({ start, end })
  }
  return spans
}

/**
 * caret 处应整体删除的贴纸区间（issue #1332）：token 在编辑器里是纯文本，
 * 逐字符删除会留下 `[:sticker:na:` 这类残缺原文。caret 紧贴 token
 * （backward 紧跟 token 之后、forward 正好在 token 起点）或落在 token 内部时，
 * token/片段作为一个原子块整体删除。其余情况返回 null，交回默认删除行为。
 */
export function stickerDeleteRangeAt(
  content: string,
  caret: number,
  direction: StickerDeleteDirection,
): StickerDeleteRange | null {
  for (const span of stickerAtomSpans(content)) {
    const hit = direction === 'backward'
      ? caret > span.start && caret <= span.end
      : caret >= span.start && caret < span.end
    if (hit) return span
  }
  return null
}

// 个人上传表情名为 u_ + 48 位小写十六进制（服务端生成规则），可被精确识别。
const PERSONAL_STICKER_NAME_RE = /^u_[0-9a-f]{48}$/
// 市场表情名为 m_ + 40 位小写十六进制（issue #1429）；单张被停用后同样无法解析。
const GENERATED_STICKER_TOKEN_RE = /\[:sticker:(?:u_[0-9a-f]{48}|m_[0-9a-f]{40}):\]/g

/** 贴纸名（不含 [:sticker:…:] 包裹）是否为服务端生成的个人贴纸名。 */
export function isPersonalStickerName(name: string): boolean {
  return PERSONAL_STICKER_NAME_RE.test(name)
}

/**
 * 消息正文渲染用的生成名表情遮蔽：待审/被拒的个人 token 与已停用的市场
 * token 无法解析，保持原文会把随机名展示给收件人（issue #1071），统一替换为
 * 本地化占位。官方与可解析 token 不受影响。
 */
export function maskPersonalStickerTokens(content: string, stickerLabel: string): string {
  return content.replace(GENERATED_STICKER_TOKEN_RE, `[${stickerLabel}]`)
}

/** 光标前正在手动输入的贴纸 token：`[:sticker:` + 至多 64 个名字字符，尚未闭合。 */
export interface StickerQuery {
  /** 片段长度（含 `[:sticker:` 前缀），选中候选时整段替换 */
  length: number
  query: string
}

const STICKER_QUERY_RE = /\[:sticker:([^\s:[\]]{0,64})$/

/** 从光标前文本识别手动输入中的 token 片段；被奇数个反斜杠转义时不算。 */
export function extractStickerQuery(prefix: string): StickerQuery | null {
  const match = STICKER_QUERY_RE.exec(prefix)
  if (!match) return null
  let slashes = 0
  for (let i = match.index - 1; i >= 0 && prefix[i] === '\\'; i--) slashes++
  if (slashes % 2 !== 0) return null
  return { length: match[0].length, query: match[1] }
}

export interface StickerSuggestion {
  name: string
  url: string
  label: string
}

/**
 * 候选排序：名字/显示名前缀命中优先，其次包含；同级保持来源顺序（我的 → 官方 → 已添加的表情包）。
 * 空查询按来源顺序取前 limit 个。名字去重，先出现的来源优先。
 */
export function rankStickerSuggestions(
  items: readonly StickerSuggestion[],
  query: string,
  limit = 8,
): StickerSuggestion[] {
  const q = query.toLowerCase()
  const seen = new Set<string>()
  const scored: Array<{ item: StickerSuggestion, score: number, index: number }> = []
  items.forEach((item, index) => {
    if (seen.has(item.name)) return
    seen.add(item.name)
    if (!q) {
      scored.push({ item, score: 0, index })
      return
    }
    const name = item.name.toLowerCase()
    const label = item.label.toLowerCase()
    const score = name === q ? 4
      : name.startsWith(q) || label.startsWith(q) ? 3
        : name.includes(q) || label.includes(q) ? 2
          : 0
    if (score > 0) scored.push({ item, score, index })
  })
  return scored
    .sort((a, b) => b.score - a.score || a.index - b.index)
    .slice(0, limit)
    .map((entry) => entry.item)
}

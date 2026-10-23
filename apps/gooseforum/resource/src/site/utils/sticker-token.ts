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

// 个人上传表情名为 u_ + 48 位小写十六进制（服务端生成规则），可被精确识别。
const PERSONAL_STICKER_TOKEN_RE = /\[:sticker:u_[0-9a-f]{48}:\]/g

/**
 * 消息正文渲染用的个人表情遮蔽：待审/被拒的个人 token 无法解析，
 * 保持原文会把随机名展示给收件人（issue #1071），统一替换为本地化占位。
 * 官方与可解析 token 不受影响。
 */
export function maskPersonalStickerTokens(content: string, stickerLabel: string): string {
  return content.replace(PERSONAL_STICKER_TOKEN_RE, `[${stickerLabel}]`)
}

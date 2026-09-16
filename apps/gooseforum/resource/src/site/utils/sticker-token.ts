/**
 * 表情包 token（[:sticker:name:]）纯文本分段解析（MADR 0030）。
 *
 * 私信消息是纯文本存储（服务端只渲染帖子 markdown 链路），聊天气泡在
 * 客户端做安全分段渲染：识别到的启用表情包渲染为内联图片，未知/停用
 * token 保持原文。正则与规则对齐服务端 markdown2html.stickerTokenRe
 * （name ≤64、无空白/冒号/方括号）。
 */

const STICKER_TOKEN_RE = /\[:sticker:([^\s:[\]]{1,64}):]/g

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

/**
 * 会话列表预览用的可读标签：把 token 缩写为 [name]，
 * 未知/停用 token 同样缩写（预览层不做启停区分）。
 */
export function stickerPreviewLabel(content: string): string {
  return content.replace(STICKER_TOKEN_RE, (_, name: string) => `[${name}]`)
}

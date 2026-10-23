import { hasRenderableStickerText } from '@/site/utils/sticker-token'

const fallbackDescriptions = [
  '(｀・ω・´)',
  '( ´ ▽ ` )ﾉ',
  '(ง •̀_•́)ง',
  '(｡･ω･｡)',
  '(￣▽￣)ノ',
  '(っ´ω`)っ',
  '( •̀ ω •́ )✧',
  '(๑•̀ㅂ•́)و✧',
]

export function topicDescription(topic: { id: number; description?: string }) {
  const description = topic.description?.trim()
  if (description) return description
  return fallbackDescriptions[Math.abs(topic.id) % fallbackDescriptions.length]
}

/**
 * 列表/链接展示文案：标题为空（无标题瞬间）时回退到摘要，再回退到按 ID 稳定的颜文字。
 * Web 各列表统一走这里；SSR 与 Flutter 列表使用同族回退，保证任何列表行都有可识别文案。
 */
export function topicDisplayLabel(id: number, title?: string, description?: string) {
  return title?.trim() || topicDescription({ id, description })
}

/**
 * 列表链接/卡片的可渲染文案。标题是纯文本（不经过 Markdown 管线），token
 * 语义不适用于标题，原样返回并与 SSR topicListLabel 对齐；只有无标题瞬间的
 * 摘要回退需要做贴纸可见性判断：token 全部无法解析时退回颜文字，避免空链接。
 */
export function topicRenderableLabel(
  id: number,
  title: string | undefined,
  description: string | undefined,
  urlByName: Map<string, string>,
) {
  if (title?.trim()) return title.trim()
  const label = topicDescription({ id, description })
  return hasRenderableStickerText(label, urlByName) ? label : topicDescription({ id, description: '' })
}

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

/**
 * 折叠光标附近的纯文本窗口（issue #1332 贴纸原子删除用）。
 *
 * 只读取光标所在的最近块级元素（段落、列表项、表格单元格、标题、代码块、引用、图注；
 * 都找不到时退到编辑器根的直接子节点）内的文本，避免把相邻块的文本拼成假 token；
 * 区间映射回 Range 时跨内联文本节点（`[:sticker:hu` 与 `aji:]` 分处不同 span 也能整段选中）。
 * 光标直接落在编辑器根上（不在任何块内）时返回 null，不读取任何块。
 */
interface CaretTextWindow {
  /** 块内以光标为中心的纯文本（前后各至多 limit 字符） */
  text: string
  /** 光标在 text 中的下标 */
  caret: number
  /** 把 text 内的字符区间映射为 DOM Range；区间无效或为空时 null */
  range(start: number, end: number): Range | null
}

/** 块级元素：光标在最近的一个处停下，相邻块之间不可能拼出跨块 token。 */
const BLOCK_ELEMENTS = new Set(['P', 'LI', 'TD', 'TH', 'DT', 'DD', 'H1', 'H2', 'H3', 'H4', 'H5', 'H6', 'PRE', 'BLOCKQUOTE', 'FIGCAPTION'])

function isBlockElement(node: Node | null): boolean {
  return node instanceof Element && BLOCK_ELEMENTS.has(node.tagName)
}

export function readCaretTextWindow(
  container: Node,
  offset: number,
  limit: number,
  element: HTMLElement,
): CaretTextWindow | null {
  if (container === element || !element.contains(container)) return null

  let block: Node = container
  if (!isBlockElement(container)) {
    for (let current = container.parentNode; current; current = current.parentNode) {
      // 先判编辑器根：不退化成读取整个编辑器（根下相邻的块会被拼成假 token）
      if (current === element) break
      block = current
      if (isBlockElement(current)) break
    }
  }

  const nodes: Array<{ node: Text; start: number }> = []
  let content = ''
  if (block.nodeType === Node.TEXT_NODE) {
    nodes.push({ node: block as Text, start: 0 })
    content = block.textContent ?? ''
  } else {
    const walker = document.createTreeWalker(block, NodeFilter.SHOW_TEXT)
    for (let node = walker.nextNode(); node; node = walker.nextNode()) {
      nodes.push({ node: node as Text, start: content.length })
      content += node.textContent ?? ''
    }
  }

  // Range.toString() 给出与文本节点拼接一致的偏移，元素容器（光标落在节点边界）也适用
  const probe = document.createRange()
  probe.setStart(block, 0)
  probe.setEnd(container, offset)
  const caretAt = probe.toString().length
  if (caretAt > content.length) return null

  const windowStart = Math.max(0, caretAt - limit)
  const windowEnd = Math.min(content.length, caretAt + limit)

  /**
   * 文本下标映射为 Range 端点。起点落在后一节点起点（不把前面的内容包进来），
   * 终点落在前一节点末尾（不把紧邻的 `<br>`/`<img>` 等非文本节点包进来）。
   */
  function pointAt(index: number, edge: 'start' | 'end'): { node: Node; offset: number } | null {
    for (const entry of nodes) {
      const length = entry.node.textContent?.length ?? 0
      const inside = edge === 'end' ? index <= entry.start + length : index < entry.start + length
      if (inside) return { node: entry.node, offset: index - entry.start }
    }
    const last = nodes[nodes.length - 1]
    return last ? { node: last.node, offset: last.node.textContent?.length ?? 0 } : null
  }

  return {
    text: content.slice(windowStart, windowEnd),
    caret: caretAt - windowStart,
    range(start, end) {
      if (start >= end) return null
      const from = pointAt(windowStart + start, 'start')
      const to = pointAt(windowStart + end, 'end')
      if (!from || !to) return null
      const range = document.createRange()
      range.setStart(from.node, from.offset)
      range.setEnd(to.node, to.offset)
      return range
    },
  }
}

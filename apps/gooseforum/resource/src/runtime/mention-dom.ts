// Caret reads and replacements must stay inside their contenteditable root.
export function textBeforeCaret(container: Node, offset: number, max: number, element: HTMLElement): string {
  if (!element.contains(container)) return ''
  const parts: string[] = []
  let remaining = max
  let node: Node | null = container
  let nodeOffset = offset
  // 容器是元素且 offset > 0：先回溯其第 offset-1 个子树末尾文本（caret 位于元素边界时）
  if (node.nodeType !== Node.TEXT_NODE && offset > 0) {
    const child = node.childNodes[offset - 1]
    if (child) {
      node = child
      while (node.lastChild) node = node.lastChild
      nodeOffset = (node.textContent ?? '').length
    }
  }
  while (node && remaining > 0) {
    if (node.nodeType === Node.TEXT_NODE) {
      const text = node.textContent ?? ''
      const take = Math.min(remaining, nodeOffset)
      if (take > 0) parts.unshift(text.slice(nodeOffset - take, nodeOffset))
      remaining -= take
      nodeOffset = 0
    }
    if (node === element) break
    const prev = node.previousSibling
    if (!prev) {
      if (node === element) break
      node = node.parentNode
      continue
    }
    node = prev
    while (node.lastChild) node = node.lastChild
    nodeOffset = (node.textContent ?? '').length
  }
  return parts.join('')
}

export function isInsideFencedCode(prefix: string): boolean {
  let fence = ''
  let inlineLength = 0
  for (const line of prefix.split('\n')) {
    const marker = line.match(/^ {0,3}(`{3,}|~{3,})(.*)$/)
    if (fence) {
      if (marker && marker[1][0] === fence[0] && marker[1].length >= fence.length && !marker[2].trim()) fence = ''
      continue
    }
    if (marker) {
      fence = marker[1]
      inlineLength = 0
      continue
    }
    for (const run of line.matchAll(/`+/g)) {
      if (!inlineLength) inlineLength = run[0].length
      else if (run[0].length === inlineLength) inlineLength = 0
    }
  }
  return Boolean(fence || inlineLength)

}

export function replaceMentionTokenInElement(element: HTMLElement, queryLength: number, replacement: string): boolean {
  const selection = window.getSelection()
  if (!selection || selection.rangeCount === 0) return false
  const range = selection.getRangeAt(0)
  if (!range.collapsed || !element.contains(range.startContainer)) return false

  let node: Node = range.startContainer
  let offset = range.startOffset
  if (node.nodeType !== Node.TEXT_NODE && offset > 0) {
    const child = node.childNodes[offset - 1]
    if (child) {
      node = child
      while (node.lastChild) node = node.lastChild
      offset = (node.textContent ?? '').length
    }
  }
  let remaining = queryLength
  while (remaining > 0) {
    if (node.nodeType === Node.TEXT_NODE) {
      const text = node.textContent ?? ''
      const take = Math.min(remaining, offset)
      offset -= take
      remaining -= take
      if (remaining === 0) break
    }
    if (node === element) return false
    const prev = node.previousSibling
    if (prev) {
      node = prev
      while (node.lastChild) node = node.lastChild
      offset = (node.textContent ?? '').length
    } else {
      const parent = node.parentNode
      if (!parent || parent === element) return false
      node = parent
    }
  }

  const tokenRange = document.createRange()
  tokenRange.setStart(node, offset)
  tokenRange.setEnd(range.startContainer, range.startOffset)
  selection.removeAllRanges()
  selection.addRange(tokenRange)
  return document.execCommand('insertText', false, `${replacement} `)
}

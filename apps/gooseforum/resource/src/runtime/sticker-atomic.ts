import { readCaretTextWindow } from '@/runtime/caret-text'
import { stickerDeleteRangeAt, type StickerDeleteDirection } from '@/site/utils/sticker-token'

/**
 * 窗口取 200 字符：需要上界是因为块内可能是超长粘贴内容，逐键扫描整块没有意义；
 * 200 远超最长 token（64 名字字符 + 前缀/闭合符），token 无论落在光标哪一侧都在窗口内。
 */
const CARET_WINDOW_LIMIT = 200

/** keydown 是否应交给贴纸原子删除处理：IME 组合中与带 Ctrl/Meta/Alt 的组合键一律不干预。 */
export function stickerDeleteDirection(event: KeyboardEvent): StickerDeleteDirection | null {
  if (event.isComposing || event.ctrlKey || event.metaKey || event.altKey) return null
  if (event.key === 'Backspace') return 'backward'
  if (event.key === 'Delete') return 'forward'
  return null
}

/**
 * 光标处删除整个贴纸 token/残缺片段（issue #1332）。
 *
 * token 在编辑器里是纯文本，逐字符删除会留下 `[:sticker:na:` 这类残缺原文；
 * 命中时选中整段并走 document.execCommand('delete') 原生 contenteditable 路径：
 * 触发 input 事件，Vditor 重新渲染并同步 modelValue/撤销栈（同 mention 插入的取舍）。
 * 返回 false 表示交回浏览器默认删除。
 */
export function deleteStickerAtCaret(element: HTMLElement, direction: StickerDeleteDirection): boolean {
  const selection = window.getSelection()
  if (!selection || selection.rangeCount === 0) return false
  const caret = selection.getRangeAt(0)
  if (!caret.collapsed || !element.contains(caret.startContainer)) return false

  const textWindow = readCaretTextWindow(caret.startContainer, caret.startOffset, CARET_WINDOW_LIMIT, element)
  if (!textWindow) return false
  const span = stickerDeleteRangeAt(textWindow.text, textWindow.caret, direction)
  if (!span) return false
  const target = textWindow.range(span.start, span.end)
  if (!target) return false

  selection.removeAllRanges()
  selection.addRange(target)
  return document.execCommand('delete')
}

import { readCaretTextWindow } from '@/runtime/caret-text'
import { STICKER_INLINE_CLASS, stickerInlineAt } from '@/runtime/sticker-inline-editor'
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

export function stickerBeforeInputDirection(event: InputEvent): StickerDeleteDirection | null {
  if (event.isComposing) return null
  if (event.inputType === 'deleteContentBackward') return 'backward'
  if (event.inputType === 'deleteContentForward') return 'forward'
  return null
}

/**
 * 光标处删除整个贴纸 token/残缺片段（issue #1332）。
 *
 * token 在编辑器里是纯文本，逐字符删除会留下 `[:sticker:na:` 这类残缺原文；
 * 命中时选中整段并走 document.execCommand('delete') 原生 contenteditable 路径：
 * 触发 input 事件，Vditor 重新渲染并同步 modelValue/撤销栈（同 mention 插入的取舍）。
 * 已显示为贴纸图的 token（sticker-inline-editor）在不可编辑块里，选区扩到整个块，
 * 否则浏览器拒绝删除不可编辑节点内的文本。
 * 返回 false 表示交回浏览器默认删除。
 */
export function deleteStickerAtCaret(element: HTMLElement, direction: StickerDeleteDirection): boolean {
  const selection = window.getSelection()
  if (!selection || selection.rangeCount === 0) return false
  const caret = selection.getRangeAt(0)
  if (!element.contains(caret.startContainer) || !element.contains(caret.endContainer)) return false
  if (!caret.collapsed) {
    const touched = [...element.querySelectorAll<HTMLElement>(`.${STICKER_INLINE_CLASS}`)]
      .filter((span) => caret.intersectsNode(span))
    if (touched.length === 0) return false
    const target = caret.cloneRange()
    const firstRange = document.createRange()
    firstRange.selectNode(touched[0])
    if (caret.compareBoundaryPoints(Range.START_TO_START, firstRange) > 0) {
      target.setStartBefore(touched[0])
    }
    const last = touched[touched.length - 1]
    const lastRange = document.createRange()
    lastRange.selectNode(last)
    if (caret.compareBoundaryPoints(Range.END_TO_END, lastRange) < 0) {
      target.setEndAfter(last)
    }
    selection.removeAllRanges()
    selection.addRange(target)
    return document.execCommand('delete')
  }

  const textWindow = readCaretTextWindow(caret.startContainer, caret.startOffset, CARET_WINDOW_LIMIT, element)
  if (!textWindow) return false
  const span = stickerDeleteRangeAt(textWindow.text, textWindow.caret, direction)
  if (!span) return false
  const target = textWindow.range(span.start, span.end)
  if (!target) return false
  const startInline = stickerInlineAt(target.startContainer)
  const endInline = stickerInlineAt(target.endContainer)
  if (startInline && element.contains(startInline)) target.setStartBefore(startInline)
  if (endInline && element.contains(endInline)) target.setEndAfter(endInline)

  selection.removeAllRanges()
  selection.addRange(target)
  return document.execCommand('delete')
}

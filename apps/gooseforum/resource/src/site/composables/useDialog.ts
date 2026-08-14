import { onBeforeUnmount, ref, watch } from 'vue'
import type { Ref } from 'vue'

export interface UseDialogOptions {
  /** 是否展示弹窗（打开后执行焦点管理/滚动锁定；关闭时恢复）。 */
  visible: Ref<boolean>
  /** 打开后自动聚焦的选择器；默认聚焦弹窗内第一个可聚焦元素。 */
  initialFocusSelector?: string
}

const FOCUSABLE =
  'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'

/**
 * 排课弹窗共享的无障碍处理（WCAG 4.1.2 / WAI-ARIA APG Dialog）：
 * - 打开：焦点移入弹窗内首个可聚焦元素；body 滚动锁定（overflow hidden + 恢复）
 * - 打开期间：Tab/Shift+Tab 焦点圈禁在弹窗内，Esc 关闭
 * - 关闭：焦点恢复到触发弹窗的元素，滚动锁定解除
 *
 * 模板侧需自行提供 role="dialog" + aria-modal="true" + aria-labelledby（标题 id）。
 * 打开/关闭事件应统一走 `closeDialog()`，保证焦点恢复与滚动解锁。
 */
export function useDialog(options: UseDialogOptions) {
  const { visible, initialFocusSelector } = options
  const dialogRef = ref<HTMLElement | null>(null)
  /** 打开前处于焦点的元素（触发按钮），关闭后焦点恢复于此。 */
  let triggerElement: HTMLElement | null = null
  let restoreScroll = false
  let keyHandler: ((event: KeyboardEvent) => void) | null = null
  let cleanupScroll: (() => void) | null = null

  function lockScroll() {
    if (typeof document === 'undefined' || restoreScroll) return
    const body = document.body
    const previous = body.style.overflow
    if (previous !== 'hidden') {
      body.style.overflow = 'hidden'
      restoreScroll = true
      cleanupScroll = () => {
        body.style.overflow = previous
        cleanupScroll = null
      }
    }
  }

  function unlockScroll() {
    cleanupScroll?.()
    restoreScroll = false
  }

  /** 弹窗内可聚焦元素；弹窗内无可聚焦元素时退回弹窗本身（保持 Tab 不逃逸）。 */
  function getFocusableElements(root: HTMLElement): HTMLElement[] {
    const focusable = Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE))
    return focusable.length > 0 ? focusable : [root]
  }

  function trapFocus(event: KeyboardEvent) {
    const root = dialogRef.value
    if (!root) return
    if (event.key === 'Escape') {
      event.preventDefault()
      closeDialog()
      return
    }
    if (event.key !== 'Tab') return
    const focusable = getFocusableElements(root)
    const first = focusable[0]
    const last = focusable[focusable.length - 1]
    const active = document.activeElement
    if (event.shiftKey) {
      if (active === first || !root.contains(active)) {
        event.preventDefault()
        last.focus()
      }
    } else if (active === last || !root.contains(active)) {
      event.preventDefault()
      first.focus()
    }
  }

  function openDialog() {
    if (typeof document === 'undefined') return
    triggerElement = document.activeElement instanceof HTMLElement ? document.activeElement : null
    lockScroll()
    if (keyHandler) document.removeEventListener('keydown', keyHandler)
    keyHandler = trapFocus
    document.addEventListener('keydown', keyHandler)
    // 下一帧聚焦：等 Transition 把面板挂载到 DOM 后再移动焦点。
    requestAnimationFrame(() => {
      const root = dialogRef.value
      if (!root) return
      const target = initialFocusSelector
        ? root.querySelector<HTMLElement>(initialFocusSelector)
        : getFocusableElements(root)[0]
      if (target) target.focus()
      else root.focus()
    })
  }

  function closeDialog() {
    if (typeof document === 'undefined') return
    if (keyHandler) {
      document.removeEventListener('keydown', keyHandler)
      keyHandler = null
    }
    unlockScroll()
    if (triggerElement && typeof triggerElement.focus === 'function') {
      triggerElement.focus()
      triggerElement = null
    }
  }

  watch(
    () => visible.value,
    (isVisible) => {
      if (isVisible) openDialog()
      else closeDialog()
    },
  )

  onBeforeUnmount(() => {
    if (keyHandler) document.removeEventListener('keydown', keyHandler)
    keyHandler = null
    unlockScroll()
  })

  return { dialogRef, closeDialog }
}

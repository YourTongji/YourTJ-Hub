import { onBeforeUnmount, ref, watch, type Ref } from 'vue'

/**
 * 弹窗无障碍处理（issue #227）：
 * - 打开：焦点移入弹窗首个可聚焦元素；body 滚动锁定
 * - Tab 圈禁：焦点在弹窗内循环，不逃逸到背景
 * - Esc 关闭；关闭后焦点恢复到触发按钮
 *
 * 用法：panelRef 绑定到弹窗最外层容器；模板需自行提供
 * role="dialog" aria-modal="true" :aria-labelledby（标题 id）。
 */
export function useDialogAccessibility(
  open: Ref<boolean>,
  opts: {
    onClose: () => void
    /** 关闭后焦点恢复目标；缺省用打开前的 activeElement */
    restoreFocusRef?: Ref<HTMLElement | null>
  },
) {
  const panelRef = ref<HTMLElement | null>(null)
  let lastFocused: HTMLElement | null = null
  let cleanup: (() => void) | null = null

  function focusablesOf(panel: HTMLElement): HTMLElement[] {
    return Array.from(
      panel.querySelectorAll<HTMLElement>(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
      ),
    ).filter((el) => !el.hasAttribute('disabled') && el.getAttribute('aria-hidden') !== 'true')
  }

  watch(
    open,
    (isOpen) => {
      if (isOpen) {
        lastFocused = document.activeElement instanceof HTMLElement ? document.activeElement : null
        requestAnimationFrame(() => {
          const panel = panelRef.value
          if (!panel) return
          const focusable = focusablesOf(panel)
          ;(focusable[0] || panel).focus()
        })
        const prevOverflow = document.body.style.overflow
        document.body.style.overflow = 'hidden'
        const onKeydown = (e: KeyboardEvent) => {
          if (e.key === 'Escape') {
            e.preventDefault()
            opts.onClose()
            return
          }
          if (e.key === 'Tab') {
            const panel = panelRef.value
            if (!panel) return
            const focusables = focusablesOf(panel)
            if (!focusables.length) return
            const first = focusables[0]
            const last = focusables[focusables.length - 1]
            const active = document.activeElement instanceof HTMLElement ? document.activeElement : null
            if (e.shiftKey && (active === first || !panel.contains(active))) {
              e.preventDefault()
              last.focus()
            } else if (!e.shiftKey && (active === last || !panel.contains(active))) {
              e.preventDefault()
              first.focus()
            }
          }
        }
        document.addEventListener('keydown', onKeydown, true)
        cleanup = () => {
          document.body.style.overflow = prevOverflow
          document.removeEventListener('keydown', onKeydown, true)
          const restore = opts.restoreFocusRef?.value ?? lastFocused
          restore?.focus?.()
        }
      } else {
        cleanup?.()
        cleanup = null
      }
    },
    { immediate: false },
  )

  onBeforeUnmount(() => {
    if (open.value) cleanup?.()
  })

  return { panelRef }
}

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
// 模块级打开栈：多弹窗叠加时只有最顶层实例处理 Esc/Tab（issue #227 对抗审查加固）。
const openDialogs: Array<{ id: number; close: () => void; contains: (el: Element) => boolean }> = []
let dialogSeq = 0

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
  let entryId = 0

  function isTopmost(): boolean {
    const last = openDialogs[openDialogs.length - 1]
    return last != null && last.id === entryId
  }

  /** 焦点是否落在某打开的 listbox 内（SiteSelect 下拉）：是则 Esc/Tab 交还给下拉自身处理。 */
  function focusInOpenListbox(): boolean {
    const active = document.activeElement
    if (!active) return false
    const listbox = active.closest('[role="listbox"]')
    return listbox != null && listbox.getAttribute('aria-hidden') !== 'true'
  }

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
        entryId = ++dialogSeq
        const panel = panelRef.value
        openDialogs.push({
          id: entryId,
          close: opts.onClose,
          contains: (el) => (panel ? panel.contains(el) : false),
        })
        lastFocused = document.activeElement instanceof HTMLElement ? document.activeElement : null
        requestAnimationFrame(() => {
          // stale 防护：RAF 执行时弹窗可能已关闭（快速开合），跳过焦点移入
          if (!open.value || !isTopmost()) return
          const p = panelRef.value
          if (!p) return
          const focusable = focusablesOf(p)
          ;(focusable[0] || p).focus()
        })
        const prevOverflow = document.body.style.overflow
        document.body.style.overflow = 'hidden'
        const onKeydown = (e: KeyboardEvent) => {
          if (!isTopmost() || focusInOpenListbox()) return
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
          openDialogs.splice(
            openDialogs.findIndex((d) => d.id === entryId),
            1,
          )
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

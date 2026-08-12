import { onBeforeUnmount, ref, watch, type Ref } from 'vue'

const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(', ')

/**
 * Minimal modal-dialog behavior for teleported overlays:
 * - moves focus to the first focusable control on open
 * - traps Tab / Shift+Tab inside the dialog
 * - closes on Escape (via onClose)
 * - marks the app root inert while open, so background content is
 *   unreachable by keyboard and assistive technology
 * - restores focus to the trigger on close
 *
 * Returns a `dialogRoot` ref to bind to the overlay element (`ref="dialog.dialogRoot"`).
 */
export function useModalDialog(active: Ref<boolean>, onClose?: () => void) {
  const dialogRoot = ref<HTMLElement | null>(null)
  let opener: HTMLElement | null = null

  function setAppInert(inert: boolean) {
    const appRoot = document.getElementById('goose-app')
    if (!appRoot) return
    if (inert) {
      appRoot.setAttribute('inert', '')
    } else {
      appRoot.removeAttribute('inert')
    }
  }

  function focusable(): HTMLElement[] {
    const root = dialogRoot.value
    if (!root) return []
    return Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)).filter(
      (el) => el.offsetParent !== null || el === document.activeElement,
    )
  }

  // 打开时优先聚焦表单控件（radio/textarea/input），否则回到第一个可聚焦元素。
  function initialFocusTarget(): HTMLElement | null {
    const root = dialogRoot.value
    if (!root) return null
    return (
      root.querySelector<HTMLElement>('input:not([disabled]), textarea:not([disabled]), select:not([disabled])') ??
      focusable()[0] ??
      null
    )
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      event.preventDefault()
      onClose?.()
      return
    }
    if (event.key !== 'Tab') return
    const els = focusable()
    if (!els.length) return
    const first = els[0]
    const last = els[els.length - 1]
    const current = document.activeElement
    const inside = dialogRoot.value?.contains(current) ?? false
    if (event.shiftKey && (current === first || !inside)) {
      event.preventDefault()
      last.focus()
    } else if (!event.shiftKey && (current === last || !inside)) {
      event.preventDefault()
      first.focus()
    }
  }

  watch(active, (open) => {
    if (open) {
      opener = document.activeElement instanceof HTMLElement ? document.activeElement : null
      setAppInert(true)
      document.addEventListener('keydown', onKeydown)
      requestAnimationFrame(() => initialFocusTarget()?.focus())
    } else {
      setAppInert(false)
      document.removeEventListener('keydown', onKeydown)
      opener?.focus()
      opener = null
    }
  })

  function bindRoot(el: unknown) {
    dialogRoot.value = (el as HTMLElement | null) ?? null
  }

  onBeforeUnmount(() => {
    document.removeEventListener('keydown', onKeydown)
    setAppInert(false)
  })

  return { dialogRoot, bindRoot }
}

import { isClickAnimationEnabled } from '@/runtime/appearance-settings'

export const RIPPLE_SELECTOR = [
  'button',
  'a[href]',
  '[role="button"]',
  '[role="menuitem"]',
  '[role="switch"]',
  '[role="tab"]',
  '[role="checkbox"]',
  '.gf-menu-item',
  '.gf-icon-button',
  '.gf-tab',
  '.gf-segmented-item',
  '[data-ripple]:not([data-ripple="false"])',
].join(',')

const RIPPLE_ANIMATION_MS = 450
const RIPPLE_MAX_ACTIVE = 8
const RIPPLE_CLEANUP_TIMEOUT_MS = 600

export function shouldTriggerRipple(input: {
  enabled: boolean
  reducedMotion: boolean
  button: number
}): boolean {
  return input.enabled && !input.reducedMotion && input.button === 0
}

export function computeWaveSize(width: number, height: number): number {
  return Math.hypot(width, height) * 2
}

const activeRipples: HTMLDivElement[] = []

export function installClickRipple(): () => void {
  const onPointerDown = (event: PointerEvent) => {
    if (
      !shouldTriggerRipple({
        enabled: isClickAnimationEnabled(),
        reducedMotion: window.matchMedia('(prefers-reduced-motion: reduce)').matches,
        button: event.button,
      })
    ) {
      return
    }
    const target = event.target
    if (!(target instanceof Element)) return
    const element = target.closest<HTMLElement>(RIPPLE_SELECTOR)
    if (!element || isDisabledElement(element)) return
    spawnRipple(element, event)
  }

  document.addEventListener('pointerdown', onPointerDown, true)
  return () => document.removeEventListener('pointerdown', onPointerDown, true)
}

function isDisabledElement(element: HTMLElement): boolean {
  return (
    element.hasAttribute('disabled') ||
    element.getAttribute('aria-disabled') === 'true' ||
    element.classList.contains('disabled')
  )
}

function spawnRipple(element: HTMLElement, event: PointerEvent) {
  const rect = element.getBoundingClientRect()
  if (rect.width === 0 || rect.height === 0) return

  while (activeRipples.length >= RIPPLE_MAX_ACTIVE) {
    activeRipples.shift()?.remove()
  }

  const layer = document.createElement('div')
  layer.className = 'gf-ripple'
  layer.style.left = `${rect.left}px`
  layer.style.top = `${rect.top}px`
  layer.style.width = `${rect.width}px`
  layer.style.height = `${rect.height}px`
  const radius = window.getComputedStyle(element).borderRadius
  if (radius && radius !== '0px') layer.style.borderRadius = radius

  const wave = document.createElement('span')
  wave.className = 'gf-ripple__wave'
  const size = computeWaveSize(rect.width, rect.height)
  wave.style.width = `${size}px`
  wave.style.height = `${size}px`
  wave.style.left = `${event.clientX - rect.left - size / 2}px`
  wave.style.top = `${event.clientY - rect.top - size / 2}px`
  layer.appendChild(wave)

  document.body.appendChild(layer)
  activeRipples.push(layer)

  const cleanup = () => {
    layer.remove()
    const index = activeRipples.indexOf(layer)
    if (index !== -1) activeRipples.splice(index, 1)
  }
  wave.addEventListener('animationend', cleanup, { once: true })
  window.setTimeout(cleanup, RIPPLE_CLEANUP_TIMEOUT_MS)
}

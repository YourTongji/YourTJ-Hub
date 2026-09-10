import { isClickAnimationEnabled } from '@/runtime/appearance-settings'
import type { TouchEffectInstance } from 'blue-archive-touch-effect'

/**
 * Blue Archive touch-effect bridge for the site "click animation" preference.
 *
 * The heavy WebGL package is loaded only after the user turns the setting on,
 * so the default bundle stays free of ogl/shaders.
 *
 * Important interaction rules:
 * - Overlay host/canvas must stay `pointer-events: none` so real UI keeps hit-testing.
 * - Never use the library's default `autoBindPointer` path: it calls
 *   `setPointerCapture` on the host and steals subsequent pointer/click events.
 * - We bind passive document listeners and only drive the cosmetic API.
 */

export const BA_TOUCH_HOST_ID = 'gf-ba-touch-effect-host'

/** Keep the effect above page chrome but below system-level overlays. */
const HOST_Z_INDEX = '2147483000'

let touchEffectInstance: TouchEffectInstance | null = null
let hostElement: HTMLDivElement | null = null
let startInFlight: Promise<void> | null = null
let reducedMotionMediaQuery: MediaQueryList | null = null
let installed = false
let pointerBindingsActive = false

const boundPointerDown = (event: PointerEvent) => {
  if (!touchEffectInstance) return
  // Match library primary interaction range (left/middle/right), skip exotic buttons.
  if (event.button < 0 || event.button > 2) return
  touchEffectInstance.triggerClickAtClient(event.clientX, event.clientY)
  touchEffectInstance.beginTrailAtClient(event.pointerId, event.clientX, event.clientY)
}

const boundPointerMove = (event: PointerEvent) => {
  if (!touchEffectInstance) return
  if (!shouldKeepTrailAlive(event)) {
    touchEffectInstance.endTrail(event.pointerId)
    return
  }
  const coalescedEvents = typeof event.getCoalescedEvents === 'function'
    ? event.getCoalescedEvents()
    : []
  const samples = coalescedEvents.length > 0 ? coalescedEvents : [event]
  for (const sample of samples) {
    touchEffectInstance.appendTrailAtClient(event.pointerId, sample.clientX, sample.clientY)
  }
}

const boundPointerEnd = (event: PointerEvent) => {
  touchEffectInstance?.endTrail(event.pointerId)
}

const boundWindowBlur = () => {
  touchEffectInstance?.endAllTrails()
}

export function prefersReducedMotion(mediaQuery: { matches: boolean } | null | undefined = null): boolean {
  if (mediaQuery) return mediaQuery.matches
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return false
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

/** Pure gate used by tests and the runtime lifecycle. */
export function shouldEnableBaTouchEffect(input: {
  clickAnimationEnabled: boolean
  reducedMotion: boolean
}): boolean {
  return input.clickAnimationEnabled && !input.reducedMotion
}

/** Pure helper: mouse/pen trails only while a button is held; touch always trails. */
export function shouldKeepTrailAlive(event: {
  pointerType: string
  buttons: number
}): boolean {
  if (event.pointerType === 'mouse' || event.pointerType === 'pen') {
    return event.buttons > 0
  }
  return true
}

/**
 * Starts or stops the BA runtime to match the current appearance preference.
 * Safe to call repeatedly (settings toggles, reduced-motion changes, boot).
 */
export function syncBaTouchEffect(): void {
  if (typeof window === 'undefined' || typeof document === 'undefined') return

  const shouldEnable = shouldEnableBaTouchEffect({
    clickAnimationEnabled: isClickAnimationEnabled(),
    reducedMotion: prefersReducedMotion(reducedMotionMediaQuery),
  })

  if (shouldEnable) {
    void ensureBaTouchEffectStarted()
    return
  }
  stopBaTouchEffect()
}

/**
 * Boot-time installer: sync once, then re-sync when the OS reduced-motion
 * preference flips. Appearance toggles call syncBaTouchEffect via applyAppearanceSettings.
 */
export function installBaTouchEffect(): () => void {
  if (installed) return () => undefined
  installed = true

  if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
    reducedMotionMediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    reducedMotionMediaQuery.addEventListener('change', syncBaTouchEffect)
  }

  syncBaTouchEffect()

  return () => {
    reducedMotionMediaQuery?.removeEventListener('change', syncBaTouchEffect)
    reducedMotionMediaQuery = null
    stopBaTouchEffect()
    installed = false
  }
}

async function ensureBaTouchEffectStarted(): Promise<void> {
  if (touchEffectInstance) return
  if (startInFlight) {
    await startInFlight
    return
  }

  startInFlight = (async () => {
    try {
      const { createTouchEffect } = await import('blue-archive-touch-effect')

      // Preference may have flipped off while the dynamic import was in flight.
      if (
        !shouldEnableBaTouchEffect({
          clickAnimationEnabled: isClickAnimationEnabled(),
          reducedMotion: prefersReducedMotion(reducedMotionMediaQuery),
        })
      ) {
        return
      }

      if (touchEffectInstance) return

      const host = ensureHostElement()
      // autoBindPointer must stay false: library default captures pointers on the host
      // and breaks real UI clicks/drags. pointerCapture is also forced off as defense.
      touchEffectInstance = createTouchEffect({
        target: host,
        autoBindPointer: false,
        pixelRatioCap: 2,
        config: {
          swipe: {
            input: {
              pointerCapture: false,
            },
          },
        },
      })
      lockOverlayPointerPassthrough(host, touchEffectInstance.canvas)
      bindPointerDrivers()
    } catch (error) {
      // Cosmetic only: log once-style warning and leave the setting inert.
      console.warn('[ba-touch-effect] failed to start Blue Archive touch effect', error)
      stopBaTouchEffect()
    } finally {
      startInFlight = null
    }
  })()

  await startInFlight
}

function ensureHostElement(): HTMLDivElement {
  const existing = document.getElementById(BA_TOUCH_HOST_ID)
  if (existing instanceof HTMLDivElement) {
    hostElement = existing
    lockOverlayPointerPassthrough(existing)
    return existing
  }

  const host = document.createElement('div')
  host.id = BA_TOUCH_HOST_ID
  host.setAttribute('aria-hidden', 'true')
  host.dataset.ripple = 'false'
  host.style.position = 'fixed'
  host.style.inset = '0'
  host.style.width = '100%'
  host.style.height = '100%'
  host.style.zIndex = HOST_Z_INDEX
  host.style.overflow = 'hidden'
  lockOverlayPointerPassthrough(host)
  document.body.appendChild(host)
  hostElement = host
  return host
}

/** Force the overlay (and its canvas) to never participate in hit-testing. */
function lockOverlayPointerPassthrough(host: HTMLElement, canvas?: HTMLCanvasElement | null) {
  host.style.pointerEvents = 'none'
  host.style.setProperty('pointer-events', 'none', 'important')
  if (canvas) {
    canvas.style.pointerEvents = 'none'
    canvas.style.setProperty('pointer-events', 'none', 'important')
  }
}

function bindPointerDrivers() {
  if (pointerBindingsActive || typeof document === 'undefined' || typeof window === 'undefined') return
  // Bubble phase + passive: observe real UI interactions without intercepting them.
  document.addEventListener('pointerdown', boundPointerDown, { passive: true })
  document.addEventListener('pointermove', boundPointerMove, { passive: true })
  document.addEventListener('pointerup', boundPointerEnd, { passive: true })
  document.addEventListener('pointercancel', boundPointerEnd, { passive: true })
  window.addEventListener('blur', boundWindowBlur)
  pointerBindingsActive = true
}

function unbindPointerDrivers() {
  if (!pointerBindingsActive || typeof document === 'undefined' || typeof window === 'undefined') return
  document.removeEventListener('pointerdown', boundPointerDown)
  document.removeEventListener('pointermove', boundPointerMove)
  document.removeEventListener('pointerup', boundPointerEnd)
  document.removeEventListener('pointercancel', boundPointerEnd)
  window.removeEventListener('blur', boundWindowBlur)
  pointerBindingsActive = false
}

function stopBaTouchEffect(): void {
  unbindPointerDrivers()
  try {
    touchEffectInstance?.endAllTrails()
  } catch {
    // ignore
  }
  try {
    touchEffectInstance?.dispose()
  } catch {
    // ignore dispose failures during teardown
  }
  touchEffectInstance = null

  if (hostElement?.isConnected) {
    hostElement.remove()
  } else {
    document.getElementById(BA_TOUCH_HOST_ID)?.remove()
  }
  hostElement = null
}

// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import type { LayoutPayload } from '@gooseforum/client'

const hintKey = 'goose:mobile-drawer-gesture-hint:v1'

function layout(): LayoutPayload {
  return {
    site: { name: 'yourtj', description: '', logo: '', favicon: '', brandType: 'text', brandText: 'yourtj', brandImage: '' },
    viewer: {
      id: 0,
      username: '',
      email: '',
      avatarUrl: '',
      isAuthenticated: false,
      canAccessAdmin: false,
      isModerator: false,
      requiresEmailVerification: false,
      adminPermissions: [],
    },
    sidebar: { main: [], resources: [], groups: [], categories: [], activeKey: 'topics' },
    footer: { links: [], primary: [] },
    unread: { notifications: false, messages: false, moderationReports: false },
    theme: { enabled: false, current: 'gf-light', themeColor: '#fbfdff' },
    umamiEnabled: false,
  }
}

function mobileMediaQuery(query: string): MediaQueryList {
  return {
    matches: query === '(max-width: 1023px)' || query === '(pointer: coarse)',
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => true,
  } as MediaQueryList
}

function touch(type: string, x: number, y: number) {
  const event = new Event(type, { bubbles: true, cancelable: true }) as TouchEvent
  const point = { identifier: 7, clientX: x, clientY: y }
  Object.defineProperty(event, 'touches', { value: type === 'touchend' ? [] : [point] })
  Object.defineProperty(event, 'changedTouches', { value: [point] })
  return event
}

async function mountFreshShell() {
  vi.resetModules()
  const [{ i18n }, { default: AppShell }] = await Promise.all([
    import('../src/runtime/i18n'),
    import('../src/site/components/AppShell.vue'),
  ])
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }],
  })
  return mount(AppShell, {
    props: { layout: layout() },
    global: { plugins: [i18n, router] },
    attachTo: document.body,
  })
}

describe('AppShell drawer gesture hint persistence', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    window.localStorage.clear()
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 390 })
    vi.spyOn(window, 'matchMedia').mockImplementation(mobileMediaQuery)
  })

  afterEach(() => {
    vi.runOnlyPendingTimers()
    vi.useRealTimers()
    document.body.innerHTML = ''
    document.body.style.pointerEvents = ''
    document.body.style.overflow = ''
    window.localStorage.clear()
    vi.restoreAllMocks()
  })

  test('showing the hint persists after 500ms without ending its display window', async () => {
    const wrapper = await mountFreshShell()

    await vi.advanceTimersByTimeAsync(800)
    await flushPromises()
    expect(document.querySelector('.gf-drawer-gesture-hint')).not.toBeNull()
    expect(window.localStorage.getItem(hintKey)).toBeNull()

    await vi.advanceTimersByTimeAsync(499)
    expect(window.localStorage.getItem(hintKey)).toBeNull()

    await vi.advanceTimersByTimeAsync(1)
    await flushPromises()
    expect(window.localStorage.getItem(hintKey)).toBe('1')
    expect(document.querySelector('.gf-drawer-gesture-hint')).not.toBeNull()

    await vi.advanceTimersByTimeAsync(3500)
    await flushPromises()
    expect(document.querySelector('.gf-drawer-gesture-hint')).toBeNull()

    wrapper.unmount()
  })

  test('leaving before 500ms of visibility does not persist it as seen', async () => {
    const wrapper = await mountFreshShell()

    await vi.advanceTimersByTimeAsync(800)
    await flushPromises()
    expect(document.querySelector('.gf-drawer-gesture-hint')).not.toBeNull()
    expect(window.localStorage.getItem(hintKey)).toBeNull()

    await vi.advanceTimersByTimeAsync(499)
    wrapper.unmount()
    await vi.advanceTimersByTimeAsync(5000)
    expect(window.localStorage.getItem(hintKey)).toBeNull()
  })

  test('leaving after one second of visibility persists the hint as seen', async () => {
    const wrapper = await mountFreshShell()
    await vi.advanceTimersByTimeAsync(1800)
    await flushPromises()
    expect(document.querySelector('.gf-drawer-gesture-hint')).not.toBeNull()
    wrapper.unmount()
    expect(window.localStorage.getItem(hintKey)).toBe('1')
  })

  test('route changes after one second of visibility do not offer the hint again', async () => {
    const wrapper = await mountFreshShell()
    await vi.advanceTimersByTimeAsync(1800)
    await wrapper.vm.$router.push('/moderation')
    await flushPromises()
    await vi.advanceTimersByTimeAsync(800)
    expect(window.localStorage.getItem(hintKey)).toBe('1')
    expect(document.querySelector('.gf-drawer-gesture-hint')).toBeNull()
    await wrapper.vm.$router.push('/')
    await flushPromises()
    await vi.advanceTimersByTimeAsync(800)
    expect(document.querySelector('.gf-drawer-gesture-hint')).toBeNull()
    wrapper.unmount()
  })

  test.each(['unmount', 'pagehide', 'route'])('%s settles visibility even when the seen timer is delayed', async (exit) => {
    const wrapper = await mountFreshShell()
    await vi.advanceTimersByTimeAsync(800)
    await flushPromises()
    vi.setSystemTime(Date.now() + 1000)
    expect(window.localStorage.getItem(hintKey)).toBeNull()

    if (exit === 'unmount') wrapper.unmount()
    else if (exit === 'pagehide') window.dispatchEvent(new Event('pagehide'))
    else await wrapper.vm.$router.push('/moderation')
    expect(window.localStorage.getItem(hintKey)).toBe('1')
    if (exit !== 'unmount') wrapper.unmount()
  })

  test('leaving before the hint is shown does not persist it as seen', async () => {
    const wrapper = await mountFreshShell()
    await vi.advanceTimersByTimeAsync(799)
    wrapper.unmount()
    await vi.advanceTimersByTimeAsync(5000)
    expect(window.localStorage.getItem(hintKey)).toBeNull()
  })

  test('a successful right swipe immediately persists the hint as seen', async () => {
    const wrapper = await mountFreshShell()

    await vi.advanceTimersByTimeAsync(800)
    await flushPromises()
    expect(window.localStorage.getItem(hintKey)).toBeNull()

    wrapper.element.dispatchEvent(touch('touchstart', 100, 160))
    wrapper.element.dispatchEvent(touch('touchmove', 185, 166))
    await flushPromises()

    expect(window.localStorage.getItem(hintKey)).toBe('1')
    expect(document.querySelector('.gf-drawer-gesture-hint')).toBeNull()

    wrapper.unmount()
  })

  test('page-level drawer swipe opt-out suppresses the one-time hint', async () => {
    const optOut = document.createElement('div')
    optOut.setAttribute('data-drawer-swipe-ignore', 'page')
    document.body.append(optOut)
    const wrapper = await mountFreshShell()

    await vi.advanceTimersByTimeAsync(4800)
    await flushPromises()

    expect(document.querySelector('.gf-drawer-gesture-hint')).toBeNull()
    expect(window.localStorage.getItem(hintKey)).toBeNull()

    wrapper.unmount()
    optOut.remove()
  })

  test('storage failures fall back to the in-memory seen latch', async () => {
    vi.resetModules()
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new DOMException('blocked', 'SecurityError')
    })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new DOMException('blocked', 'SecurityError')
    })
    const { markDrawerGestureHintSeen, shouldOfferDrawerGestureHint } = await import('../src/runtime/mobile-drawer-gesture')

    expect(shouldOfferDrawerGestureHint()).toBe(true)
    expect(() => markDrawerGestureHintSeen()).not.toThrow()
    expect(shouldOfferDrawerGestureHint()).toBe(false)
  })
})

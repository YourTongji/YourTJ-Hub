// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import type { LayoutPayload } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import { mobileDrawerGestureHintKey } from '../src/runtime/mobile-drawer-gesture'
import AppShell from '../src/site/components/AppShell.vue'

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

describe('AppShell mobile drawer gesture wiring', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 390 })
    window.localStorage.setItem(mobileDrawerGestureHintKey, '1')
    vi.spyOn(window, 'matchMedia').mockImplementation(mobileMediaQuery)
  })

  afterEach(() => {
    document.body.innerHTML = ''
    document.body.style.pointerEvents = ''
    document.body.style.overflow = ''
    vi.restoreAllMocks()
  })

  test('swipe right opens the existing drawer and swipe left on its surface closes it', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }],
    })
    const wrapper = mount(AppShell, {
      props: { layout: layout() },
      global: { plugins: [i18n, router] },
      attachTo: document.body,
    })

    wrapper.element.dispatchEvent(touch('touchstart', 100, 160))
    const openingTrack = touch('touchmove', 121, 162)
    wrapper.element.dispatchEvent(openingTrack)
    expect(openingTrack.defaultPrevented).toBe(true)
    wrapper.element.dispatchEvent(touch('touchmove', 185, 166))
    await flushPromises()
    await vi.waitFor(() => {
      expect(document.querySelector('.gf-drawer-surface')).not.toBeNull()
    })

    const drawer = document.querySelector<HTMLElement>('.gf-drawer-surface')
    expect(drawer).not.toBeNull()

    drawer!.dispatchEvent(touch('touchstart', 250, 180))
    const closingTrack = touch('touchmove', 228, 181)
    drawer!.dispatchEvent(closingTrack)
    expect(closingTrack.defaultPrevented).toBe(true)
    drawer!.dispatchEvent(touch('touchmove', 160, 184))
    await flushPromises()
    await vi.waitFor(() => {
      expect(document.querySelector('.gf-drawer-surface')).toBeNull()
    })

    wrapper.unmount()
  })
})

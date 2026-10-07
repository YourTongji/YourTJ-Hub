// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { h } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import type { LayoutPayload } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import { mobileDrawerGestureHintKey } from '../src/runtime/mobile-drawer-gesture'
import AppShell from '../src/site/components/AppShell.vue'
import TopicImageGallery from '../src/site/components/TopicImageGallery.vue'

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

function touchList(
  type: string,
  touches: Array<{ identifier: number; clientX: number; clientY: number }>,
  changedTouches = touches,
) {
  const event = new Event(type, { bubbles: true, cancelable: true }) as TouchEvent
  Object.defineProperty(event, 'touches', { value: touches })
  Object.defineProperty(event, 'changedTouches', { value: changedTouches })
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

  test('swiping the topic image gallery changes images without opening the drawer', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }],
    })
    const wrapper = mount(AppShell, {
      props: { layout: layout() },
      slots: {
        default: () => h(TopicImageGallery, { images: ['/first.jpg', '/second.jpg'] }),
      },
      global: { plugins: [i18n, router] },
      attachTo: document.body,
    })
    const gallery = wrapper.findComponent(TopicImageGallery)
    const nextButton = gallery.find(`button[aria-label="${i18n.global.t('common.nextPage')}"]`)
    expect(nextButton.exists()).toBe(true)
    await nextButton.trigger('click')
    expect(gallery.text()).toContain('2/2')

    const carousel = gallery.find('.group.relative')
    carousel.element.dispatchEvent(touch('touchstart', 100, 160))
    carousel.element.dispatchEvent(touch('touchend', 185, 160))
    await flushPromises()

    expect(gallery.text()).toContain('1/2')
    expect(document.querySelector('.gf-drawer-surface')).toBeNull()
    wrapper.unmount()
  })

  test('multi-touch and touchcancel clear the tracked finger instead of using another touch to toggle the drawer', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }],
    })
    const wrapper = mount(AppShell, {
      props: { layout: layout() },
      global: { plugins: [i18n, router] },
      attachTo: document.body,
    })

    const tracked = { identifier: 7, clientX: 100, clientY: 160 }
    const second = { identifier: 8, clientX: 190, clientY: 160 }
    wrapper.element.dispatchEvent(touchList('touchstart', [tracked], [tracked]))
    wrapper.element.dispatchEvent(touchList('touchstart', [tracked, second], [second]))
    wrapper.element.dispatchEvent(touchList('touchend', [tracked], [second]))
    await flushPromises()
    expect(document.querySelector('.gf-drawer-surface')).toBeNull()

    wrapper.element.dispatchEvent(touch('touchstart', 100, 160))
    wrapper.element.dispatchEvent(touchList('touchcancel', [], [{ identifier: 7, clientX: 100, clientY: 160 }]))
    const afterCancel = touch('touchmove', 190, 160)
    wrapper.element.dispatchEvent(afterCancel)
    expect(afterCancel.defaultPrevented).toBe(false)
    expect(document.querySelector('.gf-drawer-surface')).toBeNull()

    wrapper.element.dispatchEvent(touch('touchstart', 100, 160))
    wrapper.element.dispatchEvent(touch('touchmove', 185, 166))
    await flushPromises()
    await vi.waitFor(() => {
      expect(document.querySelector('.gf-drawer-surface')).not.toBeNull()
    })

    const drawer = document.querySelector<HTMLElement>('.gf-drawer-surface')!
    const closeTracked = { identifier: 7, clientX: 250, clientY: 180 }
    const closeSecond = { identifier: 8, clientX: 155, clientY: 180 }
    drawer.dispatchEvent(touchList('touchstart', [closeTracked], [closeTracked]))
    drawer.dispatchEvent(touchList('touchstart', [closeTracked, closeSecond], [closeSecond]))
    drawer.dispatchEvent(touchList('touchend', [closeTracked], [closeSecond]))
    await flushPromises()
    expect(document.querySelector('.gf-drawer-surface')).not.toBeNull()

    wrapper.unmount()
  })
})

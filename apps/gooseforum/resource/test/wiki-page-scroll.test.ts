// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { shallowMount, type VueWrapper } from '@vue/test-utils'
import type { LayoutPayload, WikiDetailProps } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import { saveWikiJumpState } from '../src/runtime/use-wiki-search'
import WikiPage from '../src/site/pages/WikiPage.vue'

vi.mock('../src/site/components/PostStream.vue', () => ({ default: { template: '<div />' } }))

const detail: WikiDetailProps = {
  page: {
    id: 1, topicId: 1, namespace: 'guide', path: 'guide/start', title: 'Guide',
    content: '<h2 id="heading">Heading</h2><p id="s-1">First hit</p><p id="s-2">Next hit</p>',
    toc: [], updatedAt: '', likeCount: 0, viewCount: 0, postCount: 0,
    liked: false, bookmarked: false, watched: false, canEdit: false, publishedRevisionNo: 1,
  },
  contributors: [], hotTopics: [],
}

let wrapper: VueWrapper | undefined
let frames: FrameRequestCallback[]

beforeEach(() => {
  vi.useFakeTimers()
  frames = []
  vi.spyOn(window, 'requestAnimationFrame').mockImplementation(callback => {
    frames.push(callback)
    return frames.length
  })
  vi.spyOn(window, 'scrollTo').mockImplementation(() => {})
  sessionStorage.clear()
  window.history.replaceState(null, '', '/wiki/guide/start')
})

afterEach(() => {
  window.dispatchEvent(new Event('scrollend'))
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
  vi.useRealTimers()
  vi.restoreAllMocks()
})

function mountPage() {
  wrapper = shallowMount(WikiPage, {
    props: { layout: { viewer: { isAuthenticated: false } } as LayoutPayload, props: detail },
    global: {
      plugins: [i18n],
      directives: { 'code-copy': {}, 'code-highlight': {}, 'math-render': {}, 'content-enhancements': {} },
    },
    attachTo: document.body,
  })
  vi.spyOn(document.getElementById('heading')!, 'getBoundingClientRect').mockReturnValue({ top: 488 } as DOMRect)
  vi.spyOn(document.getElementById('s-1')!, 'getBoundingClientRect').mockReturnValue({ top: 888 } as DOMRect)
  vi.spyOn(document.getElementById('s-2')!, 'getBoundingClientRect').mockReturnValue({ top: 1288 } as DOMRect)
  for (const frame of frames) frame(0)
}

describe('WikiPage anchor landing', () => {
  test('search navigation with hash and saved hits lands once instantly with the header offset', () => {
    window.history.replaceState(null, '', '/wiki/guide/start#s-1')
    saveWikiJumpState({ query: 'hit', anchors: ['s-1', 's-2'] })
    mountPage()

    expect(window.scrollTo).toHaveBeenCalledTimes(1)
    expect(window.scrollTo).toHaveBeenCalledWith({ top: 800, behavior: 'instant' })
    expect(document.getElementById('s-1')?.classList.contains('wiki-hit-flash')).toBe(false)
    vi.advanceTimersByTime(699)
    expect(document.getElementById('s-1')?.classList.contains('wiki-hit-flash')).toBe(false)
    vi.advanceTimersByTime(1)
    expect(document.getElementById('s-1')?.classList.contains('wiki-hit-flash')).toBe(true)
  })

  test('a direct heading hash lands instantly without saved search state', () => {
    window.history.replaceState(null, '', '/wiki/guide/start#heading')
    mountPage()
    expect(window.scrollTo).toHaveBeenCalledExactlyOnceWith({ top: 400, behavior: 'instant' })
  })

  test.each(['ctrlKey', 'metaKey'] as const)('%s+G keeps smooth scrolling and flashes on scrollend', modifier => {
    saveWikiJumpState({ query: 'hit', anchors: ['s-1', 's-2'] })
    mountPage()
    vi.mocked(window.scrollTo).mockClear()

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'g', [modifier]: true, cancelable: true }))
    expect(window.scrollTo).toHaveBeenCalledExactlyOnceWith({ top: 1200, behavior: 'smooth' })
    expect(document.getElementById('s-2')?.classList.contains('wiki-hit-flash')).toBe(false)
    window.dispatchEvent(new Event('scrollend'))
    expect(document.getElementById('s-2')?.classList.contains('wiki-hit-flash')).toBe(true)
  })

  test('an absent anchor does not scroll or throw', () => {
    window.history.replaceState(null, '', '/wiki/guide/start#missing')
    mountPage()
    expect(window.scrollTo).not.toHaveBeenCalled()
  })
})

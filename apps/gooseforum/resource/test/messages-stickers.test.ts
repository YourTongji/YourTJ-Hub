// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { LayoutPayload } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import MessagesPage from '../src/site/pages/MessagesPage.vue'

const { messages, resolve } = vi.hoisted(() => ({ messages: vi.fn(), resolve: vi.fn() }))
vi.mock('@/runtime/api', () => ({
  getChatMessages: messages,
  resolveForumStickers: resolve,
  markChatRead: vi.fn().mockResolvedValue(undefined),
  sendChatMessage: vi.fn(),
  sensitiveWordsFromError: () => [],
}))
vi.mock('@/runtime/private-notes', () => ({
  userDisplayName: (_id: number, username: string, nickname?: string) => nickname || username,
}))
vi.mock('@/runtime/unread-status', () => ({
  useUnreadStatus: () => ({ clearMessages: vi.fn() }),
}))

let wrapper: VueWrapper | undefined
afterEach(() => {
  wrapper?.unmount()
  window.history.replaceState({}, '', '/')
  vi.resetAllMocks()
})

it('resolves received personal tokens as inert images while preserving escaped text and unknown tokens', async () => {
  const text = '<a href="https://example.test">plain link text</a><script>alert(1)</script> [:sticker:u_x:] [:sticker:missing:]'
  messages.mockResolvedValue({
    list: [{ id: 1, content: text, isSelf: false, createdAt: '2026-09-26T00:00:00Z' }],
    hasMoreBefore: false,
  })
  resolve.mockResolvedValue([{ name: 'u_x', url: '/file/img/shared.png', isOfficial: false }])
  window.history.replaceState({}, '', '/messages?userId=2')
  wrapper = mount(MessagesPage, {
    props: {
      layout: { viewer: { id: 1, username: 'alice', avatarUrl: '' } } as LayoutPayload,
      props: {
        conversations: [{ id: 1, convId: 1, peerId: 2, peerUsername: 'bob', peerAvatar: '', lastMsg: '', lastMsgTime: '', unreadCount: 0, peerUrl: '/u/2' }],
        suggestedUsers: [],
      },
    },
    global: { plugins: [i18n], stubs: { UserAvatar: true } },
  })
  await flushPromises()
  expect(resolve).toHaveBeenCalledWith(['u_x', 'missing'])
  const image = wrapper.get('img[src="/file/img/shared.png"]')
  const bubble = image.element.parentElement!
  expect(image.attributes('alt')).toBe('[:sticker:u_x:]')
  expect(image.attributes('loading')).toBe('lazy')
  expect(image.element.closest('a, button')).toBeNull()
  expect(bubble.querySelector('a, script')).toBeNull()
  expect(bubble.textContent).toContain('<script>alert(1)</script>')
  expect(bubble.textContent).toContain('[:sticker:missing:]')
  await image.trigger('click')
  await flushPromises()
  expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  expect(window.location.pathname).toBe('/messages')
})

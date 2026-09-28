// @vitest-environment happy-dom
// 回归：备注名显示 note(display name) 依赖调用方把 payload 的 nickname 传进
// userDisplayName（issue #837）。这里把真实语义固定成 `备注(display name)`，
// 任何没传 nickname 的调用点都会渲染成 `备注(username)` 而被断言抓住。
import { afterEach, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { LayoutPayload, NotificationPayload } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import AvatarStack from '../src/site/components/AvatarStack.vue'
import NotificationsPage from '../src/site/pages/NotificationsPage.vue'

vi.mock('@/runtime/api', () => ({
  fetchNotifications: vi.fn().mockResolvedValue({ items: [], nextCursor: 0, hasNext: false, unreadCount: 0 }),
  markAllNotificationsRead: vi.fn().mockResolvedValue(undefined),
  markNotificationRead: vi.fn().mockResolvedValue(undefined),
}))
vi.mock('@/runtime/unread-status', () => ({
  useUnreadStatus: () => ({ clearNotifications: vi.fn() }),
}))
vi.mock('@/runtime/private-notes', () => ({
  userDisplayName: (_id: number, username: string, nickname?: string) => `备注(${nickname || username || '匿名'})`,
}))

let wrapper: VueWrapper | undefined
afterEach(() => {
  wrapper?.unmount()
  vi.unstubAllGlobals()
})

function createNotification(): NotificationPayload {
  return {
    id: 1,
    eventType: 'post_reply',
    isRead: true,
    createdAt: '2026-09-27T08:00:00Z',
    title: '',
    content: '回复了你的评论',
    actor: { id: 2, username: 'actor', nickname: '昵称甲', avatarUrl: '' },
    payload: { actorId: 2, actorName: 'actor' },
  }
}

it('渲染通知 actor 时传入昵称（note(display name)）', async () => {
  vi.stubGlobal('IntersectionObserver', class { observe() {} unobserve() {} disconnect() {} })
  wrapper = mount(NotificationsPage, {
    props: {
      layout: { viewer: { id: 1, username: 'alice', avatarUrl: '' } } as LayoutPayload,
      props: {
        total: 1,
        unreadCount: 0,
        notifications: [createNotification()],
        pagination: { page: 1, nextPage: 0, hasNext: false, nextUrl: '' },
      },
    },
    global: { plugins: [i18n], stubs: { UserAvatar: true } },
  })
  await flushPromises()
  // actor.name 命中备注后按当前昵称渲染；传 username 会得到 `备注(actor)`。
  expect(wrapper.text()).toContain('备注(昵称甲)')
})

it('渲染参与者头像时传入昵称（note(display name)）', () => {
  wrapper = mount(AvatarStack, {
    props: {
      users: [{ id: 3, username: 'alice', nickname: '昵称乙', avatarUrl: '/uploads/avatar.png' }],
    },
    global: { plugins: [i18n], stubs: { UserAvatar: true } },
  })
  expect(wrapper.get('a').attributes('title')).toBe('备注(昵称乙)')
})

it('参与者没有昵称时回退用户名', () => {
  wrapper = mount(AvatarStack, {
    props: {
      users: [{ id: 3, username: 'alice', avatarUrl: '/uploads/avatar.png' }],
    },
    global: { plugins: [i18n], stubs: { UserAvatar: true } },
  })
  expect(wrapper.get('a').attributes('title')).toBe('备注(alice)')
})

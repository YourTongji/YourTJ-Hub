// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { LayoutPayload } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import MessagesPage from '../src/site/pages/MessagesPage.vue'

// 组件级回归（issue #904 review）：日期分隔与气泡时间的接线。
// 纯函数测试覆盖分组规则，这里保证 MessagesPage.vue 真的把它们渲染出来，
// 重构模板或 v-if 不会静默丢掉分隔。
const { messages, dayLabel, realDayLabel } = vi.hoisted(() => ({
  messages: vi.fn(),
  dayLabel: vi.fn<(day: Date, now?: Date) => string>(),
  realDayLabel: { current: undefined as ((day: Date, now?: Date) => string) | undefined },
}))
vi.mock('@/runtime/api', () => ({
  getChatMessages: messages,
  resolveForumStickers: vi.fn().mockResolvedValue([]),
  markChatRead: vi.fn().mockResolvedValue(undefined),
  sendChatMessage: vi.fn(),
  sensitiveWordsFromError: () => [],
}))
// 统计分隔标签的求值次数：每个分隔只应计算一次（item 3）。
vi.mock('@/runtime/format', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../src/runtime/format')>()
  realDayLabel.current = actual.formatChatDayLabel
  return { ...actual, formatChatDayLabel: dayLabel }
})
vi.mock('@/runtime/private-notes', () => ({
  userDisplayName: (_id: number, username: string, nickname?: string) => nickname || username,
}))
vi.mock('@/runtime/unread-status', () => ({
  useUnreadStatus: () => ({ clearMessages: vi.fn() }),
}))

let wrapper: VueWrapper | undefined
const locale = i18n.global.locale as unknown as { value: string }
const originalLocale = locale.value

beforeEach(() => {
  // resetAllMocks 会清掉实现，重新挂上真实格式化函数。
  dayLabel.mockImplementation((day: Date, now?: Date) => realDayLabel.current!(day, now))
})

afterEach(() => {
  wrapper?.unmount()
  locale.value = originalLocale
  window.history.replaceState({}, '', '/')
  vi.resetAllMocks()
})

/** 设备本地墙钟 → 带时区的 RFC3339，断言在任何本地时区下都成立。 */
function local(
  day: number,
  hour: number,
  minute: number,
  second = 0,
  month = 9,
  year = 2026,
): string {
  return new Date(year, month - 1, day, hour, minute, second).toISOString()
}

async function mountConversation(
  list: Array<{ id: number; content: string; createdAt: string; isSelf: boolean }>,
) {
  messages.mockResolvedValue({ list, hasMoreBefore: false })
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
  return wrapper
}

it('renders one date divider per day and hides redundant bubble times', async () => {
  const page = await mountConversation([
    { id: 1, content: 'y1', createdAt: local(27, 23, 50), isSelf: false },
    { id: 2, content: 'y2', createdAt: local(27, 23, 52), isSelf: false },
    { id: 3, content: 't1', createdAt: local(28, 10, 0), isSelf: false },
    { id: 4, content: 't2', createdAt: local(28, 10, 2), isSelf: false },
    { id: 5, content: 't3', createdAt: local(28, 10, 10), isSelf: true },
  ])

  // 只统计消息流，避免侧栏会话行的时间。
  const list = page.get('[data-test="chat-message-list"]')
  // 日期分隔是标题（与移动端的标题语义一致），每天一个，标题即分隔文案。
  const dividers = list.findAll('h2')
  expect(dividers.map((node) => node.text())).toEqual(['Yesterday', 'Today'])
  expect(dividers.every((node) => node.element.tagName === 'H2')).toBe(true)
  expect(dividers.every((node) => node.element.querySelector('time') === null)).toBe(true)
  // 每个分隔只求值一次标签（item 3：避免两次绑定在午夜边界不一致）。
  expect(dayLabel).toHaveBeenCalledTimes(2)
  // 只有分组首条/间隔超过 5 分钟的气泡显示时刻。
  expect(list.findAll('time').map((node) => node.text())).toEqual(['23:50', '10:00', '10:10'])
  expect(list.findAll('time').every((node) => node.element.closest('h2') === null)).toBe(true)
})

it('localizes the divider labels (zh, with the shared i18n instance)', async () => {
  locale.value = 'zh'
  const page = await mountConversation([
    { id: 1, content: 'y1', createdAt: local(27, 23, 50), isSelf: false },
    { id: 2, content: 't1', createdAt: local(28, 10, 0), isSelf: false },
  ])

  const list = page.get('[data-test="chat-message-list"]')
  expect(list.findAll('h2').map((node) => node.text())).toEqual(['昨天', '今天'])
})

it('keeps the first-in-group time when the gap exceeds five minutes', async () => {
  const page = await mountConversation([
    { id: 1, content: 'a', createdAt: local(28, 8, 0), isSelf: false },
    { id: 2, content: 'b', createdAt: local(28, 8, 5), isSelf: false },
    { id: 3, content: 'c', createdAt: local(28, 8, 10, 1), isSelf: false },
  ])

  const list = page.get('[data-test="chat-message-list"]')
  expect(list.findAll('h2').map((node) => node.text())).toEqual(['Today'])
  expect(list.findAll('time').map((node) => node.text())).toEqual(['08:00', '08:10'])
})

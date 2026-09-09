// @vitest-environment happy-dom
import { describe, expect, test } from 'vitest'
import { extractMentionToken, rankMentionCandidates, type MentionUser } from '../src/runtime/mention'

function user(id: number, username: string, nickname?: string): MentionUser {
  return { id, username, nickname: nickname ?? username, avatarUrl: `/a${id}.png` }
}

describe('extractMentionToken（@mention token 识别，issue #564）', () => {
  test('行首 @ 触发，query 为 @ 后到光标前的文本', () => {
    expect(extractMentionToken('@wa')).toEqual({ start: 0, length: 3, query: 'wa' })
  })

  test('空白分隔的 @ 触发', () => {
    expect(extractMentionToken('你好 @wa')).toEqual({ start: 3, length: 3, query: 'wa' })
    expect(extractMentionToken('foo\n@wa')).toEqual({ start: 4, length: 3, query: 'wa' })
  })

  test('CJK 字符前 @ 触发（中文语境直接 @ 昵称）', () => {
    expect(extractMentionToken('感谢@张三')).toEqual({ start: 2, length: 3, query: '张三' })
  })

  test('常见标点分隔的 @ 触发', () => {
    expect(extractMentionToken('（@wa')).toEqual({ start: 1, length: 3, query: 'wa' })
  })

  test('仅输入 @ 时 query 为空（展示本地上下文）', () => {
    expect(extractMentionToken('@')).toEqual({ start: 0, length: 1, query: '' })
    expect(extractMentionToken('你好 @')).toEqual({ start: 3, length: 1, query: '' })
  })

  test('邮箱/标识符内嵌 @ 不触发', () => {
    expect(extractMentionToken('a@b')).toBeNull()
    expect(extractMentionToken('foo@bar')).toBeNull()
    expect(extractMentionToken('mail to a@b.com')).toBeNull()
  })

  test('@ 连写：第二个 @ 前的字符非边界，不触发', () => {
    expect(extractMentionToken('@foo@bar')).toBeNull()
  })

  test('空格/终止标点关闭 token', () => {
    expect(extractMentionToken('@wa ')).toBeNull()
    expect(extractMentionToken('@wa，')).toBeNull()
    expect(extractMentionToken('@wa。')).toBeNull()
    expect(extractMentionToken('@wa,')).toBeNull()
  })

  test('无 @ 不触发', () => {
    expect(extractMentionToken('plain text')).toBeNull()
    expect(extractMentionToken('')).toBeNull()
  })
})

describe('rankMentionCandidates（候选排序/去重/排除自己，issue #564）', () => {
  const local = [
    { ...user(1, 'target', '回复目标'), tag: 'reply-target' as const },
    { ...user(2, 'author', '主题作者'), tag: 'topic-author' as const },
    { ...user(3, 'participant', '参与者'), tag: 'participant' as const },
  ]

  test('空 query：仅本地上下文，最多 5 个，保持 回复目标 > 主题作者 > 参与者 顺序', () => {
    const result = rankMentionCandidates({ local, server: [user(9, 'anyone')], query: '', currentUserId: 0 })
    expect(result.map(u => u.id)).toEqual([1, 2, 3])
  })

  test('空 query 且无本地上下文：返回空（提示继续输入，不查服务端）', () => {
    expect(rankMentionCandidates({ local: [], server: [user(9, 'anyone')], query: '' })).toEqual([])
  })

  test('有 query：本地上下文匹配者优先，再按匹配强度排服务端结果', () => {
    const server = [
      user(20, 'alpha'),
      user(21, 'wavery'),
      user(22, 'xiaowang'),
      user(23, 'wangwu'),
    ]
    const result = rankMentionCandidates({ local, server, query: 'wa', currentUserId: 0 })
    // 本地上下文均不含 "wa" → 全部过滤；username 前缀匹配（wavery/wangwu）排在包含匹配（xiaowang）之前
    expect(result.map(u => u.username)).toEqual(['wavery', 'wangwu', 'xiaowang'])
  })

  test('username exact 优先于 username prefix，再优于 nickname exact/prefix', () => {
    const server = [
      user(30, 'li', '李四'),
      user(31, 'lisi', 'Li Si'),
      user(32, 'll', 'li'),
    ]
    const result = rankMentionCandidates({ local: [], server, query: 'li', currentUserId: 0 })
    // exact username "li"(30) > username prefix "lisi"(31) > nickname exact "li"(32)
    expect(result.map(u => u.id)).toEqual([30, 31, 32])
  })

  test('按 userId 去重：本地与服务端重复的用户只出现一次', () => {
    const server = [user(1, 'target'), user(21, 'newuser')]
    const result = rankMentionCandidates({ local, server, query: 'ta', currentUserId: 0 })
    const ids = result.map(u => u.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  test('当前用户不进入候选', () => {
    const server = [user(1, 'target'), user(2, 'author')]
    const result = rankMentionCandidates({ local, server, query: 'ta', currentUserId: 1 })
    expect(result.map(u => u.id)).not.toContain(1)
  })

  test('有 query 时展示上限 8 个', () => {
    const server = Array.from({ length: 20 }, (_, i) => user(100 + i, `user${i}`))
    const result = rankMentionCandidates({ local: [], server, query: 'user', currentUserId: 0 })
    expect(result).toHaveLength(8)
  })
})
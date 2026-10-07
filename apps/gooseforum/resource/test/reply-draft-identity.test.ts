// @vitest-environment happy-dom
import { beforeEach, expect, test } from 'vitest'
import { readReplyDraft, writeReplyDraft } from '../src/site/utils/reply-draft'
const records = new Map<string, string>()
Object.defineProperty(window, 'localStorage', { configurable: true, value: {
  getItem: (key: string) => records.get(key) ?? null,
  setItem: (key: string, value: string) => records.set(key, value),
  removeItem: (key: string) => records.delete(key),
} })
beforeEach(() => records.clear())
test('reply content and persona choice recover together only for their account and topic', () => {
  const draft = { content: '尚未提交的匿名回复', identity: 'persona' as const, targetPostId: 4 }
  writeReplyDraft(1, 2, draft)
  expect(readReplyDraft(1, 2)).toMatchObject(draft)
  expect(readReplyDraft(3, 2)).toBeNull()
  expect(readReplyDraft(1, 3)).toBeNull()
  writeReplyDraft(1, 2, { ...draft, content: '' })
  expect(readReplyDraft(1, 2)).toBeNull()
})
test('an unknown identity or expired record cannot become a member draft', () => {
  const key = 'gf:reply-draft:v1:1:2'
  records.set(key, JSON.stringify({ content: 'text', identity: 'forged', targetPostId: 0, updatedAt: Date.now() }))
  expect(readReplyDraft(1, 2)).toBeNull()
  expect(records.has(key)).toBe(false)
  records.set(key, JSON.stringify({ content: 'text', identity: 'persona', targetPostId: 0, updatedAt: 1 }))
  expect(readReplyDraft(1, 2)).toBeNull()
})

import { afterEach, describe, expect, test, vi } from 'vitest'

vi.mock('../src/runtime/i18n', () => ({
  i18n: { global: { t: (key: string) => key, te: () => false, locale: { value: 'zh' }, getLocaleMessage: () => ({}) } },
}))

import { ApiResponseError } from '../src/runtime/api'
import { closeModerationBlocked, moderationBlockKind, moderationBlockedState, showModerationBlocked } from '../src/runtime/moderation-blocked'

afterEach(() => closeModerationBlocked())

describe('moderation blocked prompt (issue #975)', () => {
  test('opens for AI policy and external-image blocks with the localized message', () => {
    expect(showModerationBlocked(new ApiResponseError('内容未通过站点发布规则', 'content.aiModeration.blocked'))).toBe(true)
    expect(moderationBlockedState).toMatchObject({ open: true, kind: 'policy', message: '内容未通过站点发布规则' })
    expect(showModerationBlocked(new ApiResponseError('不允许引用站外图片', 'content.aiModeration.externalImageBlocked'))).toBe(true)
    expect(moderationBlockedState.kind).toBe('externalImage')
  })

  test('ignores every other failure so existing inline errors stay unchanged', () => {
    for (const error of [new ApiResponseError('x', 'content.sensitive.blocked'), new ApiResponseError('x', 'common.rateLimited'), new Error('net'), null]) {
      expect(moderationBlockKind(error)).toBeNull()
      expect(showModerationBlocked(error)).toBe(false)
    }
    expect(moderationBlockedState.open).toBe(false)
  })
})

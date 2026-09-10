import { describe, expect, test } from 'vitest'
import { ApiResponseError, sensitiveWordsFromError } from '../src/runtime/api'
import { containsSensitiveText } from '../src/site/utils/sensitive-highlight'

describe('sensitive-word feedback', () => {
  test('keeps every matched word from the response and the legacy word field', () => {
    const error = new ApiResponseError('blocked', 'content.sensitive.blocked', undefined, {
      word: '赌博',
      words: ['赌博', '代考', '赌博'],
    })

    expect(sensitiveWordsFromError(error)).toEqual(['赌博', '代考'])
  })

  test('matches a field when any returned word is present', () => {
    expect(containsSensitiveText('这里是代考内容', ['赌博', '代考'])).toBe(true)
    expect(containsSensitiveText('这里是普通内容', ['赌博', '代考'])).toBe(false)
  })

  test('uses the same width and zero-width normalization as the backend', () => {
    expect(containsSensitiveText('全角ＳＰＡＭＭＥＲ', ['spammer'])).toBe(true)
    expect(containsSensitiveText('赌\u200b博', ['赌博'])).toBe(true)
  })
})

import { describe, expect, test } from 'vitest'
import { resolveActiveHeading } from '../src/site/utils/wiki-toc'

describe('resolveActiveHeading', () => {
  test('highlights the last heading (document order) whose top is at/above the reading line', () => {
    // 复现旧缺陷场景：阅读到"03"（top 80 <= offset 96）时旧实现会因 entries 乱序
    // 把更靠下的"04"（top 220）高亮。文档序 + 顶边判定下"04"位于阅读线之下，不得胜出。
    expect(resolveActiveHeading(
      [
        { id: '01', top: -120 },
        { id: '02', top: -40 },
        { id: '03', top: 80 },
        { id: '04', top: 220 },
      ],
      96,
    )).toBe('03')
  })

  test('a lower heading below the reading line never wins over one above it', () => {
    expect(resolveActiveHeading(
      [
        { id: '03', top: 80 },
        { id: '04', top: 220 },
      ],
      96,
    )).toBe('03')
  })

  test('falls back to the first heading when every heading is below the reading line', () => {
    expect(resolveActiveHeading(
      [
        { id: 'a', top: 120 },
        { id: 'b', top: 240 },
        { id: 'c', top: 400 },
      ],
      96,
    )).toBe('a')
  })

  test('returns an empty string for an empty list', () => {
    expect(resolveActiveHeading([], 96)).toBe('')
  })
})

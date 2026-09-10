import { describe, expect, test } from 'vitest'
import { REVIEW_SQID_ALPHABET, reviewSqid } from '../src/site/utils/course-review-share'

// issue #628 验收：评价短码的字母表与 minLength 必须与 YourTJCourse-Serverless
// 端（sqids@0.3.0，同字母表 + minLength 4）保持一致，同一 review id 三端同码。
// 固定 id → 固定短码的断言锁死编码参数：任何字母表/minLength 改动都会让
// 前台、管理端与 serverless 端编号失配，这里立即红。

const FIXED_EXPECTATIONS: Array<[number, string]> = [
  [1, 'Hy5N'],
  [42, 'hcR6'],
  [1234, 'Q7Pk'],
  [20250910, 'x8nghF'],
]

describe('reviewSqid（评价短码，跨端一致性锁定）', () => {
  test('固定 review id 产出固定短码（字母表 + minLength=4 与 serverless 端一致）', () => {
    for (const [id, sqid] of FIXED_EXPECTATIONS) {
      expect(reviewSqid(id)).toBe(sqid)
    }
  })

  test('短码字符全部来自自定义去易混字母表，且长度 ≥ minLength 4', () => {
    for (let id = 1; id <= 500; id++) {
      const sqid = reviewSqid(id)
      expect(sqid.length).toBeGreaterThanOrEqual(4)
      for (const char of sqid) {
        expect(REVIEW_SQID_ALPHABET).toContain(char)
      }
    }
  })

  test('同一 id 编码稳定，短码在抽样范围内不冲突（单射）', () => {
    const seen = new Map<string, number>()
    for (let id = 1; id <= 1000; id++) {
      const sqid = reviewSqid(id)
      expect(reviewSqid(id)).toBe(sqid)
      const owner = seen.get(sqid)
      expect(owner, `sqid ${sqid} already used by review ${owner}`).toBeUndefined()
      seen.set(sqid, id)
    }
  })
})

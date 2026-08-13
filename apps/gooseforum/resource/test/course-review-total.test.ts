import { describe, expect, test } from 'vitest'
import {
  nextReviewTotalOnCreate,
  nextReviewTotalOnDelete,
  resolveStatsReviewCount,
} from '../src/site/utils/course-review-count'

// issue #178：创建/删除评价后计数口径统一。
// 统计卡（statsReviewCount）与评价区标题（reviewTotal）在写入成功后应立即反映新总数，
// 而非停留在旧值直到刷新。

describe('创建评价成功后计数递增', () => {
  test('总数 +1（与列表 unshift 同步）', () => {
    expect(nextReviewTotalOnCreate(3)).toBe(4)
    expect(nextReviewTotalOnCreate(0)).toBe(1)
  })

  test('统计卡评论数在列表加载完成后以客户端实时计数为准（创建后即 +1）', () => {
    // reviewLoaded=true 时不应再被 SSR 的 reviewCount 遮盖，应展示创建后 +1 的客户端总数。
    expect(resolveStatsReviewCount(true, nextReviewTotalOnCreate(2), 2)).toBe(3)
  })
})

describe('删除评价成功后计数递减', () => {
  test('总数 -1，下限 0（与列表 filter 同步）', () => {
    expect(nextReviewTotalOnDelete(5)).toBe(4)
    expect(nextReviewTotalOnDelete(1)).toBe(0)
    expect(nextReviewTotalOnDelete(0)).toBe(0)
  })
})

describe('统计卡计数回退规则', () => {
  test('列表未加载时回退 SSR 的 reviewCount', () => {
    expect(resolveStatsReviewCount(false, 1, 2)).toBe(2)
  })

  test('列表加载后始终使用客户端实时总数', () => {
    expect(resolveStatsReviewCount(true, 4, 2)).toBe(4)
  })
})
